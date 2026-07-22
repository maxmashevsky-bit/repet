# Execution Plan

## Phase 0 - Research and Design

- Establish monorepo structure.
- Document architecture, data model, threat model, privacy model, license policy, dependency inventory, and ADRs.
- Select integration posture for identity, realtime, files, whiteboard, video, messenger, and admin.

## Phase 1 - Vertical MVP

- Build a Go modular monolith.
- Provide local PostgreSQL migrations.
- Implement development-only identity provider.
- Implement users, profiles, invitations, relations, lessons, object access checks, and audit logging.
- Build a Next.js frontend for login headers, tutor/student dashboards, invitations, relations, and lessons.
- Add tests and local CI workflow.

## Later Phases

- Replace development auth with production identity provider integration.
- Add refresh token rotation, email verification, MFA/passkeys for admins.
- Add homework, submissions, progress, files, chat, WebSocket tickets, whiteboard snapshots, video sessions, notifications, and admin UI.
- Add S3-compatible storage, malware scanning, SBOM, container scanning, secret scanning, and signed release process.

