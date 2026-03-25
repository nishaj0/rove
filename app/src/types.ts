export type ConflictPolicy = "latest_wins" | "keep_both";

export interface Game {
  id: string;
  name: string;
  path: string;
}

export interface Settings {
  port: number;
  syncIntervalSeconds: number;
  conflictPolicy: ConflictPolicy;
  autostart: boolean;
  ignorePatterns: string[];
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

export interface BackupExportRequest {
  gameId: string;
  destinationPath: string;
}

export interface BackupImportRequest {
  gameId: string;
  sourcePath: string;
  merge: boolean;
}
