# Cloud Saver

Cloud Saver is a Windows desktop app for tracking, backing up, and safely synchronizing game save folders. It is designed for games without dependable built-in cloud saves, including custom installs and non-store games.

## What it does

- Tracks arbitrary game save folders and launches configured game executables.
- Creates local ZIP exports, automatic local backups, and versioned snapshots.
- Synchronizes saves through a user-selected folder already managed by OneDrive, Dropbox, Syncthing, or another file-sync client.
- Detects concurrent edits and requires a deliberate resolution: keep local, use remote, or keep both.
- Keeps activity logs, game playtime, Windows startup, and tray support local to the device.

## Cloud sync setup

Cloud Saver is provider-neutral: it does not request provider credentials or upload directly to an account.

1. Create or choose a folder that your existing sync provider shares between computers.
2. Open **Settings** in Cloud Saver, choose that folder as **Synced folder**, and save.
3. Add a game and run **Sync This Game** while the game is closed.
4. On another computer, add that computer's local save folder for the same game.
5. Copy the first computer's **Cloud game ID** from its game page. On the other computer choose **Link Existing Cloud Game** and paste that ID.

The selected folder contains `.cloud-saver/v1`. It holds internal manifests, objects, and recovery versions; do not edit or remove it manually. ZIP backups and snapshot vaults remain separate from cloud sync.

## Sync safety

- Save contents are identified with SHA-256 hashes; unchanged files are reused.
- Revisions are immutable and are published only after referenced files are complete.
- Deleted files are represented by tombstones and propagate to paired machines.
- A remote replacement archives the displaced local file before it is applied.
- If the same path changed on two devices, sync stops without overwriting either copy.
- Snapshot restore and manual game sync are blocked while the tracked game is running.
- Ignore patterns apply to sync and snapshot scans.

## Project structure

```text
cloud-saver/
├── backend/  # Go daemon, SQLite metadata, sync engine, local HTTP API
└── app/      # Tauri + React desktop UI
```

The daemon listens only on `127.0.0.1:7331`; the desktop app calls that local API.

## Development

Prerequisites: Go 1.23+, Node.js 20+, Rust, and Visual Studio 2022 Build Tools with the C++ workload.

Start the daemon:

```powershell
cd backend
go run ./cmd/daemon --no-tray --debug
```

Start the desktop app in a second terminal:

```powershell
cd app
npm install
.\dev-tauri.cmd
```

Use `dev-tauri.cmd` because it loads the Visual Studio build environment needed by Tauri on Windows.

## Verification

```powershell
cd backend
go test ./...

cd ..\app
npm run type-check
npm run build
```

Useful local endpoints:

- `GET /status`
- `GET /games`
- `POST /sync`
- `GET /sync/jobs?gameId=...`
- `GET /sync/conflicts?gameId=...`
- `POST /sync/conflicts`
- `GET /debug/info`

## Troubleshooting

- **Backend unavailable:** start the daemon, then visit `http://127.0.0.1:7331/debug/info`.
- **Sync folder not configured:** choose a Synced folder in Settings and save.
- **Conflict detected:** open the affected game and choose a resolution; Cloud Saver has not overwritten either version.
- **Second computer created a separate cloud stream:** use **Link Existing Cloud Game** and paste the Cloud game ID from the first computer.
- **`link.exe` or Cargo missing:** start the app through `app\dev-tauri.cmd` and confirm Visual Studio Build Tools are installed.

## Release build

```powershell
cd app
npm run tauri:build
```

The packaging step stages the backend daemon into the Tauri bundle and produces an NSIS Windows installer.
