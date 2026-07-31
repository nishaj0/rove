# Provider interface

Providers implement `syncer.RevisionStore`:

```go
type RevisionStore interface {
    ReadHead(context.Context, string) (string, error)
    WriteHead(context.Context, string, string) error
    ReadManifest(context.Context, string, string) (*Manifest, error)
    WriteManifest(context.Context, string, *Manifest) error
    ReadObject(context.Context, string, string) (io.ReadCloser, error)
    WriteObject(context.Context, string, string, io.Reader) error
    ArchiveDisplaced(context.Context, string, string, string, io.Reader) error
}
```

Modern optional capabilities are `DeviceHeadStore`, `ProfileStore`, and `RecoveryPointStore`.

## Publication contract

1. Object write is content-addressed and verifies SHA-256.
2. Manifest write is immutable.
3. Per-device head is updated after the manifest.
4. Compatibility `head.json` is updated last.
5. Missing or malformed content returns an error; it must never be interpreted as an empty valid revision.

## Folder layout

```text
.cloud-saver/v1/games/<syncId>/
  game.json
  head.json
  heads/<deviceId>.json
  objects/<sha256>
  revisions/<revisionId>.json
  recovery-points/<pointId>.json
  versions/<deviceId>/...
```

## Google Drive

The Drive provider uses `drive.file`, app properties for logical type/sync ID/name, streaming media, and 8 MiB resumable chunks. Requests are paced to eight per second and the engine permits one provider transfer globally. Rate-limit responses honor Google `Retry-After`; the resulting cooldown is persisted and is not bypassed by reconnect, manual retry, or network-change events. Chunk requests have a five-minute bound, the overall retry path has a ten-minute bound, and typed errors distinguish authentication, quota, permission, rate limit, temporary outage, and connectivity failures. A local revision is never recaptured merely because its replication retry failed.

The provider owns only files created through its OAuth client. Refresh tokens use DPAPI. OAuth JSON is never copied into the repository; official builds inject the required Desktop client credential from a protected CI secret. Local setup, configuration precedence, expected diagnostic stages, and fork release configuration are documented in [Google Drive development](GOOGLE_DRIVE_DEVELOPMENT.md).

The current provider lists app-owned profile metadata. SQLite has cursor storage reserved, but Drive Changes API cursor consumption is not active yet. Add cursor-expiration recovery and persist Drive file/folder IDs before removing name-based lookup fallback.

## Adding a provider

- Keep credentials outside settings responses and diagnostics.
- Return typed/classifiable auth, rate, quota, unavailable, and corruption errors.
- Stream content; do not buffer a save object in memory.
- Make every write idempotent.
- Test interruption after object, manifest, device-head, and compatibility-head stages.
- Test two devices, deletes/tombstones, same-path conflicts, missing objects, and corrupted downloads.
- Never require a Cloud Saver VPS.
