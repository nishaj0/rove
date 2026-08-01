# Graph Report - .  (2026-07-31)

## Corpus Check
- 129 files · ~75,259 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1586 nodes · 3037 edges · 141 communities (102 shown, 39 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 288 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- Core Backend Application
- Sync Engine Operations
- API Layer
- Google Drive Integration
- Desktop Updater
- Frontend Dependencies
- Google Drive Auth
- Frontend App State
- Logging & Protection
- Frontend API Client
- Configuration & Models
- Tauri Configuration
- Settings Manager
- SQLite Storage
- API Parsers
- Backend Initialization
- Architecture Documentation
- Auto Backup Service
- Game Monitor
- Design System
- Game API Handlers
- Tauri Entry Point
- User FAQ
- React Dependencies
- TypeScript Config
- System Tray
- API Status Handlers
- Daemon Handlers
- Google Drive Dev Docs
- API Diagnostics
- Windows Credential Protection
- Game Configuration
- System Tray Windows
- Version Sync Scripts
- Branding Constants
- Agent Configuration
- Nostalgia Design Docs
- Updater Signing Scripts
- Frontend Controllers
- Tauri Capabilities
- User Guide
- Backend Staging Scripts
- Tray Status
- System Tray API
- Developer Guide
- Update Manifest Scripts
- Tray Windows Implementation
- System Tray Docs
- System Tray Menu Items
- TypeScript Node Config
- File Watcher
- Releases & Updates Docs
- Project README
- Data Normalizers
- Google Drive API Handlers
- Frontend Types
- Recovery Docs
- Agent Instructions
- Sync API Handlers
- Windows Lifecycle Docs
- Getting Started Docs
- Copilot Instructions
- Protection API
- Backup Export/Import API
- Snapshot API
- Network Constants
- Backend README
- Provider Interface Docs
- Testing Docs
- Backup Operations
- Lifecycle API Tests
- Debug API
- Logs API
- Lifecycle API
- Settings API
- Backend Bootstrap
- Release Pipeline
- Documentation Index
- Release Workflow
- Nostalgia Theme
- OpenCode Config
- Snapshot Service
- Community 82
- Community 83
- Community 84
- Community 85
- Community 86
- Community 87
- Community 88
- Community 90
- Community 92
- Community 97
- Community 98
- Community 100
- Community 101
- Community 111
- Community 113
- Community 114
- Community 115
- Community 116
- Community 117
- Community 118
- Community 119
- Community 120
- Community 121
- Community 122
- Community 123
- Community 124
- Community 125
- Community 126
- Community 127
- Community 128
- Community 130
- Community 131
- Community 132
- Community 133
- Community 134
- Community 135
- Community 136
- Community 137
- Community 138
- Community 139
- Community 140

## God Nodes (most connected - your core abstractions)
1. `writeJSON()` - 40 edges
2. `Protection Engine New` - 39 edges
3. `New()` - 32 edges
4. `writeErr()` - 31 edges
5. `Manager` - 28 edges
6. `Run protection job` - 27 edges
7. `Store` - 26 edges
8. `Provider-neutral reconciliation` - 26 edges
9. `Context` - 22 edges
10. `decodeJSON()` - 21 edges

## Surprising Connections (you probably didn't know these)
- `Claude Hooks Configuration` --references--> `Graphify Knowledge Graph Pattern`  [EXTRACTED]
  .claude/settings.json → AGENTS.md
- `Codex Hooks Configuration` --references--> `Graphify Knowledge Graph Pattern`  [EXTRACTED]
  .codex/hooks.json → AGENTS.md
- `Frontend API Client Tests` --references--> `Testing Documentation`  [EXTRACTED]
  app/src/api.test.ts → docs/TESTING.md
- `API Response Parsers Tests` --references--> `Testing Documentation`  [EXTRACTED]
  app/src/api/parsers.test.ts → docs/TESTING.md
- `Update Helpers Tests` --references--> `Testing Documentation`  [EXTRACTED]
  app/src/update.test.ts → docs/TESTING.md

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Graphify Multi-Agent Integration** — agents_md, claude_md, agents_rules_graphify_rules, agents_workflows_graphify_workflow, claude_settings, codex_hooks, opencode_opencode_json, opencode_plugins_graphify_plugin_js [EXTRACTED 0.95]
- **API Response Normalizer Pipeline** — normalizegames, normalizesettings, normalizebackuphistory, normalizesnapshots, normalizelogs, normalizesyncjobs, normalizesyncconflicts, normalizestatus [INFERRED 0.90]
- **Snapshot Lifecycle (Create/List/Restore/Delete)** — snapshot_createmanual, snapshot_list, snapshot_restore, snapshot_delete [EXTRACTED 0.95]

## Communities (141 total, 39 thin omitted)

### Community 0 - "Core Backend Application"
Cohesion: 0.05
Nodes (99): Application New, Open SQLite Log Store, GameProfile, Manager, Protection Engine New, Engine, Game, Manager (+91 more)

### Community 1 - "Sync Engine Operations"
Cohesion: 0.06
Nodes (40): Reload watcher for running games, Save Revision, Capture local revision, classifyLocalError(), classifyProviderError(), Create recovery point job, gameProfile(), Bool (+32 more)

### Community 2 - "API Layer"
Cohesion: 0.06
Nodes (49): Handler, isLoopbackOrigin(), localhostOnly(), API Middleware Tests, T, TestIsLoopbackOrigin(), TestLocalhostOnlyRejectsNonLoopbackClients(), TestLoopbackCORSSetsHeadersAndRejectsForeignOrigins() (+41 more)

### Community 3 - "Google Drive Integration"
Cohesion: 0.09
Nodes (30): Client, Context, Duration, File, GameProfile, Manifest, Mutex, ReadCloser (+22 more)

### Community 4 - "Desktop Updater"
Cohesion: 0.14
Nodes (43): candidate_is_not_newer(), check_for_update(), clear_candidate(), complete_staged_download(), decode_base64_string(), discard_candidate_files(), discard_downloaded_update(), download_to_staging_file() (+35 more)

### Community 5 - "Frontend Dependencies"
Cohesion: 0.04
Nodes (45): devDependencies, jsdom, @tauri-apps/api, @tauri-apps/cli, @testing-library/jest-dom, @testing-library/react, @testing-library/user-event, @types/react (+37 more)

### Community 6 - "Google Drive Auth"
Cohesion: 0.09
Nodes (27): accountEmail(), Client, Context, Manager, Mutex, Request, Response, RoundTripper (+19 more)

### Community 7 - "Frontend App State"
Cohesion: 0.08
Nodes (32): App(), BackupOperationState, navItems, NavSnapshot, View, viewTitle, BACKUP_HISTORY_REFRESH_INTERVAL_MS, SNAPSHOT_REFRESH_INTERVAL_MS (+24 more)

### Community 8 - "Logging & Protection"
Cohesion: 0.10
Nodes (13): ActivityEvent, boolInt(), createdTime(), formatTime(), Store, ProviderHealth, RecoveryPoint, Revision (+5 more)

### Community 9 - "Frontend API Client"
Cohesion: 0.09
Nodes (34): api, ErrorPayload, Frontend API Client Tests, DEFAULT_BACKUP_POLICY, DEFAULT_SYNC_POLICY, BackupExportRequest, BackupExportResponse, BackupHistoryEntry (+26 more)

### Community 10 - "Configuration & Models"
Cohesion: 0.11
Nodes (34): Add game with policy, ActivityEvent, BackupExportRequest, BackupHistoryEntry, BackupImportRequest, BackupPolicy, CloudProvider, ConflictPolicy (+26 more)

### Community 11 - "Tauri Configuration"
Cohesion: 0.06
Nodes (30): app, security, windows, build, beforeBuildCommand, beforeDevCommand, devUrl, frontendDist (+22 more)

### Community 12 - "Settings Manager"
Cohesion: 0.14
Nodes (20): decodeSettingsFields(), DefaultSettings(), defaultSyncPolicy(), Manager, RWMutex, Settings, SyncPolicy, normalizeSettings() (+12 more)

### Community 13 - "SQLite Storage"
Cohesion: 0.11
Nodes (11): configure(), Store, Time, isBusyError(), migrate(), BackupHistoryEntry, DB, GameStats (+3 more)

### Community 14 - "API Parsers"
Cohesion: 0.25
Nodes (28): asBoolean(), asCloudProvider(), asConflictPolicy(), asNumber(), asProtectionTarget(), asString(), asThemeMode(), isRecord() (+20 more)

### Community 15 - "Backend Initialization"
Cohesion: 0.11
Nodes (18): Application, desktopAppExecutable(), CancelFunc, Context, Engine, Manager, Monitor, Once (+10 more)

### Community 16 - "Architecture Documentation"
Cohesion: 0.09
Nodes (23): Adaptive game monitor, Explicit conflict review, Local content-addressed vault, Architecture and protection workflow, Capture, Legacy boundary, Offline queue, Provider replication (+15 more)

### Community 17 - "Auto Backup Service"
Cohesion: 0.16
Nodes (12): computeSaveSignature(), Bool, Context, Manager, Mutex, Once, Store, Time (+4 more)

### Community 18 - "Game Monitor"
Cohesion: 0.14
Nodes (13): Bool, Context, Duration, Manager, RWMutex, Store, Time, New() (+5 more)

### Community 19 - "Design System"
Cohesion: 0.09
Nodes (22): Accessibility, Brand, Buttons, Charts, Colors, Component-Level Rules, Dark mode (opt-in), Expected Behavior (+14 more)

### Community 20 - "Game API Handlers"
Cohesion: 0.22
Nodes (12): gameResponse, Server, BackupPolicy, Game, GameProtectionStatus, Request, ResponseWriter, SyncPolicy (+4 more)

### Community 21 - "Tauri Entry Point"
Cohesion: 0.22
Nodes (20): apply_platform_process_flags(), backend_http_request(), backend_running(), backend_version(), daemon_bootstrap_error(), DaemonBootstrapState, ensure_backend_running(), is_autostart_enabled() (+12 more)

### Community 22 - "User FAQ"
Cohesion: 0.10
Nodes (20): Authentication required, Conflict shown, Do I need a Google API key?, Do I uninstall Cloud Saver before updating?, Does Cloud Saver require a server or VPS?, Expected resource use, FAQ and troubleshooting, Google Drive says rate limited or unavailable (+12 more)

### Community 23 - "React Dependencies"
Cohesion: 0.12
Nodes (11): dependencies, react, react-dom, RootErrorBoundary, Game, BackupViewProps, GamesViewProps, game() (+3 more)

### Community 24 - "TypeScript Config"
Cohesion: 0.10
Nodes (19): compilerOptions, allowImportingTsExtensions, jsx, lib, module, moduleResolution, noEmit, resolveJsonModule (+11 more)

### Community 25 - "System Tray"
Cohesion: 0.19
Nodes (4): AddMenuItem(), AddMenuItemCheckbox(), MenuItem, newMenuItem()

### Community 26 - "API Status Handlers"
Cohesion: 0.19
Nodes (14): statusWriter, T, TestSettingsSaveAcceptsPartialUpdate(), TestSettingsSaveIgnoresLegacyGoogleDriveStatusFields(), Server, Engine, Game, Handler (+6 more)

### Community 27 - "Daemon Handlers"
Cohesion: 0.15
Nodes (19): Single-Instance Daemon Bootstrap, Backup Import Export Handler, Game Management Handler, Google Drive OAuth Handler, Legacy Snapshot Handler, Local HTTP API Server, Loopback API Access Control, Protection and Recovery Handler (+11 more)

### Community 28 - "Google Drive Dev Docs"
Cohesion: 0.11
Nodes (17): Access blocked or the account is not allowed, Architecture, Browser succeeds but the app reports failure, `client_secret is missing`, Configuration precedence, Confirm the active configuration, GitHub release configuration, Google Drive development (+9 more)

### Community 29 - "API Diagnostics"
Cohesion: 0.40
Nodes (6): Server, Request, ResponseWriter, ResponseWriter, writeErr(), writeJSON()

### Community 30 - "Windows Credential Protection"
Cohesion: 0.17
Nodes (9): protect(), unprotect(), systrayMenuItemSelected(), systrayTrayDoubleClicked(), GUID, Handle, menuItemInfo, notifyIconData (+1 more)

### Community 31 - "Game Configuration"
Cohesion: 0.28
Nodes (7): BackupPolicy, Manager, Game, SyncPolicy, Settings Manager New, validateBackupPolicy(), validateSyncPolicy()

### Community 32 - "System Tray Windows"
Cohesion: 0.20
Nodes (5): AddSeparator(), addSeparator(), RWMutex, SetTooltip(), winTray

### Community 33 - "Version Sync Scripts"
Cohesion: 0.23
Nodes (15): argv, checkOnly, exitWithError(), explicitVersionArg, normalizedVersion, normalizeVersion(), packageLock, readJson() (+7 more)

### Community 34 - "Branding Constants"
Cohesion: 0.19
Nodes (11): APP_DISPLAY_NAME, HOME_KICKER_LABEL, UI_ERROR_HEADING, RootErrorBoundaryState, RecoveryLibraryEntry, Status, GameDetailView(), formatBytes() (+3 more)

### Community 35 - "Agent Configuration"
Cohesion: 0.16
Nodes (14): AGENTS.md (Codex Graphify Integration), Graphify Agent Rules, Graphify Workflow, CLAUDE.md (Claude Graphify Integration), Claude Hooks Configuration, hooks, PreToolUse, Codex Hooks Configuration (+6 more)

### Community 36 - "Nostalgia Design Docs"
Cohesion: 0.14
Nodes (13): Buttons, Chart window, Color Tokens, Component Specs, Do/Don't Quick Reference, Nostalgia Design Tokens (Reference), Opacity variants (Body, per mode), Shape / Border / Shadow Tokens (+5 more)

### Community 37 - "Updater Signing Scripts"
Cohesion: 0.14
Nodes (13): args, cliPath, force, password, passwordPath, privateKeyPath, publicKey, publicKeyPath (+5 more)

### Community 38 - "Frontend Controllers"
Cohesion: 0.15
Nodes (13): Backend API Client, Backend Response Normalizers, Application Controller, DEFAULT_SETTINGS, Native File Picker, Daemon Shutdown Request, Signed Update Verification, Native Update Manager (+5 more)

### Community 39 - "Tauri Capabilities"
Cohesion: 0.15
Nodes (12): description, identifier, permissions, $schema, windows, autostart:allow-disable, autostart:allow-enable, autostart:allow-is-enabled (+4 more)

### Community 40 - "User Guide"
Cohesion: 0.15
Nodes (12): Activity and diagnostics, Application updates, Cloud Saver user guide, Conflicts, First setup, Launching, Offline behavior, Pause controls (+4 more)

### Community 41 - "Backend Staging Scripts"
Cohesion: 0.17
Nodes (11): appPackage, args, backendDir, linkerValues, oauthJSONPath, outputBinary, resourcesDir, result (+3 more)

### Community 42 - "Tray Status"
Cohesion: 0.22
Nodes (6): Time, Actions, Status, Manager, New(), Actions

### Community 43 - "System Tray API"
Cohesion: 0.22
Nodes (7): Register(), Run(), iconBytesToFilePath(), nativeLoop(), registerSystray(), SetIcon(), point

### Community 44 - "Developer Guide"
Cohesion: 0.18
Nodes (10): APIs, Cloud Saver developer guide, Current implementation boundaries, Google Drive OAuth diagnostics, Invariants, Job stages, Performance, Persistence (+2 more)

### Community 45 - "Update Manifest Scripts"
Cohesion: 0.18
Nodes (10): archive, archivePath, [bundleArg, tagArg, repositoryArg], bundleDir, checkOnly, files, manifest, outputPath (+2 more)

### Community 46 - "Tray Windows Implementation"
Cohesion: 0.27
Nodes (6): Actions, Bool, Status, Manager, loadTrayIcon(), New()

### Community 47 - "System Tray Docs"
Cohesion: 0.20
Nodes (9): API, Credits, Features, Linux, macOS, Platform notes, The Webview example, Try the example app! (+1 more)

### Community 48 - "System Tray Menu Items"
Cohesion: 0.33
Nodes (5): addOrUpdateMenuItem(), MenuItem, hideMenuItem(), SetTemplateIcon(), showMenuItem()

### Community 49 - "TypeScript Node Config"
Cohesion: 0.22
Nodes (8): compilerOptions, allowSyntheticDefaultImports, composite, module, moduleResolution, skipLibCheck, include, vite.config.ts

### Community 50 - "File Watcher"
Cohesion: 0.31
Nodes (6): addRecursive(), CancelFunc, Game, Mutex, Manager, New()

### Community 51 - "Releases & Updates Docs"
Cohesion: 0.22
Nodes (8): Application updates and release pipeline, Development behavior, Maintainer checklist, Release notes and documentation maintenance, Rollback, Runtime workflow, Signing-key ceremony, Stable release publication

### Community 52 - "Project README"
Cohesion: 0.22
Nodes (8): Local-first protection, Cloud Saver, Current protection model, Development, Documentation, Protection targets, Repository, Security

### Community 53 - "Data Normalizers"
Cohesion: 0.22
Nodes (9): normalizeBackupHistory, normalizeGames, normalizeLogs, normalizeSettings, normalizeSnapshots, normalizeStatus, normalizeSyncConflicts, normalizeSyncJobs (+1 more)

### Community 54 - "Google Drive API Handlers"
Cohesion: 0.43
Nodes (4): Server, Request, ResponseWriter, nullableEmail()

### Community 55 - "Frontend Types"
Cohesion: 0.29
Nodes (7): PauseScope, ProtectionTarget, Settings, ThemeMode, formatUpdateBytes(), SettingsView(), SettingsViewProps

### Community 56 - "Recovery Docs"
Cohesion: 0.25
Nodes (7): After reinstalling Windows or the game, Conflicts, Corruption and missing objects, Local-only limitation, Recovery and safe restore, Restoring tracked games, ZIP replacement

### Community 57 - "Agent Instructions"
Cohesion: 0.29
Nodes (6): Always-On Guidance, Codex Note, graphify, Graphify for Codex, Keeping the Graph Fresh, Precise Graph Queries

### Community 58 - "Sync API Handlers"
Cohesion: 0.57
Nodes (3): Server, Request, ResponseWriter

### Community 59 - "Windows Lifecycle Docs"
Cohesion: 0.29
Nodes (6): Connectivity, Game monitoring, Shutdown, Startup, Tray, Windows startup, daemon, and tray

### Community 60 - "Getting Started Docs"
Cohesion: 0.29
Nodes (6): Choose a destination, Choose the save folder, Getting started, Low-impact choice, Recommended first run, Second computer

### Community 61 - "Copilot Instructions"
Cohesion: 0.29
Nodes (6): coding style, comments, frontend style, project context, repository instructions, verification

### Community 62 - "Protection API"
Cohesion: 0.33
Nodes (5): recoveryLibraryEntry, baseName(), RecoveryPoint, Time, tokenizePathHint()

### Community 63 - "Backup Export/Import API"
Cohesion: 0.60
Nodes (3): Server, Request, ResponseWriter

### Community 64 - "Snapshot API"
Cohesion: 0.60
Nodes (3): Server, Request, ResponseWriter

### Community 65 - "Network Constants"
Cohesion: 0.33
Nodes (5): BACKEND_BASE_URL, BACKEND_DEBUG_INFO_URL, BACKEND_HOST, BACKEND_PORT, BACKEND_PROTOCOL

### Community 66 - "Backend README"
Cohesion: 0.33
Nodes (5): Cloud Saver backend, Important API groups, Protection model, Run locally, Tests

### Community 67 - "Provider Interface Docs"
Cohesion: 0.33
Nodes (5): Adding a provider, Folder layout, Google Drive, Provider interface, Publication contract

### Community 68 - "Testing Docs"
Cohesion: 0.33
Nodes (5): Cloud Saver testing, Contract checklist for new endpoints, Fast local paths, Higher-level verification still expected before release, Test paths

### Community 69 - "Backup Operations"
Cohesion: 0.40
Nodes (5): Run automatic backup, Export automatic game backup, Insert backup history, Read latest backup signature, Prune backup history

### Community 70 - "Lifecycle API Tests"
Cohesion: 0.60
Nodes (4): T, TestPrepareUpdateRejectsUnavailableShutdown(), TestPrepareUpdateRequiresPOST(), TestPrepareUpdateRespondsBeforeShutdownCallback()

### Community 71 - "Debug API"
Cohesion: 0.50
Nodes (3): Server, Request, ResponseWriter

### Community 72 - "Logs API"
Cohesion: 0.50
Nodes (3): Server, Request, ResponseWriter

### Community 73 - "Lifecycle API"
Cohesion: 0.50
Nodes (3): Server, Request, ResponseWriter

### Community 74 - "Settings API"
Cohesion: 0.50
Nodes (3): Server, Request, ResponseWriter

### Community 75 - "Backend Bootstrap"
Cohesion: 0.50
Nodes (4): Backend Reachability Check, Daemon Bootstrap Error State, Ensure Backend Running, Packaged Backend Binary Lookup

### Community 76 - "Release Pipeline"
Cohesion: 0.50
Nodes (4): Application updater, Draft-first release publication pipeline, Signed NSIS update package, Updater signing key

### Community 77 - "Documentation Index"
Cohesion: 0.50
Nodes (3): Cloud Saver documentation, For contributors, For players

### Community 78 - "Release Workflow"
Cohesion: 0.50
Nodes (4): Windows release workflow, Signed update manifest generator, Updater signing setup script, Version synchronization script

### Community 79 - "Nostalgia Theme"
Cohesion: 0.50
Nodes (4): Nostalgia dark mode, Nostalgia design system, Nostalgia light mode, Every panel is a window

### Community 80 - "OpenCode Config"
Cohesion: 0.50
Nodes (3): plugin, $schema, .opencode/plugins/graphify.js

### Community 81 - "Snapshot Service"
Cohesion: 0.50
Nodes (4): SnapshotService.CreateManual, SnapshotService.Delete, SnapshotService.List, SnapshotService.Restore

### Community 83 - "Community 83"
Cohesion: 0.67
Nodes (3): Application Run, Network change watcher, Run Windows tray

### Community 86 - "Community 86"
Cohesion: 0.67
Nodes (3): Recovery Library, Recovery Points, Verified exact restore

## Knowledge Gaps
- **475 isolated node(s):** `PreToolUse`, `PreToolUse`, `name`, `private`, `version` (+470 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **39 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Testing Documentation` connect `API Layer` to `Frontend API Client`, `API Parsers`, `Frontend App State`?**
  _High betweenness centrality (0.131) - this node is a cross-community bridge._
- **Why does `TestExportAndReplaceImportRoundTrip()` connect `API Layer` to `Google Drive Integration`?**
  _High betweenness centrality (0.085) - this node is a cross-community bridge._
- **Are the 34 inferred relationships involving `writeJSON()` (e.g. with `.handleActivity()` and `.handleBackupExport()`) actually correct?**
  _`writeJSON()` has 34 INFERRED edges - model-reasoned connections that need verification._
- **Are the 28 inferred relationships involving `Protection Engine New` (e.g. with `TestConcurrentProviderHeadIsPreservedForReview()` and `TestConfigurationChangedRefreshesProtectionTargetAndProviderState()`) actually correct?**
  _`Protection Engine New` has 28 INFERRED edges - model-reasoned connections that need verification._
- **Are the 22 inferred relationships involving `New()` (e.g. with `.handleBackupExport()` and `.handleBackupHistory()`) actually correct?**
  _`New()` has 22 INFERRED edges - model-reasoned connections that need verification._
- **Are the 27 inferred relationships involving `writeErr()` (e.g. with `.handleActivity()` and `.handleBackupExport()`) actually correct?**
  _`writeErr()` has 27 INFERRED edges - model-reasoned connections that need verification._
- **What connects `PreToolUse`, `PreToolUse`, `name` to the rest of the system?**
  _475 weakly-connected nodes found - possible documentation gaps or missing edges._