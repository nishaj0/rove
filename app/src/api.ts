import type {
  ActivityEvent,
  BackupExportRequest,
  BackupExportResponse,
  BackupImportRequest,
  DriveFolder,
  Game,
  GameProtectionStatus,
  PlayGameRequest,
  RecoveryLibraryEntry,
  RoveSettings,
  StorageUsage,
  RecoveryPoint,
  Revision,
  ServiceControlState,
  SettingsUpdate,
  SnapshotSummary,
  SyncGameRequest,
  SyncPolicy,
  Status,
} from "./types";
import {
  normalizeGames,
  normalizeLogs,
  normalizeSettings,
  normalizeSnapshots,
  normalizeStatus,
  normalizeSyncJobs,
  normalizeSyncConflicts,
} from "./api/parsers";
import { BACKEND_BASE_URL } from "./constants/network";

type ErrorPayload = { error?: string };

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BACKEND_BASE_URL}${path}`, {
    cache: "no-store",
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
    ...init,
  });

  if (!res.ok) {
    let msg = `${res.status} ${res.statusText}`;
    try {
      const payload = (await res.json()) as ErrorPayload;
      if (payload.error) {
        msg = payload.error;
      }
    } catch {
      // keep fallback message
    }
    throw new Error(msg);
  }

  if (res.status === 204) {
    return undefined as T;
  }

  return (await res.json()) as T;
}

export const api = {
  async getGames() {
    const payload = await request<unknown>("/games");
    return normalizeGames(payload);
  },
  addGame(input: { name: string; path: string; executablePath: string; coverImageData?: string; version?: string }) {
    return request<Game>("/games", {
      method: "POST",
      body: JSON.stringify(input),
    });
  },
  deleteGame(id: string) {
    return request<void>(`/games/${encodeURIComponent(id)}`, {
      method: "DELETE",
    });
  },
  syncGame(payload: SyncGameRequest) {
    return request<{ status: string; jobId: string }>("/games/sync", {
      method: "POST",
      body: JSON.stringify(payload),
    }).catch((error: unknown) => {
      if (error instanceof Error && (/^(404|405)\b/.test(error.message) || /method not allowed/i.test(error.message))) {
        return request<{ status: string; jobId: string }>("/sync/game", {
          method: "POST",
          body: JSON.stringify(payload),
        });
      }
      throw error;
    });
  },
  playGame(payload: PlayGameRequest) {
    return request<{ status: string }>("/games/play", {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },
  updateGameCover(payload: { gameId: string; coverImageData?: string; mode?: "manual" | "executable_icon" | "generated" }) {
    return request<Game>("/games/cover", {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },
  updateGameProtectionPolicy(payload: { gameId: string; policy: SyncPolicy }) {
    return request<Game>("/games/protection-policy", {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },
  async getStatus() {
    const payload = await request<unknown>("/status");
    return normalizeStatus(payload);
  },
  async getSettings() {
    const payload = await request<unknown>("/settings");
    return normalizeSettings(payload);
  },
  async saveSettings(payload: SettingsUpdate) {
    const response = await request<unknown>("/settings", {
      method: "POST",
      body: JSON.stringify(payload),
    });
    return normalizeSettings(response);
  },
  async getRecentLogs(limit = 50) {
    const payload = await request<unknown>(`/logs/recent?limit=${limit}`);
    return normalizeLogs(payload);
  },
  async downloadLogsZip(): Promise<void> {
    const res = await fetch(`${BACKEND_BASE_URL}/logs/download`);
    if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const disposition = res.headers.get("Content-Disposition") ?? "";
    const nameMatch = /filename="?([^"]+)"?/.exec(disposition);
    const filename = nameMatch ? nameMatch[1] : "rove-logs.zip";
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    setTimeout(() => URL.revokeObjectURL(url), 10000);
  },
  setProtectionControl(payload: { gameId?: string; protectionEnabled: boolean; pauseScope: ServiceControlState["pauseScope"]; duration?: "manual" | "one_hour" | "until_login"; reason?: string }) {
    return request<Status | GameProtectionStatus>("/protection/control", {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },
  retryProtection() {
    return request<{ status: string }>("/protection/retry", { method: "POST" });
  },
  getRevisions(gameId: string) {
    return request<Revision[]>(`/revisions?gameId=${encodeURIComponent(gameId)}`);
  },
  restoreRevision(gameId: string, revisionId: string) {
    return request<{ status: string }>("/revisions/restore", {
      method: "POST",
      body: JSON.stringify({ gameId, revisionId, confirmed: true }),
    });
  },
  getRecoveryPoints(gameId: string) {
    return request<RecoveryPoint[]>(`/recovery-points?gameId=${encodeURIComponent(gameId)}`);
  },
  createRecoveryPoint(gameId: string, name: string) {
    return request<{ status: string; jobId: string }>("/recovery-points", {
      method: "POST",
      body: JSON.stringify({ gameId, name }),
    });
  },
  deleteRecoveryPoint(id: string) {
    return request<void>(`/recovery-points?id=${encodeURIComponent(id)}`, { method: "DELETE" });
  },
  getRecoveryLibrary() {
    return request<RecoveryLibraryEntry[]>("/recovery-library");
  },
  restoreAndTrackRecoveryLibrary(syncId: string, destination: string, executablePath: string) {
    return request<Game>("/recovery-library/restore-and-track", { method: "POST", body: JSON.stringify({ syncId, destination, executablePath, confirmed: true }) });
  },
  getStorageUsage(gameId: string) {
    return request<StorageUsage>(`/storage/usage?gameId=${encodeURIComponent(gameId)}`);
  },
  getActivity() {
    return request<ActivityEvent[]>("/activity");
  },
  exportBackup(payload: BackupExportRequest) {
    return request<BackupExportResponse>("/backup/export", {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },
  importBackup(payload: BackupImportRequest) {
    return request<{ status: string }>("/backup/import", {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },
  googleDriveStatus() { return request<{ connected: boolean; ready?: boolean; configurationError?: string; accountEmail?: string; reconnectRequired?: boolean; oauthClientSource?: "built_in" | "environment" | "oauth_json"; oauthCredentialSource?: "release_embedded" | "environment" | "oauth_json" | "missing" }>("/providers/google-drive/status"); },
  connectGoogleDrive() { return request<{ connected: boolean; ready?: boolean; configurationError?: string; accountEmail?: string; reconnectRequired?: boolean; oauthClientSource?: "built_in" | "environment" | "oauth_json"; oauthCredentialSource?: "release_embedded" | "environment" | "oauth_json" | "missing" }>("/providers/google-drive/connect", { method: "POST" }); },
  disconnectGoogleDrive() { return request<void>("/providers/google-drive/disconnect", { method: "POST" }); },
  async getSyncJobs(gameId?: string) {
    const payload = await request<unknown>(`/sync/jobs${gameId ? `?gameId=${encodeURIComponent(gameId)}` : ""}`);
    return normalizeSyncJobs(payload);
  },
  async getSyncConflicts(gameId: string) {
    const payload = await request<unknown>(`/sync/conflicts?gameId=${encodeURIComponent(gameId)}`);
    return normalizeSyncConflicts(payload);
  },
  resolveSyncConflict(id: string, action: "local" | "remote" | "both") {
    return request<{ status: string }>("/sync/conflicts", { method: "POST", body: JSON.stringify({ id, action }) });
  },
  async getSnapshots(gameId: string): Promise<SnapshotSummary[]> {
    const payload = await request<unknown>(`/snapshots?gameId=${encodeURIComponent(gameId)}`);
    return normalizeSnapshots(payload);
  },
  restoreSnapshot(gameId: string, snapshotId: string) {
    return request<{ status: string }>("/snapshots/restore", {
      method: "POST",
      body: JSON.stringify({ gameId, snapshotId }),
    });
  },
  deleteSnapshot(gameId: string, snapshotId: string) {
    return request<void>(`/snapshots/${encodeURIComponent(snapshotId)}?gameId=${encodeURIComponent(gameId)}`, {
      method: "DELETE",
    });
  },
  // Google Drive folder picker
  listGoogleDriveFolders() {
    return request<DriveFolder[]>("/providers/google-drive/folders");
  },
  setGoogleDriveFolder(folderId: string, folderName: string) {
    return request<{ connected: boolean; folderId?: string; folderName?: string }>("/providers/google-drive/folder", {
      method: "POST",
      body: JSON.stringify({ folderId, folderName }),
    });
  },
  createGoogleDriveFolder(createName: string) {
    return request<{ connected: boolean; folderId?: string; folderName?: string }>("/providers/google-drive/folder", {
      method: "POST",
      body: JSON.stringify({ createName }),
    });
  },
  // Settings export / import
  async exportSettings(): Promise<void> {
    const res = await fetch(`${BACKEND_BASE_URL}/settings/export`, { cache: "no-store" });
    if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "rove-settings.json";
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    setTimeout(() => URL.revokeObjectURL(url), 10000);
  },
  importSettings(settings: RoveSettings) {
    return request<{ status: string }>("/settings/import", {
      method: "POST",
      body: JSON.stringify(settings),
    });
  },
  // Recovery point: restore to an existing tracked game
  restoreRecoveryPointToGame(recoveryPointId: string, gameId: string) {
    return request<{ status: string; revisionId: string }>("/recovery-points/restore-to-game", {
      method: "POST",
      body: JSON.stringify({ recoveryPointId, gameId, confirmed: true }),
    });
  },
};
