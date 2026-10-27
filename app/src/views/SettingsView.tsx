import { useRef, useState } from "react";
import { Save } from "lucide-react";
import type { ActivityEvent, DriveFolder, PauseScope, ProtectionTarget, Settings, ThemeMode } from "../types";
import type { AppUpdateState } from "../update";
import { DebugConsoleView } from "./DebugConsoleView";

type SettingsViewProps = {
  settings: Settings;
  activeProtectionTarget: ProtectionTarget;
  onChange: (next: Settings) => void;
  onThemeModeChange: (value: ThemeMode) => void;
  onTargetChange: (value: ProtectionTarget) => void;
  onConnectGoogleDrive: () => Promise<void>;
  onDisconnectGoogleDrive: () => Promise<void>;
  onBrowseSyncRoot: () => void;
  onAutostartChange: (value: boolean) => Promise<void>;
  onPause: (scope: PauseScope, duration: "manual" | "one_hour" | "until_login") => Promise<void>;
  onResume: () => Promise<void>;
  onProtectionEnabled: (enabled: boolean) => Promise<void>;
  onRetry: () => Promise<void>;
  onSaveSettings: () => void;
  onSaveDestination: () => void;
  updateState: AppUpdateState | null;
  updateInstallBlocked: boolean;
  onCheckForUpdate: () => void;
  onDownloadUpdate: () => void;
  onInstallUpdate: () => void;
  onDiscardUpdate: () => void;
  onAutomaticUpdateChecks: (enabled: boolean) => void;
  onViewUpdateNotes: () => void;
  onDownloadLogs: () => void;
  activityEvents: ActivityEvent[];
  safeTime: (input: string) => string;
  // Folder picker
  onListDriveFolders: () => Promise<DriveFolder[]>;
  onSetDriveFolder: (id: string, name: string) => Promise<void>;
  onCreateDriveFolder: (name: string) => Promise<void>;
  // Settings import/export
  onExportSettings: () => Promise<void>;
  onImportSettingsFile: (parsed: unknown) => Promise<void>;
};

