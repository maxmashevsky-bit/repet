# V2 Microservices Architecture

The current runnable slice uses `services/core` as a compatibility core. The target architecture is service-owned data plus page-BFF aggregation. Migration follows strangler pattern: keep user journeys working, move one bounded context at a time, then retire compatibility endpoints after contract and data migration tests pass.

## C4 Context

```mermaid
flowchart LR
  Student[Ученик] --> Web[Next.js Web]
  Teacher[Преподаватель] --> Web
  Web --> Gateway[api-gateway]
  Gateway --> BFF[Page BFFs]
  BFF --> Identity[identity-service]
  BFF --> Profile[profile-service]
  BFF --> Lessons[lesson-service]
  BFF --> Messaging[messaging-service]
  BFF --> Assignments[assignment-service]
  BFF --> Notifications[notification-service]
```

## Containers

```mermaid
flowchart TB
  Browser --> Web[apps/web]
  Web --> Gateway[apps/api-gateway]
  Gateway --> AuthBFF[auth-page-bff]
  Gateway --> DashboardBFF[dashboard-page-bff]
  Gateway --> MessagesBFF[messages-page-bff]
  Gateway --> CalendarBFF[calendar-page-bff]
  Gateway --> TasksBFF[tasks-page-bff]
  Gateway --> SettingsBFF[settings-page-bff]
  AuthBFF --> Identity
  SettingsBFF --> Profile
  MessagesBFF --> Messaging
  CalendarBFF --> Schedule
  TasksBFF --> Assignments
  DashboardBFF --> Lessons
  DashboardBFF --> Notifications
  Identity[(identity DB)]
  Profile[(profile DB)]
  Messaging[(messaging DB)]
  Assignments[(assignment DB)]
```

## Data Ownership

| Service | Owns | Does not own |
| --- | --- | --- |
| identity-service | identities, credentials, sessions | profile preferences |
| profile-service | display name, avatar, timezone, locale | credentials, role |
| relationships-service | invitations, tutor-student links | lessons, messages |
| lesson-service | lessons and lesson policy | calendar availability |
| schedule-service | calendar events, availability, reschedule records | assignment deadlines |
| messaging-service | conversations, messages, receipts | file blobs |
| assignment-service | assignments, submissions, grading | notification delivery |
| notification-service | notification inbox and delivery preferences | source domain status |
| file-service | file metadata, grants, signed URLs | message or assignment body |
| audit-service | immutable audit trail | domain operational data |

## Compatibility

`services/core` currently owns all runnable tables. It is explicitly temporary. New service migrations must use expand/migrate/contract and opaque IDs. Cross-service joins and foreign keys are forbidden once a bounded context leaves `core`.

