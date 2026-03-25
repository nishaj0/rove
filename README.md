# Cloud Saver Workspace

This workspace now has both parts of the Windows desktop solution:

- backend: Go daemon (watcher, sync engine, local API, tray)
- app: Tauri + React desktop UI

## Run backend

```powershell
cd backend
go run ./cmd/daemon --no-tray
```

## Run desktop UI (dev)

```powershell
cd app
npm install
npm run dev
```

UI expects backend API on:

- http://127.0.0.1:7331

## Current UI pages/functions

- Sync status dashboard
- Game add/remove
- Settings update (interval, policy, autostart, ignore patterns)
- Trigger sync action
- Recent logs view
- ZIP backup export/import controls
