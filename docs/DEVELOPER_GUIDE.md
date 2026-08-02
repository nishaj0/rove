# Rove developer guide

This guide covers the implemented Windows-focused, offline-first protection engine. Detailed references:

- [Architecture and workflow](developers/ARCHITECTURE.md)
- [Provider interface](developers/PROVIDER_INTERFACE.md)
- [Google Drive development and release setup](developers/GOOGLE_DRIVE_DEVELOPMENT.md)
- [Windows lifecycle and tray](developers/WINDOWS_LIFECYCLE.md)
- [Application updates and release pipeline](developers/RELEASES_AND_UPDATES.md)

## Repository map

```text
backend/internal/
  api/              loopback HTTP handlers
  app/              composition and lifecycle
  config/           settings/game JSON and migration
  gamemon/          adaptive process detection
  logstore/         SQLite state, jobs, revisions, activity
  networkwatch/     Windows connectivity-change signal
  runtimepriority/  game-time process priority control
  singleinstance/   per-user daemon mutex
  syncer/           protection engine and local/folder protocol
  syncer/drive/     Google Drive RevisionStore
  tray/             event-driven Windows tray
app/
  src/              React UI and typed API client
  src-tauri/        Tauri bootstrap, autostart, and signed persistent updater
```

## Invariants

1. Local publication completes before provider replication.
2. Objects and manifests are immutable; heads are mutable pointers published last.
3. A restore verifies SHA-256 before replacing an existing file.
4. Provider failure cannot discard the local revision or pending job.
5. Conflicting paths require review.
6. One capture worker and one provider path run serially through the engine.
7. Watch callbacks mark dirty metadata only.
8. Legacy data is preserved until an explicit user action.

## Persistence

Settings JSON retains compatibility fields but the primary model is `ProtectionTarget`, `SyncPolicy`, and `ServiceControlState`. Migration supplies Balanced/10-minute defaults, a stable device ID, per-game cloud `syncId`, auto-revision retention of three, and enabled protection/reconnect defaults.

SQLite uses WAL and persists:

- file metadata/hash cache;
- sessions and dirty state;
- durable protection jobs and retry stages;
- immutable revision metadata and Recovery Points;
- provider health and cursor slots;
- global/per-game pause state;
- profiles, conflicts, quarantine metadata, and structured activity.

The old log, backup-history, sync-baseline, conflict, and snapshot data remain readable.

`settings.json` has an explicit schema version. Loading is field-tolerant: unknown fields are retained only by newer clients and ignored by older readers, missing fields receive migration defaults, and a field whose JSON type changed is reset without discarding unrelated games or encrypted provider credentials. Saves use a same-directory temporary file and atomic replacement, with the previous valid JSON retained as `settings.json.bak`. If the primary file is malformed, Rove preserves it as `settings.json.invalid-<timestamp>`, restores the backup when possible, and records a warning in Activity. Contributors must not fix migrations by deleting user settings.

The `POST /settings` contract contains editable fields only. Provider connection details, OAuth diagnostic sources, device identity, games, and schema metadata remain daemon-owned. Known legacy envelope fields are accepted and ignored to tolerate a short UI/daemon version mismatch.

## Job stages

`waiting`, `waiting_for_network`, `stabilizing`, `capturing`, `locally_protected`, `uploading`, `publishing`, `paused`, `authentication_required`, `quota_blocked`, `conflict`, `failed`, and `completed` are represented in the API model. Jobs store revision ID, byte/file counts, progress, retry count, next attempt, and safe error code.

Transient provider retry uses jittered 15s, 30s, 1m, 2m, 5m, then 15m delays. Auth and quota states are non-spinning. A network-change event calls `RetryNow`.

## Performance

For `F` known paths and `Δ` changed bytes:

- watcher callback: path mapping and SQLite dirty mark, approximately `O(path roots)`;
- scheduled capture metadata scan: `O(F)`;
- hashing/staging: `O(Δ)`;
- manifest comparison: `O(F)`;
- memory: `O(F)` metadata plus bounded streaming buffers.

Changed files are hashed and staged in a streaming pass, then metadata is checked again. While a game is running, capture reads are limited to 4 MiB/s and the daemon is moved to Below Normal priority. Process detection polls every 10 seconds while idle and every two seconds while launching/playing; full executable paths are resolved only after a basename match.

The implementation does not yet contain benchmark enforcement for the stated `<1%` daemon CPU or `<1%` FPS acceptance targets. Treat those as release qualification goals, not guaranteed runtime telemetry.

## APIs

Important routes:

