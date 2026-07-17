import type { Settings, ThemeMode } from "../types";

type SettingsViewProps = {
  settings: Settings;
  onSyncIntervalChange: (value: number) => void;
  onThemeModeChange: (value: ThemeMode) => void;
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
    onSyncIntervalChange,
    onThemeModeChange,
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

  return (
    <section className="panel">
      <h2>Global Settings</h2>
      <p className="empty-state">Cloud changes are never applied automatically when both copies changed; conflicts require review.</p>
      <div className="settings-grid">
        <label>
          Sync interval (seconds)
          <input
            type="number"
            min={5}
            value={settings.syncIntervalSeconds}
            onChange={(e) => onSyncIntervalChange(Number(e.target.value))}
          />
        </label>
        <label>
          Theme
          <select
            value={settings.themeMode}
            onChange={(e) => onThemeModeChange(e.target.value as ThemeMode)}
          >
            <option value="dark">Dark</option>
            <option value="light">Light</option>
            <option value="system">Auto (System)</option>
          </select>
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
        <label>
          Synced folder (OneDrive, Dropbox, Syncthing, etc.)
          <input
            className="picker-input"
            value={settings.syncRoot}
            onClick={onBrowseSyncRoot}
            onChange={(e) => onSyncRootChange(e.target.value)}
            placeholder="Select a folder shared between your devices"
          />
        </label>
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
