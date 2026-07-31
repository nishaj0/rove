# Cloud Saver backend

The Go daemon owns configuration, SQLite state, incremental save capture, local revisions, provider replication, recovery, game monitoring, the Windows tray, and the loopback API used by the Tauri desktop app.

The complete contributor documentation is in the [developer guide](../docs/DEVELOPER_GUIDE.md). Google Drive contributors should also read [Google Drive development](../docs/developers/GOOGLE_DRIVE_DEVELOPMENT.md).

## Protection model

- A revision is committed to the local content-addressed vault before provider replication starts.
- Local only, Google Drive, and a user-selected synced folder use the same revision model.
- Watcher events mark paths dirty; they do not immediately hash the complete save tree.
- Offline and interrupted uploads are durable jobs stored in SQLite.
- Immutable manifests reference SHA-256 objects and tombstones. Heads are published last.
- Conflicts are review-only and preserve both versions until the user chooses local, remote, or both.
- Legacy ZIP backups and snapshot vaults remain recoverable, but their automatic schedulers are retired from the primary workflow.

## Run locally

Local-only and synced-folder development require no provider configuration:

```powershell
cd backend
$env:GOCACHE="$PWD\.cache\go-build"
go run ./cmd/daemon --no-tray --debug
```

To test Google Drive, set the downloaded Desktop OAuth JSON path in the same PowerShell process before starting the daemon:

```powershell
cd backend
$env:CLOUD_SAVER_GOOGLE_OAUTH_JSON="C:\path\to\downloaded-desktop-oauth.json"
$env:GOCACHE="$PWD\.cache\go-build"
go run ./cmd/daemon --no-tray --debug
```

Keep that JSON outside the repository. It is read in memory and is not copied into settings or SQLite. Environment variables are loaded only at daemon startup, so restart the daemon after changing them.

Confirm the active OAuth sources with:

```powershell
Invoke-RestMethod http://127.0.0.1:7331/providers/google-drive/status
```

JSON-based development should report `oauthClientSource` and `oauthCredentialSource` as `oauth_json`.

## Important API groups

- `/games`: game configuration and launch lifecycle.
- `/protection`: health, global/per-game controls, and retries.
- `/sync/jobs`: durable capture and replication jobs.
- `/sync/conflicts`: conflict review and explicit resolution.
- `/revisions`: revision history and verified restore.
- `/recovery-points`: pinned revision management.
- `/recovery-library`: fresh-install discovery and guided restore.
- `/providers/google-drive`: OAuth connection and provider status.
- `/activity` and `/diagnostics`: user history and sanitized developer diagnostics.
- `/settings`: protection target, policy, retention, startup, and service controls.

The API binds only to `127.0.0.1:7331`.

## Tests

```powershell
cd backend
$env:GOCACHE="$PWD\.cache\go-build"
go test ./...
```

The current tests cover the sync protocol, two-device update/delete/conflict behavior, local-first offline revisions, Recovery Points, exact restore, staged Recovery Library restore, OAuth PKCE parameters, OAuth JSON loading, and callback failure reporting. Remaining release-qualification gaps are listed in the [developer guide](../docs/DEVELOPER_GUIDE.md#current-implementation-boundaries).
