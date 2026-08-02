export type ConflictPolicy = "latest_wins" | "keep_both";
export type ThemeMode = "dark" | "light" | "system";
export type CloudProvider = "folder" | "google_drive";
export type ProtectionTarget = "local" | "folder" | "google_drive";
export type ProtectionMode = "balanced" | "exit_only" | "continuous" | "manual";
export type PauseScope = "none" | "uploads" | "all";
export type ProviderState = "local_only" | "online" | "offline" | "reconnecting" | "authentication_required" | "rate_limited" | "quota_full" | "provider_unavailable" | "remote_data_problem";
export interface GoogleDriveConnection {
  connected: boolean;
  accountEmail?: string;
  reconnectRequired?: boolean;
  oauthClientSource?: "built_in" | "environment" | "oauth_json";
  oauthCredentialSource?: "release_embedded" | "environment" | "oauth_json" | "missing";
}

export interface SyncPolicy {
  enabled: boolean;
  mode: ProtectionMode;
  checkpointIntervalMinutes: number;
  exitProtection: boolean;
}

export interface ServiceControlState {
  protectionEnabled: boolean;
  pauseScope: PauseScope;
  pauseUntil?: string;
  pauseUntilLogin?: boolean;
  reason?: string;
}

export interface ProviderHealth {
  state: ProviderState;
  lastSuccessAt?: string;
  lastAttemptAt?: string;
  nextRetryAt?: string;
  errorCode?: string;
  message?: string;
  action?: string;
  pendingCount: number;
  pendingBytes: number;
}

export interface GameProtectionStatus {
  gameId: string;
  policy: SyncPolicy;
  serviceControl: ServiceControlState;
  dirtyFiles: number;
  lastLocalProtection?: string;
  lastCloudUpload?: string;
  pendingCount: number;
  pendingBytes: number;
  conflictCount: number;
  activeJob?: SyncJob;
}

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
  version?: string;
  coverSource?: "manual" | "executable_icon" | "generated" | string;
  backupPolicy: BackupPolicy;
  playtimeSeconds?: number;
  lastPlayedAt?: string;
  running?: boolean;
  syncId?: string;
  protectionPolicy: SyncPolicy;
  protectionStatus: GameProtectionStatus;
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
  syncRoot: string;
  deviceId: string;
  syncRetention: number;
  reviewConflicts: boolean;
  cloudProvider: CloudProvider;
  googleDrive: GoogleDriveConnection;
  protectionTarget: ProtectionTarget;
  serviceControl: ServiceControlState;
  automaticUpload: boolean;
  uploadOnMetered: boolean;
  remoteCheckBeforeLaunch: boolean;
  importantNotifications: boolean;
  debugLogging: boolean;
  autoRevisionRetention: number;
  protectionDefaultsReady: boolean;
  games: Game[];
}

export type SettingsUpdate = Partial<Pick<Settings,
  | "port"
  | "syncIntervalSeconds"
  | "conflictPolicy"
  | "themeMode"
  | "autostart"
  | "ignorePatterns"
  | "snapshotVaultPath"
  | "snapshotMaxPerGame"
  | "snapshotOnExit"
  | "snapshotPeriodic"
  | "snapshotIntervalMinutes"
  | "syncRoot"
  | "syncRetention"
  | "reviewConflicts"
  | "protectionTarget"
  | "serviceControl"
  | "automaticUpload"
  | "uploadOnMetered"
  | "remoteCheckBeforeLaunch"
  | "importantNotifications"
  | "debugLogging"
  | "autoRevisionRetention"
>>;

export interface Status {
  running: boolean;
  protectionTarget: ProtectionTarget;
  lastSyncTime: string;
  lastError?: string;
  pending: number;
  syncConfigured: boolean;
  provider: ProviderHealth;
  serviceControl: ServiceControlState;
  lastLocalProtection?: string;
  lastCloudUpload?: string;
  pendingBytes: number;
  conflictCount: number;
  activeJobId?: string;
}

export interface SyncJob {
  id: string;
  gameId?: string;
  reason: string;
  state: "queued" | "waiting" | "waiting_for_network" | "running" | "paused" | "completed" | "conflict" | "failed" | "authentication_required" | "quota_blocked";
  startedAt?: string;
  finishedAt?: string;
  changedFiles: number;
  bytes: number;
  error?: string;
  conflictIds?: string[];
  stage?: string;
  progress: number;
  retryCount: number;
  nextRetryAt?: string;
  errorCode?: string;
  priority: number;
  revisionId?: string;
  updatedAt?: string;
}

export interface Revision {
  id: string;
  gameId: string;
  syncId: string;
  parentIds?: string[];
  kind: string;
  name?: string;
  createdAt: string;
  fileCount: number;
  totalBytes: number;
  pinned: boolean;
  uploaded: boolean;
}

export interface RecoveryPoint {
  id: string;
  gameId: string;
  revisionId: string;
  name: string;
  createdAt: string;
  fileCount: number;
  totalBytes: number;
}

export interface ActivityEvent {
  id: number;
  category: string;
  severity: "info" | "warning" | "error";
  gameId?: string;
  title: string;
  detail: string;
  timestamp: string;
  actionTarget?: string;
  resolved: boolean;
}

export interface RecoveryLibraryEntry {
  profileId: string;
  gameId?: string;
  name: string;
  version?: string;
  coverSource?: string;
  coverAvailable?: boolean;
  playtimeSeconds?: number;
  syncId: string;
  executableName?: string;
  pathHint?: string;
  latestProtected: string;
  fileCount: number;
  totalBytes: number;
  automaticCount: number;
  recoveryPoints: RecoveryPoint[];
}

export interface StorageUsage {
  gameId: string;
  uniqueBytes: number;
  automaticCount: number;
  recoveryPoints: number;
  pendingBytes: number;
  reclaimableBytes: number;
}

export interface StorageSummary {
  localBytes: number;
  cloudBytes: number;
  pendingBytes: number;
  reclaimableBytes: number;
  gameCount: number;
  cloudAvailable: boolean;
}

export interface SyncConflict {
  id: string;
  gameId: string;
  path: string;
  localHash: string;
  remoteHash: string;
  createdAt: string;
  state: string;
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
  confirmed?: boolean;
}

export interface PlayGameRequest {
  gameId: string;
  allowOffline?: boolean;
}

export interface SyncGameRequest {
  gameId: string;
}

export interface SnapshotCreateResponse {
  status: string;
  snapshot?: SnapshotSummary | null;
}
