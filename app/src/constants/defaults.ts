import type { BackupPolicy, GameProtectionStatus, Settings, SyncPolicy } from "../types";

export const DEFAULT_BACKUP_POLICY: BackupPolicy = {
  enabled: true,
  intervalMinutes: 10,
  exitDelaySeconds: 10,
  retentionCount: 5,
};

export const DEFAULT_SETTINGS: Settings = {
  port: 7331,
  syncIntervalSeconds: 30,
  conflictPolicy: "latest_wins",
  themeMode: "dark",
  autostart: false,
  ignorePatterns: ["*.tmp", "*.log", "cache"],
  snapshotVaultPath: "",
  snapshotMaxPerGame: 5,
  snapshotOnExit: true,
  snapshotPeriodic: false,
  snapshotIntervalMinutes: 15,
  syncRoot: "",
  deviceId: "",
  syncRetention: 10,
  reviewConflicts: true,
  cloudProvider: "folder",
  googleDrive: { connected: false },
  protectionTarget: "local",
  serviceControl: { protectionEnabled: true, pauseScope: "none" },
  automaticUpload: true,
  uploadOnMetered: false,
  remoteCheckBeforeLaunch: true,
  importantNotifications: true,
  debugLogging: false,
  autoRevisionRetention: 3,
  protectionDefaultsReady: true,
  games: [],
};

export const DEFAULT_SYNC_POLICY: SyncPolicy = {
  enabled: true,
  mode: "balanced",
  checkpointIntervalMinutes: 10,
  exitProtection: true,
};

export function newDefaultBackupPolicy(): BackupPolicy {
  return { ...DEFAULT_BACKUP_POLICY };
}

export function newDefaultSettings(): Settings {
  return {
    ...DEFAULT_SETTINGS,
    ignorePatterns: [...DEFAULT_SETTINGS.ignorePatterns],
    games: [],
  };
}

export function newDefaultSyncPolicy(): SyncPolicy {
  return { ...DEFAULT_SYNC_POLICY };
}

export function newDefaultProtectionStatus(gameId = ""): GameProtectionStatus {
  return {
    gameId,
    policy: newDefaultSyncPolicy(),
    serviceControl: { protectionEnabled: true, pauseScope: "none" },
    dirtyFiles: 0,
    pendingCount: 0,
    pendingBytes: 0,
    conflictCount: 0,
  };
}
