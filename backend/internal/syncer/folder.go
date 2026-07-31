package syncer

// This file implements the portable sync-folder protocol. The sync provider is
// deliberately outside the app: OneDrive, Dropbox and Syncthing all expose a
// normal directory and therefore need no stored credentials here.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"cloud-saver/backend/internal/logstore"
	"cloud-saver/backend/internal/models"
)

const protocolDirectory = ".cloud-saver/v1"

type manifestFile struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

// manifest is immutable once written. Head is only a pointer, so clients never
// consume half-written data from a third-party sync client.
type manifest struct {
	ID         string                  `json:"id"`
	ParentID   string                  `json:"parentId,omitempty"`
	ParentIDs  []string                `json:"parentIds,omitempty"`
	GameSyncID string                  `json:"gameSyncId"`
	DeviceID   string                  `json:"deviceId"`
	CreatedAt  time.Time               `json:"createdAt"`
	Files      map[string]manifestFile `json:"files"`
	Tombstones []string                `json:"tombstones,omitempty"`
}

type head struct {
	RevisionID string `json:"revisionId"`
}

// Result is deliberately transport-neutral and is used by the job engine.
type Result struct {
	ChangedFiles int
	Bytes        int64
	ConflictIDs  []string
}

// ResolveConflict applies one explicit user decision. It never guesses which
// side should win; the caller must choose local, remote, or both.
// ResolveConflict keeps the synced-folder entry point for existing callers.
func ResolveConflict(game models.Game, settings models.Settings, logs *logstore.Store, conflictID, action string) error {
	if strings.TrimSpace(settings.SyncRoot) == "" {
		return errors.New("sync folder is not configured")
	}
	return ResolveConflictWithStore(context.Background(), NewFolderStore(settings.SyncRoot), game.Path, game.ID, game.SyncID, settings.DeviceID, settings.IgnorePatterns, logs, conflictID, action)
}

// Reconcile keeps folder sync as the default while using the provider-neutral
// reconciliation algorithm shared with direct cloud stores.
func Reconcile(game models.Game, settings models.Settings, logs *logstore.Store) (Result, error) {
	if strings.TrimSpace(settings.SyncRoot) == "" {
		return Result{}, errors.New("sync folder is not configured")
	}
	if rel, err := filepath.Rel(game.Path, settings.SyncRoot); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return Result{}, errors.New("sync folder cannot be inside a game save folder")
	}
	return ReconcileWithStore(context.Background(), NewFolderStore(settings.SyncRoot), game.Path, game.ID, game.SyncID, settings.DeviceID, settings.IgnorePatterns, logs)
}

