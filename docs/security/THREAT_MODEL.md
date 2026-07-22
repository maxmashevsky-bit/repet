# Threat Model

## Assets

- Minor student personal data.
- Tutor-student relationship graph.
- Lesson metadata and future files/chat/video records.
- Authentication sessions and audit trail.

## Controls

- Deny-by-default authorization.
- Tenant checks on every protected endpoint.
- Participant checks before relation or lesson access.
- Server-owned role and organization assignment.
- Parameterized SQL through pgx.
- Security headers, CORS allowlist, request IDs, and structured logs.
- Audit events for invitations, relation acceptance, and lessons.

## Authentication Flow

```mermaid
sequenceDiagram
  participant Browser
  participant Web as Next.js
  participant API as Go API
  participant IdP as IdentityProvider
  Browser->>Web: Sign in
  Web->>IdP: Start identity flow
  IdP-->>Web: Session cookie or token exchange
  Web->>API: Authenticated request
  API->>IdP: Validate session
  API-->>Web: User context
```

Phase 1 replaces the IdP with a development-only provider that trusts local headers and is blocked in production.

## File Upload Flow

```mermaid
sequenceDiagram
  participant Client
  participant API
  participant AV as Antivirus Adapter
  participant Store as ObjectStorage
  Client->>API: Request upload
  API->>API: Check extension, quota, relation, state
  API->>Store: Write private random object
  API->>AV: Scan
  AV-->>API: Clean or rejected
  API->>API: Mark available or rejected
  API-->>Client: Signed download URL when available
```

## WebSocket Connection Flow

```mermaid
sequenceDiagram
  participant Client
  participant API
  participant WS as Realtime Gateway
  Client->>API: Request one-time ticket
  API->>API: Check authenticated participant and origin allowlist
  API-->>Client: Short-lived ticket
  Client->>WS: Connect with ticket
  WS->>API: Redeem ticket
  API-->>WS: Room permissions
  WS-->>Client: Connected with heartbeat and sequence
```

## Residual Risks

- Production identity, MFA, email verification, and refresh-token rotation are not implemented in Phase 1.
- File scanning, WebSocket tickets, and video rooms are architectural placeholders until later phases.
- Legal privacy review is required before serving minors in production.

