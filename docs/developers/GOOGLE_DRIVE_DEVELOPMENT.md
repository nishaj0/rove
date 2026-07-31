# Google Drive development

This guide is for contributors and release maintainers. Installed Cloud Saver users do not create projects, download OAuth JSON, or enter API credentials.

## Architecture

Cloud Saver uses a Google Desktop OAuth client, PKCE, a loopback callback, and the `drive.file` scope. The browser grants access, but the daemon completes the authorization-code exchange and persists the refresh token. A browser callback is successful only after the exchange and local persistence finish.

Google requires the matching Desktop client credential during token exchange even when PKCE is enabled. The credential cannot be treated as a server secret after it is distributed in a desktop executable. The project keeps it out of Git history and build logs, while user refresh tokens remain local and are protected with Windows DPAPI.

## Local setup

Obtain a Desktop OAuth client JSON from the project maintainer or create one in a Google Cloud project you control. If the consent screen is still in testing, add your Google account as a test user. Keep the downloaded JSON outside the repository.

Start the backend from the same PowerShell process in which the JSON path is set:

```powershell
cd D:\path\to\cloud-saver\backend
$env:CLOUD_SAVER_GOOGLE_OAUTH_JSON="C:\path\to\downloaded-desktop-oauth.json"
$env:GOCACHE="$PWD\.cache\go-build"
go run ./cmd/daemon --no-tray --debug
```

The JSON is parsed in memory. Cloud Saver does not copy it into settings, SQLite, or the application-data directory.

In a second terminal, install and run the desktop UI:

```powershell
cd D:\path\to\cloud-saver\app
npm install
.\dev-tauri.cmd
```

During Tauri development, keep the manually started daemon running. The desktop bootstrap reuses the process already listening on `127.0.0.1:7331`. If `dev-tauri.cmd` cannot find your Visual Studio installation, open an **x64 Native Tools Command Prompt for VS 2022**, change to `app`, and run `npm run dev`.

## Confirm the active configuration

Before selecting **Connect Google Drive**, run:

```powershell
Invoke-RestMethod http://127.0.0.1:7331/providers/google-drive/status
```

For JSON-based development, the relevant values should be:

```text
connected                 False
oauthClientSource         oauth_json
oauthCredentialSource     oauth_json
```

After a successful connection, `connected` becomes `True`. Debug output should end with `[google-drive:connect_complete]`. The status endpoint reports sources only; it never returns credential values or tokens.

Environment variables are read when the daemon starts. If either source is unexpected, stop the daemon with `Ctrl+C`, set the variables again in that terminal, and restart it.

## Configuration precedence

The daemon resolves OAuth configuration in this order:

1. `CLOUD_SAVER_GOOGLE_CLIENT_ID` and `CLOUD_SAVER_GOOGLE_CLIENT_SECRET` environment overrides.
2. Missing values from the file referenced by `CLOUD_SAVER_GOOGLE_OAUTH_JSON`.
3. The built-in official client ID and release-injected credential.

Prefer the JSON variable for local work because it keeps the matching ID and credential together without copying either into source. The individual variables are intended for CI and unusual development environments.

## Local release build

The Tauri release command stages a freshly built daemon and injects OAuth configuration using Go linker variables:

```powershell
cd D:\path\to\cloud-saver\app
$env:CLOUD_SAVER_GOOGLE_OAUTH_JSON="C:\path\to\downloaded-desktop-oauth.json"
npm run tauri:build
```

The staging script prints only the source name, such as `oauth_json`; it must never print the ID, credential, authorization code, verifier, or tokens.

## GitHub release configuration

Official releases use repository-level Actions configuration:

- `GOOGLE_OAUTH_CLIENT_SECRET`: required GitHub Actions secret containing the Desktop client credential.
- `GOOGLE_OAUTH_CLIENT_ID`: optional GitHub Actions variable overriding the built-in official client ID. Fork maintainers should set it to the ID paired with their secret.

The release workflow fails if the required credential is absent. Contributors should never commit a downloaded JSON file or put the credential in workflow YAML, source, issue text, test fixtures, or documentation.

## Sanitized diagnostic stages

Run with `--debug` or set `CLOUD_SAVER_DEBUG=1`. Expected stages include:

```text
[google-drive:connect_start]
[google-drive:callback_ready]
[google-drive:browser_opened]
[google-drive:callback_received]
[google-drive:token_exchange_start]
[google-drive:token_exchange_complete]
[google-drive:token_protected]
[google-drive:connect_complete]
```

Failures include a stable stage such as `token_exchange_failed` and a user-safe error code. Authorization codes, PKCE verifiers, tokens, client identifiers, credentials, and complete personal paths are redacted.

## Troubleshooting

### `client_secret is missing`

The running daemon has a client ID but no matching Desktop credential. Set `CLOUD_SAVER_GOOGLE_OAUTH_JSON`, stop the old daemon, and restart it from the same terminal. Confirm that `oauthCredentialSource` is `oauth_json` before retrying.

### Browser succeeds but the app reports failure

Use the newest daemon. The callback page now waits for token exchange and persistence, so it must not show success when the daemon logs `token_exchange_failed` or `settings_persist_failed`.

### Save Settings reports `unknown field "oauthClientSource"`

This was caused by an older UI payload mixing read-only OAuth diagnostics into persisted settings. Current clients send an explicit editable-only payload, and the daemon accepts and ignores the known legacy Google Drive status envelope. Restart both the desktop app and daemon so they use the current build; do not delete `settings.json`.

### `redirect_uri_mismatch`

Confirm that the downloaded JSON is a Desktop/installed application client rather than a Web application client. Cloud Saver uses a loopback callback selected by the daemon.

### Access blocked or the account is not allowed

Check the OAuth consent-screen publishing/testing state and its test-user list in the Google Cloud project that owns the client.

### Status still says `built_in` or `missing`

The environment variable was not inherited by the running process, the JSON could not be read, or an older daemon is still bound to port `7331`. Stop existing Cloud Saver daemon processes, start one daemon from the configured terminal, and check status again.

## Settings recovery during development

Do not delete `%APPDATA%\CloudSaver\settings.json` to resolve an upgrade problem. The loader migrates missing fields, ignores unknown fields, repairs individually incompatible fields, and preserves unrelated games and encrypted credentials. Every replacement keeps the previous valid document as `settings.json.bak`.

If JSON is malformed because of an interrupted or external write, Cloud Saver preserves the original as `settings.json.invalid-<timestamp>`, restores the backup when possible, and records **Settings repaired after update** in Activity. A schema produced by a newer unsupported application version is not overwritten by an older daemon.

## Security checklist

- Keep OAuth JSON outside the repository.
- Never send OAuth JSON or refresh tokens in bug reports.
- Use sanitized `--debug` logs and **Copy diagnostics** when reporting a problem.
- Never log request bodies from OAuth callback or token endpoints.
- Never add fallback credentials to tests.
- Rotate the Google credential if it is accidentally committed or published in logs.
