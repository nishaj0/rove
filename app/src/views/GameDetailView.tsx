import { useEffect, useState } from "react";
import { newDefaultBackupPolicy } from "../constants/defaults";
import type { BackupHistoryEntry, BackupPolicy, Game, SnapshotSummary } from "../types";

type GameDetailViewProps = {
  game: Game | null;
  exportPath: string;
  importPath: string;
  mergeOnImport: boolean;
  exportOperationState: "idle" | "working" | "success" | "failed";
  exportOperationMessage: string;
  importOperationState: "idle" | "working" | "success" | "failed";
  importOperationMessage: string;
  backupHistory: BackupHistoryEntry[];
  snapshots: SnapshotSummary[];
  snapshotVaultConfigured: boolean;
  onBack: () => void;
  onPlay: () => void;
  onSyncGame: () => void;
  onDelete: () => void;
  onBrowseExportPath: () => void;
  onBrowseImportPath: () => void;
  onMergeChange: (value: boolean) => void;
  onExportBackup: () => void;
  onImportBackup: () => void;
  onSaveSnapshotNow: () => void;
  onRestoreSnapshot: (snapshotId: string) => void;
  onDeleteSnapshot: (snapshotId: string) => void;
  onSaveBackupPolicy: (value: BackupPolicy) => Promise<boolean>;
  onUpdateCover: (coverImageData: string) => void;
  formatPlaytime: (seconds: number) => string;
  safeDateTime: (value: string) => string;
  formatBytes: (bytes: number) => string;
  formatSnapshotTrigger: (trigger: string) => string;
};

