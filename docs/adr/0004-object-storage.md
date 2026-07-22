# ADR 0004: Object Storage Boundary

## Status

Accepted.

## Decision

Expose object storage through an `ObjectStorage` interface. Production target is S3-compatible private storage. Local development may use a filesystem adapter after upload scanning controls are implemented.

## Consequences

File lifecycle, malware scanning, signed URLs, quotas, and audit can be enforced by our backend regardless of storage provider.

