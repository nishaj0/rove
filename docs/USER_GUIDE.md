# Cloud Saver user guide

Cloud Saver keeps game saves protected in three complementary ways:

1. **Cloud sync** copies a save between your computers through a folder already synchronized by your preferred provider.
2. **Snapshots** keep versioned, local restore points in a vault you choose.
3. **ZIP backups** create portable archives that you can export or import manually.

Cloud sync is provider-neutral. Cloud Saver never asks for a OneDrive, Dropbox, Google, or Syncthing password; it only reads and writes the synced folder that you select.

## Before you begin

- Install a file synchronization application and make sure its shared folder is available on every computer that will use the same saves.
- Do not select a synced folder that is inside a game save folder. Cloud Saver blocks this because it would create a sync loop.
- Close a game before manually syncing or restoring. Cloud Saver blocks these operations while it detects that game's executable as running.
- Keep the selected synced folder available offline long enough for your provider to finish downloading its contents before running the first sync on a new computer.

## Add a game

1. Open **Library** and choose **Add Game**.
2. Enter a recognizable game name.
3. Choose the save folder. This is the folder containing the game's save files, not the game installation folder unless they are the same.
4. Choose the game's executable. This enables launch, running-state detection, playtime tracking, and safety checks.
5. Optionally add cover artwork, then choose **Add Game**.

The game page displays its save folder, executable, playtime, running state, automatic-backup policy, snapshots, and Cloud game ID.

## Set up cloud sync on the first computer

1. Open **Settings**.
2. Under **Synced folder**, select a folder maintained by OneDrive, Dropbox, Syncthing, or another sync client.
3. Choose the number of cloud revisions to retain in the setting, then choose **Save Settings**.
4. Open the game page and choose **Sync This Game**.

The first sync creates Cloud Saver's private `.cloud-saver/v1` data inside the selected folder. Do not move, rename, edit, or delete this folder manually.

The Home screen changes from **Setup needed** to **Ready** after a sync folder is saved. The game page shows the most recent job state and the number of changed files.

### Google Drive option

In **Settings**, choose **Google Drive** and select **Connect Google Drive**. Complete sign-in in your browser, then save Settings. Cloud Saver only accesses the save-sync files it creates; it cannot browse the rest of your Drive. You can disconnect at any time or continue using a synced folder instead.

You do not need to enter a client ID, client secret, or any developer setting. The app includes the public connection information it needs and keeps your account connection protected locally.

## Link the same game on another computer

Cloud Saver must know which two local save folders represent the same game. It uses a **Cloud game ID**, not the local folder path or game name.

On the first computer:

1. Open the game's page.
2. Copy its **Cloud game ID**.

On the second computer:

1. Configure the same synced folder in Settings and add the game with that computer's local save folder and executable.
2. Open its game page and choose **Link Existing Cloud Game**.
3. Paste the Cloud game ID from the first computer.
4. If the second save folder is empty, choose **Sync This Game** to download the existing save. If it already has different files, Cloud Saver asks you to review the differences rather than overwriting them.

## How normal sync works

When you choose **Sync This Game**, Cloud Saver:

1. Scans regular files in the save folder and skips ignored patterns.
2. Hashes save contents to identify actual changes.
3. Reads the latest shared revision from the synced folder.
4. Uploads a new immutable revision, downloads an unchanged remote revision, or safely merges independent file changes.
5. Updates the job state on the game page and in **Activity**.

Files deleted on one synchronized computer are deleted on the other after a successful sync. Before applying a remote replacement or deletion, Cloud Saver copies the displaced local file into the private cloud-version store for recovery.

### Performance and storage expectations

Cloud Saver is designed to run continuously as a local background service. It
normally uses little CPU while idle, but a sync scan currently reads and hashes
all regular files in the selected save folders. A large save folder can therefore
produce disk activity every 30 seconds (the default interval), even when the
contents have not changed. A file change can also trigger an earlier scan.

Cloud Saver runs locally on your computer. With a provider-synced folder, your
OneDrive, Dropbox, Syncthing, or similar client handles Internet transfer. With
Google Drive selected, Cloud Saver makes Drive API requests only while it is
performing sync work.

