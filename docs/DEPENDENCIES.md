# Dependency Inventory

Only permissive licenses are accepted without additional legal approval.

| Component | Repository | Purpose | Version | License | Use Mode | Distributed | Risk | Safe Alternative |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| go-chi/chi | https://github.com/go-chi/chi | HTTP router and middleware | v5.2.1 | MIT | library | yes | low | stdlib mux |
| jackc/pgx | https://github.com/jackc/pgx | PostgreSQL driver and pool | v5.7.6 | MIT | library | yes | low | lib/pq style driver |
| golang.org/x/crypto | https://cs.opensource.google/go/x/crypto | bcrypt password hashing | v0.37.0 | BSD-3-Clause | library | yes | low | Argon2id package after parameter review |
| Next.js | https://github.com/vercel/next.js | frontend framework | 16.2.10 | MIT | library | yes | low | Remix |
| React | https://github.com/facebook/react | UI | 19.2.7 | MIT | library | yes | low | Preact |
| TypeScript | https://github.com/microsoft/TypeScript | type safety | 5.9.3 | Apache-2.0 | build tool | no runtime | low | none |
| PostgreSQL | https://github.com/postgres/postgres | relational database | 16-alpine image | PostgreSQL License | Docker service | no | low | managed PostgreSQL |
| Valkey | https://github.com/valkey-io/valkey | Redis-compatible local cache/realtime future | 7-alpine image | BSD-3-Clause | Docker service | no | low | managed Redis-compatible service |
| Mailpit | https://github.com/axllent/mailpit | local email capture | v1.20 image | MIT | Docker service | no | low | MailHog |

## Components Considered

| Component | License posture | Decision | Notes |
| --- | --- | --- | --- |
| Ory Kratos | Apache-2.0 | future separate service | Preferred production identity provider, deferred to avoid blocking Phase 1. |
| Excalidraw | MIT | future frontend library | Use only UI package; backend owns rooms, tickets, authorization, and snapshots. |
| LiveKit | Apache-2.0 | future separate service/provider | Integrate through `VideoProvider`; do not couple lessons to a vendor. |
| tusd | MIT | future separate service or upload adapter | Useful for resumable uploads; Phase 1 defines storage boundaries first. |
| Centrifugo | MIT | future separate realtime service | Good fit for WebSocket fanout; backend issues short-lived tickets. |
| Tinode | GPL-3.0 server | adapter only, legal review required | Do not link or embed in main Go binary. Prefer custom chat or permissive service. |
| SeaweedFS | Apache-2.0 | future separate service candidate | S3-compatible object storage can also be managed cloud S3. |
| React Admin | MIT | future admin frontend candidate | Admin surface can be built separately after API hardening. |
