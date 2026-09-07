import { type FormEvent, useEffect, useMemo, useRef, useState } from "react";
import { ChevronLeft, ChevronRight, Moon, Sun } from "lucide-react";
import { invoke } from "@tauri-apps/api/core";
import { listen } from "@tauri-apps/api/event";
import { api } from "./api";
import { newDefaultSettings } from "./constants/defaults";
import { BACKEND_DEBUG_INFO_URL } from "./constants/network";
import { SNAPSHOT_REFRESH_INTERVAL_MS, TOAST_DISMISS_DELAY_MS, VIEW_REFRESH_INTERVAL_MS } from "./constants/timing";
import roveLogo from "./assets/rove-primary-horizontal.svg";
import roveReversedLogo from "./assets/rove-reversed-horizontal.svg";
import { pickExecutable, pickFolder, pickZipFile } from "./nativeDialog";
import {
  type AppUpdateState,
  checkForAppUpdate,
  discardDownloadedUpdate,
  downloadAppUpdate,
  getUpdateStatus,
  installDownloadedUpdate,
  listenForUpdateProgress,
  setAutomaticUpdateChecks,
  updateActionLabel,
} from "./update";
import type { ActivityEvent, Game, PauseScope, RecoveryLibraryEntry, RecoveryPoint, Revision, Settings, SettingsUpdate, SnapshotSummary, Status, StorageUsage, SyncConflict, SyncJob, SyncPolicy } from "./types";
import { getSaveStatus } from "./utils/saveStatus";
import { GameDetailView } from "./views/GameDetailView";
import { GamesView } from "./views/GamesView";
import { SettingsView } from "./views/SettingsView";

type View = "games" | "game" | "settings";

type NavSnapshot = {
  view: View;
  selectedGameId: string;
};

type BackupOperationState = "idle" | "working" | "success" | "failed";

const isAutostartEnabled = () => invoke<boolean>("is_autostart_enabled");
const enableAutostart = () => invoke<boolean>("set_autostart", { enabled: true });
const disableAutostart = () => invoke<boolean>("set_autostart", { enabled: false });

const navItems: Array<{ id: View; label: string }> = [
  { id: "games", label: "Games" },
  { id: "settings", label: "Settings" },
];

const viewTitle: Record<View, string> = {
  games: "Library",
  game: "Game",
  settings: "Settings",
};

