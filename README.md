# Cloud Saver

Cloud Saver is a Windows desktop app for tracking, backing up, and safely synchronizing game save folders. It is designed for games without dependable built-in cloud saves, including custom installs and non-store games.

For step-by-step help, see the [user guide](docs/USER_GUIDE.md). Contributors should start with the [developer guide](docs/DEVELOPER_GUIDE.md).

## What it does

- Tracks arbitrary game save folders and launches configured game executables.
- Creates local ZIP exports, automatic local backups, and versioned snapshots.
- Synchronizes saves through a user-selected folder already managed by OneDrive, Dropbox, Syncthing, or another file-sync client.
- Detects concurrent edits and requires a deliberate resolution: keep local, use remote, or keep both.
- Keeps activity logs, game playtime, Windows startup, and tray support local to the device.

## Cloud sync setup

Cloud Saver supports a provider-neutral synced folder by default and an optional direct Google Drive connection. Google Drive only accesses the files Cloud Saver creates for save sync.

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

## Performance and resource notes

Cloud Saver runs as a local background daemon. The default reconciliation interval
is 30 seconds, and each reconciliation currently walks and SHA-256 hashes the
files in the save folders being synchronized. This is lightweight for normal save
folders, but very large folders can cause noticeable disk and CPU activity even
when no file changed. Filesystem events can trigger an earlier scan, while the
interval scan remains the recovery fallback.

Cloud content is deduplicated by hash, but immutable revisions and displaced-file
recovery copies are retained. Automatic cloud revision/object garbage collection
is not implemented yet, so the `.cloud-saver/v1` directory can grow over time.
The configured retention value is stored for the retention feature, but should not
currently be treated as an active deletion guarantee. Local snapshots and ZIP
backups are separate storage and have their own policies.

For the detailed algorithm, complexity, background timers, and known performance
limits, see the [developer guide](docs/DEVELOPER_GUIDE.md#performance-and-resource-model).

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
