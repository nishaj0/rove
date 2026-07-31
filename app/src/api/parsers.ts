import type {
  BackupHistoryEntry, BackupPolicy, CloudProvider, ConflictPolicy, Game,
  GameProtectionStatus, GoogleDriveConnection, LogEntry, PauseScope,
  ProtectionMode, ProtectionTarget, ProviderHealth, ProviderState,
  ServiceControlState, Settings, SnapshotSummary, Status, SyncConflict,
  SyncJob, SyncPolicy, ThemeMode,
} from "../types";
import {
  newDefaultBackupPolicy, newDefaultProtectionStatus, newDefaultSettings,
  newDefaultSyncPolicy,
} from "../constants/defaults";

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
function asCloudProvider(input: unknown, fallback: CloudProvider): CloudProvider {
  return input === "google_drive" || input === "folder" ? input : fallback;
}
function asProtectionTarget(input: unknown, fallback: ProtectionTarget): ProtectionTarget {
  return input === "local" || input === "google_drive" || input === "folder" ? input : fallback;
}

function normalizeBackupPolicy(input: unknown): BackupPolicy {
  const defaults = newDefaultBackupPolicy();
  if (!isRecord(input)) return defaults;
  return {
    enabled: asBoolean(input.enabled, defaults.enabled),
    intervalMinutes: asNumber(input.intervalMinutes, defaults.intervalMinutes),
    exitDelaySeconds: asNumber(input.exitDelaySeconds, defaults.exitDelaySeconds),
    retentionCount: asNumber(input.retentionCount, defaults.retentionCount),
  };
}

function normalizeSyncPolicy(input: unknown): SyncPolicy {
  const defaults = newDefaultSyncPolicy();
  if (!isRecord(input)) return defaults;
  const candidate = asString(input.mode, defaults.mode) as ProtectionMode;
  const mode = ["balanced", "exit_only", "continuous", "manual"].includes(candidate) ? candidate : defaults.mode;
  return {
    enabled: asBoolean(input.enabled, defaults.enabled),
    mode,
    checkpointIntervalMinutes: asNumber(input.checkpointIntervalMinutes, defaults.checkpointIntervalMinutes),
    exitProtection: asBoolean(input.exitProtection, defaults.exitProtection),
  };
}

function normalizeControl(input: unknown): ServiceControlState {
  if (!isRecord(input)) return { protectionEnabled: true, pauseScope: "none" };
  const candidate = asString(input.pauseScope, "none") as PauseScope;
  const pauseScope = ["none", "uploads", "all"].includes(candidate) ? candidate : "none";
  return {
    protectionEnabled: asBoolean(input.protectionEnabled, true),
    pauseScope,
    pauseUntil: asString(input.pauseUntil) || undefined,
    pauseUntilLogin: asBoolean(input.pauseUntilLogin, false),
    reason: asString(input.reason) || undefined,
  };
}

function normalizeProvider(input: unknown): ProviderHealth {
  if (!isRecord(input)) return { state: "local_only", pendingCount: 0, pendingBytes: 0 };
  return {
    state: asString(input.state, "local_only") as ProviderState,
    lastSuccessAt: asString(input.lastSuccessAt) || undefined,
    lastAttemptAt: asString(input.lastAttemptAt) || undefined,
    nextRetryAt: asString(input.nextRetryAt) || undefined,
    errorCode: asString(input.errorCode) || undefined,
    message: asString(input.message) || undefined,
    action: asString(input.action) || undefined,
    pendingCount: asNumber(input.pendingCount, 0),
    pendingBytes: asNumber(input.pendingBytes, 0),
  };
}

function normalizeProtectionStatus(input: unknown, gameId: string): GameProtectionStatus {
  const defaults = newDefaultProtectionStatus(gameId);
  if (!isRecord(input)) return defaults;
  return {
    gameId: asString(input.gameId, gameId),
    policy: normalizeSyncPolicy(input.policy),
    serviceControl: normalizeControl(input.serviceControl),
    dirtyFiles: asNumber(input.dirtyFiles, 0),
    lastLocalProtection: asString(input.lastLocalProtection) || undefined,
    lastCloudUpload: asString(input.lastCloudUpload) || undefined,
    pendingCount: asNumber(input.pendingCount, 0),
    pendingBytes: asNumber(input.pendingBytes, 0),
    conflictCount: asNumber(input.conflictCount, 0),
    activeJob: isRecord(input.activeJob) ? normalizeSyncJobs([input.activeJob])[0] : undefined,
  };
}

