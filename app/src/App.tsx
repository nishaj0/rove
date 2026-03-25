import { useEffect, useMemo, useState } from "react";
import { api } from "./api";
import type { ConflictPolicy, Game, LogEntry, Settings, Status } from "./types";

const defaultSettings: Settings = {
  port: 7331,
  syncIntervalSeconds: 30,
  conflictPolicy: "latest_wins",
  autostart: false,
  ignorePatterns: ["*.tmp", "*.log", "cache"],
  games: [],
};

export default function App() {
  const [status, setStatus] = useState<Status | null>(null);
  const [games, setGames] = useState<Game[]>([]);
  const [settings, setSettings] = useState<Settings>(defaultSettings);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [error, setError] = useState<string>("");
  const [toast, setToast] = useState<string>("");

  const [newName, setNewName] = useState("");
  const [newPath, setNewPath] = useState("");

  const [exportGameId, setExportGameId] = useState("");
  const [exportPath, setExportPath] = useState("");
  const [importGameId, setImportGameId] = useState("");
  const [importPath, setImportPath] = useState("");
  const [mergeOnImport, setMergeOnImport] = useState(true);

  async function refreshAll() {
    try {
      setError("");
      const [nextStatus, nextGames, nextSettings, nextLogs] = await Promise.all([
        api.getStatus(),
        api.getGames(),
        api.getSettings(),
        api.getRecentLogs(25),
      ]);
      setStatus(nextStatus);
      setGames(nextGames);
      setSettings(nextSettings);
      setLogs(nextLogs);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    void refreshAll();
    const timer = window.setInterval(() => {
      void refreshAll();
    }, 5000);
    return () => window.clearInterval(timer);
  }, []);

  useEffect(() => {
    if (!toast) {
      return;
    }
    const t = window.setTimeout(() => setToast(""), 2200);
    return () => window.clearTimeout(t);
  }, [toast]);

  const statusLabel = useMemo(() => {
    if (!status) {
      return "Connecting";
    }
    return status.running ? "Sync Running" : "Sync Paused";
  }, [status]);

  async function onAddGame(e: React.FormEvent) {
    e.preventDefault();
    try {
      await api.addGame({ name: newName.trim(), path: newPath.trim() });
      setNewName("");
      setNewPath("");
      setToast("Game folder added");
      await refreshAll();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function onDeleteGame(id: string) {
    try {
      await api.deleteGame(id);
      setToast("Game removed");
      await refreshAll();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function onSaveSettings() {
    try {
      await api.saveSettings({ ...settings, games });
      setToast("Settings saved");
      await refreshAll();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function onTriggerSync() {
    try {
      await api.triggerSync();
      setToast("Sync queued");
      await refreshAll();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function onExportBackup() {
    try {
      await api.exportBackup({ gameId: exportGameId, destinationPath: exportPath });
      setToast("Backup exported");
      await refreshAll();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function onImportBackup() {
    try {
      await api.importBackup({ gameId: importGameId, sourcePath: importPath, merge: mergeOnImport });
      setToast("Backup imported");
      await refreshAll();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <div className="app-shell">
      <header className="hero">
        <div>
          <p className="overline">Cloud Saver Desktop</p>
          <h1>Game Save Control Center</h1>
          <p className="sub">Windows app for local save tracking, sync control, and backup restore.</p>
        </div>
        <div className={`status-pill ${status?.running ? "ok" : "warn"}`}>{statusLabel}</div>
      </header>

      {error ? <div className="banner error">{error}</div> : null}
      {toast ? <div className="banner toast">{toast}</div> : null}

      <section className="grid two-up">
        <article className="card">
          <h2>Sync Status</h2>
          <dl>
            <div>
              <dt>Pending events</dt>
              <dd>{status?.pending ?? "-"}</dd>
            </div>
            <div>
              <dt>Last sync</dt>
              <dd>{status?.lastSyncTime ? new Date(status.lastSyncTime).toLocaleString() : "Never"}</dd>
            </div>
            <div>
              <dt>Tracked games</dt>
              <dd>{games.length}</dd>
            </div>
          </dl>
          <button className="btn primary" onClick={() => void onTriggerSync()}>Trigger Sync</button>
        </article>

        <article className="card">
          <h2>Global Settings</h2>
          <label>
            Sync interval (seconds)
            <input
              type="number"
              min={5}
              value={settings.syncIntervalSeconds}
              onChange={(e) => setSettings((s) => ({ ...s, syncIntervalSeconds: Number(e.target.value) }))}
            />
          </label>
          <label>
            Conflict policy
            <select
              value={settings.conflictPolicy}
              onChange={(e) =>
                setSettings((s) => ({ ...s, conflictPolicy: e.target.value as ConflictPolicy }))
              }
            >
              <option value="latest_wins">Latest modified wins</option>
              <option value="keep_both">Keep both copies</option>
            </select>
          </label>
          <label className="inline-check">
            <input
              type="checkbox"
              checked={settings.autostart}
              onChange={(e) => setSettings((s) => ({ ...s, autostart: e.target.checked }))}
            />
            Start on Windows login
          </label>
          <label>
            Ignore patterns (comma separated)
            <input
              value={settings.ignorePatterns.join(",")}
              onChange={(e) =>
                setSettings((s) => ({
                  ...s,
                  ignorePatterns: e.target.value
                    .split(",")
                    .map((v) => v.trim())
                    .filter(Boolean),
                }))
              }
            />
          </label>
          <button className="btn" onClick={() => void onSaveSettings()}>Save Settings</button>
        </article>
      </section>

      <section className="grid two-up">
        <article className="card">
          <h2>Tracked Games</h2>
          <form className="inline-form" onSubmit={onAddGame}>
            <input placeholder="Game name" value={newName} onChange={(e) => setNewName(e.target.value)} />
            <input placeholder="Save folder path" value={newPath} onChange={(e) => setNewPath(e.target.value)} />
            <button className="btn primary" type="submit">Add</button>
          </form>

          <ul className="game-list">
            {games.map((g) => (
              <li key={g.id}>
                <div>
                  <strong>{g.name}</strong>
                  <p>{g.path}</p>
                </div>
                <button className="btn danger" onClick={() => void onDeleteGame(g.id)}>Remove</button>
              </li>
            ))}
          </ul>
        </article>

        <article className="card">
          <h2>Backup Tools</h2>
          <div className="backup-grid">
            <h3>Export ZIP</h3>
            <select value={exportGameId} onChange={(e) => setExportGameId(e.target.value)}>
              <option value="">Select game</option>
              {games.map((g) => (
                <option key={g.id} value={g.id}>
                  {g.name}
                </option>
              ))}
            </select>
            <input
              placeholder="Destination zip path (e.g. D:\\Backups\\eldenring.zip)"
              value={exportPath}
              onChange={(e) => setExportPath(e.target.value)}
            />
            <button className="btn" onClick={() => void onExportBackup()}>Export</button>

            <h3>Import ZIP</h3>
            <select value={importGameId} onChange={(e) => setImportGameId(e.target.value)}>
              <option value="">Select game</option>
              {games.map((g) => (
                <option key={g.id} value={g.id}>
                  {g.name}
                </option>
              ))}
            </select>
            <input
              placeholder="Source zip path"
              value={importPath}
              onChange={(e) => setImportPath(e.target.value)}
            />
            <label className="inline-check">
              <input type="checkbox" checked={mergeOnImport} onChange={(e) => setMergeOnImport(e.target.checked)} />
              Merge into existing folder
            </label>
            <button className="btn" onClick={() => void onImportBackup()}>Import</button>
          </div>
        </article>
      </section>

      <section className="card">
        <h2>Recent Activity</h2>
        <ul className="log-list">
          {logs.map((entry) => (
            <li key={entry.id}>
              <span>{new Date(entry.timestamp).toLocaleTimeString()}</span>
              <strong>{entry.level.toUpperCase()}</strong>
              <em>{entry.event}</em>
              <p>{entry.message}</p>
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}
