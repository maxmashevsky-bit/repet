# Architecture Overview

The platform starts as a modular monolith. Business modules share one deployment unit but expose explicit service boundaries and avoid direct access to each other's private data.

```mermaid
flowchart LR
  Web[Next.js Web App] --> API[Go HTTP API]
  API --> Auth[Auth Module]
  API --> Users[Users and Profiles]
  API --> Relations[Invitations and Relations]
  API --> Lessons[Lessons]
  API --> Audit[Audit Log]
  API --> DB[(PostgreSQL)]
  API -.future.-> Storage[ObjectStorage Interface]
  API -.future.-> Messenger[Messenger Interface]
  API -.future.-> Video[VideoProvider Interface]
  API -.future.-> Identity[IdentityProvider Interface]
```

## Containers

```mermaid
flowchart TB
  Browser[Browser] --> Next[Next.js app]
  Next --> Core[Go core API]
  Core --> Postgres[(PostgreSQL)]
  Core --> Valkey[(Valkey future cache/realtime)]
  Core --> Mailpit[Mailpit local email]
  Core -.future.-> S3[S3-compatible object storage]
  Core -.future.-> LiveKit[LiveKit]
  Core -.future.-> Kratos[Ory Kratos]
```

## Module Boundaries

```mermaid
flowchart LR
  Auth --> HTTP[HTTP handlers]
  Users --> HTTP
  Invitations --> HTTP
  Relations --> HTTP
  Lessons --> HTTP
  HTTP --> Access[Access Policy]
  HTTP --> Audit
  Access --> Store[Repository interfaces]
  Store --> DB[(PostgreSQL)]
```

