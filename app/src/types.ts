export type ConflictPolicy = "latest_wins" | "keep_both";
export type ThemeMode = "dark" | "light" | "system";

export interface BackupPolicy {
  enabled: boolean;
  intervalMinutes: number;
  exitDelaySeconds: number;
  retentionCount: number;
}

export interface BackupHistoryEntry {
  id: number;
  gameId: string;
  path: string;
  createdAt: string;
  reason: string;
  signature: string;
  status: string;
}

export interface SnapshotSummary {
  id: string;
  seq: number;
  createdAt: string;
  trigger: string;
  changedFiles: number;
  totalSize: number;
}

export interface Game {
  id: string;
  name: string;
  path: string;
  executablePath: string;
  coverImageData?: string;
  backupPolicy: BackupPolicy;
  playtimeSeconds?: number;
  lastPlayedAt?: string;
  running?: boolean;
}

export interface Settings {
  port: number;
  syncIntervalSeconds: number;
  conflictPolicy: ConflictPolicy;
  themeMode: ThemeMode;
  autostart: boolean;
  ignorePatterns: string[];
  snapshotVaultPath: string;
  snapshotMaxPerGame: number;
  snapshotOnExit: boolean;
  snapshotPeriodic: boolean;
  snapshotIntervalMinutes: number;
  games: Game[];
}

export interface Status {
  running: boolean;
  lastSyncTime: string;
  lastError?: string;
  pending: number;
}

export interface LogEntry {
  id: number;
  timestamp: string;
  level: string;
  event: string;
  message: string;
}

export interface DebugInfo {
  debug: boolean;
  pid: number;
  startedAt: string;
  uptimeSec: number;
  port: number;
  configPath: string;
  sqlitePath: string;
  exePath: string;
  startupCommand?: string;
  startupCommandExists?: boolean;
}

export interface LatestReleaseInfo {
  tagName: string;
  htmlUrl: string;
}

export interface BackupExportRequest {
  gameId: string;
  destinationPath: string;
}

export interface BackupExportResponse {
  status: string;
  path: string;
}

export interface BackupImportRequest {
  gameId: string;
  sourcePath: string;
  merge: boolean;
}

export interface PlayGameRequest {
  gameId: string;
}

export interface SyncGameRequest {
  gameId: string;
}

export interface SnapshotCreateResponse {
  status: string;
  snapshot?: SnapshotSummary | null;
}
