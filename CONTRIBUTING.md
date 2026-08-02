# Contributing to Rove

Rove is a Windows-focused Go, React, and Tauri application. Start with the [developer guide](docs/DEVELOPER_GUIDE.md) for architecture, persistence, APIs, verification, and current implementation boundaries.

## First local run

Prerequisites are Go 1.23+, Node.js 20+, Rust, and Visual Studio 2022 Build Tools with the C++ workload.

```powershell
# terminal 1
cd backend
$env:GOCACHE="$PWD\.cache\go-build"
go run ./cmd/daemon --no-tray --debug

# terminal 2
cd app
npm install
.\dev-tauri.cmd
```

This is enough for local-only and synced-folder development. Google Drive work requires a Desktop OAuth client JSON referenced from the daemon terminal; follow [Google Drive development](docs/developers/GOOGLE_DRIVE_DEVELOPMENT.md). Do not copy credentials into the repository.

## Before submitting changes

```powershell
cd backend
$env:GOCACHE="$PWD\.cache\go-build"
go test ./...

cd ..\app
npm run test:run
npm run type-check
npm run build

cd src-tauri
cargo check
```

See [Testing](docs/TESTING.md) for the test-path matrix, coverage commands, API contract checklist, and Windows-only release scenarios.

Update the user and developer documentation when behavior, API contracts, settings, recovery semantics, provider configuration, or implementation boundaries change. Describe only behavior that is actually implemented.

Never resolve an upgrade test by deleting the test user's `settings.json`. Add or update a migration/regression test instead. Settings persistence must preserve daemon-owned OAuth state, game registrations, device identity, the last valid backup, and any quarantined malformed source file.
