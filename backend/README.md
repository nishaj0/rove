# Cloud Saver backend

The Go daemon owns local configuration, save reconciliation, snapshots, game monitoring, and the localhost API used by the Tauri desktop app.

## Sync model

Cloud Saver uses a provider-neutral synced folder rather than storing provider credentials. A selected OneDrive, Dropbox, Syncthing, or similar folder contains an internal `.cloud-saver/v1` store with:

- content-addressed file objects verified with SHA-256;
- immutable per-game revision manifests and tombstones;
- an atomically updated head pointer; and
- archived displaced files for recovery.

Each local game has a local ID and a separate `syncId`. The latter is shared when pairing the same game on another computer. Local sync state and unresolved conflicts are persisted in SQLite.

Conflicts are review-only. The API supports `local`, `remote`, and `both` resolutions; no reconciliation path silently overwrites concurrent changes.

## Run

```powershell
cd backend
go run ./cmd/daemon --no-tray
```

Use `--debug` for verbose logs. Settings are stored under `%AppData%/CloudSaver/settings.json` by default.

## API

- `GET`, `POST /games`
- `GET`, `PUT`, `DELETE /games/:id`
- `POST /games/sync`
- `POST /games/pair` — pair a local game with an existing `syncId`
- `POST /sync`
- `GET /sync/jobs?gameId=...`
- `GET /sync/conflicts?gameId=...`
- `POST /sync/conflicts` — resolve with `local`, `remote`, or `both`
- `GET /status`
- `GET`, `POST /settings`
- `GET /logs/recent?limit=100`
- `POST /backup/export`, `POST /backup/import`
- `GET`, `POST /snapshots`, `POST /snapshots/restore`, `DELETE /snapshots/:id`

## Tests

```powershell
cd backend
go test ./...
```

The sync integration test covers two-device initial sync, updates, deletion propagation, conflict detection, and an explicit conflict resolution.