export function GameDetailView(props: GameDetailViewProps) {
  const {
    game,
    exportPath,
    importPath,
    mergeOnImport,
    exportOperationState,
    exportOperationMessage,
    importOperationState,
    importOperationMessage,
    backupHistory,
    snapshots,
    snapshotVaultConfigured,
    onBack,
    onPlay,
    onSyncGame,
    onDelete,
    onBrowseExportPath,
    onBrowseImportPath,
    onMergeChange,
    onExportBackup,
    onImportBackup,
    onSaveSnapshotNow,
    onRestoreSnapshot,
    onDeleteSnapshot,
    onSaveBackupPolicy,
    onUpdateCover,
    formatPlaytime,
    safeDateTime,
    formatBytes,
    formatSnapshotTrigger,
  } = props;

  const [backupPolicyDraft, setBackupPolicyDraft] = useState<BackupPolicy>(game?.backupPolicy ?? newDefaultBackupPolicy());
  const [backupPolicyState, setBackupPolicyState] = useState<"idle" | "saving" | "saved" | "failed">("idle");

  useEffect(() => {
    if (!game) {
      return;
    }
    setBackupPolicyDraft(game.backupPolicy);
    setBackupPolicyState("idle");
  }, [
    game?.id,
    game?.backupPolicy.enabled,
    game?.backupPolicy.intervalMinutes,
    game?.backupPolicy.exitDelaySeconds,
    game?.backupPolicy.retentionCount,
  ]);

  async function onSavePolicy() {
    setBackupPolicyState("saving");
    const saved = await onSaveBackupPolicy(backupPolicyDraft);
    setBackupPolicyState(saved ? "saved" : "failed");
  }

  const exportStateClass = exportOperationState === "working"
    ? "syncing"
    : exportOperationState === "success"
      ? "success"
      : "error";

  const importStateClass = importOperationState === "working"
    ? "syncing"
    : importOperationState === "success"
      ? "success"
      : "error";

  const policyStateClass = backupPolicyState === "saving"
    ? "syncing"
    : backupPolicyState === "saved"
      ? "success"
      : "error";

  if (!game) {
    return (
      <section className="panel">
        <h2>Game Not Found</h2>
        <p className="empty-state">This game no longer exists. Return to Games to pick another one.</p>
        <button className="btn nav-back" onClick={onBack} type="button"><span aria-hidden="true">&#8592;</span> Back to Games</button>
      </section>
    );
  }

  return (
    <>
      <section className="panel game-detail-head">
        <div className="game-detail-nav">
          <button className="btn nav-back" onClick={onBack} type="button"><span aria-hidden="true">&#8592;</span> Back to Games</button>
        </div>
        <div className="game-detail-hero">
          <div className="game-detail-cover">
            {game.coverImageData ? <img src={game.coverImageData} alt={`${game.name} cover`} /> : <span>{game.name.slice(0, 2).toUpperCase()}</span>}
          </div>
          <div className="game-detail-meta">
            <h2>{game.name}</h2>
            <p title={game.path}>Save folder: {game.path}</p>
            <p title={game.executablePath}>Executable: {game.executablePath}</p>
            <p>Play time: {formatPlaytime(game.playtimeSeconds ?? 0)}</p>
            <p>Last played: {game.lastPlayedAt ? safeDateTime(game.lastPlayedAt) : "Never"}</p>
            <p>Status: <strong className={`state-text ${game.running ? "success" : "warning"}`}>{game.running ? "Running" : "Idle"}</strong></p>
          </div>
        </div>
        <div className="detail-actions">
          <button className="btn primary" onClick={onPlay} type="button">Play</button>
          <button className="btn" onClick={onSyncGame} type="button">Sync This Game</button>
          <button className="btn danger" onClick={onDelete} type="button">Remove Game</button>
        </div>
      </section>

      <section className="panel split">
        <div className="backup-pane">
          <h2>Export Save</h2>
          <label>
            Destination folder
            <input
              className="picker-input"
              placeholder="D:\\Backups"
              value={exportPath}
              readOnly
              disabled={exportOperationState === "working"}
              onClick={onBrowseExportPath}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  onBrowseExportPath();
                }
              }}
            />
          </label>
          <button className="btn primary backup-action" onClick={onExportBackup} type="button" disabled={exportOperationState === "working"}>
            {exportOperationState === "working" ? "Exporting..." : "Export Backup"}
          </button>
          {exportOperationState !== "idle" ? <p className={`backup-status state-text ${exportStateClass}`}>{exportOperationMessage}</p> : null}
        </div>

        <div className="backup-pane">
          <h2>Import Save</h2>
          <label>
            ZIP source path
            <input
              className="picker-input"
              placeholder="D:\\Backups\\my-game.zip"
              value={importPath}
              readOnly
              disabled={importOperationState === "working"}
              onClick={onBrowseImportPath}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  onBrowseImportPath();
                }
              }}
            />
          </label>
          <label className="inline-check">
            <input type="checkbox" checked={mergeOnImport} disabled={importOperationState === "working"} onChange={(e) => onMergeChange(e.target.checked)} />
            Merge into existing folder
          </label>
          <button className="btn primary backup-action" onClick={onImportBackup} type="button" disabled={importOperationState === "working"}>
            {importOperationState === "working" ? "Importing..." : "Import Backup"}
          </button>
          {importOperationState !== "idle" ? <p className={`backup-status state-text ${importStateClass}`}>{importOperationMessage}</p> : null}
        </div>
      </section>

      <section className="panel">
        <h2>Auto Backup Policy</h2>
        <div className="backup-policy-grid">
          <label className="inline-check">
            <input
              type="checkbox"
              checked={backupPolicyDraft.enabled}
              onChange={(e) => {
                setBackupPolicyState("idle");
                setBackupPolicyDraft((current) => ({ ...current, enabled: e.target.checked }));
              }}
            />
            Enable automatic local backups
          </label>
          <label>
            Interval minutes
            <input
              type="number"
              min={3}
              max={120}
              value={backupPolicyDraft.intervalMinutes}
              onChange={(e) => {
                setBackupPolicyState("idle");
                const value = Number.parseInt(e.target.value, 10);
                setBackupPolicyDraft((current) => ({
                  ...current,
                  intervalMinutes: Number.isFinite(value) ? value : current.intervalMinutes,
                }));
              }}
            />
          </label>
          <label>
            Exit delay seconds
            <input
              type="number"
              min={3}
              max={120}
              value={backupPolicyDraft.exitDelaySeconds}
              onChange={(e) => {
                setBackupPolicyState("idle");
                const value = Number.parseInt(e.target.value, 10);
                setBackupPolicyDraft((current) => ({
                  ...current,
                  exitDelaySeconds: Number.isFinite(value) ? value : current.exitDelaySeconds,
                }));
              }}
            />
          </label>
          <label>
            Keep latest backups
            <input
              type="number"
              min={1}
              max={30}
              value={backupPolicyDraft.retentionCount}
              onChange={(e) => {
                setBackupPolicyState("idle");
                const value = Number.parseInt(e.target.value, 10);
                setBackupPolicyDraft((current) => ({
                  ...current,
                  retentionCount: Number.isFinite(value) ? value : current.retentionCount,
                }));
              }}
            />
          </label>
        </div>
        <div className="detail-actions">
          <button className="btn primary" onClick={() => void onSavePolicy()} type="button" disabled={backupPolicyState === "saving"}>
            {backupPolicyState === "saving" ? "Saving..." : "Save Backup Policy"}
          </button>
        </div>
        {backupPolicyState !== "idle" ? (
          <p className={`backup-status state-text ${policyStateClass}`}>
            {backupPolicyState === "saving" ? "Saving backup policy..." : backupPolicyState === "saved" ? "Backup policy saved." : "Backup policy save failed."}
          </p>
        ) : null}
      </section>

      <section className="panel">
        <div className="panel-head">
          <h2>Snapshots</h2>
          <button className="btn primary" onClick={onSaveSnapshotNow} type="button" disabled={!snapshotVaultConfigured}>Save Now</button>
        </div>
        {!snapshotVaultConfigured ? (
          <p className="empty-state">Choose a snapshot vault folder in Settings to enable versioned snapshots.</p>
        ) : snapshots.length === 0 ? (
          <p className="empty-state">No snapshots yet.</p>
        ) : (
          <ul className="snapshot-list">
            {snapshots.map((snapshot) => (
              <li className="snapshot-row" key={snapshot.id}>
                <div className="snapshot-main">
                  <p>
                    <strong>#{snapshot.seq}</strong>
                    {" · "}
                    {safeDateTime(snapshot.createdAt)}
                    {" · "}
                    {formatSnapshotTrigger(snapshot.trigger)}
                  </p>
                  <p className="snapshot-meta">
                    {snapshot.changedFiles} changed file{snapshot.changedFiles === 1 ? "" : "s"} · {formatBytes(snapshot.totalSize)}
                  </p>
                </div>
                <div className="snapshot-actions">
                  <button className="btn" onClick={() => onRestoreSnapshot(snapshot.id)} type="button">Restore</button>
                  <button className="btn danger" onClick={() => onDeleteSnapshot(snapshot.id)} type="button">Delete</button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="panel">
        <h2>Local Backup History</h2>
        {backupHistory.length === 0 ? (
          <p className="empty-state">No automatic backups yet.</p>
        ) : (
          <ul className="backup-history-list">
            {backupHistory.map((entry) => (
              <li className="backup-history-row" key={entry.id}>
                <p>
                  <strong>{entry.reason === "final" ? "Final" : "Periodic"}</strong>
                  {" backup at "}
                  {safeDateTime(entry.createdAt)}
                </p>
                <p className="backup-history-path" title={entry.path}>{entry.path}</p>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="panel">
        <h2>Cover Image</h2>
        <label className="file-input-wrap">
          <span>Upload image</span>
          <input
            type="file"
            accept="image/*"
            onChange={(e) => {
              const file = e.target.files?.[0];
              if (!file) {
                return;
              }
              const reader = new FileReader();
              reader.onload = () => {
                if (typeof reader.result === "string") {
                  onUpdateCover(reader.result);
                }
              };
              reader.readAsDataURL(file);
            }}
          />
        </label>
      </section>
    </>
  );
}