| Route | Purpose |
| --- | --- |
| `GET /protection/status?gameId=` | global or per-game health |
| `POST /protection/control` | global/per-game pause and duration |
| `POST /protection/retry` | immediate provider retry |
| `GET /sync/jobs` | durable job view |
| `GET/POST /sync/conflicts` | review and explicit resolution |
| `GET /revisions`, `POST /revisions/restore` | history and exact restore |
| `GET/POST/DELETE /recovery-points` | pinned revisions |
| `GET /recovery-library` | portable provider profiles |
| `POST /recovery-library/restore` | verified restore to empty folder |
| `GET /storage/usage?gameId=` | unique retained content and pending bytes |
| `GET /activity` | structured user events |
| `GET /diagnostics` | sanitized report |
| `GET/POST /settings` | settings and target selection |
| `GET /version` | bundled daemon version compatibility |
| `POST /lifecycle/prepare-update` | refuse unsafe install or begin graceful daemon shutdown |

All endpoints bind to loopback and JSON decoding rejects unknown fields.

## Verification

```powershell
cd backend
$env:GOCACHE="$PWD\.cache\go-build"
go test ./...

cd ..\app
npm run type-check
npm run build

cd src-tauri
cargo test
```

The updater uses the official Tauri release check and signature format, but streams the NSIS updater payload into the application cache instead of retaining it only in memory. A `.part` file is never installable. Completion performs signature verification before atomic rename; installation rechecks SHA-256 and the minisign signature through a read-only mapping. The download has a 45-second no-progress watchdog, a ten-minute request bound, and three attempts. See [Application updates and releases](developers/RELEASES_AND_UPDATES.md) for the state file, key ceremony, CI publication ordering, and rollback rules.

## Google Drive OAuth diagnostics

OAuth configuration loading is in `backend/internal/auth/googledrive/service.go`. Google requires the Desktop client credential at token exchange even when PKCE is used. For local development, reference the downloaded file without copying it into the repository:

```powershell
$env:CLOUD_SAVER_GOOGLE_OAUTH_JSON="C:\path\to\downloaded-desktop-oauth.json"
cd backend
go run ./cmd/daemon --no-tray --debug
```

Set the variable in the same PowerShell process that launches the daemon. When changing it, stop the existing daemon and restart it; configuration is loaded only at startup. Confirm the active sources before connecting:

```powershell
Invoke-RestMethod http://127.0.0.1:7331/providers/google-drive/status
```

JSON-based development should report `oauthClientSource: oauth_json` and `oauthCredentialSource: oauth_json`. The complete contributor walkthrough and error guide is in [Google Drive development](developers/GOOGLE_DRIVE_DEVELOPMENT.md).

The JSON is read in memory and never copied into settings or SQLite. `CLOUD_SAVER_GOOGLE_CLIENT_ID` and `CLOUD_SAVER_GOOGLE_CLIENT_SECRET` are lower-level overrides for CI or unusual developer environments.

Official releases set the GitHub Actions secret `GOOGLE_OAUTH_CLIENT_SECRET`. Fork maintainers also set the Actions variable `GOOGLE_OAUTH_CLIENT_ID` so it matches their secret; the upstream project can retain its built-in official ID. The staging script injects these into the daemon using Go linker variables, and the release workflow refuses to publish without the credential. The value remains extractable from the installed executable—as with every public desktop OAuth client—but is absent from Git history and build logs.

`--debug` or `CLOUD_SAVER_DEBUG=1` writes sanitized stages to the terminal and SQLite log: callback listener, browser launch, verified callback, PKCE exchange, refresh-token presence, optional account metadata, DPAPI protection, and settings persistence. It never logs authorization codes, PKCE verifiers, tokens, client IDs, or client credentials. `GET /providers/google-drive/status` reports both configuration sources without returning either value.

The sync tests cover two-device updates/deletions/conflicts, local-first offline behavior, Recovery Points, exact restore, and verified staged Recovery Library restore. Add crash-injection, metered-network, Drive cursor-expiration, resumable-session, cleanup, API contract, frontend component, and Windows integration coverage before treating those paths as release-complete.

## Current implementation boundaries

- The schema includes provider cursor and cleanup/quarantine records, but Drive change-cursor consumption and physical mark-and-sweep deletion are not active yet.
- Per-device heads are published, while full multi-head graph convergence is still incomplete; the legacy compatibility head remains part of reads.
- The metered-network preference is persisted and shown, but Windows cost-aware transfer blocking still requires native integration.
- The important-notification preference is persisted, but native toast delivery is not wired yet.
- The tray currently exposes global pause controls; timed and per-game pauses are available in the desktop UI.
- Normal quit persists jobs, but the three-choice pending-work tray confirmation is not implemented.

These boundaries are documented so contributors do not mistake a type, table, or setting for completed behavior.
