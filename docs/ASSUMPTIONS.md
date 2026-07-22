# Assumptions

- The initial repository was empty and not initialized as Git.
- Phase 1 uses a safe development identity provider based on trusted reverse-proxy-style headers. It cannot run when `APP_ENV=production`.
- Production identity should use Ory Kratos or another reviewed identity provider before public launch.
- A single default organization is created per request in Phase 1 to keep tenant checks explicit while avoiding a full organization onboarding flow.
- Email delivery, password login, MFA, refresh token rotation, WebSocket, files, video, whiteboards, homework, and notifications are documented and shaped by interfaces but deferred beyond Phase 1.
- Minors may use the system; privacy defaults are restrictive and require legal review before production.
- No external reference repositories were cloned in Phase 0. Decisions are based on known public licensing/integration models and must be revalidated before procurement or launch.

