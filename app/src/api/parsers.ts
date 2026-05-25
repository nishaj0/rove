import type { BackupHistoryEntry, BackupPolicy, ConflictPolicy, Game, LogEntry, Settings, SnapshotSummary, Status, ThemeMode } from "../types";
import { newDefaultBackupPolicy, newDefaultSettings } from "../constants/defaults";

function isRecord(input: unknown): input is Record<string, unknown> {
  return typeof input === "object" && input !== null;
}

function asString(input: unknown, fallback = ""): string {
  return typeof input === "string" ? input : fallback;
}

function asNumber(input: unknown, fallback: number): number {
  return typeof input === "number" && Number.isFinite(input) ? input : fallback;
}

function asBoolean(input: unknown, fallback: boolean): boolean {
  return typeof input === "boolean" ? input : fallback;
}

function asConflictPolicy(input: unknown, fallback: ConflictPolicy): ConflictPolicy {
  return input === "keep_both" || input === "latest_wins" ? input : fallback;
}

function asThemeMode(input: unknown, fallback: ThemeMode): ThemeMode {
  return input === "dark" || input === "light" || input === "system" ? input : fallback;
}

function normalizeBackupPolicy(input: unknown): BackupPolicy {
  const defaults = newDefaultBackupPolicy();
  if (!isRecord(input)) {
    return defaults;
  }
  return {
    enabled: asBoolean(input.enabled, defaults.enabled),
    intervalMinutes: asNumber(input.intervalMinutes, defaults.intervalMinutes),
    exitDelaySeconds: asNumber(input.exitDelaySeconds, defaults.exitDelaySeconds),
    retentionCount: asNumber(input.retentionCount, defaults.retentionCount),
  };
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
        executablePath: asString(value.executablePath),
        coverImageData: asString(value.coverImageData),
        backupPolicy: normalizeBackupPolicy(value.backupPolicy),
        playtimeSeconds: asNumber(value.playtimeSeconds, 0),
        lastPlayedAt: asString(value.lastPlayedAt),
        running: asBoolean(value.running, false),
      };
    })
    .filter((value): value is Game => value !== null);
}

export function normalizeBackupHistory(input: unknown): BackupHistoryEntry[] {
  if (!Array.isArray(input)) {
    return [];
  }
  return input
    .map((value): BackupHistoryEntry | null => {
      if (!isRecord(value)) {
        return null;
      }
      return {
        id: asNumber(value.id, 0),
        gameId: asString(value.gameId),
        path: asString(value.path),
        createdAt: asString(value.createdAt),
        reason: asString(value.reason),
        signature: asString(value.signature),
        status: asString(value.status, "ok"),
      };
    })
    .filter((value): value is BackupHistoryEntry => value !== null);
}

export function normalizeSnapshots(input: unknown): SnapshotSummary[] {
  if (!Array.isArray(input)) {
    return [];
  }
  return input
    .map((value): SnapshotSummary | null => {
      if (!isRecord(value)) {
        return null;
      }
      return {
        id: asString(value.id),
        seq: asNumber(value.seq, 0),
        createdAt: asString(value.createdAt),
        trigger: asString(value.trigger),
        changedFiles: asNumber(value.changedFiles, 0),
        totalSize: asNumber(value.totalSize, 0),
      };
    })
    .filter((value): value is SnapshotSummary => value !== null);
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
  const defaults = newDefaultSettings();
  if (!isRecord(input)) {
    return defaults;
  }

  const ignorePatternsRaw = Array.isArray(input.ignorePatterns) ? input.ignorePatterns : [];
  const ignorePatterns = ignorePatternsRaw.filter((value): value is string => typeof value === "string");

  return {
    port: asNumber(input.port, defaults.port),
    syncIntervalSeconds: asNumber(input.syncIntervalSeconds, defaults.syncIntervalSeconds),
    conflictPolicy: asConflictPolicy(input.conflictPolicy, defaults.conflictPolicy),
    themeMode: asThemeMode(input.themeMode, defaults.themeMode),
    autostart: Boolean(input.autostart),
    ignorePatterns,
    snapshotVaultPath: asString(input.snapshotVaultPath),
    snapshotMaxPerGame: asNumber(input.snapshotMaxPerGame, defaults.snapshotMaxPerGame),
    snapshotOnExit: asBoolean(input.snapshotOnExit, defaults.snapshotOnExit),
    snapshotPeriodic: asBoolean(input.snapshotPeriodic, defaults.snapshotPeriodic),
    snapshotIntervalMinutes: asNumber(input.snapshotIntervalMinutes, defaults.snapshotIntervalMinutes),
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
