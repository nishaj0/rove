import { useState } from "react";
import type { CloudProvider, Settings, ThemeMode } from "../types";

type SettingsViewProps = {
  settings: Settings;
  onThemeModeChange: (value: ThemeMode) => void;
  onCloudProviderChange: (value: CloudProvider) => void;
  onConnectGoogleDrive: () => Promise<void>;
  onDisconnectGoogleDrive: () => Promise<void>;
  onSyncIntervalChange: (value: number) => void;
  onAutostartChange: (value: boolean) => void;
  onIgnorePatternsChange: (value: string[]) => void;
  onBrowseSnapshotVault: () => void;
  onBrowseSyncRoot: () => void;
  onSyncRootChange: (value: string) => void;
  onSyncRetentionChange: (value: number) => void;
  onSnapshotMaxPerGameChange: (value: number) => void;
  onSnapshotOnExitChange: (value: boolean) => void;
  onSnapshotPeriodicChange: (value: boolean) => void;
  onSnapshotIntervalChange: (value: number) => void;
  onSaveSettings: () => void;
};

export function SettingsView(props: SettingsViewProps) {
  const {
    settings,
    onThemeModeChange,
    onCloudProviderChange,
    onConnectGoogleDrive,
    onDisconnectGoogleDrive,
    onSyncIntervalChange,
    onAutostartChange,
    onIgnorePatternsChange,
    onBrowseSnapshotVault,
    onBrowseSyncRoot,
    onSyncRootChange,
    onSyncRetentionChange,
    onSnapshotMaxPerGameChange,
    onSnapshotOnExitChange,
    onSnapshotPeriodicChange,
    onSnapshotIntervalChange,
    onSaveSettings,
  } = props;
  const [googleBusy, setGoogleBusy] = useState(false);
  async function runGoogle(action: () => Promise<void>) { setGoogleBusy(true); try { await action(); } finally { setGoogleBusy(false); } }

  return (
    <section className="panel" data-window-title="SYSTEM.SETTINGS">
      <h2>Global Settings</h2>
      <p className="empty-state">Cloud changes are never applied automatically when both copies changed; conflicts require review.</p>
      <div className="provider-picker" aria-label="Cloud provider">
        <button className={`provider-card ${settings.cloudProvider === "folder" ? "selected" : ""}`} onClick={() => onCloudProviderChange("folder")} type="button"><strong>Synced folder</strong><span>Use a folder already kept in sync by OneDrive, Dropbox, Syncthing, or another app you already have.</span></button>
        <button className={`provider-card ${settings.cloudProvider === "google_drive" ? "selected" : ""}`} onClick={() => onCloudProviderChange("google_drive")} type="button"><strong>Google Drive</strong><span>Sign in with your Google account. Cloud Saver only sees the files it creates, not your whole Drive.</span></button>
      </div>
      {settings.cloudProvider === "google_drive" ? <section className="provider-status" aria-live="polite">
        {settings.googleDrive.connected ? <><p>{settings.googleDrive.reconnectRequired ? "Google Drive access needs to be reconnected." : `Connected to ${settings.googleDrive.accountEmail || "your Google account"}.`}</p><div className="detail-actions"><button className="btn primary" disabled={googleBusy || !settings.googleDrive.reconnectRequired} onClick={() => void runGoogle(onConnectGoogleDrive)} type="button">{googleBusy ? "Waiting for sign-in..." : "Reconnect Google Drive"}</button><button className="btn" disabled={googleBusy} onClick={() => void runGoogle(onDisconnectGoogleDrive)} type="button">Disconnect</button></div></> : <><p>Connect your Google account to use Google Drive for save sync.</p><button className="btn primary" disabled={googleBusy} onClick={() => void runGoogle(onConnectGoogleDrive)} type="button">{googleBusy ? "Waiting for Google sign-in in your browser..." : "Connect Google Drive"}</button></>}
      </section> : null}
      <div className="settings-grid">
        <label>
          Desktop appearance
          <select value={settings.themeMode === "dark" ? "dark" : "light"} onChange={(e) => onThemeModeChange(e.target.value as ThemeMode)}>
            <option value="light">Light desktop (default)</option>
            <option value="dark">CRT desktop</option>
          </select>
          <span className="settings-help">Light uses toasted-cream windows. CRT uses deep brown-black windows with warm orange accents.</span>
        </label>
        <label>
          Sync interval (seconds)
          <input
            type="number"
            min={5}
            value={settings.syncIntervalSeconds}
            onChange={(e) => onSyncIntervalChange(Number(e.target.value))}
          />
        </label>
        <label className="inline-check">
          <input
            type="checkbox"
            checked={settings.autostart}
            onChange={(e) => onAutostartChange(e.target.checked)}
          />
          Start on Windows login
        </label>
        <label>
          Ignore patterns (comma separated)
          <input
            value={settings.ignorePatterns.join(",")}
            onChange={(e) => onIgnorePatternsChange(e.target.value.split(",").map((v) => v.trim()).filter(Boolean))}
          />
        </label>
        <label>
          Snapshot vault folder
          <input
            className="picker-input"
            value={settings.snapshotVaultPath}
            onClick={onBrowseSnapshotVault}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                onBrowseSnapshotVault();
              }
            }}
            placeholder="Select a local vault folder"
            readOnly
          />
        </label>
        {settings.cloudProvider === "folder" ? <label>
          Synced folder (OneDrive, Dropbox, Syncthing, etc.)
          <input
            className="picker-input"
            value={settings.syncRoot}
            onClick={onBrowseSyncRoot}
            onChange={(e) => onSyncRootChange(e.target.value)}
            placeholder="Select a folder shared between your devices"
          />
        </label> : null}
        <label>
          Cloud revision retention
          <input type="number" min={1} max={50} value={settings.syncRetention} onChange={(e) => onSyncRetentionChange(Number(e.target.value))} />
        </label>
        <label>
          Max snapshots per game
          <input
            type="number"
            min={1}
            max={50}
            value={settings.snapshotMaxPerGame}
            onChange={(e) => onSnapshotMaxPerGameChange(Number(e.target.value))}
          />
        </label>
        <label className="inline-check">
          <input
            type="checkbox"
            checked={settings.snapshotOnExit}
            onChange={(e) => onSnapshotOnExitChange(e.target.checked)}
          />
          Snapshot on game exit
        </label>
        <label className="inline-check">
          <input
            type="checkbox"
            checked={settings.snapshotPeriodic}
            onChange={(e) => onSnapshotPeriodicChange(e.target.checked)}
          />
          Periodic snapshots while running
        </label>
        <label>
          Snapshot interval (minutes)
          <input
            type="number"
            min={1}
            max={1440}
            value={settings.snapshotIntervalMinutes}
            onChange={(e) => onSnapshotIntervalChange(Number(e.target.value))}
          />
        </label>
      </div>
      <button className="btn primary" onClick={onSaveSettings} type="button">Save Settings</button>
    </section>
  );
}
