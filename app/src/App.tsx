import { type FormEvent, useEffect, useMemo, useRef, useState } from "react";
import { api } from "./api";
import { newDefaultSettings } from "./constants/defaults";
import { BACKEND_DEBUG_INFO_URL } from "./constants/network";
import { BACKUP_HISTORY_REFRESH_INTERVAL_MS, TOAST_DISMISS_DELAY_MS, VIEW_REFRESH_INTERVAL_MS } from "./constants/timing";
import { pickExecutable, pickFolder, pickZipFile } from "./nativeDialog";
import type { BackupHistoryEntry, BackupPolicy, ConflictPolicy, DebugInfo, Game, LogEntry, Settings, Status } from "./types";
import { ActivityView } from "./views/ActivityView";
import { GameDetailView } from "./views/GameDetailView";
import { GamesView } from "./views/GamesView";
import { HomeView } from "./views/HomeView";
import { SettingsView } from "./views/SettingsView";

type View = "home" | "games" | "game" | "activity" | "settings";
type EffectiveTheme = "dark" | "light";

type NavSnapshot = {
  view: View;
  selectedGameId: string;
};

type BackupOperationState = "idle" | "working" | "success" | "failed";

const navItems: Array<{ id: View; label: string }> = [
  { id: "home", label: "Home" },
  { id: "games", label: "Games" },
  { id: "activity", label: "Logs" },
  { id: "settings", label: "Settings" },
];

const viewTitle: Record<View, string> = {
  home: "Home",
  games: "Library",
  game: "Game",
  activity: "Activity",
  settings: "Settings",
};

