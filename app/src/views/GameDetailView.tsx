import { useEffect, useState } from "react";
import { ImagePlus, Link2, MonitorUp, Palette, Play, RefreshCw } from "lucide-react";
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

type Tab = "saves" | "options" | "transfer" | "details";
const TABS: Array<{ id: Tab; label: string }> = [
  { id: "saves", label: "Saves" },
  { id: "options", label: "Backup options" },
  { id: "transfer", label: "Import & export" },
  { id: "details", label: "Details" },
];

export function GameDetailView(props: GameDetailViewProps) {
  const {
    game, exportPath, importPath, mergeOnImport, exportOperationState,
    exportOperationMessage, importOperationState, importOperationMessage,
    revisions, recoveryPoints, storageUsage, legacySnapshots, onPlay, onOpenSettings, onCreateShortcut, onProtect, onDelete,
    onBrowseExportPath, onBrowseImportPath, onMergeChange, onExportBackup,
    onImportBackup, onSavePolicy, onPause, onResume, onCreateRecoveryPoint,
    onRestoreRevision, onDeleteRecoveryPoint, onRestoreLegacySnapshot,
    onDeleteLegacySnapshot, onUpdateCover, formatPlaytime,
    safeDateTime, formatBytes, syncJob, provider, syncConflicts, onResolveSyncConflict,
  } = props;
  const [policy, setPolicy] = useState<SyncPolicy>(game?.protectionPolicy ?? newDefaultSyncPolicy());
  const [saving, setSaving] = useState(false);
  const [tab, setTab] = useState<Tab>("saves");

  useEffect(() => {
    setPolicy(game?.protectionPolicy ?? newDefaultSyncPolicy());
  }, [game?.id, game?.protectionPolicy]);
  useEffect(() => { setTab("saves"); }, [game?.id]);

  if (!game) {
    return <div className="pane"><div className="empty"><h2>Game not found</h2><p>It may have been removed. Pick another game from the library.</p></div></div>;
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
  const pillTone = saveStatus.tone === "success" ? "ok" : saveStatus.tone === "syncing" ? "busy" : saveStatus.tone === "error" ? "err" : "warn";
  const heroCover = game.coverImageData ? { backgroundImage: `url(${game.coverImageData})` } : undefined;

  return (
    <>
      <header className="hero">
        {heroCover ? <div className="hero-bg" style={heroCover} aria-hidden="true" /> : null}
        <div className="hero-inner">
          <div className="hero-cover"><GameCover game={game} /></div>
          <div className="hero-body">
            <h1>{game.name}</h1>
            <p className="sub">{[game.version ? `Version ${game.version}` : "", `${formatPlaytime(game.playtimeSeconds ?? 0)} played`, game.running ? "Playing now" : ""].filter(Boolean).join(" · ")}</p>
            <div className="row"><span className={`pill ${pillTone}`} role="status">{saveStatus.label}</span></div>
            <div className="hero-actions">
              <button className="btn primary lg" onClick={onPlay} type="button"><Play aria-hidden="true" size={16} /> Play</button>
              <button className="btn lg" onClick={onProtect} disabled={Boolean(active)} type="button"><RefreshCw aria-hidden="true" size={15} /> {active ? "Backing up…" : "Back up now"}</button>
            </div>
          </div>
        </div>
      </header>

      <div className="game-tabs">
        <div className="tabs" role="tablist" aria-label="Game sections">
          {TABS.map((item) => (
            <button key={item.id} className="tab" role="tab" type="button" aria-selected={tab === item.id} onClick={() => setTab(item.id)}>{item.label}{item.id === "saves" && syncConflicts.length > 0 ? ` (${syncConflicts.length})` : ""}</button>
          ))}
        </div>
      </div>

      <div className="pane" role="tabpanel">
        {saveStatus.action !== "none" ? (
          <div className="banner warning" role="alert">
            <span><strong>{saveStatus.label}.</strong> {saveStatus.detail}</span>
            {saveStatus.action === "review" ? <button className="btn sm" onClick={() => setTab("saves")} type="button">Review</button> : <button className="btn sm" onClick={onOpenSettings} type="button">Open Settings</button>}
          </div>
        ) : null}
        {syncJob?.error ? <p className="banner error">{syncJob.error}</p> : null}
        {!syncJob?.error && provider?.state === "authentication_required" && provider.message ? <p className="banner error" role="alert">{provider.message}</p> : null}

        {tab === "saves" ? (
          <>
            <div className="stat-grid">
              <div className="stat"><span>Last backup</span><strong>{health.lastLocalProtection ? safeDateTime(health.lastLocalProtection) : "Not yet"}</strong></div>
              <div className="stat"><span>Cloud backup</span><strong>{health.lastCloudUpload ? safeDateTime(health.lastCloudUpload) : "Not yet"}</strong></div>
              <div className="stat"><span>Changed files</span><strong>{health.dirtyFiles}</strong></div>
              <div className="stat"><span>Waiting to upload</span><strong>{health.pendingCount} · {formatBytes(health.pendingBytes)}</strong></div>
            </div>

            {syncConflicts.length > 0 ? (
              <section className="section" aria-label="Choose a saved version">
                <div className="section-head"><h2>Choose a saved version</h2><p>Nothing is replaced until you choose.</p></div>
                <div className="list">{syncConflicts.map((conflict) => (
                  <div className="list-row" key={conflict.id}>
                    <div className="meta"><strong>{conflict.path}</strong><span>{safeDateTime(conflict.createdAt)} · local {conflict.localHash.slice(0, 8)} · cloud {conflict.remoteHash.slice(0, 8)}</span></div>
                    <div className="actions">
                      <button className="btn sm" onClick={() => onResolveSyncConflict(conflict.id, "local")} type="button">Use this PC</button>
                      <button className="btn sm" onClick={() => onResolveSyncConflict(conflict.id, "remote")} type="button">Use cloud copy</button>
                      <button className="btn sm primary" onClick={() => onResolveSyncConflict(conflict.id, "both")} type="button">Keep both</button>
                    </div>
                  </div>
                ))}</div>
              </section>
            ) : null}

            <section className="section" aria-label="Restore points">
              <div className="section-head"><h2>Restore points</h2><button className="btn sm" onClick={onCreateRecoveryPoint} disabled={Boolean(active)} type="button">Create restore point</button></div>
              {recoveryPoints.length === 0 ? <p className="empty-note">No restore points yet. Create one before a mod, a risky boss or a game update.</p> : (
                <div className="list">{recoveryPoints.map((point) => (
                  <div className="list-row" key={point.id}>
                    <div className="meta"><strong>{point.name}</strong><span>{safeDateTime(point.createdAt)} · {point.fileCount} files · {formatBytes(point.totalBytes)}</span></div>
                    <div className="actions">
                      <button className="btn sm" onClick={() => onRestoreRevision(point.revisionId)} type="button">Restore</button>
                      <button className="btn sm danger" onClick={() => onDeleteRecoveryPoint(point.id)} type="button">Remove</button>
                    </div>
                  </div>
                ))}</div>
              )}
            </section>

            <section className="section" aria-label="Previous saves">
              <div className="section-head">
                <h2>Previous saves</h2>
                <p>{storageUsage ? `${formatBytes(storageUsage.uniqueBytes)} stored · ${storageUsage.automaticCount} automatic · ${storageUsage.recoveryPoints} pinned` : "Calculating…"}</p>
              </div>
              {revisions.length === 0 ? <p className="empty-note">No previous saves yet.</p> : (
                <div className="list">{revisions.slice(0, 20).map((revision) => (
                  <div className="list-row" key={revision.id}>
                    <div className="meta"><strong>{revision.name || revision.kind.replace(/_/g, " ")}</strong><span>{safeDateTime(revision.createdAt)} · {revision.fileCount} files · {formatBytes(revision.totalBytes)} · {revision.uploaded ? "in the cloud" : "on this PC"}</span></div>
                    <button className="btn sm" onClick={() => onRestoreRevision(revision.id)} type="button">Restore</button>
                  </div>
                ))}</div>
              )}
            </section>

            {legacySnapshots.length > 0 ? (
              <section className="section" aria-label="Older local backups">
                <div className="section-head"><h2>Older local backups</h2><p>Still available to restore.</p></div>
                <div className="list">{legacySnapshots.map((snapshot) => (
                  <div className="list-row" key={snapshot.id}>
                    <div className="meta"><strong>{safeDateTime(snapshot.createdAt)}</strong><span>{snapshot.changedFiles} changed files · {formatBytes(snapshot.totalSize)} · {snapshot.trigger}</span></div>
                    <div className="actions">
                      <button className="btn sm" type="button" onClick={() => onRestoreLegacySnapshot(snapshot.id)}>Restore</button>
                      <button className="btn sm danger" type="button" onClick={() => onDeleteLegacySnapshot(snapshot.id)}>Remove</button>
                    </div>
                  </div>
                ))}</div>
              </section>
            ) : null}
          </>
        ) : null}

        {tab === "options" ? (
          <section className="section" aria-label="Backup options">
            <div className="form-grid">
              <label className="field"><span>Backup schedule</span>
                <select value={policy.mode} onChange={(event) => setPolicy({ ...policy, mode: event.target.value as SyncPolicy["mode"] })}>
                  <option value="balanced">Balanced — every 10 minutes and exit</option>
                  <option value="exit_only">Exit only — no reads during gameplay</option>
                  <option value="continuous">Continuous — after quiet writes, at least 2 minutes</option>
                  <option value="manual">Manual only</option>
                </select>
              </label>
              <label className="field"><span>Minutes between backups</span>
                <input type="number" min={2} max={1440} disabled={policy.mode === "exit_only" || policy.mode === "manual"} value={policy.checkpointIntervalMinutes} onChange={(event) => setPolicy({ ...policy, checkpointIntervalMinutes: Number(event.target.value) })} />
              </label>
            </div>
            <label className="check"><input type="checkbox" checked={policy.enabled} onChange={(event) => setPolicy({ ...policy, enabled: event.target.checked })} /> Back up saves for this game</label>
            <label className="check"><input type="checkbox" checked={policy.exitProtection} onChange={(event) => setPolicy({ ...policy, exitProtection: event.target.checked })} /> Back up when the game closes</label>
            <div className="row">
              <button className="btn primary" disabled={saving} onClick={async () => { setSaving(true); await onSavePolicy(policy); setSaving(false); }} type="button">{saving ? "Saving…" : "Save backup options"}</button>
            </div>
            <div className="section-head"><h2>Pause</h2><p>{paused ? `Paused: ${health.serviceControl.pauseScope}` : "Backups are running"}</p></div>
            <div className="row">
              <button className="btn" onClick={() => void onPause("uploads", "manual")} type="button">Pause cloud backups</button>
              <button className="btn" onClick={() => void onPause("all", "one_hour")} type="button">Pause for one hour</button>
              <button className="btn" disabled={!paused} onClick={() => void onResume()} type="button">Resume backups</button>
            </div>
          </section>
        ) : null}

        {tab === "transfer" ? (
          <section className="split" aria-label="Import and export">
            <div>
              <h2>Export saves</h2>
              <p className="hint">Save a ZIP copy of this game’s saves anywhere you like.</p>
              <input className="picker" value={exportPath} readOnly onClick={onBrowseExportPath} placeholder="Choose a destination folder…" aria-label="Export destination" />
              <div className="row"><button className="btn primary" onClick={onExportBackup} disabled={exportOperationState === "working"} type="button">{exportOperationState === "working" ? "Exporting…" : "Export"}</button></div>
              {exportOperationState !== "idle" ? <p className="hint">{exportOperationMessage}</p> : null}
            </div>
            <div>
              <h2>Import saves</h2>
              <p className="hint">Restore from a ZIP. Rove makes a restore point first.</p>
              <input className="picker" value={importPath} readOnly onClick={onBrowseImportPath} placeholder="Choose a backup file…" aria-label="Backup file" />
              <label className="check"><input type="checkbox" checked={mergeOnImport} onChange={(event) => onMergeChange(event.target.checked)} /> Keep existing files</label>
              <div className="row"><button className="btn primary" onClick={onImportBackup} disabled={importOperationState === "working"} type="button">{importOperationState === "working" ? "Importing…" : "Import"}</button></div>
              {importOperationState !== "idle" ? <p className="hint">{importOperationMessage}</p> : null}
            </div>
          </section>
        ) : null}

        {tab === "details" ? (
          <>
            <section className="section" aria-label="Game details">
              <dl className="detail-list">
                {game.version ? <><dt>Version</dt><dd>{game.version}</dd></> : null}
                <dt>Save folder</dt><dd>{game.path}</dd>
                <dt>Executable</dt><dd>{game.executablePath}</dd>
              </dl>
              <div className="row"><button className="btn" onClick={onCreateShortcut} type="button"><Link2 aria-hidden="true" size={15} /> Create desktop shortcut</button></div>
            </section>
            <section className="section" aria-label="Cover image">
              <div className="section-head"><h2>Cover image</h2></div>
              <div className="row">
                <label className="btn file-btn"><ImagePlus aria-hidden="true" size={15} /> Upload image<input type="file" accept="image/*" onChange={(event) => {
                  const file = event.target.files?.[0];
                  if (!file) return;
                  const reader = new FileReader();
                  reader.onload = () => { if (typeof reader.result === "string") onUpdateCover(reader.result); };
                  reader.readAsDataURL(file);
                }} /></label>
                <button className="btn" type="button" onClick={() => onUpdateCover("", "executable_icon")}><MonitorUp aria-hidden="true" size={15} /> Use game icon</button>
                <button className="btn" type="button" onClick={() => onUpdateCover("", "generated")}><Palette aria-hidden="true" size={15} /> Use generated image</button>
              </div>
            </section>
            <section className="danger-zone" aria-label="Remove game">
              <h2>Remove game</h2>
              <p className="hint">This removes the game from Rove. It does not delete the game or its save files.</p>
              <div className="row"><button className="btn danger" onClick={onDelete} type="button">Remove game from Rove</button></div>
            </section>
          </>
        ) : null}
      </div>
    </>
  );
}
