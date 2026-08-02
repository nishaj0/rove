# FAQ and troubleshooting

## Does Rove require a server or VPS?

No. The desktop app communicates directly with Google Drive or reads/writes the synced folder you select.

## Do I need a Google API key?

No. Official releases include the matching installed-app OAuth configuration. You only select **Connect Google Drive**. Tokens are protected locally with Windows DPAPI.

Fork developers point `CLOUD_SAVER_GOOGLE_OAUTH_JSON` at their downloaded Desktop OAuth file or configure the release CI secret. Normal users should not set either value.

## What happens offline?

Local checkpoints, final-exit protection, manual protection, and Recovery Points continue. Upload work stays durable and resumes after a Windows connectivity event or scheduled retry.

## Upload pause versus all-protection pause?

Upload pause queues cloud work while local revisions continue. All-protection pause stops captures and transfers but preserves dirty paths, revisions, and jobs.

## Why does the cloud time lag behind local time?

The local commit succeeded but provider replication is pending, paused, rate-limited, offline, unauthenticated, or out of quota. The two timestamps are intentionally separate.

## Authentication required

Open Settings and reconnect Google Drive. Existing local revisions and queued uploads remain.

## Should I delete `settings.json` after an update error?

No. Current versions migrate missing fields, ignore obsolete fields, and keep the previous valid settings as a backup. If the primary JSON is damaged, Rove preserves it separately and restores the backup when possible. Check Activity for **Settings repaired after update** and use **Copy diagnostics** if help is still needed.

## Do I uninstall Rove before updating?

No. Choose **Download update**, continue playing if you want, then choose **Install** after tracked games and protection work finish. The signed NSIS updater replaces application files in place and restarts Rove. `%APPDATA%` settings, the SQLite database, local revisions, saves, and Windows-protected provider credentials are outside the installation files and remain in place.

## Where can I see what changed in an update?

After a signed download finishes, Rove shows a **What's new** window containing the release notes. Choosing **Install later** keeps that verified update ready without changing the app. Reopen the notes from the update banner or **Settings > Application updates** whenever you want.

## What if an update check or download fails offline?

Game launch continues. A failed or timed-out check is informational and can be retried later. Interrupted `.part` downloads are discarded on startup; a completed verified download remains available after closing and reopening Rove. If installation cannot begin, the verified download is retained and the existing version continues running.

## Why is Install disabled?

Close every tracked game and wait for the active local protection job to finish. Rove will not stop the background service or replace program files while save capture is active.

## Quota full

Free provider space or remove provider data through supported provider tools, then choose retry. Do not manually remove individual Rove objects unless you are abandoning that protected library.

## Google Drive says rate limited or unavailable

Rove keeps the revision locally and retries the same pending upload without reading the game save again. It follows Google's requested retry time when supplied and otherwise uses increasing delays, capped at 15 minutes. Repeated clicks and connectivity events do not bypass an active Google cooldown, so the app does not become stuck in a rapid-failure loop.

## Local disk full

Free local space before retrying. Rove stops unsafe writes; it does not delete the last valid revision.

## Save file locked or unstable

Let the game finish saving or close it. Rove requires two stable metadata checks and retries later rather than copying a changing file.

## Permission denied

Run Rove as the same Windows user that owns the save. Avoid protected system folders. Diagnostics sanitize personal paths.

## Conflict shown

Open the game and choose local, remote, or both. No automatic policy overwrites the conflict.

## Legacy snapshots

If a configured old vault contains snapshots, the Game page shows **Legacy local backups**. New legacy snapshots are disabled. Restore or explicitly remove the old entries; Rove does not silently delete them.

## Expected resource use

Idle monitoring uses adaptive process polling and watcher notifications. Clean gameplay does not recursively hash files. One changed file hashes only that file. Balanced in-game capture is Below Normal priority and rate-limited to 4 MiB/s. Actual CPU/FPS impact depends on save size, disk, antivirus, and the game; use Exit only for the least in-game I/O.