function resolveTheme(mode: Settings["themeMode"]): EffectiveTheme {
  if (mode === "dark" || mode === "light") {
    return mode;
  }
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

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

  const navHistoryRef = useRef<NavSnapshot[]>([{ view: "home", selectedGameId: "" }]);
  const navCursorRef = useRef(0);
  const [canGoBack, setCanGoBack] = useState(false);
  const [canGoForward, setCanGoForward] = useState(false);

  const [status, setStatus] = useState<Status | null>(null);
  const [games, setGames] = useState<Game[]>([]);
  const [settings, setSettings] = useState<Settings>(() => newDefaultSettings());
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
  const [backupHistory, setBackupHistory] = useState<BackupHistoryEntry[]>([]);
  const [exportOperationState, setExportOperationState] = useState<BackupOperationState>("idle");
  const [exportOperationMessage, setExportOperationMessage] = useState("");
  const [importOperationState, setImportOperationState] = useState<BackupOperationState>("idle");
  const [importOperationMessage, setImportOperationMessage] = useState("");

  function syncNavButtonState() {
    setCanGoBack(navCursorRef.current > 0);
    setCanGoForward(navCursorRef.current < navHistoryRef.current.length - 1);
  }

  function pushNavigation(view: View, gameId = "") {
    const current = navHistoryRef.current[navCursorRef.current];
    if (current && current.view === view && current.selectedGameId === gameId) {
      return;
    }
    const trimmed = navHistoryRef.current.slice(0, navCursorRef.current + 1);
    trimmed.push({ view, selectedGameId: gameId });
    navHistoryRef.current = trimmed;
    navCursorRef.current = trimmed.length - 1;
    setActiveView(view);
    setSelectedGameId(gameId);
    syncNavButtonState();
  }

  function goBack() {
    if (navCursorRef.current === 0) {
      return;
    }
    navCursorRef.current -= 1;
    const snapshot = navHistoryRef.current[navCursorRef.current];
    setActiveView(snapshot.view);
    setSelectedGameId(snapshot.selectedGameId);
    syncNavButtonState();
  }

  function goForward() {
    if (navCursorRef.current >= navHistoryRef.current.length - 1) {
      return;
    }
    navCursorRef.current += 1;
    const snapshot = navHistoryRef.current[navCursorRef.current];
    setActiveView(snapshot.view);
    setSelectedGameId(snapshot.selectedGameId);
    syncNavButtonState();
  }

  async function onBrowseNewSaveFolder() {
    try {
      const selected = await pickFolder(newPath || undefined);
      if (selected) {
        setNewPath(selected);
      }
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onBrowseNewExecutablePath() {
    try {
      const selected = await pickExecutable(newExecutablePath || newPath || undefined);
      if (selected) {
        setNewExecutablePath(selected);
      }
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onBrowseExportPath() {
    try {
      const selected = await pickFolder(exportPath || selectedGame?.path || undefined);
      if (selected) {
        setExportPath(selected);
      }
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onBrowseImportPath() {
    try {
      const selected = await pickZipFile(importPath || undefined);
      if (selected) {
        setImportPath(selected);
      }
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  function toFriendlyError(e: unknown): string {
    const raw = e instanceof Error ? e.message : String(e);
    if (raw.toLowerCase().includes("failed to fetch")) {
      return `Cannot reach backend service. Start backend with 'go run ./cmd/daemon --no-tray --debug', then open ${BACKEND_DEBUG_INFO_URL} to confirm it is alive.`;
    }
    return raw;
  }

  async function refreshGamesData() {
    try {
      setError("");
      const nextGames = await api.getGames();
      setGames(nextGames);
    } catch (e) {
      setError(toFriendlyError(e));
    }
  }

  async function refreshActivityData() {
    try {
      setError("");
      const [nextStatus, nextLogs] = await Promise.all([
        api.getStatus(),
        api.getRecentLogs(25),
      ]);
      const nextDebugInfo = await api.getDebugInfo().catch(() => null);
      setStatus(nextStatus);
      setLogs(nextLogs);
      setDebugInfo(nextDebugInfo);
    } catch (e) {
      setError(toFriendlyError(e));
      setDebugInfo(null);
    }
  }

  async function refreshBackupHistory(gameID: string) {
    if (!gameID) {
      setBackupHistory([]);
      return;
    }
    try {
      const entries = await api.getBackupHistory(gameID, 20);
      setBackupHistory(entries);
    } catch (e) {
      setError(toFriendlyError(e));
    }
  }

  async function loadInitialData() {
    try {
      setError("");
      const [nextGames, nextSettings] = await Promise.all([
        api.getGames(),
        api.getSettings(),
      ]);
      setGames(nextGames);
      setSettings(nextSettings);
    } catch (e) {
      setError(toFriendlyError(e));
    }
  }

  useEffect(() => {
    void loadInitialData();
  }, []);

  useEffect(() => {
    let timer: number | undefined;

    if (activeView === "activity") {
      void refreshActivityData();
      timer = window.setInterval(() => {
        void refreshActivityData();
      }, VIEW_REFRESH_INTERVAL_MS);
    } else if (activeView === "home" || activeView === "games" || activeView === "game") {
      void refreshGamesData();
      timer = window.setInterval(() => {
        void refreshGamesData();
      }, VIEW_REFRESH_INTERVAL_MS);
    }

    return () => {
      if (timer !== undefined) {
        window.clearInterval(timer);
      }
    };
  }, [activeView]);

  useEffect(() => {
    if (activeView !== "game" || !selectedGameId) {
      setBackupHistory([]);
      return;
    }

    void refreshBackupHistory(selectedGameId);
    const timer = window.setInterval(() => {
      void refreshBackupHistory(selectedGameId);
    }, BACKUP_HISTORY_REFRESH_INTERVAL_MS);

    return () => {
      window.clearInterval(timer);
    };
  }, [activeView, selectedGameId]);

  useEffect(() => {
    if (!toast) {
      return;
    }
    const t = window.setTimeout(() => setToast(""), TOAST_DISMISS_DELAY_MS);
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
        pushNavigation("games");
      }
    }
  }, [activeView, games, selectedGameId]);

  useEffect(() => {
    document.documentElement.setAttribute("data-theme", resolveTheme(settings.themeMode));
  }, [settings.themeMode]);

  useEffect(() => {
    if (settings.themeMode !== "system") {
      return;
    }

    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    const handleChange = () => {
      document.documentElement.setAttribute("data-theme", mediaQuery.matches ? "dark" : "light");
    };

    if (typeof mediaQuery.addEventListener === "function") {
      mediaQuery.addEventListener("change", handleChange);
      return () => mediaQuery.removeEventListener("change", handleChange);
    }

    mediaQuery.addListener(handleChange);
    return () => mediaQuery.removeListener(handleChange);
  }, [settings.themeMode]);

  const selectedGame = useMemo(() => {
    if (!selectedGameId) {
      return null;
    }
    return games.find((game) => game.id === selectedGameId) ?? null;
  }, [games, selectedGameId]);

  const currentViewTitle = useMemo(() => {
    if (activeView === "game" && selectedGame) {
      return selectedGame.name;
    }
    return viewTitle[activeView];
  }, [activeView, selectedGame]);

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

  async function onAddGame(e: FormEvent): Promise<boolean> {
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
      await refreshGamesData();
      return true;
    } catch (err) {
      setError(toFriendlyError(err));
      return false;
    }
  }

  async function onDeleteGame(id: string) {
    try {
      await api.deleteGame(id);
      if (selectedGameId === id) {
        setSelectedGameId("");
        pushNavigation("games");
      }
      setToast("Game removed");
      await refreshGamesData();
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onSaveSettings() {
    try {
      const saved = await api.saveSettings({ ...settings, games });
      setSettings(saved);
      setToast("Settings saved");
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onThemeModeChange(value: Settings["themeMode"]) {
    const previous = settings.themeMode;
    if (previous === value) {
      return;
    }

    setSettings((s) => ({ ...s, themeMode: value }));

    try {
      const saved = await api.saveSettings({ ...settings, games, themeMode: value });
      setSettings(saved);
    } catch (err) {
      setSettings((s) => ({ ...s, themeMode: previous }));
      setError(toFriendlyError(err));
    }
  }

  async function onTriggerSync() {
    try {
      await api.triggerSync();
      setToast("Global sync queued");
      await refreshGamesData();
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
      await refreshGamesData();
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
      await refreshGamesData();
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onExportBackup() {
    if (!selectedGame) {
      return;
    }
    setExportOperationState("working");
    setExportOperationMessage("Export in progress...");
    try {
      const result = await api.exportBackup({ gameId: selectedGame.id, destinationPath: exportPath.trim() });
      setExportOperationState("success");
      setExportOperationMessage("Export completed successfully.");
      setToast(`Backup exported to ${result.path}`);
      await refreshGamesData();
    } catch (err) {
      setExportOperationState("failed");
      setExportOperationMessage("Export failed. Please check destination folder and try again.");
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
    setImportOperationState("working");
    setImportOperationMessage("Import in progress...");
    try {
      await api.importBackup({ gameId: selectedGame.id, sourcePath: importPath.trim(), merge: mergeOnImport });
      setImportOperationState("success");
      setImportOperationMessage("Import completed successfully.");
      setToast("Backup imported");
      await refreshGamesData();
    } catch (err) {
      setImportOperationState("failed");
      setImportOperationMessage("Import failed. Please verify the ZIP and try again.");
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
      await refreshGamesData();
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onSaveSelectedGameBackupPolicy(policy: BackupPolicy): Promise<boolean> {
    if (!selectedGame) {
      return false;
    }
    try {
      const updated = await api.updateGameBackupPolicy({ gameId: selectedGame.id, backupPolicy: policy });
      setGames((existing) => existing.map((game) => (game.id === updated.id ? updated : game)));
      setToast("Backup policy saved");
      await refreshBackupHistory(updated.id);
      return true;
    } catch (err) {
      setError(toFriendlyError(err));
      return false;
    }
  }

  function openGame(gameId: string) {
    setExportPath("");
    setImportPath("");
    setMergeOnImport(true);
    setExportOperationState("idle");
    setExportOperationMessage("");
    setImportOperationState("idle");
    setImportOperationMessage("");
    setBackupHistory([]);
    pushNavigation("game", gameId);
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
              onClick={() => pushNavigation(item.id)}
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
      </aside>

      <main className="main-panel">
        <div className="app-utility">
          <div className="history-nav">
            <button className="icon-btn" disabled={!canGoBack} onClick={goBack} type="button" aria-label="Back">&#8249;</button>
            <button className="icon-btn" disabled={!canGoForward} onClick={goForward} type="button" aria-label="Forward">&#8250;</button>
          </div>
          <p className="app-utility-title">{currentViewTitle}</p>
        </div>

        {error ? <div className="banner error">{error}</div> : null}
        {toast ? <div className="banner toast">{toast}</div> : null}

        {activeView === "home" ? (
          <HomeView
            games={games}
            onOpenLibrary={() => pushNavigation("games")}
            onOpenActivity={() => pushNavigation("activity")}
            onOpenGame={openGame}
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
            onBrowseSaveFolder={() => void onBrowseNewSaveFolder()}
            onBrowseExecutable={() => void onBrowseNewExecutablePath()}
            onNewCoverImageChange={setNewCoverImageData}
            onAddGame={onAddGame}
            onOpenGame={openGame}
            initials={initials}
          />
        ) : null}

        {activeView === "game" ? (
          <GameDetailView
            game={selectedGame}
            exportPath={exportPath}
            importPath={importPath}
            mergeOnImport={mergeOnImport}
            onBack={() => pushNavigation("games")}
            onPlay={() => void onPlaySelectedGame()}
            onSyncGame={() => void onSyncSelectedGame()}
            onDelete={() => void onDeleteSelectedGame()}
            onBrowseExportPath={() => void onBrowseExportPath()}
            onBrowseImportPath={() => void onBrowseImportPath()}
            onMergeChange={setMergeOnImport}
            onExportBackup={() => void onExportBackup()}
            onImportBackup={() => void onImportBackup()}
            exportOperationState={exportOperationState}
            exportOperationMessage={exportOperationMessage}
            importOperationState={importOperationState}
            importOperationMessage={importOperationMessage}
            backupHistory={backupHistory}
            onSaveBackupPolicy={onSaveSelectedGameBackupPolicy}
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
            onRefresh={() => void refreshActivityData()}
            safeTime={safeTime}
          />
        ) : null}

        {activeView === "settings" ? (
          <SettingsView
            settings={settings}
            onSyncIntervalChange={(value) => setSettings((s) => ({ ...s, syncIntervalSeconds: value }))}
            onConflictPolicyChange={(value: ConflictPolicy) => setSettings((s) => ({ ...s, conflictPolicy: value }))}
            onThemeModeChange={(value) => void onThemeModeChange(value)}
            onAutostartChange={(value) => setSettings((s) => ({ ...s, autostart: value }))}
            onIgnorePatternsChange={(value) => setSettings((s) => ({ ...s, ignorePatterns: value }))}
            onSaveSettings={() => void onSaveSettings()}
          />
        ) : null}
      </main>
    </div>
  );
}
