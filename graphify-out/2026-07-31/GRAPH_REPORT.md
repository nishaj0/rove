# Graph Report - .  (2026-07-31)

## Corpus Check
- 110 files · ~71,350 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1321 nodes · 2807 edges · 91 communities (80 shown, 11 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 251 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- Protection Engine
- Windows Tray and Security
- Legacy Snapshots
- Application Lifecycle
- Google Drive Provider
- Signed Application Updater
- Settings and Game Models
- Google OAuth
- Activity and Diagnostics
- Revision and Recovery Models
- React Application Shell
- SQLite Persistence
- Frontend Tooling
- API Client and Types
- Tauri Packaging
- API Response Parsers
- Legacy Automatic Backups
- Game API Handlers
- Runtime Protection Control
- Tauri Bootstrap
- Status Go
- Tsconfig Json
- Handlesnapshotbyid
- Daemon Shutdown Request
- Adaptive Game Monitor
- Handleactivity
- Sync Version Mjs
- Branding Ts
- Zip Go
- Gameprofile
- Servicecontrolstate
- Setup Updater Signing Mjs
- Default Json
- Defaults Ts
- Stage Backend Daemon Mjs
- Backend Api Client
- Classifylocalerror
- Generate Update Manifest Mjs
- Handler
- Gameprotectionstatus
- Status
- Tsconfig Node Json
- Handlegoogledriveconnect
- Nativedialog Ts
- Handlestatus
- Game
- Recoverylibraryentry
- Handlebackupexport
- Network Ts
- Run Automatic Backup
- Handlers Lifecycle Test Go
- Handledebuginfo
- Handlelogsrecent
- Handleprepareupdate
- Handlesettings
- Backend Reachability Check
- Application Updater
- Persist Google Drive Connection
- Windows Release Workflow
- Nostalgia Dark Mode
- Updates Ts
- Reload Watcher For Running Games
- Application Run
- Network Stub Go
- Network Windows Go
- Recovery Library
- Vite Env D Ts
- Application Stop
- Cloud Saver
- Cloud Saver Backend
- Sqlite Log Store
- Backend Daemon Staging Script
- Windows Tray Manager

## God Nodes (most connected - your core abstractions)
1. `Engine` - 71 edges
2. `Protection Engine New` - 40 edges
3. `writeJSON()` - 39 edges
4. `New()` - 30 edges
5. `writeErr()` - 30 edges
6. `Store` - 28 edges
7. `Store` - 26 edges
8. `Service` - 25 edges
9. `Provider-neutral reconciliation` - 25 edges
10. `Monitor` - 23 edges

## Surprising Connections (you probably didn't know these)
- `Windows release workflow` --calls--> `Signed update manifest generator`  [EXTRACTED]
  .github/workflows/release-windows.yml → scripts/generate-update-manifest.mjs
- `Windows release workflow` --calls--> `Version synchronization script`  [EXTRACTED]
  .github/workflows/release-windows.yml → scripts/sync-version.mjs
- `Backend API Client` --calls--> `Local HTTP API Server`  [EXTRACTED]
  app/src/api.ts → backend/internal/api/server.go
- `Synced folder provider` --implements--> `Provider replication`  [EXTRACTED]
  README.md → docs/developers/ARCHITECTURE.md
- `install_ready_candidate()` --calls--> `backend_running()`  [INFERRED]
  app/src-tauri/src/updater.rs → app/src-tauri/src/main.rs

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Local revision protection flow** — backend_internal_syncer_engine_runonce, backend_internal_syncer_engine_capturelocal, backend_internal_syncer_store_reconcilewithstore, backend_internal_syncer_store_replicaterevisionguarded, models_revision [EXTRACTED 1.00]
- **Google Drive protection integration** — googledrive_service, googledrive_httpclient, drive_store, backend_internal_syncer_store_revisionstore [EXTRACTED 1.00]
- **Signed Windows release pipeline** — github_workflows_release_windows_workflow, scripts_sync_version_script, scripts_generate_update_manifest_script, scripts_setup_updater_signing_script [INFERRED 0.85]

## Communities (91 total, 11 thin omitted)

### Community 0 - "Protection Engine"
Cohesion: 0.06
Nodes (86): Protection Engine New, Game, Manager, Store, SyncJob, T, TestConcurrentProviderHeadIsPreservedForReview(), TestConfigurationChangedRefreshesProtectionTargetAndProviderState() (+78 more)

### Community 1 - "Windows Tray and Security"
Cohesion: 0.05
Nodes (30): protect(), unprotect(), AddMenuItem(), AddMenuItemCheckbox(), AddSeparator(), MenuItem, newMenuItem(), Register() (+22 more)

### Community 2 - "Legacy Snapshots"
Cohesion: 0.08
Nodes (49): Bool, Context, Game, Manager, Mutex, Once, Settings, SnapshotSummary (+41 more)

### Community 3 - "Application Lifecycle"
Cohesion: 0.06
Nodes (38): Application, Options, desktopAppExecutable(), CancelFunc, Context, Manager, Once, Service (+30 more)

### Community 4 - "Google Drive Provider"
Cohesion: 0.09
Nodes (30): Client, Context, Duration, File, Manifest, Mutex, ReadCloser, Reader (+22 more)

### Community 5 - "Signed Application Updater"
Cohesion: 0.14
Nodes (42): candidate_is_not_newer(), check_for_update(), clear_candidate(), complete_staged_download(), decode_base64_string(), discard_candidate_files(), discard_downloaded_update(), download_to_staging_file() (+34 more)

### Community 6 - "Settings and Game Models"
Cohesion: 0.09
Nodes (29): BackupPolicy, Manager, Game, SyncPolicy, Settings Manager, decodeSettingsFields(), defaultBackupPolicy(), defaultPath() (+21 more)

### Community 7 - "Google OAuth"
Cohesion: 0.09
Nodes (27): accountEmail(), Client, Context, Manager, Mutex, Request, Response, RoundTripper (+19 more)

### Community 8 - "Activity and Diagnostics"
Cohesion: 0.10
Nodes (13): ActivityEvent, boolInt(), createdTime(), formatTime(), Store, ProviderHealth, RecoveryPoint, Revision (+5 more)

### Community 9 - "Revision and Recovery Models"
Cohesion: 0.09
Nodes (38): Add game with policy, Save Revision, Time, Capture local revision, Create recovery point job, Run protection job, Immutable revision manifest, ActivityEvent (+30 more)

### Community 10 - "React Application Shell"
Cohesion: 0.09
Nodes (28): api, App(), BackupOperationState, navItems, NavSnapshot, View, viewTitle, newDefaultSettings() (+20 more)

### Community 11 - "SQLite Persistence"
Cohesion: 0.11
Nodes (15): configure(), Store, Time, isBusyError(), migrate(), Open SQLite Log Store, T, mustReadEqual() (+7 more)

### Community 12 - "Frontend Tooling"
Cohesion: 0.06
Nodes (33): dependencies, react, react-dom, devDependencies, @tauri-apps/api, @tauri-apps/cli, @types/react, @types/react-dom (+25 more)

### Community 13 - "API Client and Types"
Cohesion: 0.10
Nodes (29): ErrorPayload, ActivityEvent, BackupExportRequest, BackupExportResponse, BackupHistoryEntry, BackupImportRequest, BackupPolicy, CloudProvider (+21 more)

### Community 14 - "Tauri Packaging"
Cohesion: 0.06
Nodes (30): app, security, windows, build, beforeBuildCommand, beforeDevCommand, devUrl, frontendDist (+22 more)

### Community 15 - "API Response Parsers"
Cohesion: 0.31
Nodes (23): asBoolean(), asCloudProvider(), asConflictPolicy(), asNumber(), asProtectionTarget(), asString(), asThemeMode(), isRecord() (+15 more)

### Community 16 - "Legacy Automatic Backups"
Cohesion: 0.15
Nodes (13): Service, computeSaveSignature(), Bool, Context, Manager, Mutex, Once, Store (+5 more)

### Community 17 - "Game API Handlers"
Cohesion: 0.22
Nodes (12): gameResponse, Server, BackupPolicy, Game, GameProtectionStatus, Request, ResponseWriter, SyncPolicy (+4 more)

### Community 18 - "Runtime Protection Control"
Cohesion: 0.15
Nodes (8): Bool, Manager, ProviderHealth, RWMutex, Time, dirtyGame, Engine, StoreFactory

### Community 19 - "Tauri Bootstrap"
Cohesion: 0.20
Nodes (20): apply_platform_process_flags(), backend_http_request(), backend_running(), backend_version(), daemon_bootstrap_error(), DaemonBootstrapState, ensure_backend_running(), is_autostart_enabled() (+12 more)

### Community 20 - "Status Go"
Cohesion: 0.12
Nodes (11): Time, Status, Manager, New(), Bool, Status, Manager, loadTrayIcon() (+3 more)

### Community 21 - "Tsconfig Json"
Cohesion: 0.11
Nodes (18): compilerOptions, allowImportingTsExtensions, jsx, lib, module, moduleResolution, noEmit, resolveJsonModule (+10 more)

### Community 22 - "Handlesnapshotbyid"
Cohesion: 0.20
Nodes (13): T, TestSettingsSaveAcceptsPartialUpdate(), TestSettingsSaveIgnoresLegacyGoogleDriveStatusFields(), Server, Request, ResponseWriter, Server, Game (+5 more)

### Community 23 - "Daemon Shutdown Request"
Cohesion: 0.18
Nodes (17): Daemon Shutdown Request, Single-Instance Daemon Bootstrap, Backup Import Export Handler, Game Management Handler, Google Drive OAuth Handler, Legacy Snapshot Handler, Local HTTP API Server, Loopback API Access Control (+9 more)

### Community 24 - "Adaptive Game Monitor"
Cohesion: 0.12
Nodes (17): Adaptive game monitor, Explicit conflict review, Local content-addressed vault, Durable upload jobs, Google Desktop OAuth with PKCE, Google Drive provider, Immutable revision publication protocol, Loopback API (+9 more)

### Community 25 - "Handleactivity"
Cohesion: 0.40
Nodes (6): Server, Request, ResponseWriter, ResponseWriter, writeErr(), writeJSON()

### Community 26 - "Sync Version Mjs"
Cohesion: 0.23
Nodes (15): argv, checkOnly, exitWithError(), explicitVersionArg, normalizedVersion, normalizeVersion(), packageLock, readJson() (+7 more)

### Community 27 - "Branding Ts"
Cohesion: 0.17
Nodes (8): APP_DISPLAY_NAME, HOME_KICKER_LABEL, UI_ERROR_HEADING, RootErrorBoundary, RootErrorBoundaryState, formatBytes(), HomeView(), HomeViewProps

### Community 28 - "Zip Go"
Cohesion: 0.30
Nodes (11): autoBackupFolderName(), extractArchive(), generatedAutoBackupFilename(), generatedExportFilename(), File, Game, New(), resolveExportDestinationPath() (+3 more)

### Community 29 - "Gameprofile"
Cohesion: 0.21
Nodes (6): gameProfile(), Context, Game, Revision, portablePathHint(), GameProfile

### Community 30 - "Servicecontrolstate"
Cohesion: 0.32
Nodes (3): ServiceControlState, Store, normalizeControl()

### Community 31 - "Setup Updater Signing Mjs"
Cohesion: 0.14
Nodes (13): args, cliPath, force, password, passwordPath, privateKeyPath, publicKey, publicKeyPath (+5 more)

### Community 32 - "Default Json"
Cohesion: 0.15
Nodes (12): description, identifier, permissions, $schema, windows, autostart:allow-disable, autostart:allow-enable, autostart:allow-is-enabled (+4 more)

### Community 33 - "Defaults Ts"
Cohesion: 0.23
Nodes (10): DEFAULT_BACKUP_POLICY, DEFAULT_SYNC_POLICY, newDefaultProtectionStatus(), newDefaultSyncPolicy(), Settings, SyncConflict, SyncJob, SyncPolicy (+2 more)

### Community 34 - "Stage Backend Daemon Mjs"
Cohesion: 0.17
Nodes (11): appPackage, args, backendDir, linkerValues, oauthJSONPath, outputBinary, resourcesDir, result (+3 more)

### Community 35 - "Backend Api Client"
Cohesion: 0.18
Nodes (11): Backend API Client, Backend Response Normalizers, Application Controller, DEFAULT_SETTINGS, Native File Picker, Signed Update Verification, Native Update Manager, Tauri Update Bridge (+3 more)

### Community 36 - "Classifylocalerror"
Cohesion: 0.25
Nodes (8): classifyLocalError(), classifyProviderError(), Duration, providerRetryAfter(), retryDelay(), userProviderMessage(), ProviderState, syncTrigger

### Community 37 - "Generate Update Manifest Mjs"
Cohesion: 0.18
Nodes (10): archive, archivePath, [bundleArg, tagArg, repositoryArg], bundleDir, checkOnly, files, manifest, outputPath (+2 more)

### Community 38 - "Handler"
Cohesion: 0.29
Nodes (6): statusWriter, Handler, isLoopbackOrigin(), localhostOnly(), withLoopbackCORS(), Handler

### Community 39 - "Gameprotectionstatus"
Cohesion: 0.20
Nodes (5): GameProtectionStatus, RecoveryPoint, T, lenMust(), StorageUsage

### Community 41 - "Tsconfig Node Json"
Cohesion: 0.22
Nodes (8): compilerOptions, allowSyntheticDefaultImports, composite, module, moduleResolution, skipLibCheck, include, vite.config.ts

### Community 42 - "Handlegoogledriveconnect"
Cohesion: 0.43
Nodes (4): Server, Request, ResponseWriter, nullableEmail()

### Community 43 - "Nativedialog Ts"
Cohesion: 0.36
Nodes (7): DialogFilter, OpenDialogOptions, OpenDialogResult, openSinglePath(), pickExecutable(), pickFolder(), pickZipFile()

### Community 44 - "Handlestatus"
Cohesion: 0.57
Nodes (3): Server, Request, ResponseWriter

### Community 45 - "Game"
Cohesion: 0.29
Nodes (3): Game, BackupViewProps, GamesViewProps

### Community 46 - "Recoverylibraryentry"
Cohesion: 0.33
Nodes (5): recoveryLibraryEntry, baseName(), RecoveryPoint, Time, tokenizePathHint()

### Community 47 - "Handlebackupexport"
Cohesion: 0.60
Nodes (3): Server, Request, ResponseWriter

### Community 48 - "Network Ts"
Cohesion: 0.33
Nodes (5): BACKEND_BASE_URL, BACKEND_DEBUG_INFO_URL, BACKEND_HOST, BACKEND_PORT, BACKEND_PROTOCOL

### Community 49 - "Run Automatic Backup"
Cohesion: 0.40
Nodes (5): Run automatic backup, Export automatic game backup, Insert backup history, Read latest backup signature, Prune backup history

### Community 50 - "Handlers Lifecycle Test Go"
Cohesion: 0.60
Nodes (4): T, TestPrepareUpdateRejectsUnavailableShutdown(), TestPrepareUpdateRequiresPOST(), TestPrepareUpdateRespondsBeforeShutdownCallback()

### Community 51 - "Handledebuginfo"
Cohesion: 0.50
Nodes (3): Server, Request, ResponseWriter

### Community 52 - "Handlelogsrecent"
Cohesion: 0.50
Nodes (3): Server, Request, ResponseWriter

### Community 53 - "Handleprepareupdate"
Cohesion: 0.50
Nodes (3): Server, Request, ResponseWriter

### Community 54 - "Handlesettings"
Cohesion: 0.50
Nodes (3): Server, Request, ResponseWriter

### Community 55 - "Backend Reachability Check"
Cohesion: 0.50
Nodes (4): Backend Reachability Check, Daemon Bootstrap Error State, Ensure Backend Running, Packaged Backend Binary Lookup

### Community 56 - "Application Updater"
Cohesion: 0.50
Nodes (4): Application updater, Draft-first release publication pipeline, Signed NSIS update package, Updater signing key

### Community 57 - "Persist Google Drive Connection"
Cohesion: 0.50
Nodes (4): Persist Google Drive connection, Protect refresh token with DPAPI, Connect Google Drive, Finish Google Drive connection

### Community 58 - "Windows Release Workflow"
Cohesion: 0.50
Nodes (4): Windows release workflow, Signed update manifest generator, Updater signing setup script, Version synchronization script

### Community 59 - "Nostalgia Dark Mode"
Cohesion: 0.50
Nodes (4): Nostalgia dark mode, Nostalgia design system, Nostalgia light mode, Every panel is a window

### Community 61 - "Reload Watcher For Running Games"
Cohesion: 0.67
Nodes (3): Reload watcher for running games, Notify file event, Reload save file watcher

### Community 62 - "Application Run"
Cohesion: 0.67
Nodes (3): Application Run, Network change watcher, Run Windows tray

### Community 65 - "Recovery Library"
Cohesion: 0.67
Nodes (3): Recovery Library, Recovery Points, Verified exact restore

## Knowledge Gaps
- **195 isolated node(s):** `name`, `private`, `version`, `type`, `dev` (+190 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **11 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Application New` connect `Application Lifecycle` to `Protection Engine`, `Legacy Snapshots`, `Settings and Game Models`, `Google OAuth`, `SQLite Persistence`, `Zip Go`?**
  _High betweenness centrality (0.238) - this node is a cross-community bridge._
- **Why does `Local HTTP API Server` connect `Daemon Shutdown Request` to `Backend Api Client`?**
  _High betweenness centrality (0.175) - this node is a cross-community bridge._
- **Why does `Backend API Client` connect `Backend Api Client` to `Daemon Shutdown Request`?**
  _High betweenness centrality (0.175) - this node is a cross-community bridge._
- **Are the 28 inferred relationships involving `Protection Engine New` (e.g. with `NewFolderStore()` and `TestConcurrentProviderHeadIsPreservedForReview()`) actually correct?**
  _`Protection Engine New` has 28 INFERRED edges - model-reasoned connections that need verification._
- **Are the 34 inferred relationships involving `writeJSON()` (e.g. with `.handleActivity()` and `.handleBackupExport()`) actually correct?**
  _`writeJSON()` has 34 INFERRED edges - model-reasoned connections that need verification._
- **Are the 22 inferred relationships involving `New()` (e.g. with `.handleBackupExport()` and `.handleBackupHistory()`) actually correct?**
  _`New()` has 22 INFERRED edges - model-reasoned connections that need verification._
- **Are the 27 inferred relationships involving `writeErr()` (e.g. with `.handleActivity()` and `.handleBackupExport()`) actually correct?**
  _`writeErr()` has 27 INFERRED edges - model-reasoned connections that need verification._