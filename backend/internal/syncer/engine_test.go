package syncer

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cloud-saver/backend/internal/config"
	"cloud-saver/backend/internal/logstore"
	"cloud-saver/backend/internal/models"
)

func TestEngineLocalFirstRecoveryPointAndExactRestore(t *testing.T) {
	root := t.TempDir()
	saveRoot := filepath.Join(root, "save")
	if err := os.MkdirAll(saveRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(root, "game.exe")
	if err := os.WriteFile(exe, []byte("test executable"), 0o644); err != nil {
		t.Fatal(err)
	}
	save := filepath.Join(saveRoot, "slot.dat")
	if err := os.WriteFile(save, []byte("revision-one"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, logs, game := testProtectionEnvironment(t, root, saveRoot, exe)
	engine := New(cfg, logs, false)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	engine.Start(ctx)

	firstJob := engine.TriggerGame(game.ID, TriggerReasonManualGame)
	first := waitForJob(t, engine, firstJob, "completed")
	if first.RevisionID == "" {
		t.Fatal("first protection did not persist a revision id")
	}
	firstRevision := first.RevisionID

	if err := os.WriteFile(save, []byte("revision-two"), 0o644); err != nil {
		t.Fatal(err)
	}
	extra := filepath.Join(saveRoot, "new.dat")
	if err := os.WriteFile(extra, []byte("new file"), 0o644); err != nil {
		t.Fatal(err)
	}
	engine.NotifyFileEvent(save)
	engine.NotifyFileEvent(extra)
	secondJob := engine.TriggerGame(game.ID, TriggerReasonManualGame)
	waitForJob(t, engine, secondJob, "completed")

	pointJob := engine.CreateRecoveryPoint(game.ID, "Before boss")
	waitForJob(t, engine, pointJob, "completed")
	points, err := engine.RecoveryPoints(game.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 1 || points[0].Name != "Before boss" {
		t.Fatalf("unexpected recovery points: %#v", points)
	}

	if err := engine.RestoreLocalRevision(context.Background(), game, firstRevision); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(save)
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != "revision-one" {
		t.Fatalf("restore materialized %q, want revision-one", payload)
	}
	if _, err := os.Stat(extra); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("exact restore left a file absent from the revision: %v", err)
	}
}

func TestEngineKeepsLocalRevisionWhenProviderIsOffline(t *testing.T) {
	root := t.TempDir()
	saveRoot := filepath.Join(root, "save")
	if err := os.MkdirAll(saveRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(root, "game.exe")
	if err := os.WriteFile(exe, []byte("test executable"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(saveRoot, "slot.dat"), []byte("offline progress"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, logs, game := testProtectionEnvironment(t, root, saveRoot, exe)
	settings := cfg.Get()
	settings.ProtectionTarget = models.ProtectionTargetFolder
	settings.CloudProvider = models.CloudProviderFolder
	settings.SyncRoot = filepath.Join(root, "provider")
	if err := cfg.Update(settings); err != nil {
		t.Fatal(err)
	}
	engine := New(cfg, logs, false)
	engine.SetStoreFactory(func(context.Context, models.Settings) (RevisionStore, error) {
		return nil, errors.New("network connection unavailable")
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	engine.Start(ctx)

	jobID := engine.TriggerGame(game.ID, TriggerReasonManualGame)
	job := waitForJob(t, engine, jobID, "waiting_for_network")
	if job.RevisionID == "" {
		t.Fatal("offline provider failure lost the committed local revision")
	}
	revision, err := logs.Revision(job.RevisionID)
	if err != nil {
		t.Fatal(err)
	}
	if revision.Uploaded {
		t.Fatal("offline revision was incorrectly marked uploaded")
	}
	status := engine.Status()
	if status.Provider.State != models.ProviderOffline || status.Pending == 0 {
		t.Fatalf("unexpected offline status: %#v", status)
	}
}

func TestConfigurationChangedRefreshesProtectionTargetAndProviderState(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	cfg, err := config.New(filepath.Join(root, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	logs, err := logstore.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = logs.Close() })
	engine := New(cfg, logs, false)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	engine.Start(ctx)
	if state := engine.Status().Provider.State; state != models.ProviderLocalOnly {
		t.Fatalf("expected initial local provider state, got %q", state)
	}

	if err := cfg.SetGoogleDriveConnection("player@example.test", "protected-token"); err != nil {
		t.Fatal(err)
	}
	settings := cfg.Get()
	settings.ProtectionTarget = models.ProtectionTargetDrive
	if err := cfg.Update(settings); err != nil {
		t.Fatal(err)
	}
	engine.ConfigurationChanged()
	status := engine.Status()
	if status.ProtectionTarget != models.ProtectionTargetDrive {
		t.Fatalf("expected Google Drive target, got %q", status.ProtectionTarget)
	}
	if status.Provider.State != models.ProviderReconnecting {
		t.Fatalf("expected a fresh Drive target to reconnect, got %q", status.Provider.State)
	}
}

func TestStructuredProviderRateLimitClassification(t *testing.T) {
	t.Parallel()
	retryAfter := 7 * time.Minute
	err := fmt.Errorf("upload object: %w", &ProviderOperationError{
		Code:       "rate_limited",
		Retryable:  true,
		RetryAfter: retryAfter,
		Err:        errors.New("provider rate limit"),
	})
	code, state, _, retryable := classifyProviderError(err)
	if code != "rate_limited" || state != models.ProviderRateLimited || !retryable {
		t.Fatalf("unexpected rate classification: code=%q state=%q retryable=%t", code, state, retryable)
	}
	if got := providerRetryAfter(err); got != retryAfter {
		t.Fatalf("expected %s Retry-After, got %s", retryAfter, got)
	}
}

func TestRecoveryLibraryRestorePublishesOnlyVerifiedRevision(t *testing.T) {
	root := t.TempDir()
	saveRoot := filepath.Join(root, "save")
	if err := os.MkdirAll(saveRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(root, "game.exe")
	if err := os.WriteFile(exe, []byte("test executable"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(saveRoot, "slot.dat"), []byte("cloud progress"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, logs, game := testProtectionEnvironment(t, root, saveRoot, exe)
	providerRoot := filepath.Join(root, "provider")
	settings := cfg.Get()
	settings.ProtectionTarget = models.ProtectionTargetFolder
	settings.CloudProvider = models.CloudProviderFolder
	settings.SyncRoot = providerRoot
	if err := cfg.Update(settings); err != nil {
		t.Fatal(err)
	}
	engine := New(cfg, logs, false)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	engine.Start(ctx)

	job := waitForJob(t, engine, engine.TriggerGame(game.ID, TriggerReasonManualGame), "completed")
	destination := filepath.Join(root, "restored")
	if err := engine.RestoreCloudProfile(context.Background(), game.SyncID, destination); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(filepath.Join(destination, "slot.dat"))
	if err != nil || string(payload) != "cloud progress" {
		t.Fatalf("unexpected restored payload %q, error %v", payload, err)
	}

	revision, err := logs.Revision(job.RevisionID)
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := json.Unmarshal([]byte(revision.ManifestJSON), &manifest); err != nil {
		t.Fatal(err)
	}
	var objectHash string
	for _, entry := range manifest.Files {
		objectHash = entry.Hash
		break
	}
	objectPath := filepath.Join(providerRoot, protocolDirectory, "games", game.SyncID, "objects", objectHash)
	if err := os.WriteFile(objectPath, []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	corruptDestination := filepath.Join(root, "corrupt-restore")
	if err := engine.RestoreCloudProfile(context.Background(), game.SyncID, corruptDestination); err == nil {
		t.Fatal("corrupt provider object was accepted")
	}
	if _, err := os.Stat(corruptDestination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed recovery became visible at the destination: %v", err)
	}
}

func TestConcurrentProviderHeadIsPreservedForReview(t *testing.T) {
	root := t.TempDir()
	saveOne := filepath.Join(root, "save-one")
	saveTwo := filepath.Join(root, "save-two")
	for _, path := range []string{saveOne, saveTwo} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	exe := filepath.Join(root, "game.exe")
	if err := os.WriteFile(exe, []byte("test executable"), 0o644); err != nil {
		t.Fatal(err)
	}
	slotOne := filepath.Join(saveOne, "slot.dat")
	if err := os.WriteFile(slotOne, []byte("common"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, logsOne, game := testProtectionEnvironment(t, root, saveOne, exe)
	providerRoot := filepath.Join(root, "provider")
	settings := cfg.Get()
	settings.ProtectionTarget = models.ProtectionTargetFolder
	settings.CloudProvider = models.CloudProviderFolder
	settings.SyncRoot = providerRoot
	if err := cfg.Update(settings); err != nil {
		t.Fatal(err)
	}
	engine := New(cfg, logsOne, false)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	engine.Start(ctx)
	waitForJob(t, engine, engine.TriggerGame(game.ID, TriggerReasonManualGame), "completed")

	provider := NewFolderStore(providerRoot)
	commonHead, err := provider.ReadHead(context.Background(), game.SyncID)
	if err != nil || commonHead == "" {
		t.Fatalf("missing common provider head: %q, %v", commonHead, err)
	}
	deviceTwoRoot := filepath.Join(root, "device-two-vault")
	deviceTwo := NewFolderStore(deviceTwoRoot)
	if _, err := ReplicateRevision(context.Background(), provider, deviceTwo, game.SyncID, commonHead); err != nil {
		t.Fatal(err)
	}
	if _, err := RestoreRevision(context.Background(), deviceTwo, saveTwo, game.SyncID, commonHead, "device-two", nil); err != nil {
		t.Fatal(err)
	}
	deviceTwoState := filepath.Join(root, "device-two-state")
	if err := os.MkdirAll(deviceTwoState, 0o755); err != nil {
		t.Fatal(err)
	}
	logsTwo, err := logstore.Open(deviceTwoState)
	if err != nil {
		t.Fatal(err)
	}
	defer logsTwo.Close()
	if _, err := ReconcileWithStore(context.Background(), deviceTwo, saveTwo, "game-two", game.SyncID, "device-two", nil, logsTwo); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(saveTwo, "slot.dat"), []byte("device two"), 0o644); err != nil {
		t.Fatal(err)
	}
	localTwoResult, err := ReconcileWithStore(context.Background(), deviceTwo, saveTwo, "game-two", game.SyncID, "device-two", nil, logsTwo)
	if err != nil || localTwoResult.ChangedFiles == 0 {
		t.Fatalf("device two publication failed: %#v, %v", localTwoResult, err)
	}
	deviceTwoHead, err := deviceTwo.ReadHead(context.Background(), game.SyncID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReplicateRevisionGuarded(context.Background(), deviceTwo, provider, game.SyncID, deviceTwoHead, "game-two", logsTwo); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(slotOne, []byte("device one"), 0o644); err != nil {
		t.Fatal(err)
	}
	engine.NotifyFileEvent(slotOne)
	conflictJob := waitForJob(t, engine, engine.TriggerGame(game.ID, TriggerReasonManualGame), "conflict")
	if conflictJob.ErrorCode != "concurrent_revision" {
		t.Fatalf("unexpected conflict job: %#v", conflictJob)
	}
	after, err := provider.ReadHead(context.Background(), game.SyncID)
	if err != nil {
		t.Fatal(err)
	}
	if after != deviceTwoHead {
		t.Fatalf("concurrent upload overwrote provider head: got %s, want %s", after, deviceTwoHead)
	}
	conflicts, err := logsOne.SyncConflicts(game.ID)
	if err != nil || len(conflicts) == 0 {
		t.Fatalf("concurrent revision did not create review records: %#v, %v", conflicts, err)
	}
}

func testProtectionEnvironment(t *testing.T, root, saveRoot, exe string) (*config.Manager, *logstore.Store, models.Game) {
	t.Helper()
	cfg, err := config.New(filepath.Join(root, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	game, err := cfg.AddGame("Test Game", saveRoot, exe, "")
	if err != nil {
		t.Fatal(err)
	}
	logs, err := logstore.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = logs.Close() })
	return cfg, logs, game
}

func waitForJob(t *testing.T, engine *Engine, id, state string) models.SyncJob {
	t.Helper()
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		for _, job := range engine.Jobs("") {
			if job.ID == id && job.State == state {
				return job
			}
			if job.ID == id && job.State == "failed" && state != "failed" {
				t.Fatalf("job failed: %#v", job)
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("job %s did not reach %s: %#v", id, state, engine.Jobs(""))
	return models.SyncJob{}
}