export function normalizeGames(input: unknown): Game[] {
  if (!Array.isArray(input)) return [];
  return input.map((value): Game | null => {
    if (!isRecord(value)) return null;
    const id = asString(value.id);
    return {
      id,
      name: asString(value.name),
      path: asString(value.path),
      executablePath: asString(value.executablePath),
      coverImageData: asString(value.coverImageData),
      backupPolicy: normalizeBackupPolicy(value.backupPolicy),
      playtimeSeconds: asNumber(value.playtimeSeconds, 0),
      lastPlayedAt: asString(value.lastPlayedAt),
      running: asBoolean(value.running, false),
      syncId: asString(value.syncId),
      protectionPolicy: normalizeSyncPolicy(value.protectionPolicy),
      protectionStatus: normalizeProtectionStatus(value.protectionStatus, id),
    };
  }).filter((value): value is Game => value !== null);
}

export function normalizeBackupHistory(input: unknown): BackupHistoryEntry[] {
  if (!Array.isArray(input)) return [];
  return input.map((value): BackupHistoryEntry | null => {
    if (!isRecord(value)) return null;
    return {
      id: asNumber(value.id, 0), gameId: asString(value.gameId),
      path: asString(value.path), createdAt: asString(value.createdAt),
      reason: asString(value.reason), signature: asString(value.signature),
      status: asString(value.status, "ok"),
    };
  }).filter((value): value is BackupHistoryEntry => value !== null);
}

function normalizeGoogleDrive(input: unknown): GoogleDriveConnection {
  return isRecord(input) ? {
    connected: asBoolean(input.connected, false),
    accountEmail: asString(input.accountEmail),
    reconnectRequired: asBoolean(input.reconnectRequired, false),
    oauthClientSource: ["built_in", "environment", "oauth_json"].includes(asString(input.oauthClientSource))
      ? asString(input.oauthClientSource) as "built_in" | "environment" | "oauth_json"
      : undefined,
    oauthCredentialSource: ["release_embedded", "environment", "oauth_json", "missing"].includes(asString(input.oauthCredentialSource))
      ? asString(input.oauthCredentialSource) as "release_embedded" | "environment" | "oauth_json" | "missing"
      : undefined,
  } : { connected: false };
}

export function normalizeSnapshots(input: unknown): SnapshotSummary[] {
  if (!Array.isArray(input)) return [];
  return input.map((value): SnapshotSummary | null => {
    if (!isRecord(value)) return null;
    return {
      id: asString(value.id), seq: asNumber(value.seq, 0),
      createdAt: asString(value.createdAt), trigger: asString(value.trigger),
      changedFiles: asNumber(value.changedFiles, 0), totalSize: asNumber(value.totalSize, 0),
    };
  }).filter((value): value is SnapshotSummary => value !== null);
}

export function normalizeLogs(input: unknown): LogEntry[] {
  if (!Array.isArray(input)) return [];
  return input.map((value): LogEntry | null => {
    if (!isRecord(value)) return null;
    return {
      id: asNumber(value.id, 0), timestamp: asString(value.timestamp),
      level: asString(value.level, "info"), event: asString(value.event, "app"),
      message: asString(value.message),
    };
  }).filter((value): value is LogEntry => value !== null);
}

