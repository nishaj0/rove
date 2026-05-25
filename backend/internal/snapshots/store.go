package snapshots

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"cloud-saver/backend/internal/models"
)

const (
	indexFilename = "index.json"
)

var invalidSnapshotNameChars = regexp.MustCompile(`[<>:"/\\|?*]+`)

type fingerprint struct {
	Size            int64 `json:"size"`
	ModTimeUnixNano int64 `json:"modTimeUnixNano"`
}

type snapshotFile struct {
	Path            string `json:"path"`
	Size            int64  `json:"size"`
	ModTimeUnixNano int64  `json:"modTimeUnixNano"`
}

type snapshotMetadata struct {
	SnapshotID   string         `json:"snapshotId"`
	Seq          int            `json:"seq"`
	CreatedAt    time.Time      `json:"createdAt"`
	Trigger      string         `json:"trigger"`
	CopiedFiles  []snapshotFile `json:"copiedFiles"`
	DeletedPaths []string       `json:"deletedPaths"`
	TotalSize    int64          `json:"totalSize"`
}

type indexSnapshot struct {
	ID           string    `json:"id"`
	Seq          int       `json:"seq"`
	CreatedAt    time.Time `json:"createdAt"`
	Trigger      string    `json:"trigger"`
	FolderName   string    `json:"folderName"`
	MetadataFile string    `json:"metadataFile"`
	ChangedFiles int       `json:"changedFiles"`
	TotalSize    int64     `json:"totalSize"`
}

