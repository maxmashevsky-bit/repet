# ADR 0004: Object Storage Boundary

## Status

Accepted.

## Decision

Expose object storage through an `ObjectStorage` interface. Production target is S3-compatible private storage. Local development may use a filesystem adapter after upload scanning controls are implemented.

## Consequences

File lifecycle, malware scanning, signed URLs, quotas, and audit can be enforced by our backend regardless of storage provider.

## Restricted Chat Upload Policy (2026-10-30)

Chat v1 uses positive content validation and content disarm/reconstruction rather than accepting arbitrary documents: only bounded PNG/JPEG are decoded and re-encoded without source metadata/trailing bytes; UTF-8 TXT rejects binary/control/format characters and is downloaded as plain text with attachment disposition. Original image bytes never enter storage. This is a scanning control for this narrow allowlist, NOT a general-purpose antivirus guarantee. PDF, office files, SVG/HTML, executables and archives remain rejected until a quarantine/scanner policy exists for them. Do not broaden the allowlist with MIME sniffing alone.

ObjectStorage remains the sole boundary. Chat keys never leave backend; authenticated downloads recheck active conversation participation, with no-store/nosniff/sandbox headers. Server-serialized quotas are50MiB/100 objects per owner, source and canonical bytes≤4MiB, bitmap≤4M pixels. Unsent objects expire after24h and are reclaimed on the owner's next upload; explicit cancellation also deletes them. Attached files remain immutable. Upload/download/delete audit events are recorded. Cleanup after an uncertain DB commit must first confirm the object was not recorded; a general orphan reconciler and production S3/AV remain deployment work, not claims of this development adapter.