export function SettingsView(props: SettingsViewProps) {
  const {
    settings, activeProtectionTarget, onChange, onThemeModeChange, onTargetChange, onConnectGoogleDrive,
    onDisconnectGoogleDrive, onBrowseSyncRoot, onAutostartChange, onPause,
    onResume, onProtectionEnabled, onRetry, onSaveSettings, onSaveDestination, updateState,
    updateInstallBlocked, onCheckForUpdate, onDownloadUpdate, onInstallUpdate,
    onDiscardUpdate, onAutomaticUpdateChecks, onViewUpdateNotes, onDownloadLogs, activityEvents, safeTime,
    onListDriveFolders, onSetDriveFolder, onCreateDriveFolder, onExportSettings, onImportSettingsFile,
  } = props;
  const [googleBusy, setGoogleBusy] = useState(false);
  const [folderPickerOpen, setFolderPickerOpen] = useState(false);
  const [driveFolders, setDriveFolders] = useState<DriveFolder[]>([]);
  const [folderLoading, setFolderLoading] = useState(false);
  const [newFolderName, setNewFolderName] = useState("");
  const [folderError, setFolderError] = useState("");
  const [settingsImportError, setSettingsImportError] = useState("");
  const importFileRef = useRef<HTMLInputElement>(null);

  async function runGoogle(action: () => Promise<void>) {
    setGoogleBusy(true);
    try { await action(); } finally { setGoogleBusy(false); }
  }

  async function openFolderPicker() {
    setFolderError("");
    setFolderLoading(true);
    setFolderPickerOpen(true);
    try {
      const folders = await onListDriveFolders();
      setDriveFolders(folders);
    } catch (e) {
      setFolderError(e instanceof Error ? e.message : "Could not load folders");
    } finally {
      setFolderLoading(false);
    }
  }

  async function selectFolder(id: string, name: string) {
    setFolderError("");
    try {
      await onSetDriveFolder(id, name);
      setFolderPickerOpen(false);
    } catch (e) {
      setFolderError(e instanceof Error ? e.message : "Could not set folder");
    }
  }

  async function createFolder() {
    if (!newFolderName.trim()) return;
    setFolderError("");
    setFolderLoading(true);
    try {
      await onCreateDriveFolder(newFolderName.trim());
      setNewFolderName("");
      setFolderPickerOpen(false);
    } catch (e) {
      setFolderError(e instanceof Error ? e.message : "Could not create folder");
    } finally {
      setFolderLoading(false);
    }
  }

  function handleImportFile(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = async () => {
      try {
        const parsed = JSON.parse(reader.result as string);
        await props.onImportSettingsFile(parsed);
        setSettingsImportError("");
      } catch (err) {
        setSettingsImportError(err instanceof SyntaxError ? "That file is not a valid Rove settings file." : err instanceof Error ? err.message : "Could not import settings.");
      }
    };
    reader.readAsText(file);
  }

  const paused = settings.serviceControl.pauseScope !== "none";
  const targetName = (target: ProtectionTarget) => target === "google_drive" ? "Google Drive" : target === "folder" ? "Synced folder" : "Local only";

  return (
    <>
      <section className="panel" data-window-title="SAVE.LOCATION">
        <h2>Save location</h2>
        <p className="empty-state">Choose where Rove keeps backup copies of your game saves.</p>
        <div className="provider-picker provider-picker-three" aria-label="Save location">
          <button className={`provider-card ${settings.protectionTarget === "local" ? "selected" : ""}`} onClick={() => onTargetChange("local")} type="button">
            <strong>This computer</strong><span>Keep backup copies on this PC.</span>
          </button>
          <button className={`provider-card ${settings.protectionTarget === "folder" ? "selected" : ""}`} onClick={() => onTargetChange("folder")} type="button">
            <strong>A synced folder</strong><span>Use a folder managed by another app.</span>
          </button>
          <button className={`provider-card ${settings.protectionTarget === "google_drive" ? "selected" : ""}`} onClick={() => onTargetChange("google_drive")} type="button">
            <strong>Google Drive</strong><span>Keep an extra copy in Google Drive.</span>
          </button>
        </div>
        <p className="snapshot-meta" role="status">
          Current save location: {targetName(activeProtectionTarget)}.
          {settings.protectionTarget !== activeProtectionTarget ? ` Save settings to switch to ${targetName(settings.protectionTarget)}.` : ""}
        </p>

        {settings.protectionTarget === "google_drive" ? (
          <div className="provider-status" aria-live="polite">
            <p>{settings.googleDrive.connected && settings.googleDrive.ready !== false ? `Connected to ${settings.googleDrive.accountEmail || "Google Drive"}.` : "Connect Google Drive to keep an extra copy of your saves."}</p>
			{settings.googleDrive.configurationError ? <p className="banner warning" role="alert">Google Drive needs attention. Reconnect to continue cloud backups.</p> : null}
            {settings.googleDrive.connected && (
              <p className="snapshot-meta">
                Save folder: <strong>{settings.googleDrive.folderName || "My Drive (root)"}</strong>
                {" "}<button className="btn" type="button" onClick={() => void openFolderPicker()}>Change folder</button>
              </p>
            )}
            {folderPickerOpen && (
              <div className="folder-picker" role="dialog" aria-label="Choose Drive folder">
                <div className="folder-picker-head">
                  <strong>Choose a Drive folder</strong>
                  <button className="btn" type="button" onClick={() => { setFolderPickerOpen(false); setFolderError(""); }}>Cancel</button>
                </div>
                {folderError ? <p className="banner error" role="alert">{folderError}</p> : null}
                {folderLoading ? <p className="snapshot-meta">Loading folders…</p> : (
                  <ul className="folder-list">
                    <li><button className="btn" type="button" onClick={() => void selectFolder("root", "My Drive (root)")}>My Drive (root)</button></li>
                    {driveFolders.map((folder) => (
                      <li key={folder.id}>
                        <button className="btn" type="button" onClick={() => void selectFolder(folder.id, folder.name)}>{folder.name}</button>
                      </li>
                    ))}
                  </ul>
                )}
                <div className="folder-create">
                  <input
                    type="text"
                    placeholder="New folder name"
                    value={newFolderName}
                    onChange={(e) => setNewFolderName(e.target.value)}
                    onKeyDown={(e) => { if (e.key === "Enter") void createFolder(); }}
                  />
                  <button className="btn primary" type="button" disabled={!newFolderName.trim() || folderLoading} onClick={() => void createFolder()}>
                    Create &amp; select
                  </button>
                </div>
              </div>
            )}
            <div className="detail-actions">
              <button className="btn primary" disabled={googleBusy} onClick={() => void runGoogle(onConnectGoogleDrive)} type="button">{googleBusy ? "Waiting for sign-in…" : settings.googleDrive.connected ? "Reconnect" : "Connect Google Drive"}</button>
              {settings.googleDrive.connected ? <button className="btn" disabled={googleBusy} onClick={() => void runGoogle(onDisconnectGoogleDrive)} type="button">Disconnect</button> : null}
            </div>
          </div>
        ) : null}

        {settings.protectionTarget === "folder" ? (
          <label>
            Provider-synced folder
            <input className="picker-input" value={settings.syncRoot} onClick={onBrowseSyncRoot} readOnly placeholder="Choose a folder shared between devices" />
          </label>
        ) : null}
        {settings.protectionTarget !== activeProtectionTarget ? <button className="btn primary destination-save" type="button" onClick={onSaveDestination}><Save aria-hidden="true" size={15} /> Save destination</button> : null}
      </section>

      <section className="panel" data-window-title="BACKUPS">
        <div className="panel-head">
          <h2>Backups</h2>
          <span className={`state-text ${paused ? "warning" : "success"}`}>{paused ? "Paused" : "On"}</span>
        </div>
        <label className="inline-check">
          <input
            type="checkbox"
            checked={settings.serviceControl.protectionEnabled}
            onChange={(event) => void onProtectionEnabled(event.target.checked)}
          />
          Protect my game saves
        </label>
        <label className="inline-check">
          <input type="checkbox" checked={settings.autostart} onChange={(event) => void onAutostartChange(event.target.checked)} />
          Start Rove when I sign in to Windows
        </label>
        <details className="advanced-disclosure settings-disclosure">
          <summary>Advanced backup options</summary>
          <div className="detail-actions">
            <button className="btn" type="button" onClick={() => void onPause("uploads", "manual")}>Pause cloud backups</button>
            <button className="btn" type="button" onClick={() => void onPause("all", "one_hour")}>Pause for one hour</button>
            <button className="btn" type="button" onClick={() => void onPause("all", "until_login")}>Pause until I restart Rove</button>
            <button className="btn primary" type="button" disabled={!paused} onClick={() => void onResume()}>Resume backups</button>
            <button className="btn" type="button" onClick={() => void onRetry()}>Try connection again</button>
          </div>
        </details>
      </section>

      <section className="panel advanced-setup-panel" data-window-title="RECENT.ACTIVITY">
        <details className="advanced-disclosure settings-disclosure">
          <summary>Recent activity</summary>
          <p className="disclosure-help">Your latest backups and restores.</p>
          {activityEvents.length === 0 ? (
            <p className="empty-state">Your backup and restore activity will appear here.</p>
          ) : (
            <ul className="log-feed">
              {activityEvents.map((entry) => (
                <li className="log-row" key={entry.id}>
                  <div className="log-meta"><span>{safeTime(entry.timestamp)}</span></div>
                  <p><strong>{entry.title}</strong></p>
                  <p>{entry.detail}</p>
                </li>
              ))}
            </ul>
          )}
        </details>
      </section>

      <section className="panel advanced-setup-panel" data-window-title="APP.SUPPORT">
        <details className="advanced-disclosure settings-disclosure">
          <summary>Advanced app & support</summary>
          <p className="disclosure-help">Updates, appearance, and troubleshooting tools.</p>
          <div className="advanced-section">
        <div className="panel-head">
          <h2>Application updates</h2>
          <span className={`state-text ${updateState?.lifecycle === "error" ? "warning" : updateState?.lifecycle === "ready" ? "success" : ""}`}>
            {(updateState?.lifecycle ?? "unavailable").replace(/_/g, " ")}
          </span>
        </div>
        <p className="empty-state">
          Signed updates replace the installed application in place. Saves, settings, local revisions, and provider credentials remain in your application-data folder.
        </p>
        {updateState ? (
          <div className="settings-grid update-settings-grid">
            <div><span className="snapshot-meta">Installed version</span><strong>{updateState.currentVersion}</strong></div>
            <div><span className="snapshot-meta">Available version</span><strong>{updateState.availableVersion ?? "Up to date"}</strong></div>
            <div><span className="snapshot-meta">Last successful check</span><strong>{updateState.lastCheckedAt ? new Date(updateState.lastCheckedAt * 1000).toLocaleString() : "Not checked yet"}</strong></div>
            <label className="inline-check">
              <input
                type="checkbox"
                checked={updateState.automaticChecksEnabled}
                disabled={updateState.development}
                onChange={(event) => onAutomaticUpdateChecks(event.target.checked)}
              />
              Check automatically every six hours
            </label>
          </div>
        ) : <p className="empty-state">Native update status is unavailable in browser preview mode.</p>}
        {updateState?.notes ? <div className="update-notes" aria-label="Release notes">{updateState.notes}</div> : null}
        {updateState?.lifecycle === "downloading" ? (
          <div className="update-progress" aria-live="polite">
            <progress max={updateState.totalBytes ?? Math.max(1, updateState.downloadedBytes)} value={updateState.downloadedBytes} />
            <span>{formatUpdateBytes(updateState.downloadedBytes)}{updateState.totalBytes ? ` of ${formatUpdateBytes(updateState.totalBytes)}` : " downloaded"}</span>
          </div>
        ) : null}
        {updateState?.error ? <p className="banner error" role="alert">{updateState.error}</p> : null}
        {updateInstallBlocked && updateState?.lifecycle === "ready" ? <p className="snapshot-meta">Close tracked games and wait for active sync work to finish before installing.</p> : null}
        {updateState?.development ? <p className="snapshot-meta">Installation is disabled in source/development builds. Release builds embed the public signing key.</p> : null}
        <div className="detail-actions">
          <button className="btn" type="button" disabled={updateState?.lifecycle === "checking" || updateState?.lifecycle === "downloading" || updateState?.development} onClick={onCheckForUpdate}>Check now</button>
          {updateState?.lifecycle === "available" ? <button className="btn primary" type="button" onClick={onDownloadUpdate}>Download update</button> : null}
          {updateState?.lifecycle === "ready" ? <button className="btn primary" type="button" disabled={updateInstallBlocked} onClick={onInstallUpdate}>Install v{updateState.availableVersion}</button> : null}
          {updateState?.lifecycle === "ready" ? <button className="btn" type="button" onClick={onViewUpdateNotes}>What's new</button> : null}
          {updateState?.lifecycle === "ready" ? <button className="btn" type="button" onClick={onDiscardUpdate}>Discard download</button> : null}
          {updateState?.lifecycle === "error" && updateState.retryable ? <button className="btn primary" type="button" onClick={onCheckForUpdate}>Retry</button> : null}
        </div>
          </div>
        </details>
      </section>

      <section className="panel advanced-setup-panel" data-window-title="APP.SETTINGS">
        <details className="advanced-disclosure settings-disclosure">
          <summary>More settings</summary>
          <div className="advanced-section">
        <h2>Defaults and behavior</h2>
        <div className="settings-grid">
          <label>
            Desktop appearance
            <select value={settings.themeMode === "dark" ? "dark" : "light"} onChange={(event) => onThemeModeChange(event.target.value as ThemeMode)}>
              <option value="light">Light desktop (default)</option>
              <option value="dark">CRT desktop</option>
            </select>
          </label>
          <label className="inline-check">
            <input type="checkbox" checked={settings.automaticUpload} onChange={(event) => onChange({ ...settings, automaticUpload: event.target.checked })} />
            Upload automatically after reconnecting
          </label>
          <label className="inline-check">
            <input type="checkbox" checked={settings.remoteCheckBeforeLaunch} onChange={(event) => onChange({ ...settings, remoteCheckBeforeLaunch: event.target.checked })} />
            Check provider before Rove launch
          </label>
          <label>
            Automatic revisions to keep
            <input type="number" min={1} max={50} value={settings.autoRevisionRetention} onChange={(event) => onChange({ ...settings, autoRevisionRetention: Number(event.target.value) })} />
          </label>
          <label>
            Ignore patterns
            <input value={settings.ignorePatterns.join(",")} onChange={(event) => onChange({ ...settings, ignorePatterns: event.target.value.split(",").map((item) => item.trim()).filter(Boolean) })} />
          </label>
          <label className="inline-check">
            <input
              type="checkbox"
              checked={settings.debugLogging}
              onChange={(event) => onChange({ ...settings, debugLogging: event.target.checked })}
            />
            Enable debug logging
          </label>
        </div>
        <div className="detail-actions">
          <button className="btn primary" onClick={onSaveSettings} type="button">Save settings</button>
          <button className="btn" type="button" onClick={() => void onExportSettings()}>Export setup</button>
          <button className="btn" type="button" onClick={() => importFileRef.current?.click()}>Import setup</button>
        </div>
        <input
          ref={importFileRef}
          type="file"
          accept=".json,application/json"
          style={{ display: "none" }}
          onChange={handleImportFile}
        />
        {settingsImportError ? <p className="banner error" role="alert">{settingsImportError}</p> : null}
        {settings.debugLogging ? <DebugConsoleView onDownloadLogs={onDownloadLogs} /> : null}
          </div>
        </details>
      </section>
    </>
  );
}

function formatUpdateBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
