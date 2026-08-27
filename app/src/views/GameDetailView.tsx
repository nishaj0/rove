import { useEffect, useState } from "react";
import { ImagePlus, Link2, MonitorUp, Palette } from "lucide-react";
import { GameCover } from "../components/GameCover";
import { newDefaultSyncPolicy } from "../constants/defaults";
import type { Game, PauseScope, ProviderHealth, RecoveryPoint, Revision, SnapshotSummary, StorageUsage, SyncConflict, SyncJob, SyncPolicy } from "../types";

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
    revisions, recoveryPoints, storageUsage, legacySnapshots, onBack, onPlay, onCreateShortcut, onProtect, onDelete,
    onBrowseExportPath, onBrowseImportPath, onMergeChange, onExportBackup,
    onImportBackup, onSavePolicy, onPause, onResume, onCreateRecoveryPoint,
    onRestoreRevision, onDeleteRecoveryPoint, onRestoreLegacySnapshot,
    onDeleteLegacySnapshot, onUpdateCover, formatPlaytime,
    safeDateTime, formatBytes, syncJob, provider, syncConflicts, onResolveSyncConflict,
  } = props;
  const [policy, setPolicy] = useState<SyncPolicy>(game?.protectionPolicy ?? newDefaultSyncPolicy());
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setPolicy(game?.protectionPolicy ?? newDefaultSyncPolicy());
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
            {game.version ? <p>Version: {game.version}</p> : null}
            <p title={game.path}>Save folder: {game.path}</p>
            <p>Play time: {formatPlaytime(game.playtimeSeconds ?? 0)}</p>
            <p>Status: <strong className={`state-text ${game.running ? "success" : "warning"}`}>{game.running ? "Playing" : "Idle"}</strong></p>
          </div>
        </div>
        <div className="detail-actions">
          <button className="btn primary" onClick={onPlay} type="button">Play with pre-check</button>
          <button className="btn" onClick={onCreateShortcut} type="button"><Link2 aria-hidden="true" size={15} />Create Rove shortcut</button>
          <button className="btn" onClick={onProtect} disabled={Boolean(active)} type="button">{active ? `${syncJob?.stage?.replace(/_/g, " ")} ${syncJob?.progress ?? 0}%` : "Sync now"}</button>
          <button className="btn" onClick={onCreateRecoveryPoint} disabled={Boolean(active)} type="button">Create Recovery Point</button>
          <button className="btn danger" onClick={onDelete} type="button">Remove Game</button>
        </div>
      </section>

      <section className="panel" data-window-title="CLOUD.SYNC.HEALTH">
        <div className="panel-head">
          <h2>Cloud Sync health</h2>
          <span className={`state-text ${paused ? "warning" : health.conflictCount ? "error" : "success"}`}>{paused ? `Paused: ${health.serviceControl.pauseScope}` : health.conflictCount ? "Needs review" : "Active"}</span>
        </div>
        <div className="health-grid">
          <div><span>Last local sync</span><strong>{health.lastLocalProtection ? safeDateTime(health.lastLocalProtection) : "Not yet"}</strong></div>
          <div><span>Cloud upload</span><strong>{health.lastCloudUpload ? safeDateTime(health.lastCloudUpload) : "Not yet"}</strong></div>
          <div><span>Dirty files</span><strong>{health.dirtyFiles}</strong></div>
          <div><span>Pending uploads</span><strong>{health.pendingCount} · {formatBytes(health.pendingBytes)}</strong></div>
        </div>
        {syncJob?.error ? <p className="banner error">{syncJob.error}</p> : null}
        {!syncJob?.error && provider?.state === "authentication_required" && provider.message ? <p className="banner error" role="alert">{provider.message}</p> : null}
      </section>

      <section className="panel" data-window-title="CLOUD.SYNC.POLICY">
        <h2>Cloud Sync policy</h2>
        <div className="settings-grid">
          <label className="inline-check">
            <input type="checkbox" checked={policy.enabled} onChange={(event) => setPolicy({ ...policy, enabled: event.target.checked })} />
            Cloud Sync enabled for this game
          </label>
          <label>
            Mode
            <select value={policy.mode} onChange={(event) => setPolicy({ ...policy, mode: event.target.value as SyncPolicy["mode"] })}>
              <option value="balanced">Balanced — every 10 minutes and exit</option>
              <option value="exit_only">Exit only — no reads during gameplay</option>
              <option value="continuous">Continuous — after quiet writes, at least 2 minutes</option>
              <option value="manual">Manual only</option>
            </select>
          </label>
          <label>
            Checkpoint interval (minutes)
            <input type="number" min={2} max={1440} disabled={policy.mode === "exit_only" || policy.mode === "manual"} value={policy.checkpointIntervalMinutes} onChange={(event) => setPolicy({ ...policy, checkpointIntervalMinutes: Number(event.target.value) })} />
          </label>
          <label className="inline-check">
            <input type="checkbox" checked={policy.exitProtection} onChange={(event) => setPolicy({ ...policy, exitProtection: event.target.checked })} />
            Sync after game exit
          </label>
        </div>
        <div className="detail-actions">
          <button className="btn primary" disabled={saving} onClick={async () => { setSaving(true); await onSavePolicy(policy); setSaving(false); }} type="button">{saving ? "Saving…" : "Save policy"}</button>
          <button className="btn" onClick={() => void onPause("uploads", "manual")} type="button">Pause uploads</button>
          <button className="btn" onClick={() => void onPause("all", "one_hour")} type="button">Pause all for one hour</button>
          <button className="btn" onClick={() => void onPause("all", "until_login")} type="button">Pause until next login</button>
          <button className="btn" disabled={!paused} onClick={() => void onResume()} type="button">Resume</button>
        </div>
      </section>

      <section className="panel" data-window-title="RECOVERY.POINTS">
        <div className="panel-head"><h2>Recovery Points</h2><span className="state-text success">{recoveryPoints.length} pinned</span></div>
        {recoveryPoints.length === 0 ? <p className="empty-state">No pinned Recovery Points yet. Pinned points are never removed automatically.</p> : (
          <ul className="snapshot-list">{recoveryPoints.map((point) => (
            <li className="snapshot-row" key={point.id}>
              <div className="snapshot-main"><p><strong>{point.name}</strong></p><p className="snapshot-meta">{safeDateTime(point.createdAt)} · {point.fileCount} files · {formatBytes(point.totalBytes)}</p></div>
              <div className="snapshot-actions">
                <button className="btn" onClick={() => onRestoreRevision(point.revisionId)} type="button">Restore</button>
                <button className="btn danger" onClick={() => onDeleteRecoveryPoint(point.id)} type="button">Delete pin</button>
              </div>
            </li>
          ))}</ul>
        )}
      </section>

      <section className="panel" data-window-title="REVISION.HISTORY">
        <div className="panel-head">
          <h2>Automatic history</h2>
          <span className="state-text">{storageUsage ? `${formatBytes(storageUsage.uniqueBytes)} unique` : "Calculating"}</span>
        </div>
        {storageUsage ? <p className="snapshot-meta">{storageUsage.automaticCount} automatic · {storageUsage.recoveryPoints} pinned · {formatBytes(storageUsage.pendingBytes)} pending upload</p> : null}
        {revisions.length === 0 ? <p className="empty-state">No unified revisions yet.</p> : (
          <ul className="snapshot-list">{revisions.slice(0, 20).map((revision) => (
            <li className="snapshot-row" key={revision.id}>
              <div className="snapshot-main"><p><strong>{revision.name || revision.kind.replace(/_/g, " ")}</strong></p><p className="snapshot-meta">{safeDateTime(revision.createdAt)} · {revision.fileCount} files · {formatBytes(revision.totalBytes)} · {revision.uploaded ? "cloud uploaded" : "local"}</p></div>
              <button className="btn" onClick={() => onRestoreRevision(revision.id)} type="button">Restore</button>
            </li>
          ))}</ul>
        )}
      </section>

      {legacySnapshots.length > 0 ? (
        <section className="panel" data-window-title="LEGACY.LOCAL.BACKUPS">
          <div className="panel-head">
            <h2>Legacy local backups</h2>
            <span className="state-text warning">Read-only format</span>
          </div>
          <p className="empty-state">These older snapshot-vault entries are preserved. New snapshots cannot be created; use Recovery Points instead.</p>
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
        <section className="panel" data-window-title="CONFLICT.REVIEW">
          <h2>Conflicts need review</h2>
          <p className="empty-state">Neither copy is overwritten until you choose.</p>
          <ul className="snapshot-list">{syncConflicts.map((conflict) => (
            <li className="snapshot-row" key={conflict.id}>
              <div className="snapshot-main"><p><strong>{conflict.path}</strong></p><p className="snapshot-meta">{safeDateTime(conflict.createdAt)} · local {conflict.localHash.slice(0, 8)} · remote {conflict.remoteHash.slice(0, 8)}</p></div>
              <div className="snapshot-actions">
                <button className="btn" onClick={() => onResolveSyncConflict(conflict.id, "local")} type="button">Use local</button>
                <button className="btn" onClick={() => onResolveSyncConflict(conflict.id, "remote")} type="button">Use remote</button>
                <button className="btn primary" onClick={() => onResolveSyncConflict(conflict.id, "both")} type="button">Keep both</button>
              </div>
            </li>
          ))}</ul>
        </section>
      ) : null}

      <section className="panel split" data-window-title="MANUAL.ZIP.UTILITY">
        <div className="backup-pane">
          <h2>Export ZIP</h2>
          <input className="picker-input" value={exportPath} readOnly onClick={onBrowseExportPath} placeholder="Choose destination folder" />
          <button className="btn primary" onClick={onExportBackup} disabled={exportOperationState === "working"} type="button">{exportOperationState === "working" ? "Exporting…" : "Export"}</button>
          {exportOperationState !== "idle" ? <p className="snapshot-meta">{exportOperationMessage}</p> : null}
        </div>
        <div className="backup-pane">
          <h2>Import ZIP</h2>
          <input className="picker-input" value={importPath} readOnly onClick={onBrowseImportPath} placeholder="Choose ZIP file" />
          <label className="inline-check"><input type="checkbox" checked={mergeOnImport} onChange={(event) => onMergeChange(event.target.checked)} />Merge into existing folder</label>
          <button className="btn primary" onClick={onImportBackup} disabled={importOperationState === "working"} type="button">{importOperationState === "working" ? "Importing…" : "Import"}</button>
          {importOperationState !== "idle" ? <p className="snapshot-meta">{importOperationMessage}</p> : null}
        </div>
      </section>

      <section className="panel" data-window-title="COVER.ART">
        <h2>Cover image</h2>
        <div className="detail-actions"><label className="file-input-wrap"><span><ImagePlus aria-hidden="true" size={15} /> Upload image</span><input type="file" accept="image/*" onChange={(event) => {
          const file = event.target.files?.[0];
          if (!file) return;
          const reader = new FileReader();
          reader.onload = () => { if (typeof reader.result === "string") onUpdateCover(reader.result); };
          reader.readAsDataURL(file);
        }} /></label><button className="btn" type="button" onClick={() => onUpdateCover("", "executable_icon")}><MonitorUp aria-hidden="true" size={15} />Use executable icon</button><button className="btn" type="button" onClick={() => onUpdateCover("", "generated")}><Palette aria-hidden="true" size={15} />Use generated cover</button></div>
      </section>
    </>
  );
}