type gameIndex struct {
	GameID      string                 `json:"gameId"`
	GameName    string                 `json:"gameName"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	LatestState map[string]fingerprint `json:"latestState"`
	Snapshots   []indexSnapshot        `json:"snapshots"`
}

type CreateResult struct {
	Status   string
	Snapshot *models.SnapshotSummary
}

type stateCapture struct {
	entry   indexSnapshot
	state   map[string]fingerprint
	origins map[string]string
}

func listSnapshotsForGame(game models.Game, settings models.Settings) ([]models.SnapshotSummary, error) {
	if strings.TrimSpace(settings.SnapshotVaultPath) == "" {
		return []models.SnapshotSummary{}, nil
	}

	index, err := loadGameIndex(snapshotGameDir(settings.SnapshotVaultPath, game))
	if err != nil {
		return nil, err
	}

	result := make([]models.SnapshotSummary, 0, len(index.Snapshots))
	for i := len(index.Snapshots) - 1; i >= 0; i-- {
		entry := index.Snapshots[i]
		result = append(result, toSummary(entry))
	}
	return result, nil
}

func createSnapshotForGame(game models.Game, settings models.Settings, trigger string) (CreateResult, error) {
	vaultRoot := strings.TrimSpace(settings.SnapshotVaultPath)
	if vaultRoot == "" {
		return CreateResult{Status: "skipped"}, nil
	}
	if err := os.MkdirAll(vaultRoot, 0o755); err != nil {
		return CreateResult{}, fmt.Errorf("create snapshot vault: %w", err)
	}

	gameDir := snapshotGameDir(vaultRoot, game)
	if err := os.MkdirAll(gameDir, 0o755); err != nil {
		return CreateResult{}, fmt.Errorf("create game snapshot dir: %w", err)
	}

	index, err := loadGameIndex(gameDir)
	if err != nil {
		return CreateResult{}, err
	}
	index.GameID = game.ID
	index.GameName = game.Name

	currentState, err := scanLiveState(game.Path, settings.IgnorePatterns)
	if err != nil {
		return CreateResult{}, err
	}

	changedPaths, deletedPaths := diffState(index.LatestState, currentState)
	shouldPersistEmpty := trigger == models.SnapshotTriggerManual || trigger == models.SnapshotTriggerExit || trigger == models.SnapshotTriggerPreRestore
	if len(changedPaths) == 0 && len(deletedPaths) == 0 && !shouldPersistEmpty {
		return CreateResult{Status: "skipped"}, nil
	}

	now := time.Now().UTC()
	seq := len(index.Snapshots) + 1
	folderName := snapshotFolderName(seq, now)
	metadataFile := snapshotMetadataFilename(folderName)
	snapshotDir := filepath.Join(gameDir, folderName)
	if err := os.MkdirAll(snapshotDir, 0o755); err != nil {
		return CreateResult{}, fmt.Errorf("create snapshot dir: %w", err)
	}

	copiedFiles := make([]snapshotFile, 0, len(changedPaths))
	var totalSize int64
	for _, relPath := range changedPaths {
		fp := currentState[relPath]
		src := filepath.Join(game.Path, filepath.FromSlash(relPath))
		dst := filepath.Join(snapshotDir, filepath.FromSlash(relPath))
		if err := copyFileWithModTime(src, dst, time.Unix(0, fp.ModTimeUnixNano)); err != nil {
			return CreateResult{}, err
		}
		copiedFiles = append(copiedFiles, snapshotFile{
			Path:            relPath,
			Size:            fp.Size,
			ModTimeUnixNano: fp.ModTimeUnixNano,
		})
		totalSize += fp.Size
	}

	meta := snapshotMetadata{
		SnapshotID:   uuid.NewString(),
		Seq:          seq,
		CreatedAt:    now,
		Trigger:      trigger,
		CopiedFiles:  copiedFiles,
		DeletedPaths: append([]string{}, deletedPaths...),
		TotalSize:    totalSize,
	}
	if err := writeJSONAtomic(filepath.Join(gameDir, metadataFile), meta); err != nil {
		return CreateResult{}, err
	}

	entry := indexSnapshot{
		ID:           meta.SnapshotID,
		Seq:          seq,
		CreatedAt:    now,
		Trigger:      trigger,
		FolderName:   folderName,
		MetadataFile: metadataFile,
		ChangedFiles: len(copiedFiles),
		TotalSize:    totalSize,
	}
	index.Snapshots = append(index.Snapshots, entry)
	index.LatestState = cloneFingerprintMap(currentState)
	index.UpdatedAt = now

	keepFrom := 0
	if len(index.Snapshots) > settings.SnapshotMaxPerGame {
		keepFrom = len(index.Snapshots) - settings.SnapshotMaxPerGame
	}
	if keepFrom > 0 {
		rebuilt, err := rewriteGameSnapshots(gameDir, game, index, index.Snapshots[keepFrom:])
		if err != nil {
			return CreateResult{}, err
		}
		index = rebuilt
	} else if err := writeJSONAtomic(filepath.Join(gameDir, indexFilename), index); err != nil {
		return CreateResult{}, err
	}

	latest := index.Snapshots[len(index.Snapshots)-1]
	return CreateResult{
		Status:   "created",
		Snapshot: ptr(toSummary(latest)),
	}, nil
}

func restoreSnapshotForGame(game models.Game, settings models.Settings, snapshotID string) error {
	vaultRoot := strings.TrimSpace(settings.SnapshotVaultPath)
	if vaultRoot == "" {
		return errors.New("snapshot vault path is not configured")
	}

	gameDir := snapshotGameDir(vaultRoot, game)
	index, err := loadGameIndex(gameDir)
	if err != nil {
		return err
	}
	if _, ok := findSnapshot(index.Snapshots, snapshotID); !ok {
		return errors.New("snapshot not found")
	}

	restoreSourceRoot, err := os.MkdirTemp("", "cloud-saver-restore-source-*")
	if err != nil {
		return fmt.Errorf("create restore temp dir: %w", err)
	}
	defer os.RemoveAll(restoreSourceRoot)

	if err := materializeSnapshotState(gameDir, index.Snapshots, snapshotID, restoreSourceRoot); err != nil {
		return err
	}

	preRestore, err := createSnapshotForGame(game, settings, models.SnapshotTriggerPreRestore)
	if err != nil {
		return err
	}
	if preRestore.Status != "created" {
		return errors.New("failed to create pre-restore snapshot")
	}

	return copyDirectoryContents(restoreSourceRoot, game.Path)
}

func deleteSnapshotForGame(game models.Game, settings models.Settings, snapshotID string) error {
	vaultRoot := strings.TrimSpace(settings.SnapshotVaultPath)
	if vaultRoot == "" {
		return errors.New("snapshot vault path is not configured")
	}

	gameDir := snapshotGameDir(vaultRoot, game)
	index, err := loadGameIndex(gameDir)
	if err != nil {
		return err
	}

	kept := make([]indexSnapshot, 0, len(index.Snapshots))
	found := false
	for _, entry := range index.Snapshots {
		if entry.ID == snapshotID {
			found = true
			continue
		}
		kept = append(kept, entry)
	}
	if !found {
		return errors.New("snapshot not found")
	}

	_, err = rewriteGameSnapshots(gameDir, game, index, kept)
	return err
}

func rewriteGameSnapshots(gameDir string, game models.Game, original gameIndex, kept []indexSnapshot) (gameIndex, error) {
	captures, err := collectStateCaptures(gameDir, original, kept)
	if err != nil {
		return gameIndex{}, err
	}

	parent := filepath.Dir(gameDir)
	tempDir := filepath.Join(parent, filepath.Base(gameDir)+".rewrite-"+uuid.NewString())
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return gameIndex{}, fmt.Errorf("create rewrite temp dir: %w", err)
	}

	newIndex := gameIndex{
		GameID:      game.ID,
		GameName:    game.Name,
		UpdatedAt:   time.Now().UTC(),
		LatestState: map[string]fingerprint{},
		Snapshots:   []indexSnapshot{},
	}

	prevState := map[string]fingerprint{}
	for i, capture := range captures {
		seq := i + 1
		folderName := snapshotFolderName(seq, capture.entry.CreatedAt)
		metadataFile := snapshotMetadataFilename(folderName)
		snapshotDir := filepath.Join(tempDir, folderName)
		if err := os.MkdirAll(snapshotDir, 0o755); err != nil {
			return gameIndex{}, fmt.Errorf("create rewritten snapshot dir: %w", err)
		}

		changedPaths, deletedPaths := diffState(prevState, capture.state)
		copiedFiles := make([]snapshotFile, 0, len(changedPaths))
		var totalSize int64
		for _, relPath := range changedPaths {
			origin := capture.origins[relPath]
			fp := capture.state[relPath]
			dst := filepath.Join(snapshotDir, filepath.FromSlash(relPath))
			if err := copyFileWithModTime(origin, dst, time.Unix(0, fp.ModTimeUnixNano)); err != nil {
				return gameIndex{}, err
			}
			copiedFiles = append(copiedFiles, snapshotFile{
				Path:            relPath,
				Size:            fp.Size,
				ModTimeUnixNano: fp.ModTimeUnixNano,
			})
			totalSize += fp.Size
		}

		meta := snapshotMetadata{
			SnapshotID:   capture.entry.ID,
			Seq:          seq,
			CreatedAt:    capture.entry.CreatedAt,
			Trigger:      capture.entry.Trigger,
			CopiedFiles:  copiedFiles,
			DeletedPaths: deletedPaths,
			TotalSize:    totalSize,
		}
		if err := writeJSONAtomic(filepath.Join(tempDir, metadataFile), meta); err != nil {
			return gameIndex{}, err
		}

		newIndex.Snapshots = append(newIndex.Snapshots, indexSnapshot{
			ID:           capture.entry.ID,
			Seq:          seq,
			CreatedAt:    capture.entry.CreatedAt,
			Trigger:      capture.entry.Trigger,
			FolderName:   folderName,
			MetadataFile: metadataFile,
			ChangedFiles: len(copiedFiles),
			TotalSize:    totalSize,
		})
		prevState = cloneFingerprintMap(capture.state)
	}

	newIndex.LatestState = cloneFingerprintMap(prevState)
	if err := writeJSONAtomic(filepath.Join(tempDir, indexFilename), newIndex); err != nil {
		return gameIndex{}, err
	}

	if err := os.RemoveAll(gameDir); err != nil && !os.IsNotExist(err) {
		return gameIndex{}, fmt.Errorf("remove previous snapshot dir: %w", err)
	}
	if err := os.Rename(tempDir, gameDir); err != nil {
		return gameIndex{}, fmt.Errorf("swap rewritten snapshot dir: %w", err)
	}
	return newIndex, nil
}

func collectStateCaptures(gameDir string, original gameIndex, kept []indexSnapshot) ([]stateCapture, error) {
	keepSet := make(map[string]struct{}, len(kept))
	for _, entry := range kept {
		keepSet[entry.ID] = struct{}{}
	}

	currentState := map[string]fingerprint{}
	currentOrigins := map[string]string{}
	captures := make([]stateCapture, 0, len(kept))
	for _, entry := range original.Snapshots {
		meta, err := loadSnapshotMetadata(filepath.Join(gameDir, entry.MetadataFile))
		if err != nil {
			return nil, err
		}
		for _, relPath := range meta.DeletedPaths {
			delete(currentState, relPath)
			delete(currentOrigins, relPath)
		}
		for _, copied := range meta.CopiedFiles {
			currentState[copied.Path] = fingerprint{
				Size:            copied.Size,
				ModTimeUnixNano: copied.ModTimeUnixNano,
			}
			currentOrigins[copied.Path] = filepath.Join(gameDir, entry.FolderName, filepath.FromSlash(copied.Path))
		}
		if _, ok := keepSet[entry.ID]; ok {
			captures = append(captures, stateCapture{
				entry:   entry,
				state:   cloneFingerprintMap(currentState),
				origins: cloneStringMap(currentOrigins),
			})
		}
	}
	return captures, nil
}

func materializeSnapshotState(gameDir string, entries []indexSnapshot, targetID string, outputDir string) error {
	for _, entry := range entries {
		meta, err := loadSnapshotMetadata(filepath.Join(gameDir, entry.MetadataFile))
		if err != nil {
			return err
		}
		for _, relPath := range meta.DeletedPaths {
			_ = os.Remove(filepath.Join(outputDir, filepath.FromSlash(relPath)))
		}
		for _, copied := range meta.CopiedFiles {
			src := filepath.Join(gameDir, entry.FolderName, filepath.FromSlash(copied.Path))
			dst := filepath.Join(outputDir, filepath.FromSlash(copied.Path))
			if err := copyFileWithModTime(src, dst, time.Unix(0, copied.ModTimeUnixNano)); err != nil {
				return err
			}
		}
		if entry.ID == targetID {
			return nil
		}
	}
	return errors.New("snapshot not found")
}

func copyDirectoryContents(srcRoot, dstRoot string) error {
	return filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFileWithModTime(path, filepath.Join(dstRoot, rel), info.ModTime())
	})
}

func copyFileWithModTime(src, dst string, modTime time.Time) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create parent dir for %s: %w", dst, err)
	}

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create target file %s: %w", dst, err)
	}

	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return fmt.Errorf("copy file %s: %w", src, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close target file %s: %w", dst, closeErr)
	}
	if err := os.Chtimes(dst, modTime, modTime); err != nil {
		return fmt.Errorf("preserve mod time for %s: %w", dst, err)
	}
	return nil
}

func scanLiveState(root string, ignorePatterns []string) (map[string]fingerprint, error) {
	state := make(map[string]fingerprint)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if shouldIgnore(path, ignorePatterns) {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldIgnore(path, ignorePatterns) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		state[filepath.ToSlash(rel)] = fingerprint{
			Size:            info.Size(),
			ModTimeUnixNano: info.ModTime().UnixNano(),
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan live state: %w", err)
	}
	return state, nil
}

func diffState(previous, current map[string]fingerprint) ([]string, []string) {
	changed := make([]string, 0)
	deleted := make([]string, 0)

	for relPath, currentFP := range current {
		prevFP, ok := previous[relPath]
		if !ok || prevFP != currentFP {
			changed = append(changed, relPath)
		}
	}
	for relPath := range previous {
		if _, ok := current[relPath]; !ok {
			deleted = append(deleted, relPath)
		}
	}

	sort.Strings(changed)
	sort.Strings(deleted)
	return changed, deleted
}

func loadGameIndex(gameDir string) (gameIndex, error) {
	path := filepath.Join(gameDir, indexFilename)
	payload, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return gameIndex{
			LatestState: map[string]fingerprint{},
			Snapshots:   []indexSnapshot{},
		}, nil
	}
	if err != nil {
		return gameIndex{}, fmt.Errorf("read snapshot index: %w", err)
	}

	var index gameIndex
	if err := json.Unmarshal(payload, &index); err != nil {
		return gameIndex{}, fmt.Errorf("decode snapshot index: %w", err)
	}
	if index.LatestState == nil {
		index.LatestState = map[string]fingerprint{}
	}
	if index.Snapshots == nil {
		index.Snapshots = []indexSnapshot{}
	}
	return index, nil
}

func loadSnapshotMetadata(path string) (snapshotMetadata, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return snapshotMetadata{}, fmt.Errorf("read snapshot metadata: %w", err)
	}

	var meta snapshotMetadata
	if err := json.Unmarshal(payload, &meta); err != nil {
		return snapshotMetadata{}, fmt.Errorf("decode snapshot metadata: %w", err)
	}
	if meta.CopiedFiles == nil {
		meta.CopiedFiles = []snapshotFile{}
	}
	if meta.DeletedPaths == nil {
		meta.DeletedPaths = []string{}
	}
	return meta, nil
}

func writeJSONAtomic(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create json parent dir: %w", err)
	}
	bytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	bytes = append(bytes, '\n')

	tmpPath := path + ".tmp-" + uuid.NewString()
	if err := os.WriteFile(tmpPath, bytes, 0o644); err != nil {
		return fmt.Errorf("write temp json: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace json: %w", err)
	}
	return nil
}

func snapshotGameDir(vaultRoot string, game models.Game) string {
	return filepath.Join(vaultRoot, sanitizeSnapshotName(game.Name)+"__"+game.ID)
}

func snapshotFolderName(seq int, createdAt time.Time) string {
	return fmt.Sprintf("%06d_%s", seq, createdAt.UTC().Format("20060102_150405"))
}

func snapshotMetadataFilename(folderName string) string {
	return folderName + ".json"
}

func sanitizeSnapshotName(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "game"
	}
	withoutInvalid := invalidSnapshotNameChars.ReplaceAllString(trimmed, "_")
	compactSpaces := strings.Join(strings.Fields(withoutInvalid), "_")
	compact := strings.Trim(compactSpaces, "._ ")
	if compact == "" {
		return "game"
	}
	return compact
}

func shouldIgnore(path string, patterns []string) bool {
	name := filepath.Base(path)
	for _, pattern := range patterns {
		p := strings.TrimSpace(pattern)
		if p == "" {
			continue
		}
		if strings.EqualFold(name, p) {
			return true
		}
		if matched, _ := filepath.Match(p, name); matched {
			return true
		}
		if strings.Contains(strings.ToLower(path), strings.ToLower(string(os.PathSeparator)+p+string(os.PathSeparator))) {
			return true
		}
	}
	return false
}

func toSummary(entry indexSnapshot) models.SnapshotSummary {
	return models.SnapshotSummary{
		ID:           entry.ID,
		Seq:          entry.Seq,
		CreatedAt:    entry.CreatedAt,
		Trigger:      entry.Trigger,
		ChangedFiles: entry.ChangedFiles,
		TotalSize:    entry.TotalSize,
	}
}

func findSnapshot(entries []indexSnapshot, snapshotID string) (indexSnapshot, bool) {
	for _, entry := range entries {
		if entry.ID == snapshotID {
			return entry, true
		}
	}
	return indexSnapshot{}, false
}

func cloneFingerprintMap(input map[string]fingerprint) map[string]fingerprint {
	result := make(map[string]fingerprint, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func cloneStringMap(input map[string]string) map[string]string {
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func ptr[T any](value T) *T {
	return &value
}
