import { useEffect, useState } from "react";
import { ImagePlus, Link2, MonitorUp, Palette } from "lucide-react";
import { GameCover } from "../components/GameCover";
import { newDefaultSyncPolicy } from "../constants/defaults";
import type { Game, PauseScope, ProviderHealth, RecoveryPoint, Revision, SnapshotSummary, StorageUsage, SyncConflict, SyncJob, SyncPolicy } from "../types";
import { getSaveStatus } from "../utils/saveStatus";

type GameDetailViewProps = {
  game: Game | null;
  exportPath: string;
  importPath: string;
  mergeOnImport: boolean;
  exportOperationState: "idle" | "working" | "success" | "failed";
  exportOperationMessage: string;
  importOperationState: "idle" | "working" | "success" | "failed";
  importOperationMessage: string;
  revisions: Revision[];
  recoveryPoints: RecoveryPoint[];
  storageUsage: StorageUsage | null;
  legacySnapshots: SnapshotSummary[];
  onBack: () => void;
  onPlay: () => void;
  onOpenSettings: () => void;
  onCreateShortcut: () => void;
  onProtect: () => void;
  onDelete: () => void;
  onBrowseExportPath: () => void;
  onBrowseImportPath: () => void;
  onMergeChange: (value: boolean) => void;
  onExportBackup: () => void;
  onImportBackup: () => void;
  onSavePolicy: (value: SyncPolicy) => Promise<boolean>;
  onPause: (scope: PauseScope, duration: "manual" | "one_hour" | "until_login") => Promise<void>;
  onResume: () => Promise<void>;
  onCreateRecoveryPoint: () => void;
  onRestoreRevision: (revisionId: string) => void;
  onDeleteRecoveryPoint: (id: string) => void;
  onRestoreLegacySnapshot: (id: string) => void;
  onDeleteLegacySnapshot: (id: string) => void;
  onUpdateCover: (coverImageData: string, mode?: "manual" | "executable_icon" | "generated") => void;
  formatPlaytime: (seconds: number) => string;
  safeDateTime: (value: string) => string;
  formatBytes: (bytes: number) => string;
  syncJob: SyncJob | null;
  provider?: ProviderHealth;
  syncConflicts: SyncConflict[];
  onResolveSyncConflict: (id: string, action: "local" | "remote" | "both") => void;
};

