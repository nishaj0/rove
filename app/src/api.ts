import type {
  BackupHistoryEntry,
  BackupPolicy,
  BackupExportRequest,
  BackupExportResponse,
  BackupImportRequest,
  DebugInfo,
  Game,
  LogEntry,
  PlayGameRequest,
  Settings,
  SyncGameRequest,
  Status,
} from "./types";
import {
  normalizeBackupHistory,
  normalizeGames,
  normalizeLogs,
  normalizeSettings,
  normalizeStatus,
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
  addGame(input: { name: string; path: string; executablePath: string; coverImageData?: string }) {
    return request<Game>("/games", {
      method: "POST",
      body: JSON.stringify(input),
    });
  },
  getGame(id: string) {
    return request<Game>(`/games/${encodeURIComponent(id)}`);
  },
  updateGame(id: string, input: { name: string; path: string; executablePath: string; coverImageData?: string }) {
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
    return request<{ status: string }>("/sync", { method: "POST" });
  },
  syncGame(payload: SyncGameRequest) {
    return request<{ status: string }>("/games/sync", {
      method: "POST",
      body: JSON.stringify(payload),
    }).catch((error: unknown) => {
      if (error instanceof Error && (/^(404|405)\b/.test(error.message) || /method not allowed/i.test(error.message))) {
        return request<{ status: string }>("/sync/game", {
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
  updateGameCover(payload: { gameId: string; coverImageData: string }) {
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
  async getStatus() {
    const payload = await request<unknown>("/status");
    return normalizeStatus(payload);
  },
  async getSettings() {
    const payload = await request<unknown>("/settings");
    return normalizeSettings(payload);
  },
  saveSettings(payload: Settings) {
    return request<Settings>("/settings", {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },
  async getRecentLogs(limit = 50) {
    const payload = await request<unknown>(`/logs/recent?limit=${limit}`);
    return normalizeLogs(payload);
  },
  getDebugInfo() {
    return request<DebugInfo>("/debug/info");
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
};
