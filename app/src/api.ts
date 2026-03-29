import type {
  BackupExportRequest,
  BackupImportRequest,
  ConflictPolicy,
  DebugInfo,
  Game,
  LogEntry,
  Settings,
  Status,
} from "./types";

const BASE_URL = "http://127.0.0.1:7331";

type ErrorPayload = { error?: string };

const defaultSettings: Settings = {
  port: 7331,
  syncIntervalSeconds: 30,
  conflictPolicy: "latest_wins",
  autostart: false,
  ignorePatterns: ["*.tmp", "*.log", "cache"],
  games: [],
};

function isRecord(input: unknown): input is Record<string, unknown> {
  return typeof input === "object" && input !== null;
}

function asString(input: unknown, fallback = ""): string {
  return typeof input === "string" ? input : fallback;
}

function asNumber(input: unknown, fallback: number): number {
  return typeof input === "number" && Number.isFinite(input) ? input : fallback;
}

function asConflictPolicy(input: unknown, fallback: ConflictPolicy): ConflictPolicy {
  return input === "keep_both" || input === "latest_wins" ? input : fallback;
}

function normalizeGames(input: unknown): Game[] {
  if (!Array.isArray(input)) {
    return [];
  }
  return input
    .map((value): Game | null => {
      if (!isRecord(value)) {
        return null;
      }
      return {
        id: asString(value.id),
        name: asString(value.name),
        path: asString(value.path),
      };
    })
    .filter((value): value is Game => value !== null);
}

function normalizeLogs(input: unknown): LogEntry[] {
  if (!Array.isArray(input)) {
    return [];
  }
  return input
    .map((value): LogEntry | null => {
      if (!isRecord(value)) {
        return null;
      }
      return {
        id: asNumber(value.id, 0),
        timestamp: asString(value.timestamp),
        level: asString(value.level, "info"),
        event: asString(value.event, "app"),
        message: asString(value.message),
      };
    })
    .filter((value): value is LogEntry => value !== null);
}

function normalizeSettings(input: unknown): Settings {
  if (!isRecord(input)) {
    return { ...defaultSettings };
  }

  const ignorePatternsRaw = Array.isArray(input.ignorePatterns) ? input.ignorePatterns : [];
  const ignorePatterns = ignorePatternsRaw.filter((value): value is string => typeof value === "string");

  return {
    port: asNumber(input.port, defaultSettings.port),
    syncIntervalSeconds: asNumber(input.syncIntervalSeconds, defaultSettings.syncIntervalSeconds),
    conflictPolicy: asConflictPolicy(input.conflictPolicy, defaultSettings.conflictPolicy),
    autostart: Boolean(input.autostart),
    ignorePatterns,
    games: normalizeGames(input.games),
  };
}

function normalizeStatus(input: unknown): Status {
  if (!isRecord(input)) {
    return { running: false, lastSyncTime: "", pending: 0 };
  }
  return {
    running: Boolean(input.running),
    lastSyncTime: asString(input.lastSyncTime),
    lastError: typeof input.lastError === "string" ? input.lastError : undefined,
    pending: asNumber(input.pending, 0),
  };
}

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
  async getGames() {
    const payload = await request<unknown>("/games");
    return normalizeGames(payload);
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
