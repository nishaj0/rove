import type {
  ActivityEvent,
  BackupHistoryEntry,
  BackupPolicy,
  BackupExportRequest,
  BackupExportResponse,
  BackupImportRequest,
  DebugInfo,
  Game,
  GameProtectionStatus,
  LogEntry,
  PlayGameRequest,
  RecoveryLibraryEntry,
  StorageSummary,
  StorageUsage,
  RecoveryPoint,
  Revision,
  ServiceControlState,
  Settings,
  SettingsUpdate,
  SnapshotCreateResponse,
  SnapshotSummary,
  SyncGameRequest,
  SyncPolicy,
  Status,
} from "./types";
import {
  normalizeBackupHistory,
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
  getGame(id: string) {
    return request<Game>(`/games/${encodeURIComponent(id)}`);
  },
  updateGame(id: string, input: { name: string; path: string; executablePath: string; coverImageData?: string; version?: string }) {
    return request<Game>(`/games/${encodeURIComponent(id)}`, {
      method: "PUT",
      body: JSON.stringify(input),
    });
  },
  deleteGame(id: string) {
    return request<void>(`/games/${encodeURIComponent(id)}`, {
      method: "DELETE",
    });
  },
  triggerSync() {
    return request<{ status: string; jobId: string }>("/sync", { method: "POST" });
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
  updateGameBackupPolicy(payload: { gameId: string; backupPolicy: BackupPolicy }) {
    return request<Game>("/games/backup-policy", {
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
  getDebugInfo() {
    return request<DebugInfo>("/debug/info");
  },
  async downloadLogsZip(): Promise<void> {
    const res = await fetch(`${BACKEND_BASE_URL}/logs/download`);
    if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const disposition = res.headers.get("Content-Disposition") ?? "";
    const nameMatch = /filename="?([^"]+)"?/.exec(disposition);
    const filename = nameMatch ? nameMatch[1] : "cloud-saver-logs.zip";
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    setTimeout(() => URL.revokeObjectURL(url), 10000);
  },
  getProtectionStatus(gameId?: string) {
    return request<Status | GameProtectionStatus>(`/protection/status${gameId ? `?gameId=${encodeURIComponent(gameId)}` : ""}`);
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
  restoreRecoveryLibrary(syncId: string, destination: string) {
    return request<{ status: string }>("/recovery-library/restore", {
      method: "POST",
      body: JSON.stringify({ syncId, destination, confirmed: true }),
    });
  },
  restoreAndTrackRecoveryLibrary(syncId: string, destination: string, executablePath: string) {
    return request<Game>("/recovery-library/restore-and-track", { method: "POST", body: JSON.stringify({ syncId, destination, executablePath, confirmed: true }) });
  },
  getStorageUsage(gameId: string) {
    return request<StorageUsage>(`/storage/usage?gameId=${encodeURIComponent(gameId)}`);
  },
  getStorageSummary() { return request<StorageSummary>("/storage/summary"); },
  getActivity() {
    return request<ActivityEvent[]>("/activity");
  },
  getDiagnostics() {
    return request<Record<string, unknown>>("/diagnostics");
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
  async getBackupHistory(gameId: string, limit = 20): Promise<BackupHistoryEntry[]> {
    const payload = await request<unknown>(`/backup/history?gameId=${encodeURIComponent(gameId)}&limit=${limit}`);
    return normalizeBackupHistory(payload);
  },
  googleDriveStatus() { return request<{ connected: boolean; accountEmail?: string; reconnectRequired?: boolean; oauthClientSource?: "built_in" | "environment" | "oauth_json"; oauthCredentialSource?: "release_embedded" | "environment" | "oauth_json" | "missing" }>("/providers/google-drive/status"); },
  connectGoogleDrive() { return request<{ connected: boolean; accountEmail?: string; reconnectRequired?: boolean; oauthClientSource?: "built_in" | "environment" | "oauth_json"; oauthCredentialSource?: "release_embedded" | "environment" | "oauth_json" | "missing" }>("/providers/google-drive/connect", { method: "POST" }); },
  disconnectGoogleDrive() { return request<void>("/providers/google-drive/disconnect", { method: "POST" }); },
  linkGameToCloud(payload: { gameId: string; syncId: string }) {
    return request<Game>("/games/pair", { method: "POST", body: JSON.stringify(payload) });
  },
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
  createSnapshot(gameId: string) {
    return request<SnapshotCreateResponse>("/snapshots", {
      method: "POST",
      body: JSON.stringify({ gameId }),
    });
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
};
