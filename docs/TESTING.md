# Rove testing

Rove is tested in layers. Fast tests should run on every change; integration and release checks should run before merging or publishing an installer.

## Fast local paths

```powershell
# Backend unit and integration tests
cd backend
$env:GOCACHE="$PWD\.cache\go-build"
go test ./...

# Frontend unit and component tests
cd ..\app
npm run test:run

# Frontend type and production checks
npm run type-check
npm run build

# Native shell compilation
cd ..\app\src-tauri
cargo check
```

For local coverage reports:

```powershell
cd backend
go test ./... -coverprofile coverage.out
go tool cover -html coverage.out -o coverage.html

cd ..\app
npm run test:coverage
```

## Test paths

| Area | Test path | Required behavior |
| --- | --- | --- |
| Frontend parser boundary | `app/src/api/parsers.test.ts` | Malformed payloads become safe defaults; nested policies, provider state, jobs, conflicts, logs, history, and snapshots are normalized without trusting invalid types. |
| Frontend HTTP client | `app/src/api.test.ts` | JSON headers/bodies, URL encoding, 204 responses, server error messages, and the legacy `/sync/game` compatibility fallback. |
| Native update bridge | `app/src/update.test.ts` | Lifecycle labels, progress subscription cleanup, and exact Tauri command arguments. |
| Library UI | `app/src/views/GamesView.test.tsx` | Empty state, search filtering, add-game modal submission, and game selection. |
| Library UI | `app/src/views/GamesView.test.tsx`, `app/src/components/Onboarding.test.tsx` | Empty state, cloud-only restore list, newer-version block, first-run setup flow. |
| Home UI (legacy) | `app/src/views/HomeView.test.tsx` | Paused/offline provider messaging, byte formatting, recovery restore, and navigation actions. |
| API security boundary | `backend/internal/api/middleware_test.go` | Loopback-only clients, loopback origin allow-list, preflight headers, and foreign-origin rejection. |
| API response boundary | `backend/internal/api/server_test.go` | Unknown JSON fields are rejected, JSON content type is present, and route method boundaries are enforced. |
| Lifecycle API | `backend/internal/api/handlers_lifecycle_test.go` | POST-only update preparation, unavailable callback behavior, and asynchronous callback execution. |
| Settings API/configuration | `backend/internal/api/handlers_settings_test.go`, `backend/internal/config/manager_test.go`, `backend/internal/models/models_test.go` | Partial updates, legacy payload compatibility, malformed-file recovery, OAuth preservation, and daemon-owned field protection. |
| Game/configuration behavior | `backend/internal/config/games.go`, `backend/internal/api/handlers_games.go` | Add/update/delete validation, duplicate paths, policy validation, path/executable checks, and API status/error mapping. Add handler integration cases when changing these routes. |
| Backup integrity | `backend/internal/backup/zip_test.go` | Export naming, nested-file round trip, replace import semantics, and ZIP path traversal rejection. |
| Snapshot integrity | `backend/internal/snapshots/service_test.go` | Manual create/list/restore/delete flow and per-game retention. |
| Local sync engine | `backend/internal/syncer/folder_test.go`, `backend/internal/syncer/engine_test.go` | Publish/download/delete, conflicts, exact restore, offline local-first commits, provider state transitions, concurrent heads, and verified cloud restore. |
| Google Drive provider | `backend/internal/syncer/drive/store_test.go`, `backend/internal/auth/googledrive/service_test.go` | Rate limits, quota errors, retry-after parsing, cancellation, PKCE, callback failures, and OAuth JSON loading. |

## Contract checklist for new endpoints

When adding or changing an endpoint, add tests for:

- every supported HTTP method and unsupported-method response;
- malformed JSON, unknown fields, missing identifiers, invalid identifiers, and not-found resources;
- success payload shape and `Content-Type`;
- authorization boundary: loopback `RemoteAddr` and `Origin` behavior;
- state mutation and persistence, including daemon-owned fields and credentials;
- concurrent/running-game behavior where the operation could corrupt a save;
- provider offline, rate-limited, authentication-required, quota-full, and remote-data-corrupt cases;
- idempotency or retry behavior for repeated requests;
- logs/activity output when the user-visible history contract changes.

## Higher-level verification still expected before release

These paths require a real Windows desktop or CI integration environment and are intentionally not faked in unit tests:

- Tauri command wiring, native folder/file dialogs, autostart, tray menu, process launch, and Windows DPAPI;
- daemon startup/shutdown, single-instance mutex, watcher events, and tray lifecycle;
- a live Google Drive OAuth callback and upload/download against a disposable test account;
- signed updater download, signature failure, rollback, and install blocking while a game/Cloud Sync job is active;
- packaged NSIS installer upgrade/uninstall behavior;
- accessibility smoke checks with keyboard navigation and reduced-motion settings;
- a full user flow: first launch → add game → protect locally → disconnect provider → reconnect → conflict review → restore.

These are release-gate scenarios, not reasons to weaken unit tests with production credentials or real save data.