export function GameDetailView(props: GameDetailViewProps) {
  const {
    game, exportPath, importPath, mergeOnImport, exportOperationState,
    exportOperationMessage, importOperationState, importOperationMessage,
    revisions, recoveryPoints, storageUsage, legacySnapshots, onBack, onPlay, onOpenSettings, onCreateShortcut, onProtect, onDelete,
    onBrowseExportPath, onBrowseImportPath, onMergeChange, onExportBackup,
    onImportBackup, onSavePolicy, onPause, onResume, onCreateRecoveryPoint,
    onRestoreRevision, onDeleteRecoveryPoint, onRestoreLegacySnapshot,
    onDeleteLegacySnapshot, onUpdateCover, formatPlaytime,
    safeDateTime, formatBytes, syncJob, provider, syncConflicts, onResolveSyncConflict,
  } = props;
  const [policy, setPolicy] = useState<SyncPolicy>(game?.protectionPolicy ?? newDefaultSyncPolicy());
  const [saving, setSaving] = useState(false);
  const [advancedOpen, setAdvancedOpen] = useState(false);

  useEffect(() => {
    setPolicy(game?.protectionPolicy ?? newDefaultSyncPolicy());
    setAdvancedOpen(false);
  }, [game?.id, game?.protectionPolicy]);

  if (!game) {
    return (
      <section className="panel" data-window-title="MISSING.GAME">
        <h2>Game not found</h2>
        <button className="btn" onClick={onBack} type="button">Back to Games</button>
      </section>
    );
  }

  const health = game.protectionStatus;
  const active = syncJob && ["queued", "waiting", "running"].includes(syncJob.state);
  const paused = health.serviceControl.pauseScope !== "none";
  const saveStatus = getSaveStatus({
    provider,
    serviceControl: health.serviceControl,
    conflictCount: health.conflictCount,
    pendingCount: health.pendingCount,
    active: Boolean(active),
  });

  return (
    <>
      <section className="panel game-detail-head" data-window-title="GAME.PROFILE">
        <button className="btn nav-back" onClick={onBack} type="button">← Back to Games</button>
        <div className="game-detail-hero">
          <div className="game-detail-cover">
            <GameCover game={game} />
          </div>
          <div className="game-detail-meta">
            <h1>{game.name}</h1>
            <p>{game.running ? "Playing now" : "Ready to play"}</p>
            <p>Time played: {formatPlaytime(game.playtimeSeconds ?? 0)}</p>
            <div className="save-summary game-save-summary" role={saveStatus.action === "none" ? "status" : "alert"}>
              <span className={`state-text ${saveStatus.tone}`}>{saveStatus.label}</span>
              <p>{saveStatus.detail}</p>
            </div>
          </div>
        </div>
        <div className="detail-actions">
          <button className="btn primary" onClick={onPlay} type="button">Play</button>
          <button className="btn" onClick={onProtect} disabled={Boolean(active)} type="button">{active ? "Backing up…" : "Back up now"}</button>
        </div>
        {saveStatus.action !== "none" ? <div className="banner warning" role="alert"><strong>{saveStatus.label}</strong><span>{saveStatus.detail}</span>{saveStatus.action === "review" ? <button className="btn" onClick={() => setAdvancedOpen(true)} type="button">Review saves</button> : <button className="btn" onClick={onOpenSettings} type="button">Open Settings</button>}</div> : null}
      </section>

      <section className="panel advanced-setup-panel" data-window-title="ADVANCED.SETUP">
        <details className="advanced-disclosure" open={advancedOpen} onToggle={(event) => setAdvancedOpen(event.currentTarget.open)}>
          <summary>Advanced setup</summary>
          <p className="disclosure-help">Backup options, restore tools, and game details.</p>
          <div className="advanced-section">
            <h2>Game details</h2>
            <dl className="detail-list">
              {game.version ? <><dt>Version</dt><dd>{game.version}</dd></> : null}
              <dt>Save folder</dt><dd>{game.path}</dd>
            </dl>
            <button className="btn" onClick={onCreateShortcut} type="button"><Link2 aria-hidden="true" size={15} />Create Rove shortcut</button>
          </div>

      <section className="advanced-section" data-window-title="SAVE.STATUS">
        <div className="panel-head">
          <h2>Save status</h2>
          <span className={`state-text ${paused ? "warning" : health.conflictCount ? "error" : "success"}`}>{paused ? `Paused: ${health.serviceControl.pauseScope}` : health.conflictCount ? "Needs review" : "Active"}</span>
        </div>
        <div className="health-grid">
          <div><span>Last backup</span><strong>{health.lastLocalProtection ? safeDateTime(health.lastLocalProtection) : "Not yet"}</strong></div>
          <div><span>Cloud backup</span><strong>{health.lastCloudUpload ? safeDateTime(health.lastCloudUpload) : "Not yet"}</strong></div>
          <div><span>Changed files</span><strong>{health.dirtyFiles}</strong></div>
          <div><span>Waiting to upload</span><strong>{health.pendingCount} · {formatBytes(health.pendingBytes)}</strong></div>
        </div>
        {syncJob?.error ? <p className="banner error">{syncJob.error}</p> : null}
        {!syncJob?.error && provider?.state === "authentication_required" && provider.message ? <p className="banner error" role="alert">{provider.message}</p> : null}
      </section>

      <section className="advanced-section" data-window-title="BACKUP.OPTIONS">
        <h2>Backup options</h2>
        <div className="settings-grid">
          <label className="inline-check">
            <input type="checkbox" checked={policy.enabled} onChange={(event) => setPolicy({ ...policy, enabled: event.target.checked })} />
            Back up saves for this game
          </label>
          <label>
            Backup schedule
            <select value={policy.mode} onChange={(event) => setPolicy({ ...policy, mode: event.target.value as SyncPolicy["mode"] })}>
              <option value="balanced">Balanced — every 10 minutes and exit</option>
              <option value="exit_only">Exit only — no reads during gameplay</option>
              <option value="continuous">Continuous — after quiet writes, at least 2 minutes</option>
              <option value="manual">Manual only</option>
            </select>
          </label>
          <label>
            Minutes between backups
            <input type="number" min={2} max={1440} disabled={policy.mode === "exit_only" || policy.mode === "manual"} value={policy.checkpointIntervalMinutes} onChange={(event) => setPolicy({ ...policy, checkpointIntervalMinutes: Number(event.target.value) })} />
          </label>
          <label className="inline-check">
            <input type="checkbox" checked={policy.exitProtection} onChange={(event) => setPolicy({ ...policy, exitProtection: event.target.checked })} />
            Back up when the game closes
          </label>
        </div>
        <div className="detail-actions">
          <button className="btn primary" disabled={saving} onClick={async () => { setSaving(true); await onSavePolicy(policy); setSaving(false); }} type="button">{saving ? "Saving…" : "Save backup options"}</button>
          <button className="btn" onClick={() => void onPause("uploads", "manual")} type="button">Pause cloud backups</button>
          <button className="btn" onClick={() => void onPause("all", "one_hour")} type="button">Pause for one hour</button>
          <button className="btn" disabled={!paused} onClick={() => void onResume()} type="button">Resume backups</button>
        </div>
      </section>

      <section className="advanced-section" data-window-title="RESTORE.POINTS">
        <div className="panel-head"><h2>Restore points</h2><button className="btn" onClick={onCreateRecoveryPoint} disabled={Boolean(active)} type="button">Create restore point</button></div>
        {recoveryPoints.length === 0 ? <p className="empty-state">No restore points yet.</p> : (
          <ul className="snapshot-list">{recoveryPoints.map((point) => (
            <li className="snapshot-row" key={point.id}>
              <div className="snapshot-main"><p><strong>{point.name}</strong></p><p className="snapshot-meta">{safeDateTime(point.createdAt)} · {point.fileCount} files · {formatBytes(point.totalBytes)}</p></div>
              <div className="snapshot-actions">
                <button className="btn" onClick={() => onRestoreRevision(point.revisionId)} type="button">Restore</button>
                <button className="btn danger" onClick={() => onDeleteRecoveryPoint(point.id)} type="button">Remove</button>
              </div>
            </li>
          ))}</ul>
        )}
      </section>

      <section className="advanced-section" data-window-title="PREVIOUS.SAVES">
        <div className="panel-head">
          <h2>Previous saves</h2>
          <span className="state-text">{storageUsage ? `${formatBytes(storageUsage.uniqueBytes)} unique` : "Calculating"}</span>
        </div>
        {storageUsage ? <p className="snapshot-meta">{storageUsage.automaticCount} automatic · {storageUsage.recoveryPoints} pinned · {formatBytes(storageUsage.pendingBytes)} pending upload</p> : null}
        {revisions.length === 0 ? <p className="empty-state">No previous saves yet.</p> : (
          <ul className="snapshot-list">{revisions.slice(0, 20).map((revision) => (
            <li className="snapshot-row" key={revision.id}>
              <div className="snapshot-main"><p><strong>{revision.name || revision.kind.replace(/_/g, " ")}</strong></p><p className="snapshot-meta">{safeDateTime(revision.createdAt)} · {revision.fileCount} files · {formatBytes(revision.totalBytes)} · {revision.uploaded ? "cloud uploaded" : "local"}</p></div>
              <button className="btn" onClick={() => onRestoreRevision(revision.id)} type="button">Restore</button>
            </li>
          ))}</ul>
        )}
      </section>

      {legacySnapshots.length > 0 ? (
        <section className="advanced-section" data-window-title="OLDER.LOCAL.BACKUPS">
          <div className="panel-head">
            <h2>Older local backups</h2>
          </div>
          <p className="empty-state">These older backups are still available to restore.</p>
          <ul className="snapshot-list">
            {legacySnapshots.map((snapshot) => (
              <li className="snapshot-row" key={snapshot.id}>
                <div className="snapshot-main">
                  <p><strong>{safeDateTime(snapshot.createdAt)}</strong></p>
                  <p className="snapshot-meta">{snapshot.changedFiles} changed files · {formatBytes(snapshot.totalSize)} · {snapshot.trigger}</p>
                </div>
                <div className="snapshot-actions">
                  <button className="btn" type="button" onClick={() => onRestoreLegacySnapshot(snapshot.id)}>Restore</button>
                  <button className="btn danger" type="button" onClick={() => onDeleteLegacySnapshot(snapshot.id)}>Remove</button>
                </div>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      {syncConflicts.length > 0 ? (
        <section className="advanced-section" data-window-title="SAVED.VERSION.REVIEW">
          <h2>Choose a saved version</h2>
          <p className="empty-state">Nothing is replaced until you choose.</p>
          <ul className="snapshot-list">{syncConflicts.map((conflict) => (
            <li className="snapshot-row" key={conflict.id}>
              <div className="snapshot-main"><p><strong>{conflict.path}</strong></p><p className="snapshot-meta">{safeDateTime(conflict.createdAt)} · local {conflict.localHash.slice(0, 8)} · remote {conflict.remoteHash.slice(0, 8)}</p></div>
              <div className="snapshot-actions">
                <button className="btn" onClick={() => onResolveSyncConflict(conflict.id, "local")} type="button">Use this PC</button>
                <button className="btn" onClick={() => onResolveSyncConflict(conflict.id, "remote")} type="button">Use cloud copy</button>
                <button className="btn primary" onClick={() => onResolveSyncConflict(conflict.id, "both")} type="button">Keep both</button>
              </div>
            </li>
          ))}</ul>
        </section>
      ) : null}

      <section className="advanced-section split" data-window-title="IMPORT.EXPORT">
        <div className="backup-pane">
          <h2>Export saves</h2>
          <input className="picker-input" value={exportPath} readOnly onClick={onBrowseExportPath} placeholder="Choose destination folder" />
          <button className="btn primary" onClick={onExportBackup} disabled={exportOperationState === "working"} type="button">{exportOperationState === "working" ? "Exporting…" : "Export"}</button>
          {exportOperationState !== "idle" ? <p className="snapshot-meta">{exportOperationMessage}</p> : null}
        </div>
        <div className="backup-pane">
          <h2>Import saves</h2>
          <input className="picker-input" value={importPath} readOnly onClick={onBrowseImportPath} placeholder="Choose a backup file" />
          <label className="inline-check"><input type="checkbox" checked={mergeOnImport} onChange={(event) => onMergeChange(event.target.checked)} />Keep existing files</label>
          <button className="btn primary" onClick={onImportBackup} disabled={importOperationState === "working"} type="button">{importOperationState === "working" ? "Importing…" : "Import"}</button>
          {importOperationState !== "idle" ? <p className="snapshot-meta">{importOperationMessage}</p> : null}
        </div>
      </section>

      <section className="advanced-section" data-window-title="GAME.APPEARANCE">
        <h2>Game appearance</h2>
        <div className="detail-actions"><label className="file-input-wrap"><span><ImagePlus aria-hidden="true" size={15} /> Upload image</span><input type="file" accept="image/*" onChange={(event) => {
          const file = event.target.files?.[0];
          if (!file) return;
          const reader = new FileReader();
          reader.onload = () => { if (typeof reader.result === "string") onUpdateCover(reader.result); };
          reader.readAsDataURL(file);
        }} /></label><button className="btn" type="button" onClick={() => onUpdateCover("", "executable_icon")}><MonitorUp aria-hidden="true" size={15} />Use game icon</button><button className="btn" type="button" onClick={() => onUpdateCover("", "generated")}><Palette aria-hidden="true" size={15} />Use generated image</button></div>
      </section>
          <section className="advanced-section danger-zone" data-window-title="REMOVE.GAME">
            <h2>Remove game</h2>
            <p className="disclosure-help">This removes the game from Rove. It does not delete the game or its save files.</p>
            <button className="btn danger" onClick={onDelete} type="button">Remove game from Rove</button>
          </section>
        </details>
      </section>
    </>
  );
}
