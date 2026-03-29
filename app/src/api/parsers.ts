import type { ConflictPolicy, Game, LogEntry, Settings, Status } from "../types";

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

export function normalizeGames(input: unknown): Game[] {
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

export function normalizeLogs(input: unknown): LogEntry[] {
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

export function normalizeSettings(input: unknown): Settings {
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

export function normalizeStatus(input: unknown): Status {
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
