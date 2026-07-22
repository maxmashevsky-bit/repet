# Docker

Local infrastructure is defined in `compose.yml`.

Phase 1 intentionally does not publish Valkey or internal service ports to the host. PostgreSQL and Mailpit are bound to `127.0.0.1` for local development.

