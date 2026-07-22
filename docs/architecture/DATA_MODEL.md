# Data Model

Phase 1 tables:

- `organizations`
- `users`
- `tutor_profiles`
- `student_profiles`
- `invitations`
- `relations`
- `lessons`
- `audit_events`

```mermaid
erDiagram
  organizations ||--o{ users : owns
  users ||--o| tutor_profiles : has
  users ||--o| student_profiles : has
  users ||--o{ invitations : creates
  invitations ||--o| relations : becomes
  users ||--o{ relations : tutor
  users ||--o{ relations : student
  relations ||--o{ lessons : schedules
  users ||--o{ audit_events : acts
```

Server-owned fields include organization, role, ownership, creator, and state transitions. Clients submit only workflow input.