function NavIcon({ view }: { view: View }) {
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
  const [activeView, setActiveView] = useState<View>("games");
  const [selectedGameId, setSelectedGameId] = useState("");
	const gamesRefreshSequence = useRef(0);

  const navHistoryRef = useRef<NavSnapshot[]>([{ view: "games", selectedGameId: "" }]);
  const navCursorRef = useRef(0);
  const mainPanelRef = useRef<HTMLElement>(null);
  const [canGoBack, setCanGoBack] = useState(false);
  const [canGoForward, setCanGoForward] = useState(false);

  const [status, setStatus] = useState<Status | null>(null);
  const [syncJobs, setSyncJobs] = useState<SyncJob[]>([]);
  const [syncConflicts, setSyncConflicts] = useState<SyncConflict[]>([]);
  const [games, setGames] = useState<Game[]>([]);
  const [settings, setSettings] = useState<Settings>(() => newDefaultSettings());
  const [activityEvents, setActivityEvents] = useState<ActivityEvent[]>([]);
  const [revisions, setRevisions] = useState<Revision[]>([]);
  const [recoveryPoints, setRecoveryPoints] = useState<RecoveryPoint[]>([]);
  const [recoveryLibrary, setRecoveryLibrary] = useState<RecoveryLibraryEntry[]>([]);
  const [storageUsage, setStorageUsage] = useState<StorageUsage | null>(null);
  const [error, setError] = useState<string>("");
  const [toast, setToast] = useState<string>("");
  const [updateState, setUpdateState] = useState<AppUpdateState | null>(null);
  const [updateDismissed, setUpdateDismissed] = useState(false);
  const [launchUpdatePromptOpen, setLaunchUpdatePromptOpen] = useState(false);
  const [updateNotesOpen, setUpdateNotesOpen] = useState(false);
  const launchUpdateChoiceRef = useRef<((choice: "download" | "later" | "cancel") => void) | null>(null);
  const previousUpdateLifecycleRef = useRef<AppUpdateState["lifecycle"] | null>(null);

  const [newName, setNewName] = useState("");
  const [newPath, setNewPath] = useState("");
  const [newExecutablePath, setNewExecutablePath] = useState("");
  const [newCoverImageData, setNewCoverImageData] = useState("");
  const [newVersion, setNewVersion] = useState("");

  const [exportPath, setExportPath] = useState("");
  const [importPath, setImportPath] = useState("");
  const [mergeOnImport, setMergeOnImport] = useState(true);
  const [snapshots, setSnapshots] = useState<SnapshotSummary[]>([]);
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

  async function onBrowseSyncRoot() {
    try {
      const selected = await pickFolder(settings.syncRoot || undefined);
      if (selected) setSettings((current) => ({ ...current, syncRoot: selected }));
    } catch (err) { setError(toFriendlyError(err)); }
  }

  function toFriendlyError(e: unknown): string {
    const raw = e instanceof Error ? e.message : String(e);
    if (raw.toLowerCase().includes("failed to fetch")) {
      return `Cannot reach Rove's background service. If running from source, start backend with 'go run ./cmd/daemon --no-tray --debug'. If using the installed app, restart Rove and allow its background service through security prompts. Then open ${BACKEND_DEBUG_INFO_URL} to confirm it is alive.`;
    }
    if (raw.toLowerCase().includes("json: unknown field")) {
      return "The Rove desktop interface and background service are from different versions. Quit Rove completely, restart it, and try again. Your existing settings have not been deleted.";
    }
    return raw;
  }

  async function refreshGamesData() {
    const sequence = ++gamesRefreshSequence.current;
    try {
      const [nextGames, nextStatus, nextJobs, nextRecoveryLibrary] = await Promise.all([
        api.getGames(),
        api.getStatus(),
        api.getSyncJobs(),
        api.getRecoveryLibrary().catch(() => recoveryLibrary),
      ]);
      // Do not let an older in-flight response overwrite newer process state.
      if (sequence !== gamesRefreshSequence.current) return;
      setError("");
      setGames(nextGames);
      setStatus(nextStatus);
      setSyncJobs(nextJobs);
      setRecoveryLibrary(nextRecoveryLibrary);
    } catch (e) {
      if (sequence !== gamesRefreshSequence.current) return;
      setError(toFriendlyError(e));
    }
  }

  async function refreshActivityData() {
    try {
      setError("");
      const [nextStatus, nextEvents, nextJobs] = await Promise.all([
        api.getStatus(),
        api.getActivity(),
        api.getSyncJobs(),
      ]);
      setStatus(nextStatus);
      setActivityEvents(nextEvents);
      setSyncJobs(nextJobs);
    } catch (e) {
      setError(toFriendlyError(e));
    }
  }

  async function refreshRecoveryData(gameID: string) {
    if (!gameID) {
      setRevisions([]);
      setRecoveryPoints([]);
      setStorageUsage(null);
      return;
    }
    try {
      const [nextRevisions, nextPoints, nextUsage] = await Promise.all([
        api.getRevisions(gameID),
        api.getRecoveryPoints(gameID),
        api.getStorageUsage(gameID),
      ]);
      setRevisions(nextRevisions);
      setRecoveryPoints(nextPoints);
      setStorageUsage(nextUsage);
    } catch (e) {
      setError(toFriendlyError(e));
    }
  }

  async function refreshSnapshots(gameID: string) {
    if (!gameID) {
      setSnapshots([]);
      return;
    }
    try {
      const entries = await api.getSnapshots(gameID);
      setSnapshots(entries);
    } catch (e) {
      setError(toFriendlyError(e));
    }
  }

  async function refreshSyncConflicts(gameID: string) {
    if (!gameID) { setSyncConflicts([]); return; }
    try { setSyncConflicts(await api.getSyncConflicts(gameID)); } catch (e) { setError(toFriendlyError(e)); }
  }

  async function loadInitialData() {
    try {
      setError("");
      const daemonBootstrapError = await invoke<string | null>("daemon_bootstrap_error").catch(() => null);
      const [nextGames, nextSettings, nextStatus, nextRecoveryLibrary, googleDriveStatus] = await Promise.all([
        api.getGames(),
        api.getSettings(),
        api.getStatus(),
        api.getRecoveryLibrary().catch(() => []),
        api.googleDriveStatus().catch(() => null),
      ]);
      const nativeAutostart = await isAutostartEnabled().catch(() => nextSettings.autostart);
      setGames(nextGames);
      setSettings({
        ...nextSettings,
        autostart: nativeAutostart,
        googleDrive: googleDriveStatus ? { ...nextSettings.googleDrive, ...googleDriveStatus } : nextSettings.googleDrive,
      });
      setStatus(nextStatus);
      setRecoveryLibrary(nextRecoveryLibrary);
      const launchGameID = await invoke<string | null>("take_launch_game").catch(() => null);
      if (launchGameID) void launchGameFromShortcut(launchGameID);
      if (daemonBootstrapError) setError(daemonBootstrapError);
    } catch (e) {
      const daemonBootstrapError = await invoke<string | null>("daemon_bootstrap_error").catch(() => null);
      setError(daemonBootstrapError || toFriendlyError(e));
    }
  }

  useEffect(() => {
    void loadInitialData();
  }, []);

  useEffect(() => {
    let unlisten: (() => void) | undefined;
    void listen<string>("rove://launch-game", (event) => {
      void launchGameFromShortcut(event.payload);
    }).then((dispose) => { unlisten = dispose; }).catch(() => {
      // Browser preview has no Tauri event bridge.
    });
    return () => unlisten?.();
  }, []);

  useEffect(() => {
    let cancelled = false;
    let unlisten: (() => void) | undefined;

    async function loadUpdateInfo() {
      try {
        const restored = await getUpdateStatus();
        if (cancelled) return;
        setUpdateState(restored);
        unlisten = await listenForUpdateProgress((next) => {
          if (!cancelled) setUpdateState(next);
        });
        const checked = await checkForAppUpdate(false);
        if (!cancelled) {
          setUpdateState(checked);
          if (checked.lifecycle === "available") {
            setUpdateDismissed(false);
            setToast(`New version ${checked.availableVersion} is available`);
          }
        }
      } catch {
        // The browser-only Vite preview has no native updater. Installed builds
        // report actionable update errors through UpdateState instead.
      }
    }

    void loadUpdateInfo();
    return () => {
      cancelled = true;
      unlisten?.();
    };
  }, []);

  useEffect(() => {
    mainPanelRef.current?.scrollTo({ top: 0, behavior: "auto" });

    let timer: number | undefined;

    if (activeView === "settings") {
      void refreshActivityData();
      timer = window.setInterval(() => {
        void refreshActivityData();
      }, VIEW_REFRESH_INTERVAL_MS);
    } else if (activeView === "games" || activeView === "game") {
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
    if (activeView !== "games" && activeView !== "game" && activeView !== "settings") return;
    const refreshOnFocus = () => void (activeView === "settings" ? refreshActivityData() : refreshGamesData());
    window.addEventListener("focus", refreshOnFocus);
    return () => window.removeEventListener("focus", refreshOnFocus);
  }, [activeView]);

  useEffect(() => {
    if (!launchUpdatePromptOpen) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") resolveLaunchUpdateChoice("cancel");
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [launchUpdatePromptOpen]);

  useEffect(() => {
    if (!updateNotesOpen) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setUpdateNotesOpen(false);
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [updateNotesOpen]);

  useEffect(() => {
    const previousLifecycle = previousUpdateLifecycleRef.current;
    if (previousLifecycle !== null && previousLifecycle !== "ready" && updateState?.lifecycle === "ready") {
      setUpdateNotesOpen(true);
    }
    previousUpdateLifecycleRef.current = updateState?.lifecycle ?? null;
  }, [updateState?.lifecycle]);

  useEffect(() => {
    if (activeView !== "game" || !selectedGameId) {
      setRevisions([]);
      setRecoveryPoints([]);
      return;
    }

    void refreshRecoveryData(selectedGameId);
    void refreshSyncConflicts(selectedGameId);
    void refreshSnapshots(selectedGameId);
    const recoveryTimer = window.setInterval(() => {
      void refreshRecoveryData(selectedGameId);
      void refreshSyncConflicts(selectedGameId);
      void refreshSnapshots(selectedGameId);
    }, SNAPSHOT_REFRESH_INTERVAL_MS);

    return () => {
      window.clearInterval(recoveryTimer);
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

  const selectedGame = useMemo(() => {
    if (!selectedGameId) {
      return null;
    }
    return games.find((game) => game.id === selectedGameId) ?? null;
  }, [games, selectedGameId]);

  const isDarkTheme = settings.themeMode === "dark";

  const currentViewTitle = useMemo(() => {
    if (activeView === "game" && selectedGame) {
      return selectedGame.name;
    }
    return viewTitle[activeView];
  }, [activeView, selectedGame]);

  async function onAddGame(e: FormEvent): Promise<boolean> {
    e.preventDefault();
    try {
      await api.addGame({
        name: newName.trim(),
        path: newPath.trim(),
        executablePath: newExecutablePath.trim(),
        coverImageData: newCoverImageData,
        version: newVersion.trim(),
      });
      setNewName("");
      setNewPath("");
      setNewExecutablePath("");
      setNewCoverImageData("");
      setNewVersion("");
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
      const update: SettingsUpdate = {
        port: settings.port,
        syncIntervalSeconds: settings.syncIntervalSeconds,
        conflictPolicy: settings.conflictPolicy,
        themeMode: settings.themeMode,
        autostart: settings.autostart,
        ignorePatterns: [...settings.ignorePatterns],
        snapshotVaultPath: settings.snapshotVaultPath,
        snapshotMaxPerGame: settings.snapshotMaxPerGame,
        snapshotOnExit: settings.snapshotOnExit,
        snapshotPeriodic: settings.snapshotPeriodic,
        snapshotIntervalMinutes: settings.snapshotIntervalMinutes,
        syncRoot: settings.syncRoot,
        syncRetention: settings.syncRetention,
        reviewConflicts: settings.reviewConflicts,
        protectionTarget: settings.protectionTarget,
        serviceControl: { ...settings.serviceControl },
        automaticUpload: settings.automaticUpload,
        uploadOnMetered: settings.uploadOnMetered,
        remoteCheckBeforeLaunch: settings.remoteCheckBeforeLaunch,
        importantNotifications: settings.importantNotifications,
        debugLogging: settings.debugLogging,
        autoRevisionRetention: settings.autoRevisionRetention,
      };
      const saved = await api.saveSettings(update);
      const [googleDriveStatus, nextStatus] = await Promise.all([
        api.googleDriveStatus().catch(() => null),
        api.getStatus().catch(() => null),
      ]);
      setSettings({
        ...saved,
        googleDrive: googleDriveStatus ? { ...saved.googleDrive, ...googleDriveStatus } : saved.googleDrive,
      });
      if (nextStatus) setStatus(nextStatus);
      setToast("Settings saved");
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onSyncSelectedGame() {
    if (!selectedGame) {
      return;
    }
    try {
      const result = await api.syncGame({ gameId: selectedGame.id });
      setToast(`Backup started (${result.jobId.slice(0, 8)})`);
      await refreshGamesData();
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onPlaySelectedGame() {
    if (!selectedGame) {
      return;
    }
    let latest = updateState;
    try {
      latest = await checkForAppUpdate(false);
      setUpdateState(latest);
    } catch {
      // Update availability never blocks launching a game.
    }

    let downloadAfterLaunch = false;
    if (latest?.lifecycle === "available") {
      const choice = await new Promise<"download" | "later" | "cancel">((resolve) => {
        launchUpdateChoiceRef.current = resolve;
        setLaunchUpdatePromptOpen(true);
      });
      if (choice === "cancel") return;
      downloadAfterLaunch = choice === "download";
    }

    try {
      const result = await api.playGame({ gameId: selectedGame.id });
      setToast(result.status === "already_running" ? "Game is already running" : "Game launched");
      if (downloadAfterLaunch) void onDownloadUpdate();
      await refreshGamesData();
    } catch (err) {
      const message = toFriendlyError(err);
      if (/pre-launch check failed/i.test(message) && window.confirm(`${message}\n\nLaunch offline using the protected local save?`)) {
        try {
          await api.playGame({ gameId: selectedGame.id, allowOffline: true });
          setToast("Game launched offline. Cloud changes will reconcile later.");
          await refreshGamesData();
          return;
        } catch (offlineError) {
          setError(toFriendlyError(offlineError));
          return;
        }
      }
      setError(message);
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
    if (!mergeOnImport && !window.confirm("Replace this save folder from the ZIP? Rove will create a Recovery Point first, and files absent from the ZIP will be removed.")) {
      return;
    }
    setImportOperationState("working");
    setImportOperationMessage("Import in progress...");
    try {
      await api.importBackup({
        gameId: selectedGame.id,
        sourcePath: importPath.trim(),
        merge: mergeOnImport,
        confirmed: !mergeOnImport,
      });
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

  async function onUpdateCover(coverImageData: string, mode: "manual" | "executable_icon" | "generated" = "manual") {
    if (!selectedGame) {
      return;
    }
    try {
      await api.updateGameCover({ gameId: selectedGame.id, coverImageData, mode });
      setToast("Cover updated");
      await refreshGamesData();
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onSaveProtectionPolicy(policy: SyncPolicy): Promise<boolean> {
    if (!selectedGame) return false;
    try {
      const updated = await api.updateGameProtectionPolicy({ gameId: selectedGame.id, policy });
      setGames((existing) => existing.map((game) => game.id === updated.id ? updated : game));
      setToast("Backup options saved");
      return true;
    } catch (err) {
      setError(toFriendlyError(err));
      return false;
    }
  }

  async function launchGameFromShortcut(gameID: string) {
    try {
      const result = await api.playGame({ gameId: gameID });
      setToast(result.status === "already_running" ? "Game is already running" : "Game launched through Rove");
      await refreshGamesData();
    } catch (err) {
      setError(`Rove shortcut could not launch the game: ${toFriendlyError(err)}`);
    }
  }

  async function onCreateRoveShortcut() {
    if (!selectedGame) return;
    try {
      const path = await invoke<string>("create_game_shortcut", { gameId: selectedGame.id, gameName: selectedGame.name });
      setToast(`Rove shortcut created: ${path}`);
    } catch (err) {
      setError(`Could not create Rove shortcut: ${toFriendlyError(err)}`);
    }
  }

  async function onQuickThemeToggle() {
    const themeMode = settings.themeMode === "dark" ? "light" : "dark";
    setSettings((current) => ({ ...current, themeMode }));
    try { await api.saveSettings({ themeMode }); } catch (err) { setError(toFriendlyError(err)); }
  }

  async function onSaveDestination() {
    try {
      if (settings.protectionTarget === "folder" && !settings.syncRoot.trim()) throw new Error("Choose a synced folder before saving this destination.");
      if (settings.protectionTarget === "google_drive" && !settings.googleDrive.connected) throw new Error("Connect Google Drive before saving this destination.");
      const saved = await api.saveSettings({ protectionTarget: settings.protectionTarget, syncRoot: settings.syncRoot });
      setSettings((current) => ({ ...current, ...saved }));
      setStatus(await api.getStatus());
      setToast("Save location updated");
    } catch (err) { setError(toFriendlyError(err)); }
  }

  function resolveLaunchUpdateChoice(choice: "download" | "later" | "cancel") {
    setLaunchUpdatePromptOpen(false);
    const resolve = launchUpdateChoiceRef.current;
    launchUpdateChoiceRef.current = null;
    resolve?.(choice);
  }

  async function onSetProtectionControl(
    gameId: string | undefined,
    pauseScope: PauseScope,
    duration: "manual" | "one_hour" | "until_login" = "manual",
    protectionEnabled = true,
  ) {
    try {
      await api.setProtectionControl({
        gameId,
        protectionEnabled,
        pauseScope,
        duration,
        reason: !protectionEnabled ? "Backups turned off by user" : pauseScope === "uploads" ? "Cloud backups paused by user" : pauseScope === "all" ? "Backups paused by user" : "",
      });
      if (!gameId) {
        setSettings((current) => ({ ...current, serviceControl: { protectionEnabled, pauseScope } }));
      }
      setToast(!protectionEnabled ? "Backups turned off" : pauseScope === "none" ? "Backups resumed" : pauseScope === "uploads" ? "Cloud backups paused; local backups continue" : "Backups paused");
      await refreshGamesData();
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onCreateRecoveryPoint() {
    if (!selectedGame) return;
    const name = window.prompt("Name this Recovery Point", `${selectedGame.name} — ${new Date().toLocaleString()}`);
    if (!name?.trim()) return;
    try {
      await api.createRecoveryPoint(selectedGame.id, name.trim());
      setToast("Recovery Point is being created");
      await refreshRecoveryData(selectedGame.id);
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onRestoreRevision(revisionId: string) {
    if (!selectedGame) return;
    if (!window.confirm("Restore this revision exactly? Files absent from it will be removed after a pre-restore Recovery Point is created.")) return;
    try {
      await api.restoreRevision(selectedGame.id, revisionId);
      setToast("Revision restored");
      await Promise.all([refreshGamesData(), refreshRecoveryData(selectedGame.id)]);
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onDeleteRecoveryPoint(id: string) {
    if (!selectedGame || !window.confirm("Delete this Recovery Point pin? Its data may later become reclaimable.")) return;
    try {
      await api.deleteRecoveryPoint(id);
      setToast("Recovery Point pin deleted");
      await refreshRecoveryData(selectedGame.id);
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onAutostartChange(value: boolean) {
    try {
      if (value) await enableAutostart(); else await disableAutostart();
      const actual = await isAutostartEnabled();
      setSettings((current) => ({ ...current, autostart: actual }));
      setToast(actual ? "Rove will start silently with Windows" : "Windows startup disabled");
    } catch (err) {
      setError(`Windows startup could not be changed: ${toFriendlyError(err)}`);
    }
  }

  async function onRestoreRecoveryEntry(entry: RecoveryLibraryEntry) {
    try {
      const executablePath = await pickExecutable();
      if (!executablePath) return;
      const destination = await pickFolder();
      if (!destination) return;
      if (!window.confirm(`Restore and track ${entry.name} in this empty folder?\n\n${destination}\n\nRove will verify the complete revision before publishing it.`)) return;
      await api.restoreAndTrackRecoveryLibrary(entry.syncId, destination, executablePath);
      setToast(`${entry.name} restored and added to Rove.`);
      await refreshGamesData();
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onConnectGoogleDrive() {
    try {
      const status = await api.connectGoogleDrive();
      setSettings((s) => ({ ...s, cloudProvider: "google_drive", protectionTarget: "google_drive", googleDrive: status }));
		await refreshGamesData();
      setToast("Google Drive connected. Save Settings to use it for backups.");
    } catch (err) {
      setError(`Google Drive could not be connected: ${toFriendlyError(err)}`);
    }
  }
  async function onDisconnectGoogleDrive() {
    try {
      await api.disconnectGoogleDrive();
      const [nextSettings, nextStatus] = await Promise.all([
        api.getSettings(),
        api.getStatus().catch(() => null),
      ]);
      setSettings({ ...nextSettings, googleDrive: { connected: false } });
      if (nextStatus) setStatus(nextStatus);
      setToast("Google Drive disconnected; local sync remains active");
    } catch (err) {
      setError(`Google Drive could not be disconnected: ${toFriendlyError(err)}`);
    }
  }

  async function onRestoreSnapshot(snapshotId: string) {
    if (!selectedGame) {
      return;
    }
    const confirmed = window.confirm("Restore this snapshot? Rove will save a pre-restore snapshot first.");
    if (!confirmed) {
      return;
    }
    try {
      await api.restoreSnapshot(selectedGame.id, snapshotId);
      setToast("Snapshot restored");
      await Promise.all([refreshGamesData(), refreshSnapshots(selectedGame.id)]);
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onDeleteSnapshot(snapshotId: string) {
    if (!selectedGame) {
      return;
    }
    const confirmed = window.confirm("Delete this snapshot?");
    if (!confirmed) {
      return;
    }
    try {
      await api.deleteSnapshot(selectedGame.id, snapshotId);
      setToast("Snapshot deleted");
      await refreshSnapshots(selectedGame.id);
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onResolveSyncConflict(id: string, action: "local" | "remote" | "both") {
    if (!selectedGame) return;
    const confirmed = window.confirm(`Resolve this conflict by ${action === "local" ? "keeping your local save" : action === "remote" ? "using the remote save" : "keeping both saves"}?`);
    if (!confirmed) return;
    try { await api.resolveSyncConflict(id, action); setToast("Sync conflict resolved"); await Promise.all([refreshSyncConflicts(selectedGame.id), refreshGamesData()]); } catch (err) { setError(toFriendlyError(err)); }
  }

  function openGame(gameId: string) {
    setExportPath("");
    setImportPath("");
    setMergeOnImport(true);
    setExportOperationState("idle");
    setExportOperationMessage("");
    setImportOperationState("idle");
    setImportOperationMessage("");
    setSnapshots([]);
    pushNavigation("game", gameId);
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

  function formatBytes(bytes: number): string {
    const value = Math.max(0, bytes);
    if (value < 1024) {
      return `${value} B`;
    }
    if (value < 1024 * 1024) {
      return `${(value / 1024).toFixed(1)} KB`;
    }
    if (value < 1024 * 1024 * 1024) {
      return `${(value / (1024 * 1024)).toFixed(1)} MB`;
    }
    return `${(value / (1024 * 1024 * 1024)).toFixed(1)} GB`;
  }

  async function onDownloadUpdate() {
    try {
      setUpdateDismissed(false);
      if (updateState?.lifecycle === "available" || updateState?.lifecycle === "error") {
        const refreshed = await checkForAppUpdate(true);
        setUpdateState(refreshed);
        if (refreshed.lifecycle !== "available") return;
      }
      setUpdateState(await downloadAppUpdate());
    } catch (err) {
      setError(toFriendlyError(err));
      setUpdateState(await getUpdateStatus().catch(() => updateState));
    }
  }

  async function onInstallUpdate() {
    try {
      setUpdateState(await installDownloadedUpdate());
    } catch (err) {
      setError(toFriendlyError(err));
      setUpdateState(await getUpdateStatus().catch(() => updateState));
    }
  }

  async function onCheckForUpdate() {
    try {
      setUpdateState(await checkForAppUpdate(true));
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onDiscardUpdate() {
    try {
      setUpdateState(await discardDownloadedUpdate());
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  async function onAutomaticUpdateChecks(enabled: boolean) {
    try {
      setUpdateState(await setAutomaticUpdateChecks(enabled));
    } catch (err) {
      setError(toFriendlyError(err));
    }
  }

  const sidebarSaveStatus = getSaveStatus({
    provider: status?.provider,
    serviceControl: status?.serviceControl,
    conflictCount: status?.conflictCount,
    pendingCount: status?.pending,
    active: Boolean(status?.activeJobId),
  });

  return (
    <div className="app-layout" data-theme={isDarkTheme ? "dark" : "light"}>
      <header className="app-utility">
        <div className="topbar-brand">
          <img className="topbar-brand-logo" src={isDarkTheme ? roveReversedLogo : roveLogo} alt="Rove" />
        </div>
        <div className="history-nav">
          <button className="icon-btn" disabled={!canGoBack} onClick={goBack} type="button" aria-label="Back"><ChevronLeft aria-hidden="true" size={18} /></button>
          <button className="icon-btn" disabled={!canGoForward} onClick={goForward} type="button" aria-label="Forward"><ChevronRight aria-hidden="true" size={18} /></button>
        </div>
        <p className="app-utility-title">{currentViewTitle}</p>
        <div className="app-utility-actions">
          <button className="icon-btn theme-toggle" type="button" onClick={() => void onQuickThemeToggle()} aria-label={isDarkTheme ? "Switch to light theme" : "Switch to dark theme"} title={isDarkTheme ? "Switch to light theme" : "Switch to dark theme"}>{isDarkTheme ? <Sun aria-hidden="true" size={17} /> : <Moon aria-hidden="true" size={17} />}</button>
          {updateState && ["available", "downloading", "ready", "installing", "error"].includes(updateState.lifecycle) ? (
            <button
              className="update-indicator"
              type="button"
              onClick={() => updateState.lifecycle === "ready" ? void onInstallUpdate() : pushNavigation("settings")}
              disabled={updateState.lifecycle === "downloading" || updateState.lifecycle === "installing"}
            >
              {updateActionLabel(updateState)}
            </button>
          ) : null}
        </div>
      </header>

      <aside className="sidebar">
        <nav className="side-nav">
          {navItems.map((item) => {
            const isActive = activeView === item.id || (item.id === "games" && activeView === "game");
            const count = item.id === "games" ? games.length : 0;
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
              {count > 0 ? <span className="nav-count">{count}</span> : null}
            </button>
            );
          })}
        </nav>
        <div className="sidebar-status">
          <span>Game saves</span>
          <strong className={sidebarSaveStatus.tone}>{sidebarSaveStatus.label}</strong>
        </div>
      </aside>

      <main className="main-panel" ref={mainPanelRef}>
        {error ? <div className="banner error">{error}</div> : null}
        {updateState && !updateDismissed && ["available", "downloading", "ready", "error"].includes(updateState.lifecycle) ? (
          <div className="banner toast update-banner">
            <p>
              {updateState.lifecycle === "ready"
                ? `Version ${updateState.availableVersion} is verified and ready to install.`
                : updateState.lifecycle === "downloading"
                  ? `Downloading version ${updateState.availableVersion}: ${formatBytes(updateState.downloadedBytes)}${updateState.totalBytes ? ` of ${formatBytes(updateState.totalBytes)}` : ""}.`
                  : updateState.lifecycle === "error"
                    ? updateState.error ?? "The update needs attention. Your current installation is unchanged."
                    : `New version ${updateState.availableVersion} is available. You are on ${updateState.currentVersion}.`}
            </p>
            <div className="update-banner-actions">
              {updateState.lifecycle === "available" || updateState.lifecycle === "error" ? <button className="btn primary" type="button" onClick={() => void (updateState.lifecycle === "available" ? onDownloadUpdate() : onCheckForUpdate())}>{updateState.lifecycle === "available" ? "Download" : "Retry"}</button> : null}
              {updateState.lifecycle === "ready" ? <button className="btn primary" type="button" disabled={games.some((game) => game.running) || Boolean(status?.activeJobId)} onClick={() => void onInstallUpdate()}>Install update</button> : null}
              {updateState.lifecycle === "ready" ? <button className="btn" type="button" onClick={() => setUpdateNotesOpen(true)}>What's new</button> : null}
              <button className="btn" type="button" onClick={() => setUpdateDismissed(true)}>Dismiss</button>
            </div>
          </div>
        ) : null}
        {toast ? <div className="banner toast">{toast}</div> : null}

        {activeView === "games" ? (
          <GamesView
            games={games}
            newName={newName}
            newPath={newPath}
            newExecutablePath={newExecutablePath}
            newCoverImageData={newCoverImageData}
            newVersion={newVersion}
            onNewNameChange={setNewName}
            onBrowseSaveFolder={() => void onBrowseNewSaveFolder()}
            onBrowseExecutable={() => void onBrowseNewExecutablePath()}
            onNewCoverImageChange={setNewCoverImageData}
            onNewVersionChange={setNewVersion}
            onAddGame={onAddGame}
            onOpenGame={openGame}
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
            onOpenSettings={() => pushNavigation("settings")}
            onCreateShortcut={() => void onCreateRoveShortcut()}
            onProtect={() => void onSyncSelectedGame()}
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
            revisions={revisions}
            recoveryPoints={recoveryPoints}
            storageUsage={storageUsage}
            legacySnapshots={snapshots}
            onSavePolicy={onSaveProtectionPolicy}
            onPause={(scope, duration) => onSetProtectionControl(selectedGameId, scope, duration)}
            onResume={() => onSetProtectionControl(selectedGameId, "none")}
            onCreateRecoveryPoint={() => void onCreateRecoveryPoint()}
            onRestoreRevision={(revisionId) => void onRestoreRevision(revisionId)}
            onDeleteRecoveryPoint={(id) => void onDeleteRecoveryPoint(id)}
            onRestoreLegacySnapshot={(id) => void onRestoreSnapshot(id)}
            onDeleteLegacySnapshot={(id) => void onDeleteSnapshot(id)}
            onUpdateCover={(coverImageData, mode) => void onUpdateCover(coverImageData, mode)}
            formatPlaytime={formatPlaytime}
            safeDateTime={safeDateTime}
            formatBytes={formatBytes}
            syncJob={syncJobs.find((job) => job.gameId === selectedGameId) ?? null}
			provider={status?.provider}
            syncConflicts={syncConflicts}
            onResolveSyncConflict={(id, action) => void onResolveSyncConflict(id, action)}
          />
        ) : null}

        {activeView === "settings" ? (
          <SettingsView
            settings={settings}
            activeProtectionTarget={status?.protectionTarget ?? "local"}
            onChange={setSettings}
            onThemeModeChange={(value) => setSettings((s) => ({ ...s, themeMode: value }))}
            onTargetChange={(value) => setSettings((s) => ({ ...s, protectionTarget: value, cloudProvider: value === "google_drive" ? "google_drive" : "folder" }))}
            onConnectGoogleDrive={onConnectGoogleDrive}
            onDisconnectGoogleDrive={onDisconnectGoogleDrive}
            onAutostartChange={onAutostartChange}
            onBrowseSyncRoot={() => void onBrowseSyncRoot()}
            onPause={(scope, duration) => onSetProtectionControl(undefined, scope, duration)}
            onResume={() => onSetProtectionControl(undefined, "none")}
            onProtectionEnabled={(enabled) => onSetProtectionControl(undefined, enabled ? "none" : "all", "manual", enabled)}
            onRetry={async () => { await api.retryProtection(); setToast("Provider retry requested"); }}
            onSaveSettings={() => void onSaveSettings()}
            onSaveDestination={() => void onSaveDestination()}
            updateState={updateState}
            updateInstallBlocked={games.some((game) => game.running) || Boolean(status?.activeJobId)}
            onCheckForUpdate={() => void onCheckForUpdate()}
            onDownloadUpdate={() => void onDownloadUpdate()}
            onInstallUpdate={() => void onInstallUpdate()}
            onDiscardUpdate={() => void onDiscardUpdate()}
            onAutomaticUpdateChecks={(enabled) => void onAutomaticUpdateChecks(enabled)}
            onViewUpdateNotes={() => setUpdateNotesOpen(true)}
            onDownloadLogs={() => { api.downloadLogsZip().catch((err: unknown) => setError(toFriendlyError(err))); }}
            activityEvents={activityEvents}
            safeTime={safeTime}
            recoveryLibrary={recoveryLibrary}
            onRestoreRecovery={(entry) => void onRestoreRecoveryEntry(entry)}
            onOpenGame={openGame}
            onListDriveFolders={() => api.listGoogleDriveFolders()}
            onSetDriveFolder={async (id, name) => {
              await api.setGoogleDriveFolder(id, name);
              setSettings((s) => ({ ...s, googleDrive: { ...s.googleDrive, folderId: id, folderName: name } }));
              setToast(`Drive folder set to "${name}".`);
            }}
            onCreateDriveFolder={async (name) => {
              const result = await api.createGoogleDriveFolder(name);
              setSettings((s) => ({ ...s, googleDrive: { ...s.googleDrive, folderId: result.folderId, folderName: result.folderName ?? name } }));
              setToast(`Created and selected Drive folder "${result.folderName ?? name}".`);
            }}
            onExportSettings={async () => {
              try { await api.exportSettings(); } catch (err) { setError(toFriendlyError(err)); }
            }}
            onImportSettingsFile={() => {
              // The file picker and parsing happen inside SettingsView;
              // we expose the api method for a direct JSON import if needed.
              void api.importSettings({ schemaVersion: 1 });
            }}
          />
        ) : null}
      </main>

      {launchUpdatePromptOpen && updateState?.availableVersion ? (
        <div className="add-modal-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) resolveLaunchUpdateChoice("cancel"); }}>
          <section className="add-modal update-launch-dialog" role="dialog" aria-modal="true" aria-labelledby="update-launch-title">
            <div className="modal-head">
              <div>
                <span className="eyebrow">APPLICATION.UPDATE</span>
                <h2 id="update-launch-title">Version {updateState.availableVersion} is available</h2>
              </div>
            </div>
            <p>Rove can download the signed update in the background while your game launches. Installation will wait until every tracked game and sync job has finished.</p>
            {updateState.notes ? <div className="update-notes">{updateState.notes}</div> : null}
            <div className="modal-actions">
              <button className="btn primary" autoFocus type="button" onClick={() => resolveLaunchUpdateChoice("download")}>Download and play</button>
              <button className="btn" type="button" onClick={() => resolveLaunchUpdateChoice("later")}>Not now</button>
              <button className="btn" type="button" onClick={() => resolveLaunchUpdateChoice("cancel")}>Cancel launch</button>
            </div>
          </section>
        </div>
      ) : null}

      {updateNotesOpen && updateState?.lifecycle === "ready" && updateState.availableVersion ? (
        <div className="add-modal-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) setUpdateNotesOpen(false); }}>
          <section className="add-modal update-notes-dialog" role="dialog" aria-modal="true" aria-labelledby="update-notes-title" aria-describedby="update-notes-copy">
            <div className="modal-head">
              <div>
                <span className="eyebrow">APPLICATION.UPDATE</span>
                <h2 id="update-notes-title">What's new in v{updateState.availableVersion}</h2>
              </div>
            </div>
            <p id="update-notes-copy">The signed update is downloaded and verified. Review the release notes, then install whenever no tracked game or sync job is active.</p>
            <div className="update-notes" tabIndex={0} aria-label={`Release notes for version ${updateState.availableVersion}`}>
              {updateState.notes || "No release notes were supplied for this update."}
            </div>
            {games.some((game) => game.running) || Boolean(status?.activeJobId) ? <p className="snapshot-meta">Installation is waiting for tracked games and active sync work to finish.</p> : null}
            <div className="modal-actions">
              <button className="btn primary" autoFocus type="button" disabled={games.some((game) => game.running) || Boolean(status?.activeJobId)} onClick={() => { setUpdateNotesOpen(false); void onInstallUpdate(); }}>Install update</button>
              <button className="btn" type="button" onClick={() => setUpdateNotesOpen(false)}>Install later</button>
            </div>
          </section>
        </div>
      ) : null}
    </div>
  );
}
