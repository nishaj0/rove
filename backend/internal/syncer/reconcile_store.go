package syncer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"cloud-saver/backend/internal/logstore"
	"cloud-saver/backend/internal/models"
)

func publishToStore(ctx context.Context, store RevisionStore, next, base *Manifest, parentID, gamePath, syncID, deviceID string) (*Manifest, Result, error) {
	if next == nil {
		return nil, Result{}, errors.New("missing manifest")
	}
	published := cloneManifest(next)
	published.ID, published.ParentID, published.GameSyncID, published.DeviceID = uuid.NewString(), parentID, syncID, deviceID
	if parentID != "" {
		published.ParentIDs = []string{parentID}
	} else {
		published.ParentIDs = nil
	}
	published.CreatedAt, published.Tombstones = time.Now().UTC(), deletedPaths(base, next)
	result := Result{}
	for path, entry := range published.Files {
		if base != nil {
			if old, ok := base.Files[path]; ok && old.Hash == entry.Hash {
				continue
			}
		}
		source := filepath.Join(gamePath, filepath.FromSlash(path))
		got, _, err := hashFile(source)
		if err != nil {
			return nil, Result{}, err
		}
		if got != entry.Hash {
			return nil, Result{}, errors.New("save changed while it was being copied")
		}
		in, err := os.Open(source)
		if err != nil {
			return nil, Result{}, err
		}
		err = store.WriteObject(ctx, syncID, entry.Hash, in)
		closeErr := in.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			return nil, Result{}, err
		}
		result.ChangedFiles++
		result.Bytes += entry.Size
	}
	result.ChangedFiles += len(published.Tombstones)
	if err := store.WriteManifest(ctx, syncID, published); err != nil {
		return nil, Result{}, err
	}
	if err := store.WriteHead(ctx, syncID, published.ID); err != nil {
		return nil, Result{}, err
	}
	return published, result, nil
}

func applyFromStore(ctx context.Context, store RevisionStore, gamePath string, current, desired *Manifest, syncID, deviceID string) (Result, error) {
	result := Result{}
	for path, local := range current.Files {
		remote, exists := desired.Files[path]
		if exists && remote.Hash == local.Hash {
			continue
		}
		in, err := os.Open(filepath.Join(gamePath, filepath.FromSlash(path)))
		if err != nil {
			return Result{}, fmt.Errorf("open displaced save: %w", err)
		}
		err = store.ArchiveDisplaced(ctx, syncID, deviceID, path, in)
		closeErr := in.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			return Result{}, fmt.Errorf("archive displaced save: %w", err)
		}
	}
	for path, remote := range desired.Files {
		if local, ok := current.Files[path]; ok && local.Hash == remote.Hash {
			continue
		}
		in, err := store.ReadObject(ctx, syncID, remote.Hash)
		if err != nil {
			return Result{}, fmt.Errorf("read remote object: %w", err)
		}
		err = copyReaderAtomicVerified(in, filepath.Join(gamePath, filepath.FromSlash(path)), remote.Hash)
		closeErr := in.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			return Result{}, fmt.Errorf("apply remote save: %w", err)
		}
		result.ChangedFiles++
		result.Bytes += remote.Size
	}
	for path := range current.Files {
		if _, keep := desired.Files[path]; !keep {
			if err := os.Remove(filepath.Join(gamePath, filepath.FromSlash(path))); err != nil && !errors.Is(err, os.ErrNotExist) {
				return Result{}, fmt.Errorf("remove stale save: %w", err)
			}
			result.ChangedFiles++
		}
	}
	return result, nil
}

func copyReaderAtomic(in io.Reader, target string) error {
	return copyReaderAtomicVerified(in, target, "")
}

func copyReaderAtomicVerified(in io.Reader, target, expectedHash string) error {
	tmp := target + ".rove-" + uuid.NewString() + ".tmp"
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if expectedHash != "" {
		actual, _, err := hashFile(tmp)
		if err != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("verify restored object: %w", err)
		}
		if actual != expectedHash {
			_ = os.Remove(tmp)
			return errors.New("restored object failed SHA-256 verification")
		}
	}
	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func localManifestFrom(logs *logstore.Store, gameID string) (*Manifest, error) {
	return localManifest(logs, gameID)
}
func saveLocalManifestTo(logs *logstore.Store, gameID string, value *Manifest) error {
	return saveLocalManifest(logs, gameID, value)
}
func recordConflictsTo(logs *logstore.Store, gameID string, local, remote *Manifest, only []string) (Result, error) {
	return recordConflicts(logs, gameID, local, remote, only)
}

// ResolveConflictWithStore applies the same explicit conflict choice against
// any revision provider; no provider chooses a winner automatically.
func ResolveConflictWithStore(ctx context.Context, store RevisionStore, gamePath, gameID, syncID, deviceID string, ignore []string, logs *logstore.Store, conflictID, action string) error {
	conflict, err := logs.SyncConflict(conflictID)
	if err != nil {
		return err
	}
	if conflict.GameID != gameID || conflict.State != "open" {
		return errors.New("sync conflict is not open for this game")
	}
	if action != "local" && action != "remote" && action != "both" {
		return errors.New("invalid conflict action")
	}
	if !safeRelativePath(conflict.Path) {
		return errors.New("invalid conflict path")
	}
	headID, err := store.ReadHead(ctx, syncID)
	if err != nil {
		return err
	}
	if headID == "" {
		return errors.New("remote revision is unavailable")
	}
	remote, err := store.ReadManifest(ctx, syncID, headID)
	if err != nil {
		return err
	}
	remoteFile, remoteExists := remote.Files[conflict.Path]
	if remoteExists && remoteFile.Hash != conflict.RemoteHash {
		return errors.New("conflict is stale; sync again before resolving")
	}
	if !remoteExists && conflict.RemoteHash != "" {
		return errors.New("conflict is stale; sync again before resolving")
	}
	target := filepath.Join(gamePath, filepath.FromSlash(conflict.Path))
	if action == "remote" && remoteExists {
		in, err := store.ReadObject(ctx, syncID, remoteFile.Hash)
		if err != nil {
			return err
		}
		err = copyReaderAtomicVerified(in, target, remoteFile.Hash)
		closeErr := in.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			return err
		}
	} else if action == "remote" {
		if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if action == "both" && remoteExists {
		ext := filepath.Ext(target)
		copyTarget := target[:len(target)-len(ext)] + ".cloud-conflict-" + remoteFile.Hash[:8] + ext
		in, err := store.ReadObject(ctx, syncID, remoteFile.Hash)
		if err != nil {
			return err
		}
		err = copyReaderAtomicVerified(in, copyTarget, remoteFile.Hash)
		closeErr := in.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			return err
		}
	}
	if err := saveLocalManifestTo(logs, gameID, remote); err != nil {
		return err
	}
	if err := logs.ResolveSyncConflict(conflictID); err != nil {
		return err
	}
	if action == "local" || action == "both" {
		_, err = ReconcileWithStore(ctx, store, gamePath, gameID, syncID, deviceID, ignore, logs)
	}
	return err
}

// Keep encoding/json and models referenced here so provider packages can rely
// on the same marshaling and conflict records without copy-pasting protocol.
var _ = json.Marshal
var _ = models.SyncConflict{}