export function normalizeSettings(input: unknown): Settings {
  const defaults = newDefaultSettings();
  if (!isRecord(input)) return defaults;
  const patterns = Array.isArray(input.ignorePatterns)
    ? input.ignorePatterns.filter((value): value is string => typeof value === "string")
    : [...defaults.ignorePatterns];
  return {
    port: asNumber(input.port, defaults.port),
    syncIntervalSeconds: asNumber(input.syncIntervalSeconds, defaults.syncIntervalSeconds),
    conflictPolicy: asConflictPolicy(input.conflictPolicy, defaults.conflictPolicy),
    themeMode: asThemeMode(input.themeMode, defaults.themeMode),
    autostart: asBoolean(input.autostart, defaults.autostart),
    ignorePatterns: patterns,
    snapshotVaultPath: asString(input.snapshotVaultPath),
    snapshotMaxPerGame: asNumber(input.snapshotMaxPerGame, defaults.snapshotMaxPerGame),
    snapshotOnExit: asBoolean(input.snapshotOnExit, defaults.snapshotOnExit),
    snapshotPeriodic: asBoolean(input.snapshotPeriodic, defaults.snapshotPeriodic),
    snapshotIntervalMinutes: asNumber(input.snapshotIntervalMinutes, defaults.snapshotIntervalMinutes),
    syncRoot: asString(input.syncRoot),
    deviceId: asString(input.deviceId),
    syncRetention: asNumber(input.syncRetention, defaults.syncRetention),
    reviewConflicts: asBoolean(input.reviewConflicts, true),
    cloudProvider: asCloudProvider(input.cloudProvider, defaults.cloudProvider),
    googleDrive: normalizeGoogleDrive(input.googleDrive),
    protectionTarget: asProtectionTarget(input.protectionTarget, defaults.protectionTarget),
    serviceControl: normalizeControl(input.serviceControl),
    automaticUpload: asBoolean(input.automaticUpload, defaults.automaticUpload),
    uploadOnMetered: asBoolean(input.uploadOnMetered, defaults.uploadOnMetered),
    remoteCheckBeforeLaunch: asBoolean(input.remoteCheckBeforeLaunch, defaults.remoteCheckBeforeLaunch),
    importantNotifications: asBoolean(input.importantNotifications, defaults.importantNotifications),
    debugLogging: asBoolean(input.debugLogging, defaults.debugLogging),
    autoRevisionRetention: asNumber(input.autoRevisionRetention, defaults.autoRevisionRetention),
    protectionDefaultsReady: asBoolean(input.protectionDefaultsReady, true),
    games: normalizeGames(input.games),
  };
}

export function normalizeStatus(input: unknown): Status {
  const fallback: Status = {
    running: false, protectionTarget: "local", lastSyncTime: "", pending: 0, syncConfigured: false,
    provider: { state: "local_only", pendingCount: 0, pendingBytes: 0 },
    serviceControl: { protectionEnabled: true, pauseScope: "none" },
    pendingBytes: 0, conflictCount: 0,
  };
  if (!isRecord(input)) return fallback;
  return {
    running: asBoolean(input.running, false),
    protectionTarget: asProtectionTarget(input.protectionTarget, "local"),
    lastSyncTime: asString(input.lastSyncTime),
    lastError: asString(input.lastError) || undefined,
    pending: asNumber(input.pending, 0),
    syncConfigured: asBoolean(input.syncConfigured, false),
    provider: normalizeProvider(input.provider),
    serviceControl: normalizeControl(input.serviceControl),
    lastLocalProtection: asString(input.lastLocalProtection) || undefined,
    lastCloudUpload: asString(input.lastCloudUpload) || undefined,
    pendingBytes: asNumber(input.pendingBytes, 0),
    conflictCount: asNumber(input.conflictCount, 0),
    activeJobId: asString(input.activeJobId) || undefined,
  };
}

export function normalizeSyncJobs(input: unknown): SyncJob[] {
  if (!Array.isArray(input)) return [];
  return input.filter(isRecord).map((value) => ({
    id: asString(value.id), gameId: asString(value.gameId), reason: asString(value.reason),
    state: asString(value.state, "failed") as SyncJob["state"],
    startedAt: asString(value.startedAt), finishedAt: asString(value.finishedAt),
    changedFiles: asNumber(value.changedFiles, 0), bytes: asNumber(value.bytes, 0),
    error: asString(value.error), conflictIds: Array.isArray(value.conflictIds)
      ? value.conflictIds.filter((id): id is string => typeof id === "string") : [],
    stage: asString(value.stage), progress: asNumber(value.progress, 0),
    retryCount: asNumber(value.retryCount, 0), nextRetryAt: asString(value.nextRetryAt),
    errorCode: asString(value.errorCode), priority: asNumber(value.priority, 0),
    revisionId: asString(value.revisionId), updatedAt: asString(value.updatedAt),
  }));
}

export function normalizeSyncConflicts(input: unknown): SyncConflict[] {
  if (!Array.isArray(input)) return [];
  return input.filter(isRecord).map((value) => ({
    id: asString(value.id), gameId: asString(value.gameId), path: asString(value.path),
    localHash: asString(value.localHash), remoteHash: asString(value.remoteHash),
    createdAt: asString(value.createdAt), state: asString(value.state),
  }));
}
