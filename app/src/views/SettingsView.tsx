import type { ConflictPolicy, Settings } from "../types";

type SettingsViewProps = {
  settings: Settings;
  onSyncIntervalChange: (value: number) => void;
  onConflictPolicyChange: (value: ConflictPolicy) => void;
  onAutostartChange: (value: boolean) => void;
  onIgnorePatternsChange: (value: string[]) => void;
  onSaveSettings: () => void;
};

export function SettingsView(props: SettingsViewProps) {
  const {
    settings,
    onSyncIntervalChange,
    onConflictPolicyChange,
    onAutostartChange,
    onIgnorePatternsChange,
    onSaveSettings,
  } = props;

  return (
    <section className="panel">
      <h2>Global Settings</h2>
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
          Conflict policy
          <select
            value={settings.conflictPolicy}
            onChange={(e) => onConflictPolicyChange(e.target.value as ConflictPolicy)}
          >
            <option value="latest_wins">Latest modified wins</option>
            <option value="keep_both">Keep both copies</option>
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
      </div>
      <button className="btn primary" onClick={onSaveSettings} type="button">Save Settings</button>
    </section>
  );
}
