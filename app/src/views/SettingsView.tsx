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

type Section = "location" | "backups" | "general" | "updates" | "activity" | "advanced";
const SECTIONS: Array<{ id: Section; label: string }> = [
  { id: "location", label: "Save location" },
  { id: "backups", label: "Backups" },
  { id: "general", label: "General" },
  { id: "updates", label: "Updates" },
  { id: "activity", label: "Activity" },
  { id: "advanced", label: "Advanced" },
];

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
  const [section, setSection] = useState<Section>("location");

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

  const saveButton = <div className="row"><button className="btn primary" onClick={onSaveSettings} type="button">Save settings</button></div>;

  return (
    <div className="prefs">
      <nav className="prefs-nav" aria-label="Settings sections">
        <h2>Settings</h2>
        {SECTIONS.map((item) => (
          <button key={item.id} className="side-item" type="button" aria-current={section === item.id ? "page" : undefined} onClick={() => setSection(item.id)}>{item.label}</button>
        ))}
      </nav>

      <div className="prefs-body">
        {section === "location" ? (
          <>
            <div className="section-head"><h2>Save location</h2></div>
            <p className="hint">Choose where Rove keeps backup copies of your game saves. You can change this any time.</p>
            <div className="choice-grid" role="radiogroup" aria-label="Save location">
              <button className="choice" role="radio" aria-checked={settings.protectionTarget === "local"} onClick={() => onTargetChange("local")} type="button">
                <strong>This computer</strong><span>Keep backup copies on this PC.</span>
              </button>
              <button className="choice" role="radio" aria-checked={settings.protectionTarget === "folder"} onClick={() => onTargetChange("folder")} type="button">
                <strong>A synced folder</strong><span>Use a folder managed by another app.</span>
              </button>
              <button className="choice" role="radio" aria-checked={settings.protectionTarget === "google_drive"} onClick={() => onTargetChange("google_drive")} type="button">
                <strong>Google Drive</strong><span>Keep an extra copy in Google Drive.</span>
              </button>
            </div>
            <p className="hint" role="status">
              Current save location: {targetName(activeProtectionTarget)}.
              {settings.protectionTarget !== activeProtectionTarget ? ` Save to switch to ${targetName(settings.protectionTarget)}.` : ""}
            </p>

            {settings.protectionTarget === "google_drive" ? (
              <div className="stack" aria-live="polite">
                <p>{settings.googleDrive.connected && settings.googleDrive.ready !== false ? `Connected to ${settings.googleDrive.accountEmail || "Google Drive"}.` : "Connect Google Drive to keep an extra copy of your saves."}</p>
                {settings.googleDrive.configurationError ? <p className="banner warning" role="alert">Google Drive needs attention. Reconnect to continue cloud backups.</p> : null}
                {settings.googleDrive.connected ? (
                  <div className="row">
                    <span className="hint">Save folder: <strong>{settings.googleDrive.folderName || "My Drive (root)"}</strong></span>
                    <button className="btn sm" type="button" onClick={() => void openFolderPicker()}>Change folder</button>
                  </div>
                ) : null}
                {folderPickerOpen ? (
                  <div className="folder-picker" role="dialog" aria-label="Choose Drive folder">
                    <div className="folder-picker-head">
                      <strong>Choose a Drive folder</strong>
                      <button className="btn sm" type="button" onClick={() => { setFolderPickerOpen(false); setFolderError(""); }}>Cancel</button>
                    </div>
                    {folderError ? <p className="banner error" role="alert">{folderError}</p> : null}
                    {folderLoading ? <p className="hint">Loading folders…</p> : (
                      <ul className="folder-list">
                        <li><button className="btn ghost sm" type="button" onClick={() => void selectFolder("root", "My Drive (root)")}>My Drive (root)</button></li>
                        {driveFolders.map((folder) => (
                          <li key={folder.id}><button className="btn ghost sm" type="button" onClick={() => void selectFolder(folder.id, folder.name)}>{folder.name}</button></li>
                        ))}
                      </ul>
                    )}
                    <div className="folder-create">
                      <input type="text" placeholder="New folder name" value={newFolderName} onChange={(e) => setNewFolderName(e.target.value)} onKeyDown={(e) => { if (e.key === "Enter") void createFolder(); }} />
                      <button className="btn primary" type="button" disabled={!newFolderName.trim() || folderLoading} onClick={() => void createFolder()}>Create &amp; select</button>
                    </div>
                  </div>
                ) : null}
                <div className="row">
                  <button className="btn primary" disabled={googleBusy} onClick={() => void runGoogle(onConnectGoogleDrive)} type="button">{googleBusy ? "Waiting for sign-in…" : settings.googleDrive.connected ? "Reconnect" : "Connect Google Drive"}</button>
                  {settings.googleDrive.connected ? <button className="btn" disabled={googleBusy} onClick={() => void runGoogle(onDisconnectGoogleDrive)} type="button">Disconnect</button> : null}
                </div>
              </div>
            ) : null}

            {settings.protectionTarget === "folder" ? (
              <label className="field"><span>Synced folder</span>
                <input className="picker" value={settings.syncRoot} onClick={onBrowseSyncRoot} readOnly placeholder="Choose a folder that OneDrive, Dropbox or similar keeps in sync…" />
              </label>
            ) : null}
            {settings.protectionTarget !== activeProtectionTarget ? <div className="row"><button className="btn primary" type="button" onClick={onSaveDestination}><Save aria-hidden="true" size={15} /> Save destination</button></div> : null}
          </>
        ) : null}

        {section === "backups" ? (
          <>
            <div className="section-head"><h2>Backups</h2><span className={`pill ${paused ? "warn" : "ok"}`}>{paused ? "Paused" : "On"}</span></div>
            <label className="check">
              <input type="checkbox" checked={settings.serviceControl.protectionEnabled} onChange={(event) => void onProtectionEnabled(event.target.checked)} />
              Protect my game saves
            </label>
            <div className="section-head"><h2>Pause and retry</h2></div>
            <div className="row">
              <button className="btn" type="button" onClick={() => void onPause("uploads", "manual")}>Pause cloud backups</button>
              <button className="btn" type="button" onClick={() => void onPause("all", "one_hour")}>Pause for one hour</button>
              <button className="btn" type="button" onClick={() => void onPause("all", "until_login")}>Pause until I restart Rove</button>
              <button className="btn" type="button" disabled={!paused} onClick={() => void onResume()}>Resume backups</button>
              <button className="btn" type="button" onClick={() => void onRetry()}>Try connection again</button>
            </div>
          </>
        ) : null}

        {section === "general" ? (
          <>
            <div className="section-head"><h2>General</h2></div>
            <label className="field"><span>Appearance</span>
              <select value={settings.themeMode} onChange={(event) => onThemeModeChange(event.target.value as ThemeMode)}>
                <option value="dark">Dark</option>
                <option value="light">Light</option>
                <option value="system">Match Windows</option>
              </select>
            </label>
            <label className="check">
              <input type="checkbox" checked={settings.autostart} onChange={(event) => void onAutostartChange(event.target.checked)} />
              Start Rove when I sign in to Windows
            </label>
            <label className="check">
              <input type="checkbox" checked={settings.automaticUpload} onChange={(event) => onChange({ ...settings, automaticUpload: event.target.checked })} />
              Upload automatically after reconnecting
            </label>
            <label className="check">
              <input type="checkbox" checked={settings.remoteCheckBeforeLaunch} onChange={(event) => onChange({ ...settings, remoteCheckBeforeLaunch: event.target.checked })} />
              Check for newer saves before launching a game from Rove
            </label>
            {saveButton}
          </>
        ) : null}

        {section === "updates" ? (
          <>
            <div className="section-head">
              <h2>Application updates</h2>
              <span className={`pill ${updateState?.lifecycle === "error" ? "warn" : updateState?.lifecycle === "ready" ? "ok" : ""}`}>{(updateState?.lifecycle ?? "unavailable").replace(/_/g, " ")}</span>
            </div>
            <p className="hint">Signed updates replace the installed app in place. Saves, settings and credentials stay in your application-data folder.</p>
            {updateState ? (
              <div className="stat-grid">
                <div className="stat"><span>Installed version</span><strong>{updateState.currentVersion}</strong></div>
                <div className="stat"><span>Available version</span><strong>{updateState.availableVersion ?? "Up to date"}</strong></div>
                <div className="stat"><span>Last successful check</span><strong>{updateState.lastCheckedAt ? new Date(updateState.lastCheckedAt * 1000).toLocaleString() : "Not checked yet"}</strong></div>
              </div>
            ) : <p className="empty-note">Native update status is unavailable in browser preview mode.</p>}
            {updateState ? (
              <label className="check">
                <input type="checkbox" checked={updateState.automaticChecksEnabled} disabled={updateState.development} onChange={(event) => onAutomaticUpdateChecks(event.target.checked)} />
                Check automatically every six hours
              </label>
            ) : null}
            {updateState?.notes ? <div className="update-notes" aria-label="Release notes">{updateState.notes}</div> : null}
            {updateState?.lifecycle === "downloading" ? (
              <div className="stack" aria-live="polite">
                <progress max={updateState.totalBytes ?? Math.max(1, updateState.downloadedBytes)} value={updateState.downloadedBytes} />
                <span className="hint">{formatUpdateBytes(updateState.downloadedBytes)}{updateState.totalBytes ? ` of ${formatUpdateBytes(updateState.totalBytes)}` : " downloaded"}</span>
              </div>
            ) : null}
            {updateState?.error ? <p className="banner error" role="alert">{updateState.error}</p> : null}
            {updateInstallBlocked && updateState?.lifecycle === "ready" ? <p className="hint">Close tracked games and wait for active sync work to finish before installing.</p> : null}
            {updateState?.development ? <p className="hint">Installation is disabled in source/development builds. Release builds embed the public signing key.</p> : null}
            <div className="row">
              <button className="btn" type="button" disabled={updateState?.lifecycle === "checking" || updateState?.lifecycle === "downloading" || updateState?.development} onClick={onCheckForUpdate}>Check now</button>
              {updateState?.lifecycle === "available" ? <button className="btn primary" type="button" onClick={onDownloadUpdate}>Download update</button> : null}
              {updateState?.lifecycle === "ready" ? <button className="btn primary" type="button" disabled={updateInstallBlocked} onClick={onInstallUpdate}>Install v{updateState.availableVersion}</button> : null}
              {updateState?.lifecycle === "ready" ? <button className="btn" type="button" onClick={onViewUpdateNotes}>What's new</button> : null}
              {updateState?.lifecycle === "ready" ? <button className="btn" type="button" onClick={onDiscardUpdate}>Discard download</button> : null}
              {updateState?.lifecycle === "error" && updateState.retryable ? <button className="btn primary" type="button" onClick={onCheckForUpdate}>Retry</button> : null}
            </div>
          </>
        ) : null}

        {section === "activity" ? (
          <>
            <div className="section-head"><h2>Recent activity</h2><p>Your latest backups and restores</p></div>
            {activityEvents.length === 0 ? (
              <p className="empty-note">Your backup and restore activity will appear here.</p>
            ) : (
              <div className="list">
                {activityEvents.map((entry) => (
                  <div className="log-row" key={entry.id}>
                    <strong>{entry.title}</strong>
                    {entry.detail ? <span className="hint">{entry.detail}</span> : null}
                    <time>{safeTime(entry.timestamp)}</time>
                  </div>
                ))}
              </div>
            )}
          </>
        ) : null}

        {section === "advanced" ? (
          <>
            <div className="section-head"><h2>Advanced</h2></div>
            <div className="form-grid">
              <label className="field"><span>Automatic revisions to keep</span>
                <input type="number" min={1} max={50} value={settings.autoRevisionRetention} onChange={(event) => onChange({ ...settings, autoRevisionRetention: Number(event.target.value) })} />
              </label>
              <label className="field"><span>Ignore patterns (comma separated)</span>
                <input value={settings.ignorePatterns.join(",")} onChange={(event) => onChange({ ...settings, ignorePatterns: event.target.value.split(",").map((item) => item.trim()).filter(Boolean) })} />
              </label>
            </div>
            <label className="check">
              <input type="checkbox" checked={settings.debugLogging} onChange={(event) => onChange({ ...settings, debugLogging: event.target.checked })} />
              Enable debug logging
            </label>
            <div className="row">
              <button className="btn primary" onClick={onSaveSettings} type="button">Save settings</button>
              <button className="btn" type="button" onClick={() => void onExportSettings()}>Export setup</button>
              <button className="btn" type="button" onClick={() => importFileRef.current?.click()}>Import setup</button>
            </div>
            <input ref={importFileRef} type="file" accept=".json,application/json" style={{ display: "none" }} onChange={handleImportFile} />
            {settingsImportError ? <p className="banner error" role="alert">{settingsImportError}</p> : null}
            {settings.debugLogging ? <DebugConsoleView onDownloadLogs={onDownloadLogs} /> : null}
          </>
        ) : null}
      </div>
    </div>
  );
}

function formatUpdateBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
