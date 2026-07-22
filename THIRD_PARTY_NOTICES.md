# Third Party Notices

This file records third-party dependencies used by the Phase 1 implementation.

| Name | Source | Version | License | Use | Distributed with product | Risk | Alternative |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Go | https://go.dev | local toolchain | BSD-3-Clause | compiler/runtime | runtime-linked | low | none |
| go-chi/chi | https://github.com/go-chi/chi | v5.2.1 | MIT | HTTP routing | yes, library | low | stdlib router |
| jackc/pgx | https://github.com/jackc/pgx | v5.7.6 | MIT | PostgreSQL driver/pool | yes, library | low | database/sql driver |
| golang.org/x/crypto | https://cs.opensource.google/go/x/crypto | v0.37.0 | BSD-3-Clause | bcrypt password hashing | yes, library | low | Argon2id package after parameter review |
| Next.js | https://github.com/vercel/next.js | 16.2.10 | MIT | frontend framework | yes, library | low | Remix |
| React | https://github.com/facebook/react | 19.2.7 | MIT | UI library | yes, library | low | Preact |
| TypeScript | https://github.com/microsoft/TypeScript | 5.9.3 | Apache-2.0 | type checking | no runtime | low | none |
| PostgreSQL image | https://github.com/docker-library/postgres | 16-alpine | PostgreSQL License | local Docker service | no | low | managed PostgreSQL |
| Valkey image | https://github.com/valkey-io/valkey | 7-alpine | BSD-3-Clause | local Docker service | no | low | Redis-compatible managed service |
| Mailpit image | https://github.com/axllent/mailpit | v1.20 | MIT | local email testing | no | low | MailHog |
