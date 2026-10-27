# Recovery and safe restore

## After reinstalling Windows or the game

1. Install Rove.
2. Select Local only, connect Google Drive, or select the same synced folder.
3. Open **Home** (or look at the sidebar) and find **In your cloud**.
4. Choose the game and select **Restore**. A save marked **Needs a newer Rove** was written by a newer version; update Rove first.
5. Choose the installed game executable and a new or empty destination save folder.
6. Confirm the destination. Rove verifies and stages the complete revision, restores it, and links the restored game to the same cloud identity.

Cloud metadata stores environment-tokenized hints such as `%APPDATA%` and executable basenames. It does not publish the previous Windows username or an absolute personal path.

Rove also carries an optional game version, a compact cover image (when one is available), and playtime metadata. Covers use your manual image first, then the executable icon, then a stable generated cover. Save contents remain separate immutable objects.

## Restoring tracked games

From the Game page, select a revision or Recovery Point. Restore is refused while the executable is running. Rove first creates a local pre-restore Recovery Point, verifies every copied object's SHA-256, and restores exact state—including deletion of files absent from the selected revision.

## ZIP replacement

An unchecked merge option means replacement. The backend requires confirmation, refuses a running game, creates a Recovery Point, validates/extracts the archive into a sibling staging directory, then swaps it into place. If staging fails, the live destination is not published.

## Conflicts

Do not choose a winner until you understand which save is newer. **Keep both** is the safest exploratory choice for formats where the game can recognize a renamed slot; some games cannot. Rove never parses proprietary save formats.

## Local-only limitation

Local-only revisions survive game uninstall but not loss/reset of the disk containing Rove's application data. Use Google Drive, a synced folder, or manual ZIP export for disaster recovery.

## Corruption and missing objects

A hash mismatch prevents replacement. An incomplete remote revision cannot be restored because its head is published only after objects and manifest complete; provider-side deletion can still make an older revision incomplete, in which case Rove reports a remote data problem and retains local data.
