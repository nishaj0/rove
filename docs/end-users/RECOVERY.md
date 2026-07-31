# Recovery and safe restore

## After reinstalling Windows or the game

1. Install Cloud Saver.
2. Select Local only, connect Google Drive, or select the same synced folder.
3. Open the Home **Recovery Library**.
4. Choose a remote-only game and select a new or empty destination folder.
5. Confirm the destination.
6. After restore, add the game and its executable so monitoring resumes.

Cloud metadata stores environment-tokenized hints such as `%APPDATA%` and executable basenames. It does not publish the previous Windows username or an absolute personal path.

## Restoring tracked games

From the Game page, select a revision or Recovery Point. Restore is refused while the executable is running. Cloud Saver first creates a local pre-restore Recovery Point, verifies every copied object's SHA-256, and restores exact state—including deletion of files absent from the selected revision.

## ZIP replacement

An unchecked merge option means replacement. The backend requires confirmation, refuses a running game, creates a Recovery Point, validates/extracts the archive into a sibling staging directory, then swaps it into place. If staging fails, the live destination is not published.

## Conflicts

Do not choose a winner until you understand which save is newer. **Keep both** is the safest exploratory choice for formats where the game can recognize a renamed slot; some games cannot. Cloud Saver never parses proprietary save formats.

## Local-only limitation

Local-only revisions survive game uninstall but not loss/reset of the disk containing Cloud Saver's application data. Use Google Drive, a synced folder, or manual ZIP export for disaster recovery.

## Corruption and missing objects

A hash mismatch prevents replacement. An incomplete remote revision cannot be restored because its head is published only after objects and manifest complete; provider-side deletion can still make an older revision incomplete, in which case Cloud Saver reports a remote data problem and retains local data.
