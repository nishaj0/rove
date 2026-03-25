# Cloud Saver Backend (MVP Foundation)

This backend runs a local daemon for game save sync orchestration.

## Implemented in this step

- Local settings store (JSON)
- Game CRUD config management
- Local HTTP API on localhost
- Sync scheduler (manual + 30s interval)
- Recursive filesystem watcher integration
- SQLite log storage
- ZIP export/import backup endpoints
- Windows autostart helpers
- Windows tray scaffolding (systray)

## Run

```powershell
cd backend
go mod tidy
go run ./cmd/daemon --no-tray
```

By default, settings are stored at:

- `%AppData%/CloudSaver/settings.json`

## API (current)

- `GET /games`
- `POST /games`
- `PUT /games/:id`
- `DELETE /games/:id`
- `POST /sync`
- `GET /status`
- `GET /settings`
- `POST /settings`
- `GET /logs/recent?limit=100`
- `POST /backup/export`
- `POST /backup/import`

## Notes

- Current sync engine tracks file changes incrementally and logs cycles.
- Cloud provider implementation is pending (next step: provider abstraction + manual mode plumbing + Google Drive path).
