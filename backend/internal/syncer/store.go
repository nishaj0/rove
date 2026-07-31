package syncer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"cloud-saver/backend/internal/logstore"
	"cloud-saver/backend/internal/models"
)

// Manifest and ManifestFile are the immutable, provider-independent revision
// format. Providers transport these documents; they do not interpret saves.
type Manifest = manifest
type ManifestFile = manifestFile

// RevisionStore is the only provider-specific boundary in reconciliation.
// Implementations must publish objects and manifests before moving the head.
type RevisionStore interface {
	ReadHead(ctx context.Context, syncID string) (string, error)
	WriteHead(ctx context.Context, syncID, revisionID string) error
	ReadManifest(ctx context.Context, syncID, revisionID string) (*Manifest, error)
	WriteManifest(ctx context.Context, syncID string, m *Manifest) error
	ReadObject(ctx context.Context, syncID, sha256 string) (io.ReadCloser, error)
	WriteObject(ctx context.Context, syncID, sha256 string, r io.Reader) error
	ArchiveDisplaced(ctx context.Context, syncID, deviceID, relPath string, r io.Reader) error
}

// DeviceHeadStore is implemented by modern providers. Independent heads keep
// concurrent device history reachable even while legacy clients use head.json.
type DeviceHeadStore interface {
	WriteDeviceHead(ctx context.Context, syncID, deviceID, revisionID string) error
}

// ProfileStore exposes the provider-side Recovery Library without requiring a
// remembered local game ID.
type ProfileStore interface {
	WriteGameProfile(ctx context.Context, profile models.GameProfile) error
	ListGameProfiles(ctx context.Context) ([]models.GameProfile, error)
}

type RecoveryPointStore interface {
	WriteRecoveryPoint(ctx context.Context, syncID string, point models.RecoveryPoint) error
	DeleteRecoveryPoint(ctx context.Context, syncID, pointID string) error
}

var ErrConcurrentRevision = errors.New("provider head changed on another device")

// ProviderOperationError carries provider-neutral retry instructions across
// the RevisionStore boundary. Implementations may wrap SDK-specific errors,
// but the engine only depends on this stable contract.
type ProviderOperationError struct {
	Code       string
	Retryable  bool
	RetryAfter time.Duration
	Err        error
}

func (e *ProviderOperationError) Error() string {
	if e == nil || e.Err == nil {
		return "provider operation failed"
	}
	return e.Err.Error()
}

func (e *ProviderOperationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// StageRevision copies immutable content and publishes the originating device
// head without moving the shared compatibility head.
func StageRevision(ctx context.Context, source, destination RevisionStore, syncID, revisionID string) (*Manifest, Result, error) {
	if source == nil || destination == nil {
		return nil, Result{}, errors.New("revision source and destination are required")
	}
	value, err := source.ReadManifest(ctx, syncID, revisionID)
	if err != nil {
		return nil, Result{}, err
	}
	result := Result{}
	for _, entry := range value.Files {
		in, err := source.ReadObject(ctx, syncID, entry.Hash)
		if err != nil {
			return nil, Result{}, fmt.Errorf("open staged object %s: %w", entry.Hash, err)
		}
		err = destination.WriteObject(ctx, syncID, entry.Hash, in)
		closeErr := in.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			return nil, Result{}, fmt.Errorf("upload staged object %s: %w", entry.Hash, err)
		}
		result.ChangedFiles++
		result.Bytes += entry.Size
	}
	if err := destination.WriteManifest(ctx, syncID, value); err != nil {
		return nil, Result{}, err
	}
	if modern, ok := destination.(DeviceHeadStore); ok {
		if err := modern.WriteDeviceHead(ctx, syncID, value.DeviceID, revisionID); err != nil {
			return nil, Result{}, err
		}
	}
	return value, result, nil
}

// ReplicateRevision copies one already committed immutable revision to another
// provider. Objects and the manifest are complete before the compatibility
// head is moved, so interrupted uploads never expose a partial revision.
func ReplicateRevision(ctx context.Context, source, destination RevisionStore, syncID, revisionID string) (Result, error) {
	_, result, err := StageRevision(ctx, source, destination, syncID, revisionID)
	if err != nil {
		return Result{}, err
	}
	if err := destination.WriteHead(ctx, syncID, revisionID); err != nil {
		return Result{}, err
	}
	return result, nil
}

