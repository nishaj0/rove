import type {
  BackupExportRequest,
  BackupImportRequest,
  DebugInfo,
  Game,
  LogEntry,
  Settings,
  Status,
} from "./types";

const BASE_URL = "http://127.0.0.1:7331";

type ErrorPayload = { error?: string };

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
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
  getGames() {
    return request<Game[]>("/games");
  },
  addGame(input: { name: string; path: string }) {
    return request<Game>("/games", {
      method: "POST",
      body: JSON.stringify(input),
    });
  },
  updateGame(id: string, input: { name: string; path: string }) {
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
  getStatus() {
    return request<Status>("/status");
  },
  getSettings() {
    return request<Settings>("/settings");
  },
  saveSettings(payload: Settings) {
    return request<Settings>("/settings", {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },
  getRecentLogs(limit = 50) {
    return request<LogEntry[]>(`/logs/recent?limit=${limit}`);
  },
  getDebugInfo() {
    return request<DebugInfo>("/debug/info");
  },
  exportBackup(payload: BackupExportRequest) {
    return request<{ status: string }>("/backup/export", {
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
};
