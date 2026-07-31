# Application updates and release pipeline

Cloud Saver updates directly from stable GitHub Releases. There is no VPS, custom update server, end-user key, or manual uninstall step.

## Runtime workflow

The React client calls native Tauri commands in `app/src-tauri/src/updater.rs`:

1. `update_status` restores the schema-versioned state from the application update cache.
2. `check_for_update` uses the official Tauri updater parser against `https://github.com/njnis/cloud-saver/releases/latest/download/latest.json`.
3. A successful check is cached for six hours. Startup and pre-launch calls reuse it; a check is bounded to three seconds.
4. `download_update` streams the signed NSIS updater payload (`-setup.exe` in the current Tauri output, with archive compatibility) to a versioned `.part` file. A 45-second no-progress watchdog, ten-minute request limit, and three attempts bound failure.
5. The completed archive is minisign-verified, SHA-256 recorded, synced, and atomically renamed. Only then does the state become `ready`.
6. When a download first becomes `ready` in a desktop session, the client opens a keyboard-safe **What's new** dialog. It displays the persisted release notes and offers **Install update** or **Install later**. Escape and clicking the backdrop choose the safe later option. The dialog is not repeatedly forced after a restart; **What's new** remains available from the update banner and Settings.
7. `install_downloaded_update` streams the checksum again, re-verifies the signature from a read-only file mapping, safely extracts the NSIS executable, and asks the daemon to prepare for update.
8. The daemon refuses while a tracked game or active capture is present. Otherwise it stops API intake, waits for workers, closes SQLite, and releases its executable.
9. The desktop waits for the loopback port to close and starts NSIS with passive update/restart flags. The existing installation is replaced in place.

The public UI states are `idle`, `checking`, `available`, `downloading`, `ready`, `installing`, `error`, and `development`. Completed downloads survive restarts. Interrupted checks/downloads/install attempts recover to `available` or `ready`; `.part` files are removed. Only one update operation runs at a time and downgrade candidates are discarded.

Application updates do not modify the game save folders or `%APPDATA%\CloudSaver`, which contains settings, SQLite state, the local vault, and encrypted provider credentials.

## Development behavior

Debug/source builds report `development` and cannot download or install releases. This prevents an installer from overwriting a contributor build or workspace. Frontend browser preview catches the absence of the native bridge and continues normally.

The bundled daemon receives the same package version through Go linker flags in `scripts/stage-backend-daemon.mjs`. `GET /version` lets a release desktop detect a stale daemon. A mismatch gets one graceful restart through `/lifecycle/prepare-update`; Cloud Saver does not terminate an unknown process occupying its port.

## Signing-key ceremony

The committed `app/src-tauri/updater-public-key.txt` verifies every updater archive. The matching private key must remain stable for the lifetime of installs using that public key.

The initial key was generated with:

```powershell
node scripts/setup-updater-signing.mjs
```

The script refuses to overwrite existing material unless `--force` is supplied. It places the private key and generated password under ignored `.secrets/` files and copies only the public key into the source tree. Back up both secret files in an access-controlled password manager or secret vault, then configure these GitHub Actions secrets:

- `TAURI_SIGNING_PRIVATE_KEY`: complete contents of `.secrets/cloud-saver-updater.key`;
- `TAURI_SIGNING_PRIVATE_KEY_PASSWORD`: contents of `.secrets/cloud-saver-updater-password.txt`.

Do not put either value in source, workflow YAML, issues, logs, release assets, or contributor setup. Ordinary contributors and all end users need neither value. Losing the private key means existing installations cannot trust future updates; rotating it requires an installer release that deliberately migrates the public key.

The Google Drive release variables remain separate:

- repository variable `GOOGLE_OAUTH_CLIENT_ID`;
- repository secret `GOOGLE_OAUTH_CLIENT_SECRET`.

## Release notes and documentation maintenance

The updater dialog uses the `notes` field in `latest.json`. During the release workflow, GitHub generates those notes from the tagged changes and the workflow supplies them to `scripts/generate-update-manifest.mjs`. They are persisted alongside the verified download, so the same text can be reviewed after a restart without another network request.

Before creating a stable release, the maintainer should make the generated notes useful to non-technical players:

1. Use clear, user-facing PR titles and labels before merging; GitHub uses this history to generate the release notes.
2. Review the generated draft release notes for the version tag. Call out protection behavior, restore/migration implications, provider changes, and any action a user must take. Do not include credentials, personal paths, tokens, save data, or internal-only diagnostics.
3. Update the relevant user documentation when behavior changes: `README.md`, `docs/USER_GUIDE.md`, and `docs/end-users/FAQ.md` for player-facing changes; `docs/developers/` for architecture, provider, lifecycle, build, or release-process changes.
4. Keep this document accurate when changing the update manifest schema, signing procedure, cache location, release assets, release endpoint, or required CI secrets. Treat those changes as updater protocol changes and test an upgrade from the previous stable release.
5. Add migration/recovery notes before publishing whenever settings, database data, provider credentials, or local revisions need special handling.

The dialog must remain plain-text only: release notes are rendered as text, never HTML. This keeps a remotely served release description from becoming executable application content.

## Stable release publication

`.github/workflows/release-windows.yml` runs for version tags and rejects anything except stable `vX.Y.Z` for the updater channel. It then:

1. synchronizes the package JSON, package lock, Tauri config, Cargo package, frontend fallback, and linked daemon version;
2. runs TypeScript and backend build checks;
3. validates OAuth and updater signing configuration;
4. builds the NSIS bundle with `createUpdaterArtifacts` and Tauri signing variables;
5. generates `latest.json` for `windows-x86_64-nsis` from the actual archive and `.sig`;
6. verifies the manifest checksum/size/URL/signature and tests the signature against the committed public key;
7. generates release checksums;
8. creates a draft GitHub Release and uploads installer, updater archive, signature, manifest, and checksums;
9. confirms every required draft asset exists, then publishes the release last.

Publishing last prevents clients from observing a manifest whose archive or signature is missing. Do not manually replace `latest.json` without regenerating it from the exact signed archive.

### Maintainer checklist

- Confirm the version tag is a stable `vX.Y.Z`; prereleases do not enter the automatic-update channel.
- Verify user-facing release notes and all affected documentation before the tag is published.
- Ensure the repository OAuth variables and Tauri signing secrets are configured; never substitute local secrets into tracked files.
- Let CI build, sign, verify, and upload the complete draft. Inspect the draft assets, `latest.json`, checksum, signature, and release notes before it becomes public.
- Run or confirm the Windows upgrade qualification from the previous stable version with settings, local revisions, pending uploads, provider credentials, and startup registration present.
- After publishing, install the release on a non-development build, download it through the app, review the **What's new** dialog, and verify in-place installation preserves application data.
- If the release process or updater behavior changes, update this guide in the same pull request; documentation is part of the release contract.

## Rollback

Updates refuse downgrades. If a release is defective:

1. mark the defective GitHub Release as non-latest or remove it from the update channel;
2. create a higher patch version containing the fix;
3. run the normal signed draft-first pipeline;
4. publish only after validation succeeds.

Do not reuse a version number or point its manifest at a different archive. Users who already downloaded the defective version retain a signature-bound candidate; publish the higher patch promptly and provide release notes. Save data and provider credentials are independent of application binaries, but release qualification should still include an N-to-N+1 Windows install test with populated settings, SQLite state, provider credentials, autostart registration, and pending jobs.
