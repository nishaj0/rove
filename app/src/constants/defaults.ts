import type { BackupPolicy, Settings } from "../types";

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
  games: [],
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