func resolveConflictFolderLegacy(game models.Game, settings models.Settings, logs *logstore.Store, conflictID, action string) error {
	conflict, err := logs.SyncConflict(conflictID)
	if err != nil {
		return err
	}
	if conflict.GameID != game.ID || conflict.State != "open" {
		return errors.New("sync conflict is not open for this game")
	}
	if action != "local" && action != "remote" && action != "both" {
		return errors.New("invalid conflict action")
	}
	if !safeRelativePath(conflict.Path) {
		return errors.New("invalid conflict path")
	}
	root := filepath.Join(settings.SyncRoot, protocolDirectory, "games", game.SyncID)
	remote, _, err := loadHead(root)
	if err != nil {
		return err
	}
	if remote == nil {
		return errors.New("remote revision is unavailable")
	}
	remoteFile, remoteExists := remote.Files[conflict.Path]
	if remoteExists && remoteFile.Hash != conflict.RemoteHash {
		return errors.New("conflict is stale; sync again before resolving")
	}
	if !remoteExists && conflict.RemoteHash != "" {
		return errors.New("conflict is stale; sync again before resolving")
	}
	target := filepath.Join(game.Path, filepath.FromSlash(conflict.Path))
	if action == "remote" {
		if remoteExists {
			if err := copyFileAtomic(filepath.Join(root, "objects", remoteFile.Hash), target); err != nil {
				return err
			}
		} else if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if action == "both" && remoteExists {
		ext := filepath.Ext(target)
		base := strings.TrimSuffix(target, ext)
		target = base + ".cloud-conflict-" + remoteFile.Hash[:8] + ext
		if err := copyFileAtomic(filepath.Join(root, "objects", remoteFile.Hash), target); err != nil {
			return err
		}
	}
	if err := saveLocalManifest(logs, game.ID, remote); err != nil {
		return err
	}
	if err := logs.ResolveSyncConflict(conflictID); err != nil {
		return err
	}
	if action == "local" || action == "both" {
		_, err = Reconcile(game, settings, logs)
	}
	return err
}

// Reconcile performs one safe two-way reconciliation for a game.
func reconcileFolderLegacy(game models.Game, settings models.Settings, logs *logstore.Store) (Result, error) {
	if strings.TrimSpace(settings.SyncRoot) == "" {
		return Result{}, errors.New("sync folder is not configured")
	}
	if strings.TrimSpace(game.SyncID) == "" {
		return Result{}, errors.New("game is not paired with a cloud save")
	}
	if strings.TrimSpace(settings.DeviceID) == "" {
		return Result{}, errors.New("sync device id is missing")
	}
	if rel, err := filepath.Rel(game.Path, settings.SyncRoot); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return Result{}, errors.New("sync folder cannot be inside a game save folder")
	}

	root := filepath.Join(settings.SyncRoot, protocolDirectory, "games", game.SyncID)
	if err := os.MkdirAll(filepath.Join(root, "objects"), 0o755); err != nil {
		return Result{}, fmt.Errorf("create sync store: %w", err)
	}
	current, err := scanManifest(game.Path, settings.IgnorePatterns)
	if err != nil {
		return Result{}, err
	}

	previous, err := localManifest(logs, game.ID)
	if err != nil {
		return Result{}, err
	}
	remote, remoteID, err := loadHead(root)
	if err != nil {
		return Result{}, err
	}

	if remote == nil {
		published, result, err := publish(root, current, previous, "", game, settings)
		if err != nil {
			return Result{}, err
		}
		return result, saveLocalManifest(logs, game.ID, published)
	}

	if previous == nil {
		if manifestsEqual(current, remote) {
			return Result{}, saveLocalManifest(logs, game.ID, remote)
		}
		if len(current.Files) == 0 {
			result, err := applyManifest(root, game.Path, current, remote, game, settings)
			if err != nil {
				return Result{}, err
			}
			return result, saveLocalManifest(logs, game.ID, remote)
		}
		return recordConflicts(logs, game.ID, current, remote, nil)
	}

	localChanged := changedPaths(previous, current)
	remoteChanged := changedPaths(previous, remote)
	if remoteID == previous.ID {
		if len(localChanged) == 0 {
			return Result{}, nil
		}
		published, result, err := publish(root, current, previous, remote.ID, game, settings)
		if err != nil {
			return Result{}, err
		}
		return result, saveLocalManifest(logs, game.ID, published)
	}

	if len(localChanged) == 0 {
		result, err := applyManifest(root, game.Path, current, remote, game, settings)
		if err != nil {
			return Result{}, err
		}
		return result, saveLocalManifest(logs, game.ID, remote)
	}

	conflictPaths := overlappingChanges(localChanged, remoteChanged, current, remote)
	if len(conflictPaths) > 0 {
		return recordConflicts(logs, game.ID, current, remote, conflictPaths)
	}

	// Independent edits are safe to merge. Publish the merged revision rather
	// than mutating the remote state in place, so a later client can reconcile it.
	merged := cloneManifest(remote)
	for path := range localChanged {
		if entry, ok := current.Files[path]; ok {
			merged.Files[path] = entry
		} else {
			delete(merged.Files, path)
		}
	}
	published, result, err := publish(root, merged, remote, remote.ID, game, settings)
	if err != nil {
		return Result{}, err
	}
	return result, saveLocalManifest(logs, game.ID, published)
}

func scanManifest(root string, ignorePatterns []string) (*manifest, error) {
	m := &manifest{Files: make(map[string]manifestFile)}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return nil
		}
		if shouldIgnore(path, ignorePatterns) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if !safeRelativePath(filepath.ToSlash(rel)) {
			return errors.New("unsafe save path")
		}
		hash, size, err := hashFile(path)
		if err != nil {
			return err
		}
		m.Files[filepath.ToSlash(rel)] = manifestFile{Hash: hash, Size: size}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan save folder: %w", err)
	}
	return m, nil
}

