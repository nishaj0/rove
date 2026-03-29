# Cloud Saver

Cloud Saver is a Windows desktop app to track and back up game save folders.

It is designed for games that do not have reliable cloud sync, including custom installs and non-store games.

## Project Overview

The app runs as a lightweight local service and a desktop UI:

- `backend`: Go daemon for watching save folders, running sync jobs, handling tray behavior, and exposing a local API
- `app`: Tauri + React desktop UI for settings, game management, logs, and backup actions

## Core Goals

- Automatically sync changed game saves
- Let users manually map any game save folder
- Keep CPU/RAM usage low while running in background
- Support safe conflict handling
- Provide simple UI with tray and startup behavior on Windows

## Current MVP Features

- Add and remove tracked game folders
- Sync status dashboard
- Manual sync trigger
- Interval sync scheduling
- Ignore pattern settings
- Conflict policy setting (`latest_wins` or `keep_both`)
- Local activity logs
- ZIP export/import backup tools
- Windows tray support
- Windows startup setting (autostart)

## Architecture

1. Go daemon watches folders and manages sync state
2. Daemon exposes local HTTP API on `127.0.0.1:7331`
3. Tauri desktop UI calls the local API
4. User interacts through desktop UI and tray menu

## Tech Stack

- Backend: Go
- UI: Tauri + React + TypeScript + Vite
- File watcher: fsnotify
- Local metadata/log storage: SQLite

## Project Structure

```text
cloud-saver/
	backend/   # Go daemon
	app/       # Tauri + React desktop app
```

## Windows Prerequisites

Install once:

1. Go 1.23+
2. Node.js 20+
3. Rust (cargo/rustc)
4. Visual Studio 2022 Build Tools (C++ workload)

## Development Run Guide

Use 2 terminals.

Terminal A: backend

```powershell
cd backend
go run ./cmd/daemon --no-tray
```

Terminal B: desktop UI

```powershell
cd app
npm install
.\dev-tauri.cmd
```

Why use `.\dev-tauri.cmd` instead of `npm run dev`:

- It loads Visual Studio build environment
- It ensures Cargo is available in PATH

## How to Verify It Is Working

If you are new, use this checklist.

1. Start backend in Terminal A:

```powershell
cd backend
go run ./cmd/daemon --no-tray
```

If this command shows no extra output and just keeps running, that is normal.

2. Open backend status endpoint in browser:

- http://127.0.0.1:7331/status

Expected result: JSON like this:

```json
{"running":true,"lastSyncTime":"...","pending":0}
```

3. (Optional) Verify backend using Postman:

- `GET http://127.0.0.1:7331/status`
- `GET http://127.0.0.1:7331/games`
- `POST http://127.0.0.1:7331/sync`

4. Start UI in Terminal B:

```powershell
cd app
.\dev-tauri.cmd
```

Expected result:

- Vite shows `Local: http://localhost:5173/`
- Tauri starts and opens the desktop window

5. Quick functional test in UI:

- Add one game folder in Games view
- Click Sync Now
- Open Logs view and confirm new activity entries

## Tray + Startup Apps

Yes, you can run this from tray and enable startup with Windows.

For reliable startup app behavior, use a built daemon EXE instead of `go run`.

Build daemon EXE:

```powershell
cd backend
go build -o cloud-saver-daemon.exe ./cmd/daemon
```

Run daemon with tray:

```powershell
cd backend
.\cloud-saver-daemon.exe
```

Then in UI:

1. Open Settings
2. Enable "Start on Windows login"
3. Save settings

This writes a per-user Run registry entry used by Startup Apps.

## Build for Release

Desktop package:

```powershell
cd app
npm run tauri:build
```

Optional daemon build:

```powershell
cd backend
go build -o cloud-saver-daemon.exe ./cmd/daemon
```

## Troubleshooting

- `cargo metadata ... program not found`
	- Run from `app` using `.\dev-tauri.cmd`
- `Unexpected token '??='` while running Vite
	- Your Node.js is too old (usually Node 14)
	- Install/use Node 20+ (`nvm use 24.12.0` if using nvm-windows)
- `link.exe not found`
	- Install Visual Studio 2022 Build Tools with C++ tools
- UI cannot connect to backend
	- Start backend first and confirm service on `127.0.0.1:7331`

## Roadmap (Next)

- Real cloud provider integration (Google Drive path)
- Better conflict visibility in UI
- Production packaging flow improvements
- Optional one-click dev/run scripts