The `.cloud-saver/v1` directory contains immutable revisions, deduplicated file
objects, and recovery copies. Do not delete it manually. Automatic pruning of old
cloud revisions is not implemented yet, so this directory may grow as revisions
accumulate. The **Cloud revision retention** setting is stored, but it should not
currently be understood as an active deletion guarantee. Snapshots and ZIP backups
are separate and follow their own retention settings.

## Resolve a sync conflict

A conflict means the same file changed on both computers after their last common sync. Cloud Saver does not pick a winner automatically.

Open the affected game. In **Sync conflicts need review**, each conflicting path lists shortened local and remote hashes. Pick one action:

- **Keep local**: preserve the file currently on this computer and publish it as the next shared version.
- **Use remote**: replace the local file with the remote version.
- **Keep both**: keep the local file and save the remote copy beside it with `.cloud-conflict-<hash>` in the file name.

If a conflict is stale because another computer synced again, synchronize first and review the current conflict instead.

## Snapshots and restore

Snapshots are local versioned save points. They are not the same as cloud revisions.

1. In **Settings**, choose a **Snapshot vault folder** and save.
2. Use **Save Now** from a game page, or enable snapshot-on-exit and periodic snapshots.
3. To restore, select a snapshot and confirm the prompt.

Cloud Saver creates a pre-restore snapshot first. Restore is exact: files that do not exist in the selected snapshot are removed from the live save folder.

## Local backups and ZIP files

- **Automatic local backups** run according to the per-game policy while a tracked game is running and when it exits.
- **Export Save** creates a ZIP archive in a destination folder you select.
- **Import Save** imports a ZIP archive. With **Merge into existing folder** checked, existing files not represented in the archive remain. Unchecked import behavior may replace content, so make a snapshot first.

## Settings reference

| Setting | What it controls |
| --- | --- |
| Sync interval | How often the daemon checks for scheduled sync work. |
| Ignore patterns | Comma-separated filenames, wildcard patterns, or directory names to skip, such as `*.tmp`, `*.log`, or `cache`. |
| Synced folder | The shared local folder used for cloud revisions. |
| Cloud revision retention | The saved retention policy for future cloud revision cleanup; automatic pruning is not yet active. |
| Snapshot vault | Local directory that stores snapshots. |
| Max snapshots per game | Number of local snapshots retained for each game. |
| Snapshot on game exit | Creates a snapshot when a monitored game stops. |
| Periodic snapshots | Creates snapshots at the configured interval while a game runs. |
| Start on Windows login | Starts the desktop/daemon experience with Windows. |

## FAQ

### Does Cloud Saver support Google Drive directly?

Yes, as an optional choice in Settings. You can also keep using a synced folder; Google Drive is never required.

### What can Cloud Saver see in my Google account?

Only the save-sync files Cloud Saver creates. It cannot browse the rest of your Drive.

### Is my provider password stored in Cloud Saver?

No provider password is stored. Google Drive keeps a Windows-protected connection token locally after you choose to connect it.

### Why is Sync This Game disabled or rejected?

Configure and save a synced folder first. It is also blocked while the configured game executable is detected as running.

### Why does a sync use disk or CPU when I did not change a save?

The current safety fallback scans and hashes the save folder on each scheduled
reconciliation. This makes missed filesystem events recoverable, but it means a
large save directory can be expensive to scan. Close unnecessary games, increase
the sync interval, and use ignore patterns for temporary files when appropriate.

### Does Cloud Saver delete old cloud revisions automatically?

Not yet. Cloud Saver keeps immutable revisions and displaced-file recovery copies
so they remain available for safety. The retention setting is persisted for the
planned cleanup feature, but old cloud objects are currently not removed
automatically.

### What if I accidentally choose the wrong Cloud game ID?

Linking changes the destination stream for that local game. Do not sync until you verify the ID. If the folders contain unrelated saves, Cloud Saver should surface conflicts instead of automatically overwriting files.

### Can I delete `.cloud-saver/v1` to save space?

No. That data is the shared revision history and object store. Deleting it breaks pairing and recovery. Use your provider's normal recovery tools or remove the cloud setup only when you no longer need the shared save history.

### My game saves in the cloud already. Should I use Cloud Saver too?

Only if you understand which folders each tool manages. Never point two sync systems at different interpretations of the same save state without testing on a copy first.

### Where can I see errors?

Open **Activity** for the latest backend error and logs. If the app cannot contact the daemon, start the backend and check `http://127.0.0.1:7331/debug/info`.
