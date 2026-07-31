package syncer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"cloud-saver/backend/internal/logstore"
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
	current, err := scanManifest(gamePath, ignorePatterns)
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
