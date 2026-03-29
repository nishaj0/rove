import { useEffect, useMemo, useState } from "react";
import { api } from "./api";
import type { ConflictPolicy, DebugInfo, Game, LogEntry, Settings, Status } from "./types";

type View = "home" | "games" | "backup" | "activity" | "settings";

type SyncState = {
  label: string;
  className: "success" | "warning" | "error" | "syncing";
};

const navItems: Array<{ id: View; label: string; icon: string }> = [
  { id: "home", label: "Home", icon: "⌂" },
  { id: "games", label: "Games", icon: "▦" },
  { id: "backup", label: "Backup", icon: "⇅" },
  { id: "activity", label: "Logs", icon: "◷" },
  { id: "settings", label: "Settings", icon: "⚙" },
];

const titleByView: Record<View, string> = {
  home: "Control Center",
  games: "Game Library",
  backup: "Backup Tools",
  activity: "Recent Activity",
  settings: "Settings",
};

const defaultSettings: Settings = {
  port: 7331,
  syncIntervalSeconds: 30,
  conflictPolicy: "latest_wins",
  autostart: false,
  ignorePatterns: ["*.tmp", "*.log", "cache"],
  games: [],
};

export default function App() {
  const [activeView, setActiveView] = useState<View>("home");
  const [status, setStatus] = useState<Status | null>(null);
  const [games, setGames] = useState<Game[]>([]);
  const [settings, setSettings] = useState<Settings>(defaultSettings);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [debugInfo, setDebugInfo] = useState<DebugInfo | null>(null);
  const [error, setError] = useState<string>("");
  const [toast, setToast] = useState<string>("");

  const [newName, setNewName] = useState("");
  const [newPath, setNewPath] = useState("");

  const [exportGameId, setExportGameId] = useState("");
  const [exportPath, setExportPath] = useState("");
  const [importGameId, setImportGameId] = useState("");
  const [importPath, setImportPath] = useState("");
  const [mergeOnImport, setMergeOnImport] = useState(true);

  function toFriendlyError(e: unknown): string {
    const raw = e instanceof Error ? e.message : String(e);
    if (raw.toLowerCase().includes("failed to fetch")) {
      return "Cannot reach backend service. Start backend with 'go run ./cmd/daemon --no-tray --debug', then open http://127.0.0.1:7331/debug/info to confirm it is alive.";
    }
    return raw;
  }

  async function refreshAll() {
    try {
      setError("");
      const [nextStatus, nextGames, nextSettings, nextLogs] = await Promise.all([
        api.getStatus(),
        api.getGames(),
        api.getSettings(),
        api.getRecentLogs(25),
      ]);
      const nextDebugInfo = await api.getDebugInfo().catch(() => null);
      setStatus(nextStatus);
      setGames(nextGames);
      setSettings(nextSettings);
      setLogs(nextLogs);
      setDebugInfo(nextDebugInfo);
    } catch (e) {
      setError(toFriendlyError(e));
      setDebugInfo(null);
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

  const syncState = useMemo<SyncState>(() => {
    if (error) {
      return { label: "Error", className: "error" };
    }
    if (!status) {
      return { label: "Connecting", className: "warning" };
    }
    if (status.pending > 0) {
      return { label: "Syncing", className: "syncing" };
    }
    if (!status.running) {
      return { label: "Paused", className: "warning" };
    }
    return { label: "Synced", className: "success" };
  }, [error, status]);

  const lastSyncLabel = useMemo(() => {
    if (!status?.lastSyncTime || status.lastSyncTime.startsWith("0001-")) {
      return "Never";
    }
    return new Date(status.lastSyncTime).toLocaleString();
  }, [status]);

  const uptimeLabel = useMemo(() => {
    if (!debugInfo) {
      return "-";
    }
    const total = Math.max(0, debugInfo.uptimeSec);
    const h = Math.floor(total / 3600);
    const m = Math.floor((total % 3600) / 60);
    const s = total % 60;
    return `${h}h ${m}m ${s}s`;
  }, [debugInfo]);

  async function onAddGame(e: React.FormEvent) {
    e.preventDefault();
    try {
      await api.addGame({ name: newName.trim(), path: newPath.trim() });
      setNewName("");
      setNewPath("");
      setToast("Game folder added");
      await refreshAll();
    } catch (e) {
      setError(toFriendlyError(e));
    }
  }

  async function onDeleteGame(id: string) {
    try {
      await api.deleteGame(id);
      setToast("Game removed");
      await refreshAll();
    } catch (e) {
      setError(toFriendlyError(e));
    }
  }

  async function onSaveSettings() {
    try {
      await api.saveSettings({ ...settings, games });
      setToast("Settings saved");
      await refreshAll();
    } catch (e) {
      setError(toFriendlyError(e));
    }
  }

  async function onTriggerSync() {
    try {
      await api.triggerSync();
      setToast("Sync queued");
      await refreshAll();
    } catch (e) {
      setError(toFriendlyError(e));
    }
  }

  async function onExportBackup() {
    try {
      await api.exportBackup({ gameId: exportGameId, destinationPath: exportPath });
      setToast("Backup exported");
      await refreshAll();
    } catch (e) {
      setError(toFriendlyError(e));
    }
  }

  async function onQuickExport(gameId: string) {
    setExportGameId(gameId);
    setActiveView("backup");
    setToast("Selected game for export");
  }

  async function onQuickImport(gameId: string) {
    setImportGameId(gameId);
    setActiveView("backup");
    setToast("Selected game for import");
  }

  async function onQuickEnableSync() {
    await onTriggerSync();
  }

  async function onImportBackup() {
    try {
      await api.importBackup({ gameId: importGameId, sourcePath: importPath, merge: mergeOnImport });
      setToast("Backup imported");
      await refreshAll();
    } catch (e) {
      setError(toFriendlyError(e));
    }
  }

  function initials(name: string): string {
    const parts = name.trim().split(/\s+/).filter(Boolean);
    if (parts.length === 0) {
      return "GS";
    }
    if (parts.length === 1) {
      return parts[0].slice(0, 2).toUpperCase();
    }
    return `${parts[0][0]}${parts[1][0]}`.toUpperCase();
  }

  return (
    <div className="app-layout">
      <aside className="sidebar">
        <div className="brand">
          <div className="brand-mark">CS</div>
          <div className="brand-copy">
            <strong>Cloud Saver</strong>
            <p>Desktop sync</p>
          </div>
        </div>
        <nav className="side-nav">
          {navItems.map((item) => (
            <button
              key={item.id}
              className={`side-link ${activeView === item.id ? "active" : ""}`}
              onClick={() => setActiveView(item.id)}
              type="button"
            >
              <span className="side-icon">{item.icon}</span>
              <span className="side-label">{item.label}</span>
            </button>
          ))}
        </nav>

        <div className="sidebar-status">
          <span className={`dot ${syncState.className}`} />
          <span>{syncState.label}</span>
        </div>
      </aside>

      <main className="main-panel">
        <header className="topbar">
          <div>
            <p className="kicker">Cloud Saver Desktop</p>
            <h1>{titleByView[activeView]}</h1>
          </div>
          <div className="topbar-actions">
            <button className="btn" onClick={() => void refreshAll()} type="button">Refresh</button>
            <button className="btn primary" onClick={() => void onTriggerSync()} type="button">Sync Now</button>
          </div>
        </header>

        {error ? <div className="banner error">{error}</div> : null}
        {toast ? <div className="banner toast">{toast}</div> : null}

        {activeView === "home" ? (
          <>
            <section className="stats-grid">
              <article className="stat-card">
                <h3>Pending events</h3>
                <p>{status?.pending ?? 0}</p>
              </article>
              <article className="stat-card">
                <h3>Tracked games</h3>
                <p>{games.length}</p>
              </article>
              <article className="stat-card">
                <h3>Last sync</h3>
                <p>{lastSyncLabel}</p>
              </article>
              <article className="stat-card">
                <h3>Autostart</h3>
                <p>{settings.autostart ? "Enabled" : "Disabled"}</p>
              </article>
              <article className="stat-card">
                <h3>Sync state</h3>
                <p className={`state-text ${syncState.className}`}>{syncState.label}</p>
              </article>
            </section>

            <section className="panel">
              <div className="panel-head">
                <h2>Library</h2>
                <button className="btn" onClick={() => setActiveView("games")} type="button">Manage</button>
              </div>
              {games.length === 0 ? (
                <p className="empty-state">No games tracked yet. Open Games to add your first save folder.</p>
              ) : (
                <div className="game-grid">
                  {games.slice(0, 6).map((g) => (
                    <article className="game-card" key={g.id}>
                      <div className="game-cover">
                        <span>{initials(g.name)}</span>
                      </div>
                      <div className="game-meta">
                        <strong>{g.name}</strong>
                        <p title={g.path}>{g.path}</p>
                      </div>
                    </article>
                  ))}
                </div>
              )}
            </section>
          </>
        ) : null}

        {activeView === "games" ? (
          <>
            <section className="panel">
              <h2>Add New Game Folder</h2>
              <form className="add-game-form" onSubmit={onAddGame}>
                <input placeholder="Game name" value={newName} onChange={(e) => setNewName(e.target.value)} />
                <input placeholder="Save folder path (e.g. D:\\Games\\EldenRing\\Save)" value={newPath} onChange={(e) => setNewPath(e.target.value)} />
                <button className="btn primary" type="submit">Add Game</button>
              </form>
            </section>

            <section className="panel">
              <h2>Tracked Games</h2>
              {games.length === 0 ? (
                <p className="empty-state">No games added yet.</p>
              ) : (
                <div className="game-grid">
                  {games.map((g) => (
                    <article className="game-card" key={g.id}>
                      <div className="game-cover">
                        <span>{initials(g.name)}</span>
                        <span className={`game-state ${syncState.className}`}>{syncState.label}</span>
                      </div>
                      <div className="game-meta">
                        <strong>{g.name}</strong>
                        <p title={g.path}>{g.path}</p>
                      </div>
                      <div className="game-actions">
                        <button className="btn" onClick={() => void onQuickExport(g.id)} type="button">Export</button>
                        <button className="btn" onClick={() => void onQuickImport(g.id)} type="button">Import</button>
                        <button className="btn primary" onClick={() => void onQuickEnableSync()} type="button">Enable Sync</button>
                      </div>
                      <div className="game-actions game-actions-danger">
                        <button className="btn danger" onClick={() => void onDeleteGame(g.id)} type="button">Remove</button>
                      </div>
                    </article>
                  ))}
                </div>
              )}
            </section>
          </>
        ) : null}

        {activeView === "backup" ? (
          <section className="panel split">
            <div>
              <h2>Export ZIP</h2>
              <label>
                Game
                <select value={exportGameId} onChange={(e) => setExportGameId(e.target.value)}>
                  <option value="">Select game</option>
                  {games.map((g) => (
                    <option key={g.id} value={g.id}>{g.name}</option>
                  ))}
                </select>
              </label>
              <label>
                Destination path
                <input
                  placeholder="D:\\Backups\\my-game.zip"
                  value={exportPath}
                  onChange={(e) => setExportPath(e.target.value)}
                />
              </label>
              <button className="btn primary" onClick={() => void onExportBackup()} type="button">Export Backup</button>
            </div>

            <div>
              <h2>Import ZIP</h2>
              <label>
                Game
                <select value={importGameId} onChange={(e) => setImportGameId(e.target.value)}>
                  <option value="">Select game</option>
                  {games.map((g) => (
                    <option key={g.id} value={g.id}>{g.name}</option>
                  ))}
                </select>
              </label>
              <label>
                ZIP source path
                <input
                  placeholder="D:\\Backups\\my-game.zip"
                  value={importPath}
                  onChange={(e) => setImportPath(e.target.value)}
                />
              </label>
              <label className="inline-check">
                <input type="checkbox" checked={mergeOnImport} onChange={(e) => setMergeOnImport(e.target.checked)} />
                Merge into existing folder
              </label>
              <button className="btn primary" onClick={() => void onImportBackup()} type="button">Import Backup</button>
            </div>
          </section>
        ) : null}

        {activeView === "activity" ? (
          <section className="panel">
            <div className="panel-head">
              <h2>Recent Activity</h2>
              <button className="btn" type="button" onClick={() => void refreshAll()}>Refresh</button>
            </div>

              <div className="debug-grid">
                <div className="debug-item">
                  <span>Backend debug</span>
                  <strong>{debugInfo ? (debugInfo.debug ? "enabled" : "disabled") : "offline"}</strong>
                </div>
                <div className="debug-item">
                  <span>PID</span>
                  <strong>{debugInfo?.pid ?? "-"}</strong>
                </div>
                <div className="debug-item">
                  <span>Uptime</span>
                  <strong>{uptimeLabel}</strong>
                </div>
                <div className="debug-item full">
                  <span>Config file</span>
                  <strong>{debugInfo?.configPath ?? "-"}</strong>
                </div>
                <div className="debug-item full">
                  <span>Log database</span>
                  <strong>{debugInfo?.sqlitePath ?? "-"}</strong>
                </div>
              </div>

              {status?.lastError ? (
                <div className="banner error">Last backend error: {status.lastError}</div>
              ) : null}

            {logs.length === 0 ? (
              <p className="empty-state">No logs yet.</p>
            ) : (
              <ul className="activity-list">
                {logs.map((entry) => (
                  <li key={entry.id}>
                    <span>{new Date(entry.timestamp).toLocaleTimeString()}</span>
                    <strong>{entry.level.toUpperCase()}</strong>
                    <em>{entry.event}</em>
                    <p>{entry.message}</p>
                  </li>
                ))}
              </ul>
            )}
          </section>
        ) : null}

        {activeView === "settings" ? (
          <section className="panel">
            <h2>Global Settings</h2>
            <div className="settings-grid">
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
                  onChange={(e) => setSettings((s) => ({ ...s, conflictPolicy: e.target.value as ConflictPolicy }))}
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
                      ignorePatterns: e.target.value.split(",").map((v) => v.trim()).filter(Boolean),
                    }))
                  }
                />
              </label>
            </div>
            <button className="btn primary" onClick={() => void onSaveSettings()} type="button">Save Settings</button>
          </section>
        ) : null}
      </main>
    </div>
  );
}