// scanManifestCached walks metadata but only opens files whose stable metadata
// changed or which the watcher marked dirty. All candidates share one stability
// delay, avoiding a one-second delay per file.
func scanManifestCached(ctx context.Context, root string, ignorePatterns []string, logs *logstore.Store, gameID string) (*manifest, error) {
	if logs == nil || strings.TrimSpace(gameID) == "" {
		return scanManifest(root, ignorePatterns)
	}
	cache, err := logs.FileStates(gameID)
	if err != nil {
		return nil, err
	}
	type candidate struct {
		path string
		rel  string
		info fs.FileInfo
	}
	result := &manifest{Files: map[string]manifestFile{}}
	candidates := []candidate{}
	seen := map[string]struct{}{}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && shouldIgnore(path, ignorePatterns) {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() || shouldIgnore(path, ignorePatterns) {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil || !safeRelativePath(filepath.ToSlash(rel)) {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		seen[rel] = struct{}{}
		cached, ok := cache[rel]
		if ok && !cached.Dirty && cached.SHA256 != "" && cached.Size == info.Size() && cached.ModifiedNS == info.ModTime().UnixNano() {
			result.Files[rel] = manifestFile{Hash: cached.SHA256, Size: cached.Size}
			return nil
		}
		candidates = append(candidates, candidate{path: path, rel: rel, info: info})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(candidates) > 0 {
		time.Sleep(time.Second)
	}
	for _, item := range candidates {
		stable, statErr := os.Stat(item.path)
		if statErr != nil {
			if errors.Is(statErr, os.ErrNotExist) {
				_ = logs.DeleteFileState(gameID, item.rel)
				continue
			}
			return nil, statErr
		}
		if stable.Size() != item.info.Size() || stable.ModTime().UnixNano() != item.info.ModTime().UnixNano() {
			return nil, fmt.Errorf("save file %q is still changing; protection will retry", item.rel)
		}
		hash, size, hashErr := hashFileWithContext(ctx, item.path)
		if hashErr != nil {
			return nil, hashErr
		}
		after, statErr := os.Stat(item.path)
		if statErr != nil || after.Size() != stable.Size() || after.ModTime().UnixNano() != stable.ModTime().UnixNano() {
			return nil, fmt.Errorf("save file %q changed during capture; protection will retry", item.rel)
		}
		result.Files[item.rel] = manifestFile{Hash: hash, Size: size}
		if err := logs.UpsertFileState(logstore.FileState{
			GameID:       gameID,
			RelativePath: item.rel,
			Size:         size,
			ModifiedNS:   after.ModTime().UnixNano(),
			SHA256:       hash,
			Dirty:        false,
		}); err != nil {
			return nil, err
		}
	}
	for rel := range cache {
		if _, ok := seen[rel]; !ok {
			if err := logs.DeleteFileState(gameID, rel); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

type captureLimitKey struct{}

// WithCaptureLimit applies a best-effort read ceiling to changed-file hashing.
func WithCaptureLimit(ctx context.Context, bytesPerSecond int64) context.Context {
	return context.WithValue(ctx, captureLimitKey{}, bytesPerSecond)
}

func hashFileWithContext(ctx context.Context, path string) (string, int64, error) {
	limit, _ := ctx.Value(captureLimitKey{}).(int64)
	if limit <= 0 {
		return hashFile(path)
	}
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	hash := sha256.New()
	buffer := make([]byte, 64*1024)
	started := time.Now()
	var total int64
	for {
		count, readErr := file.Read(buffer)
		if count > 0 {
			if _, err := hash.Write(buffer[:count]); err != nil {
				return "", 0, err
			}
			total += int64(count)
			expected := time.Duration(float64(total) / float64(limit) * float64(time.Second))
			if delay := expected - time.Since(started); delay > 0 {
				timer := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return "", 0, ctx.Err()
				case <-timer.C:
				}
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return "", 0, readErr
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), total, nil
}

func shouldIgnore(path string, patterns []string) bool {
	name := filepath.Base(path)
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if strings.EqualFold(name, pattern) {
			return true
		}
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
		needle := strings.ToLower(string(os.PathSeparator) + pattern + string(os.PathSeparator))
		if strings.Contains(strings.ToLower(path), needle) {
			return true
		}
	}
	return false
}

func safeRelativePath(path string) bool {
	clean := filepath.Clean(filepath.FromSlash(path))
	return clean != "." && !filepath.IsAbs(clean) && clean != ".." && !strings.HasPrefix(clean, ".."+string(os.PathSeparator))
}

func publish(root string, next, base *manifest, parentID string, game models.Game, settings models.Settings) (*manifest, Result, error) {
	if next == nil {
		return nil, Result{}, errors.New("missing manifest")
	}
	published := cloneManifest(next)
	published.ID = uuid.NewString()
	published.ParentID = parentID
	published.GameSyncID = game.SyncID
	published.DeviceID = settings.DeviceID
	published.CreatedAt = time.Now().UTC()
	published.Tombstones = deletedPaths(base, next)

	result := Result{}
	for path, entry := range published.Files {
		if base != nil {
			if old, ok := base.Files[path]; ok && old.Hash == entry.Hash {
				continue
			}
		}
		source := filepath.Join(game.Path, filepath.FromSlash(path))
		written, err := writeObject(filepath.Join(root, "objects", entry.Hash), source, entry.Hash)
		if err != nil {
			return nil, Result{}, err
		}
		result.ChangedFiles++
		result.Bytes += written
	}
	for range published.Tombstones {
		result.ChangedFiles++
	}
	if err := writeJSONAtomic(filepath.Join(root, "revisions", published.ID+".json"), published); err != nil {
		return nil, Result{}, err
	}
	if err := writeJSONAtomic(filepath.Join(root, "head.json"), head{RevisionID: published.ID}); err != nil {
		return nil, Result{}, err
	}
	return published, result, nil
}

func loadHead(root string) (*manifest, string, error) {
	payload, err := os.ReadFile(filepath.Join(root, "head.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("read sync head: %w", err)
	}
	var pointer head
	if err := json.Unmarshal(payload, &pointer); err != nil || pointer.RevisionID == "" {
		return nil, "", errors.New("sync head is invalid; waiting for provider to finish copying")
	}
	manifestPayload, err := os.ReadFile(filepath.Join(root, "revisions", pointer.RevisionID+".json"))
	if err != nil {
		return nil, "", fmt.Errorf("read sync revision: %w", err)
	}
	var remote manifest
	if err := json.Unmarshal(manifestPayload, &remote); err != nil || remote.ID != pointer.RevisionID || remote.Files == nil {
		return nil, "", errors.New("sync revision is invalid; waiting for provider to finish copying")
	}
	return &remote, pointer.RevisionID, nil
}

func localManifest(logs *logstore.Store, gameID string) (*manifest, error) {
	state, err := logs.SyncState(gameID)
	if err != nil || state.ManifestJSON == "" {
		return nil, err
	}
	var value manifest
	if err := json.Unmarshal([]byte(state.ManifestJSON), &value); err != nil {
		return nil, fmt.Errorf("decode local sync state: %w", err)
	}
	return &value, nil
}

func saveLocalManifest(logs *logstore.Store, gameID string, value *manifest) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return logs.SaveSyncState(gameID, value.ID, string(payload))
}

func applyManifest(root, gameRoot string, current, desired *manifest, game models.Game, settings models.Settings) (Result, error) {
	result := Result{}
	archiveRoot := filepath.Join(root, "versions", settings.DeviceID, game.SyncID, time.Now().UTC().Format("20060102T150405.000000000Z"))
	for path, local := range current.Files {
		remote, exists := desired.Files[path]
		if exists && remote.Hash == local.Hash {
			continue
		}
		if _, err := copyFile(filepath.Join(gameRoot, filepath.FromSlash(path)), filepath.Join(archiveRoot, filepath.FromSlash(path))); err != nil {
			return Result{}, fmt.Errorf("archive displaced save: %w", err)
		}
	}
	for path, remote := range desired.Files {
		if local, ok := current.Files[path]; ok && local.Hash == remote.Hash {
			continue
		}
		source := filepath.Join(root, "objects", remote.Hash)
		target := filepath.Join(gameRoot, filepath.FromSlash(path))
		if err := copyFileAtomic(source, target); err != nil {
			return Result{}, fmt.Errorf("apply remote save: %w", err)
		}
		result.ChangedFiles++
		result.Bytes += remote.Size
	}
	for path := range current.Files {
		if _, keep := desired.Files[path]; keep {
			continue
		}
		if err := os.Remove(filepath.Join(gameRoot, filepath.FromSlash(path))); err != nil && !errors.Is(err, os.ErrNotExist) {
			return Result{}, fmt.Errorf("remove stale save: %w", err)
		}
		result.ChangedFiles++
	}
	return result, nil
}

func recordConflicts(logs *logstore.Store, gameID string, local, remote *manifest, only []string) (Result, error) {
	paths := only
	if len(paths) == 0 {
		paths = unionPaths(local, remote)
	}
	result := Result{}
	for _, path := range paths {
		id := uuid.NewString()
		localHash := local.Files[path].Hash
		remoteHash := remote.Files[path].Hash
		if err := logs.WriteConflict(gameID, path, "review_required", fmt.Sprintf("sync conflict %s local=%s remote=%s", id, localHash, remoteHash)); err != nil {
			return Result{}, err
		}
		if err := logs.CreateSyncConflict(models.SyncConflict{ID: id, GameID: gameID, Path: path, LocalHash: localHash, RemoteHash: remoteHash, CreatedAt: time.Now().UTC(), State: "open"}); err != nil {
			return Result{}, err
		}
		result.ConflictIDs = append(result.ConflictIDs, id)
	}
	return result, nil
}

func changedPaths(base, next *manifest) map[string]struct{} {
	changed := make(map[string]struct{})
	if base == nil {
		for path := range next.Files {
			changed[path] = struct{}{}
		}
		return changed
	}
	for path, entry := range next.Files {
		if prior, ok := base.Files[path]; !ok || prior.Hash != entry.Hash {
			changed[path] = struct{}{}
		}
	}
	for path := range base.Files {
		if _, ok := next.Files[path]; !ok {
			changed[path] = struct{}{}
		}
	}
	return changed
}

func overlappingChanges(a, b map[string]struct{}, local, remote *manifest) []string {
	paths := make([]string, 0)
	for path := range a {
		if _, ok := b[path]; !ok {
			continue
		}
		if local.Files[path].Hash != remote.Files[path].Hash {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths
}

func deletedPaths(base, next *manifest) []string {
	if base == nil {
		return nil
	}
	paths := make([]string, 0)
	for path := range base.Files {
		if _, ok := next.Files[path]; !ok {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths
}

func unionPaths(a, b *manifest) []string {
	set := changedPaths(a, b)
	for path := range changedPaths(b, a) {
		set[path] = struct{}{}
	}
	paths := make([]string, 0, len(set))
	for path := range set {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func manifestsEqual(a, b *manifest) bool {
	return len(changedPaths(a, b)) == 0 && len(changedPaths(b, a)) == 0
}
func cloneManifest(value *manifest) *manifest {
	copy := &manifest{Files: make(map[string]manifestFile, len(value.Files))}
	*copy = *value
	copy.ParentIDs = append([]string{}, value.ParentIDs...)
	for path, entry := range value.Files {
		copy.Files[path] = entry
	}
	copy.Tombstones = append([]string{}, value.Tombstones...)
	return copy
}

func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	size, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), size, nil
}

func writeObject(target, source, expectedHash string) (int64, error) {
	if _, err := os.Stat(target); err == nil {
		return 0, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return 0, err
	}
	tmp := target + ".tmp-" + uuid.NewString()
	bytes, err := copyFile(source, tmp)
	if err != nil {
		_ = os.Remove(tmp)
		return 0, err
	}
	hash, _, err := hashFile(tmp)
	if err != nil || hash != expectedHash {
		_ = os.Remove(tmp)
		return 0, errors.New("save changed while it was being copied")
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return 0, err
	}
	return bytes, nil
}

func copyFile(source, target string) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return 0, err
	}
	in, err := os.Open(source)
	if err != nil {
		return 0, err
	}
	defer in.Close()
	out, err := os.Create(target)
	if err != nil {
		return 0, err
	}
	n, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return 0, copyErr
	}
	return n, closeErr
}

func copyFileAtomic(source, target string) error {
	tmp := target + ".cloud-saver-" + uuid.NewString() + ".tmp"
	if _, err := copyFile(source, tmp); err != nil {
		return err
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

func writeJSONAtomic(path string, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp-" + uuid.NewString()
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
