import { useState } from "react";
import type { PauseScope, ProtectionTarget, Settings, ThemeMode } from "../types";
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
  updateState: AppUpdateState | null;
  updateInstallBlocked: boolean;
  onCheckForUpdate: () => void;
  onDownloadUpdate: () => void;
  onInstallUpdate: () => void;
  onDiscardUpdate: () => void;
  onAutomaticUpdateChecks: (enabled: boolean) => void;
  onViewUpdateNotes: () => void;
  onDownloadLogs: () => void;
};

export function SettingsView(props: SettingsViewProps) {
  const {
    settings, activeProtectionTarget, onChange, onThemeModeChange, onTargetChange, onConnectGoogleDrive,
    onDisconnectGoogleDrive, onBrowseSyncRoot, onAutostartChange, onPause,
    onResume, onProtectionEnabled, onRetry, onSaveSettings, updateState,
    updateInstallBlocked, onCheckForUpdate, onDownloadUpdate, onInstallUpdate,
    onDiscardUpdate, onAutomaticUpdateChecks, onViewUpdateNotes, onDownloadLogs,
  } = props;
  const [googleBusy, setGoogleBusy] = useState(false);
  async function runGoogle(action: () => Promise<void>) {
    setGoogleBusy(true);
    try { await action(); } finally { setGoogleBusy(false); }
  }
  const paused = settings.serviceControl.pauseScope !== "none";
  const targetName = (target: ProtectionTarget) => target === "google_drive" ? "Google Drive" : target === "folder" ? "Synced folder" : "Local only";

  return (
    <>
      <section className="panel" data-window-title="PROTECTION.DESTINATION">
        <h2>Protection destination</h2>
        <p className="empty-state">Every revision is committed locally first. Cloud replication happens afterward and can wait safely while offline.</p>
        <div className="provider-picker provider-picker-three" aria-label="Protection destination">
          <button className={`provider-card ${settings.protectionTarget === "local" ? "selected" : ""}`} onClick={() => onTargetChange("local")} type="button">
            <strong>Local only</strong><span>No account required. This cannot survive losing or resetting this disk.</span>
          </button>
          <button className={`provider-card ${settings.protectionTarget === "folder" ? "selected" : ""}`} onClick={() => onTargetChange("folder")} type="button">
            <strong>Synced folder</strong><span>Use OneDrive, Dropbox, Syncthing, or another folder provider.</span>
          </button>
          <button className={`provider-card ${settings.protectionTarget === "google_drive" ? "selected" : ""}`} onClick={() => onTargetChange("google_drive")} type="button">
            <strong>Google Drive</strong><span>Direct connection using drive.file. No VPS or user API key.</span>
          </button>
        </div>
        <p className="snapshot-meta" role="status">
          Active destination: {targetName(activeProtectionTarget)}.
          {settings.protectionTarget !== activeProtectionTarget ? ` Save settings to switch to ${targetName(settings.protectionTarget)}.` : ""}
        </p>

        {settings.protectionTarget === "google_drive" ? (
          <div className="provider-status" aria-live="polite">
            <p>{settings.googleDrive.connected ? `Connected to ${settings.googleDrive.accountEmail || "Google Drive"}.` : "Connect Google Drive to replicate protected revisions."}</p>
            {settings.googleDrive.oauthClientSource ? <p className="snapshot-meta">OAuth client: {settings.googleDrive.oauthClientSource.replace(/_/g, " ")} · credential: {(settings.googleDrive.oauthCredentialSource ?? "unknown").replace(/_/g, " ")}</p> : null}
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
      </section>

      <section className="panel" data-window-title="SERVICE.CONTROLS">
        <div className="panel-head">
          <h2>Protection controls</h2>
          <span className={`state-text ${paused ? "warning" : "success"}`}>{paused ? `Paused: ${settings.serviceControl.pauseScope}` : "Active"}</span>
        </div>
        <label className="inline-check">
          <input
            type="checkbox"
            checked={settings.serviceControl.protectionEnabled}
            onChange={(event) => void onProtectionEnabled(event.target.checked)}
          />
          Protection enabled globally
        </label>
        <div className="detail-actions">
          <button className="btn" type="button" onClick={() => void onPause("uploads", "manual")}>Pause cloud uploads</button>
          <button className="btn" type="button" onClick={() => void onPause("all", "one_hour")}>Pause all for one hour</button>
          <button className="btn" type="button" onClick={() => void onPause("all", "until_login")}>Pause until next login</button>
          <button className="btn primary" type="button" disabled={!paused} onClick={() => void onResume()}>Resume protection</button>
          <button className="btn" type="button" onClick={() => void onRetry()}>Retry provider</button>
        </div>
      </section>

      <section className="panel" data-window-title="APPLICATION.UPDATES">
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
        {updateInstallBlocked && updateState?.lifecycle === "ready" ? <p className="snapshot-meta">Close tracked games and wait for active protection to finish before installing.</p> : null}
        {updateState?.development ? <p className="snapshot-meta">Installation is disabled in source/development builds. Release builds embed the public signing key.</p> : null}
        <div className="detail-actions">
          <button className="btn" type="button" disabled={updateState?.lifecycle === "checking" || updateState?.lifecycle === "downloading" || updateState?.development} onClick={onCheckForUpdate}>Check now</button>
          {updateState?.lifecycle === "available" ? <button className="btn primary" type="button" onClick={onDownloadUpdate}>Download update</button> : null}
          {updateState?.lifecycle === "ready" ? <button className="btn primary" type="button" disabled={updateInstallBlocked} onClick={onInstallUpdate}>Install v{updateState.availableVersion}</button> : null}
          {updateState?.lifecycle === "ready" ? <button className="btn" type="button" onClick={onViewUpdateNotes}>What's new</button> : null}
          {updateState?.lifecycle === "ready" ? <button className="btn" type="button" onClick={onDiscardUpdate}>Discard download</button> : null}
          {updateState?.lifecycle === "error" && updateState.retryable ? <button className="btn primary" type="button" onClick={onCheckForUpdate}>Retry</button> : null}
        </div>
      </section>

      <section className="panel" data-window-title="SYSTEM.SETTINGS">
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
            <input type="checkbox" checked={settings.autostart} onChange={(event) => void onAutostartChange(event.target.checked)} />
            Start silently on Windows login
          </label>
          <label className="inline-check">
            <input type="checkbox" checked={settings.automaticUpload} onChange={(event) => onChange({ ...settings, automaticUpload: event.target.checked })} />
            Upload automatically after reconnecting
          </label>
          <label className="inline-check">
            <input type="checkbox" checked={settings.uploadOnMetered} disabled title="Windows metered-network enforcement is not active yet" readOnly />
            Upload on metered connections (integration pending)
          </label>
          <label className="inline-check">
            <input type="checkbox" checked={settings.remoteCheckBeforeLaunch} onChange={(event) => onChange({ ...settings, remoteCheckBeforeLaunch: event.target.checked })} />
            Check provider before Rove launch
          </label>
          <label className="inline-check">
            <input type="checkbox" checked={settings.importantNotifications} disabled title="Native Windows toast delivery is not active yet" readOnly />
            Important Windows notifications (integration pending)
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
        <button className="btn primary" onClick={onSaveSettings} type="button">Save settings</button>
      </section>

      {settings.debugLogging ? (
        <DebugConsoleView onDownloadLogs={onDownloadLogs} />
      ) : null}
    </>
  );
}

function formatUpdateBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