// ReplicateRevisionGuarded stages a local revision but moves the compatibility
// head only if it still matches an ancestor of that local revision.
func ReplicateRevisionGuarded(ctx context.Context, source, destination RevisionStore, syncID, revisionID, gameID string, logs *logstore.Store) (Result, error) {
	remoteHead, err := destination.ReadHead(ctx, syncID)
	if err != nil {
		return Result{}, err
	}
	local, result, err := StageRevision(ctx, source, destination, syncID, revisionID)
	if err != nil {
		return Result{}, err
	}
	if remoteHead == "" || remoteHead == revisionID || manifestHasParent(local, remoteHead) {
		if err := destination.WriteHead(ctx, syncID, revisionID); err != nil {
			return Result{}, err
		}
		return result, nil
	}
	remote, err := destination.ReadManifest(ctx, syncID, remoteHead)
	if err != nil {
		return Result{}, err
	}
	if manifestsEqual(local, remote) {
		if err := destination.WriteHead(ctx, syncID, revisionID); err != nil {
			return Result{}, err
		}
		return result, nil
	}
	differences := changedPaths(remote, local)
	paths := make([]string, 0, len(differences))
	for path := range differences {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	if _, err := recordConflictsTo(logs, gameID, local, remote, paths); err != nil {
		return Result{}, err
	}
	return result, ErrConcurrentRevision
}

func manifestHasParent(value *Manifest, revisionID string) bool {
	if value == nil || revisionID == "" {
		return false
	}
	if value.ParentID == revisionID {
		return true
	}
	for _, parent := range value.ParentIDs {
		if parent == revisionID {
			return true
		}
	}
	return false
}

// RestoreRevision exactly materializes an immutable revision, including
// deletion of paths absent from the revision.
func RestoreRevision(ctx context.Context, store RevisionStore, gamePath, syncID, revisionID, deviceID string, ignorePatterns []string) (Result, error) {
	desired, err := store.ReadManifest(ctx, syncID, revisionID)
	if err != nil {
		return Result{}, err
	}
	current, err := scanManifest(gamePath, ignorePatterns)
	if err != nil {
		return Result{}, err
	}
	return applyFromStore(ctx, store, gamePath, current, desired, syncID, deviceID)
}

// FolderStore keeps the original provider-synced-folder behavior intact.
type FolderStore struct{ root string }

func NewFolderStore(syncRoot string) *FolderStore { return &FolderStore{root: syncRoot} }
func (s *FolderStore) gameRoot(syncID string) string {
	return filepath.Join(s.root, protocolDirectory, "games", syncID)
}

func (s *FolderStore) ReadHead(_ context.Context, syncID string) (string, error) {
	payload, err := os.ReadFile(filepath.Join(s.gameRoot(syncID), "head.json"))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read sync head: %w", err)
	}
	var value head
	if err := json.Unmarshal(payload, &value); err != nil || value.RevisionID == "" {
		return "", errors.New("sync head is invalid; waiting for provider to finish copying")
	}
	return value.RevisionID, nil
}
func (s *FolderStore) WriteHead(_ context.Context, syncID, revisionID string) error {
	return writeJSONAtomic(filepath.Join(s.gameRoot(syncID), "head.json"), head{RevisionID: revisionID})
}
func (s *FolderStore) WriteDeviceHead(_ context.Context, syncID, deviceID, revisionID string) error {
	if strings.TrimSpace(deviceID) == "" {
		return errors.New("device id is required")
	}
	return writeJSONAtomic(filepath.Join(s.gameRoot(syncID), "heads", deviceID+".json"), head{RevisionID: revisionID})
}
func (s *FolderStore) WriteGameProfile(_ context.Context, profile models.GameProfile) error {
	if strings.TrimSpace(profile.SyncID) == "" {
		return errors.New("profile sync id is required")
	}
	return writeJSONAtomic(filepath.Join(s.gameRoot(profile.SyncID), "game.json"), profile)
}
func (s *FolderStore) ListGameProfiles(_ context.Context) ([]models.GameProfile, error) {
	root := filepath.Join(s.root, protocolDirectory, "games")
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return []models.GameProfile{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := []models.GameProfile{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		payload, err := os.ReadFile(filepath.Join(root, entry.Name(), "game.json"))
		if err != nil {
			continue
		}
		var profile models.GameProfile
		if json.Unmarshal(payload, &profile) == nil && profile.SyncID != "" {
			result = append(result, profile)
		}
	}
	return result, nil
}
func (s *FolderStore) WriteRecoveryPoint(_ context.Context, syncID string, point models.RecoveryPoint) error {
	return writeJSONAtomic(filepath.Join(s.gameRoot(syncID), "recovery-points", point.ID+".json"), point)
}
func (s *FolderStore) DeleteRecoveryPoint(_ context.Context, syncID, pointID string) error {
	err := os.Remove(filepath.Join(s.gameRoot(syncID), "recovery-points", pointID+".json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
func (s *FolderStore) ReadManifest(_ context.Context, syncID, revisionID string) (*Manifest, error) {
	payload, err := os.ReadFile(filepath.Join(s.gameRoot(syncID), "revisions", revisionID+".json"))
	if err != nil {
		return nil, fmt.Errorf("read sync revision: %w", err)
	}
	var value Manifest
	if err := json.Unmarshal(payload, &value); err != nil || value.ID != revisionID || value.Files == nil {
		return nil, errors.New("sync revision is invalid; waiting for provider to finish copying")
	}
	return &value, nil
}
func (s *FolderStore) WriteManifest(_ context.Context, syncID string, m *Manifest) error {
	return writeJSONAtomic(filepath.Join(s.gameRoot(syncID), "revisions", m.ID+".json"), m)
}
func (s *FolderStore) ReadObject(_ context.Context, syncID, sha string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(s.gameRoot(syncID), "objects", sha))
}
func (s *FolderStore) WriteObject(_ context.Context, syncID, sha string, r io.Reader) error {
	target := filepath.Join(s.gameRoot(syncID), "objects", sha)
	if _, err := os.Stat(target); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	tmp := target + ".tmp-" + uuid.NewString()
	defer os.Remove(tmp)
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err = io.Copy(out, r); err == nil {
		err = out.Close()
	} else {
		_ = out.Close()
	}
	if err != nil {
		return err
	}
	got, _, err := hashFile(tmp)
	if err != nil {
		return err
	}
	if got != sha {
		return errors.New("save changed while it was being copied")
	}
	return os.Rename(tmp, target)
}
func (s *FolderStore) ArchiveDisplaced(_ context.Context, syncID, deviceID, relPath string, r io.Reader) error {
	target := filepath.Join(s.gameRoot(syncID), "versions", deviceID, time.Now().UTC().Format("20060102T150405.000000000Z"), filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, r)
	return err
}

// ReconcileWithStore contains the one conflict/manifest algorithm shared by
// every provider. The store only moves immutable protocol artifacts.
func ReconcileWithStore(ctx context.Context, store RevisionStore, gamePath, gameID, syncID, deviceID string, ignorePatterns []string, logs *logstore.Store) (Result, error) {
	if strings.TrimSpace(syncID) == "" {
		return Result{}, errors.New("game is not paired with a cloud save")
	}
	if strings.TrimSpace(deviceID) == "" {
		return Result{}, errors.New("sync device id is missing")
	}
	current, err := scanManifestCached(ctx, gamePath, ignorePatterns, logs, gameID)
	if err != nil {
		return Result{}, err
	}
	previous, err := localManifestFrom(logs, gameID)
	if err != nil {
		return Result{}, err
	}
	remoteID, err := store.ReadHead(ctx, syncID)
	if err != nil {
		return Result{}, err
	}
	var remote *Manifest
	if remoteID != "" {
		remote, err = store.ReadManifest(ctx, syncID, remoteID)
		if err != nil {
			return Result{}, err
		}
	}
	if remote == nil {
		published, result, err := publishToStore(ctx, store, current, previous, "", gamePath, syncID, deviceID)
		if err != nil {
			return Result{}, err
		}
		return result, saveLocalManifestTo(logs, gameID, published)
	}
	if previous == nil {
		if manifestsEqual(current, remote) {
			return Result{}, saveLocalManifestTo(logs, gameID, remote)
		}
		if len(current.Files) == 0 {
			result, err := applyFromStore(ctx, store, gamePath, current, remote, syncID, deviceID)
			if err != nil {
				return Result{}, err
			}
			return result, saveLocalManifestTo(logs, gameID, remote)
		}
		return recordConflictsTo(logs, gameID, current, remote, nil)
	}
	localChanged, remoteChanged := changedPaths(previous, current), changedPaths(previous, remote)
	if remoteID == previous.ID {
		if len(localChanged) == 0 {
			return Result{}, nil
		}
		published, result, err := publishToStore(ctx, store, current, previous, remote.ID, gamePath, syncID, deviceID)
		if err != nil {
			return Result{}, err
		}
		return result, saveLocalManifestTo(logs, gameID, published)
	}
	if len(localChanged) == 0 {
		result, err := applyFromStore(ctx, store, gamePath, current, remote, syncID, deviceID)
		if err != nil {
			return Result{}, err
		}
		return result, saveLocalManifestTo(logs, gameID, remote)
	}
	if paths := overlappingChanges(localChanged, remoteChanged, current, remote); len(paths) > 0 {
		return recordConflictsTo(logs, gameID, current, remote, paths)
	}
	merged := cloneManifest(remote)
	for path := range localChanged {
		if entry, ok := current.Files[path]; ok {
			merged.Files[path] = entry
		} else {
			delete(merged.Files, path)
		}
	}
	published, result, err := publishToStore(ctx, store, merged, remote, remote.ID, gamePath, syncID, deviceID)
	if err != nil {
		return Result{}, err
	}
	return result, saveLocalManifestTo(logs, gameID, published)
}
