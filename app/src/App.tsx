import { type FormEvent, useEffect, useMemo, useState } from "react";
import { api } from "./api";
import type { ConflictPolicy, DebugInfo, Game, LogEntry, Settings, Status } from "./types";
import { ActivityView } from "./views/ActivityView";
import { GameDetailView } from "./views/GameDetailView";
import { GamesView } from "./views/GamesView";
import { HomeView } from "./views/HomeView";
import { SettingsView } from "./views/SettingsView";

type View = "home" | "games" | "game" | "activity" | "settings";

type SyncState = {
  label: string;
  className: "success" | "warning" | "error" | "syncing";
};

const navItems: Array<{ id: View; label: string }> = [
  { id: "home", label: "Home" },
  { id: "games", label: "Games" },
  { id: "activity", label: "Logs" },
  { id: "settings", label: "Settings" },
];

const titleByView: Record<View, string> = {
  home: "Control Center",
  games: "Game Library",
  game: "Game Profile",
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

function NavIcon({ view }: { view: View }) {
  if (view === "home") {
    return (
      <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <path d="M3 10.5L12 3l9 7.5" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
        <path d="M6 9.8V20h12V9.8" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
    );
  }

  if (view === "games" || view === "game") {
    return (
      <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <rect x="4" y="7.5" width="16" height="9" rx="3" stroke="currentColor" strokeWidth="1.8" />
        <path d="M9 10.5v3M7.5 12h3" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
        <circle cx="15.7" cy="11.2" r="1" fill="currentColor" />
        <circle cx="17.8" cy="13.3" r="1" fill="currentColor" />
      </svg>
    );
  }

  if (view === "activity") {
    return (
      <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <path d="M5 6h14M5 12h14M5 18h9" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
      </svg>
    );
  }

  return (
    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <circle cx="12" cy="12" r="3" stroke="currentColor" strokeWidth="1.8" />
      <path
        d="M19.5 12a7.5 7.5 0 0 0-.08-1.08l2.04-1.58-2-3.46-2.45.7a7.72 7.72 0 0 0-1.88-1.08L14.7 3h-4.02l-.43 2.5a7.72 7.72 0 0 0-1.88 1.08l-2.45-.7-2 3.46 2.04 1.58a7.5 7.5 0 0 0 0 2.16L3.92 14.66l2 3.46 2.45-.7a7.72 7.72 0 0 0 1.88 1.08l.43 2.5h4.02l.43-2.5a7.72 7.72 0 0 0 1.88-1.08l2.45.7 2-3.46-2.04-1.58c.05-.35.08-.71.08-1.08z"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

export default function App() {
  const [activeView, setActiveView] = useState<View>("home");
  const [selectedGameId, setSelectedGameId] = useState("");

  const [status, setStatus] = useState<Status | null>(null);
  const [games, setGames] = useState<Game[]>([]);
  const [settings, setSettings] = useState<Settings>(defaultSettings);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [debugInfo, setDebugInfo] = useState<DebugInfo | null>(null);
  const [error, setError] = useState<string>("");
  const [toast, setToast] = useState<string>("");

  const [newName, setNewName] = useState("");
  const [newPath, setNewPath] = useState("");
  const [newExecutablePath, setNewExecutablePath] = useState("");
  const [newCoverImageData, setNewCoverImageData] = useState("");

  const [exportPath, setExportPath] = useState("");
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

  useEffect(() => {
    if (!selectedGameId) {
      return;
    }
    const stillExists = games.some((game) => game.id === selectedGameId);
    if (!stillExists) {
      setSelectedGameId("");
      if (activeView === "game") {
        setActiveView("games");
      }
    }
  }, [activeView, games, selectedGameId]);

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

  const selectedGame = useMemo(() => {
    if (!selectedGameId) {
      return null;
    }
    return games.find((game) => game.id === selectedGameId) ?? null;
  }, [games, selectedGameId]);

  const pageTitle = useMemo(() => {
    if (activeView === "game" && selectedGame) {
      return selectedGame.name;
    }
    return titleByView[activeView];
  }, [activeView, selectedGame]);

  const lastSyncLabel = useMemo(() => {
    if (!status?.lastSyncTime || status.lastSyncTime.startsWith("0001-")) {
      return "Never";
    }
    return safeDateTime(status.lastSyncTime);
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

  async function onAddGame(e: FormEvent) {
    e.preventDefault();
    try {
      await api.addGame({
        name: newName.trim(),
        path: newPath.trim(),
        executablePath: newExecutablePath.trim(),
        coverImageData: newCoverImageData,
      });
      setNewName("");
      setNewPath("");
      setNewExecutablePath("");
      setNewCoverImageData("");
      setToast("Game added");
      await refreshAll();
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onDeleteGame(id: string) {
    try {
      await api.deleteGame(id);
      if (selectedGameId === id) {
        setSelectedGameId("");
        setActiveView("games");
      }
      setToast("Game removed");
      await refreshAll();
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onSaveSettings() {
    try {
      await api.saveSettings({ ...settings, games });
      setToast("Settings saved");
      await refreshAll();
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onTriggerSync() {
    try {
      await api.triggerSync();
      setToast("Global sync queued");
      await refreshAll();
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onSyncSelectedGame() {
    if (!selectedGame) {
      return;
    }
    try {
      await api.syncGame({ gameId: selectedGame.id });
      setToast("Game sync queued");
      await refreshAll();
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onPlaySelectedGame() {
    if (!selectedGame) {
      return;
    }
    try {
      const result = await api.playGame({ gameId: selectedGame.id });
      setToast(result.status === "already_running" ? "Game is already running" : "Game launched");
      await refreshAll();
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onExportBackup() {
    if (!selectedGame) {
      return;
    }
    try {
      const result = await api.exportBackup({ gameId: selectedGame.id, destinationPath: exportPath.trim() });
      setToast(`Backup exported to ${result.path}`);
      await refreshAll();
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onDeleteSelectedGame() {
    if (!selectedGame) {
      return;
    }
    const confirmed = window.confirm(`Remove ${selectedGame.name} from tracked games?`);
    if (!confirmed) {
      return;
    }
    await onDeleteGame(selectedGame.id);
  }

  async function onImportBackup() {
    if (!selectedGame) {
      return;
    }
    try {
      await api.importBackup({ gameId: selectedGame.id, sourcePath: importPath.trim(), merge: mergeOnImport });
      setToast("Backup imported");
      await refreshAll();
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onUpdateCover(coverImageData: string) {
    if (!selectedGame) {
      return;
    }
    try {
      await api.updateGameCover({ gameId: selectedGame.id, coverImageData });
      setToast("Cover updated");
      await refreshAll();
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  function openGame(gameId: string) {
    setSelectedGameId(gameId);
    setExportPath("");
    setImportPath("");
    setMergeOnImport(true);
    setActiveView("game");
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

  function safeDateTime(input: string): string {
    const parsed = new Date(input);
    if (Number.isNaN(parsed.getTime())) {
      return "Invalid time";
    }
    return parsed.toLocaleString();
  }

  function safeTime(input: string): string {
    const parsed = new Date(input);
    if (Number.isNaN(parsed.getTime())) {
      return "--:--:--";
    }
    return parsed.toLocaleTimeString();
  }

  function formatPlaytime(seconds: number): string {
    const total = Math.max(0, Math.floor(seconds));
    const h = Math.floor(total / 3600);
    const m = Math.floor((total % 3600) / 60);
    if (h === 0) {
      return `${m}m`;
    }
    return `${h}h ${m}m`;
  }

  return (
    <div className="app-layout">
      <aside className="sidebar">
        <nav className="side-nav">
          {navItems.map((item) => {
            const isActive = activeView === item.id || (item.id === "games" && activeView === "game");
            return (
            <button
              key={item.id}
              className={`side-link ${isActive ? "active" : ""}`}
              onClick={() => setActiveView(item.id)}
              type="button"
            >
              <span className="side-icon">
                <NavIcon view={item.id} />
              </span>
              <span className="side-label">{item.label}</span>
            </button>
            );
          })}
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
            <h1>{pageTitle}</h1>
          </div>
          <div className="topbar-actions">
            <button className="btn" onClick={() => void refreshAll()} type="button">Refresh</button>
            {activeView !== "game" ? (
              <button className="btn primary" onClick={() => void onTriggerSync()} type="button">Sync All</button>
            ) : null}
          </div>
        </header>

        {error ? <div className="banner error">{error}</div> : null}
        {toast ? <div className="banner toast">{toast}</div> : null}

        {activeView === "home" ? (
          <HomeView
            pending={status?.pending ?? 0}
            games={games}
            autostart={settings.autostart}
            lastSyncLabel={lastSyncLabel}
            syncLabel={syncState.label}
            syncClass={syncState.className}
            onManageGames={() => setActiveView("games")}
            initials={initials}
          />
        ) : null}

        {activeView === "games" ? (
          <GamesView
            games={games}
            newName={newName}
            newPath={newPath}
            newExecutablePath={newExecutablePath}
            newCoverImageData={newCoverImageData}
            onNewNameChange={setNewName}
            onNewPathChange={setNewPath}
            onNewExecutablePathChange={setNewExecutablePath}
            onNewCoverImageChange={setNewCoverImageData}
            onAddGame={onAddGame}
            onOpenGame={openGame}
            initials={initials}
            formatPlaytime={formatPlaytime}
          />
        ) : null}

        {activeView === "game" ? (
          <GameDetailView
            game={selectedGame}
            exportPath={exportPath}
            importPath={importPath}
            mergeOnImport={mergeOnImport}
            onBack={() => setActiveView("games")}
            onPlay={() => void onPlaySelectedGame()}
            onSyncGame={() => void onSyncSelectedGame()}
            onDelete={() => void onDeleteSelectedGame()}
            onExportPathChange={setExportPath}
            onImportPathChange={setImportPath}
            onMergeChange={setMergeOnImport}
            onExportBackup={() => void onExportBackup()}
            onImportBackup={() => void onImportBackup()}
            onUpdateCover={(coverImageData) => void onUpdateCover(coverImageData)}
            formatPlaytime={formatPlaytime}
            safeDateTime={safeDateTime}
          />
        ) : null}

        {activeView === "activity" ? (
          <ActivityView
            debugInfo={debugInfo}
            uptimeLabel={uptimeLabel}
            lastError={status?.lastError ?? ""}
            logs={logs}
            onRefresh={() => void refreshAll()}
            safeTime={safeTime}
          />
        ) : null}

        {activeView === "settings" ? (
          <SettingsView
            settings={settings}
            onSyncIntervalChange={(value) => setSettings((s) => ({ ...s, syncIntervalSeconds: value }))}
            onConflictPolicyChange={(value: ConflictPolicy) => setSettings((s) => ({ ...s, conflictPolicy: value }))}
            onAutostartChange={(value) => setSettings((s) => ({ ...s, autostart: value }))}
            onIgnorePatternsChange={(value) => setSettings((s) => ({ ...s, ignorePatterns: value }))}
            onSaveSettings={() => void onSaveSettings()}
          />
        ) : null}
      </main>
    </div>
  );
}
