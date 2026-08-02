# Rove

Rove is a Windows desktop utility for games that do not have dependable built-in cloud saves. It syncs ordinary save folders locally first, then optionally replicates immutable revisions directly to Google Drive or a user-selected folder managed by OneDrive, Dropbox, Syncthing, or another sync client.

**Game cloud saves for everyone.**

There is no Rove server or VPS. End users never enter developer API keys.

Rove uses its own warm, editorial brand system across the desktop UI, application icon, and documentation. See the [brand implementation guide](docs/BRAND.md) for the approved assets and rules.

## Current Cloud Sync model

- **Balanced** is the default: changed, stable files are checkpointed every 10 minutes while a game is running, followed by a final sync after exit.
- **Exit only**, **Continuous**, and **Manual** policies are available per game.
- Watcher events mark individual paths dirty. Clean gameplay does not recursively hash save folders.
- Every revision is committed to the local content-addressed vault before cloud upload begins.
- Offline uploads are durable and retry automatically after connectivity returns.
- The latest three automatic revision records are retained. Named Recovery Points stay pinned until the user removes them.
- Conflicting local and remote edits require an explicit choice; Rove does not silently pick a winner.
- Global and per-game controls can pause uploads while keeping local sync active, or pause all sync while leaving the tray process available.
- Rove can use the icon embedded in a selected game executable as its cover. A manual image wins; otherwise Rove uses a stable generated cover.
- Optional game version, cover, portable playtime, and restore hints travel with Rove's `game.json` metadata. The immutable save layout continues to use the stable Sync ID, not a title-derived folder name.
- Home reports Rove-managed local/cloud storage only; it never reports unrelated files in a provider account.

## Sync destinations

1. **Local only** — no account or Internet required. It does not survive loss of the disk containing Rove's vault.
2. **Google Drive** — direct desktop-to-Drive replication using the narrow `drive.file` scope. OAuth tokens are protected with Windows DPAPI.
3. **Synced folder** — provider-neutral storage under `.cloud-saver/v1` in a folder synchronized by another application.

Objects use SHA-256 names, manifests are immutable, and a head is published only after its referenced objects and manifest are complete. Modern publications also write a per-device head while continuing to update the legacy compatibility head.

## Documentation

- [User guide](docs/USER_GUIDE.md)
- [Developer guide](docs/DEVELOPER_GUIDE.md)
- [End-user getting started](docs/end-users/GETTING_STARTED.md)
- [Recovery and restore](docs/end-users/RECOVERY.md)
- [FAQ and troubleshooting](docs/end-users/FAQ.md)
- [Architecture and workflow](docs/developers/ARCHITECTURE.md)
- [Provider interface](docs/developers/PROVIDER_INTERFACE.md)
- [Google Drive development](docs/developers/GOOGLE_DRIVE_DEVELOPMENT.md)
- [Windows startup and tray](docs/developers/WINDOWS_LIFECYCLE.md)
- [Application updates and releases](docs/developers/RELEASES_AND_UPDATES.md)
- [Brand implementation guide](docs/BRAND.md)

## Repository

```text
cloud-saver/
├── backend/  Go daemon, SQLite state, Cloud Sync engine, providers, local API
├── app/      Tauri 2 + React desktop application
└── docs/     end-user and contributor documentation
```

The daemon binds only to `127.0.0.1:7331`.

## Development

Prerequisites: Go 1.23+, Node.js 20+, Rust, and Visual Studio 2022 Build Tools with the C++ workload.

See [CONTRIBUTING.md](CONTRIBUTING.md) for the contributor entry point and [the complete developer guide](docs/DEVELOPER_GUIDE.md) for architecture and implementation boundaries.

Install the frontend dependencies once:

```powershell
cd app
npm install
```

For local-only and synced-folder development, no provider credential is required. To test Google Drive, set `CLOUD_SAVER_GOOGLE_OAUTH_JSON` in the same PowerShell window that starts the daemon. The file must be a Google Desktop OAuth client download and must remain outside this repository.

```powershell
# terminal 1
cd backend
$env:CLOUD_SAVER_GOOGLE_OAUTH_JSON="C:\path\to\downloaded-desktop-oauth.json" # omit when not testing Drive
$env:GOCACHE="$PWD\.cache\go-build"
go run ./cmd/daemon --no-tray --debug

# terminal 2
cd app
.\dev-tauri.cmd
```

Changing an OAuth environment variable does not update a daemon that is already running. Stop it with `Ctrl+C`, set the variable, and start it again. Verify the active configuration with:

```powershell
Invoke-RestMethod http://127.0.0.1:7331/providers/google-drive/status
```

For JSON-based development, both `oauthClientSource` and `oauthCredentialSource` should be `oauth_json`. See the [Google Drive contributor setup](docs/developers/GOOGLE_DRIVE_DEVELOPMENT.md) for OAuth setup, release builds, sanitized logs, and troubleshooting.

Verification:

```powershell
cd backend
go test ./...

cd ..\app
npm run type-check
npm run build

cd src-tauri
cargo check
```

The Tauri release build uses the official autostart plugin and registers `Rove.exe --background`. The background bootstrap ensures the single daemon is running, opens no main window, and exits.

Installed releases check the stable GitHub Releases update manifest at startup and before a game launch when the last successful check is more than six hours old. Checks time out after three seconds and never block play. Accepted updates download as signed NSIS updater archives in the background, survive application restarts, and install in place after tracked games and active Cloud Sync work close. Manual uninstall is neither required nor recommended. Source/development builds can check the UI state but cannot install updates.

## Security

Never commit Google OAuth JSON downloads, client credentials, refresh tokens, save contents, or personal paths. Google requires the Desktop client credential during token exchange even though it cannot be confidential in a distributed desktop binary. Official releases inject it from a protected CI secret. Local developers point `CLOUD_SAVER_GOOGLE_OAUTH_JSON` at their downloaded file; normal users enter nothing. User refresh tokens remain local and Windows-protected.

The updater public key is committed at `app/src-tauri/updater-public-key.txt`. Its private key and password must never enter Git; the local key ceremony stores them under ignored `.secrets/` files, and releases read them from GitHub Actions secrets. End users and ordinary contributors do not supply signing keys.

The OAuth loading and release-injection paths are defined in `backend/internal/auth/googledrive/service.go` and `scripts/stage-backend-daemon.mjs`. For sanitized OAuth-stage logs during local development, start the daemon with `--debug` or set `CLOUD_SAVER_DEBUG=1`.
