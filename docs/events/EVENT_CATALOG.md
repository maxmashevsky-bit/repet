# Event Catalog

All events use this envelope:

```json
{
  "eventId": "uuid",
  "eventType": "assignment.published.v1",
  "occurredAt": "2026-07-22T12:00:00Z",
  "producer": "assignment-service",
  "schemaVersion": 1,
  "correlationId": "request-id",
  "actorId": "uuid",
  "organizationId": "uuid",
  "payload": {}
}
```

## Versioned Events

- `identity.user_registered.v1`
- `relationship.invitation_accepted.v1`
- `lesson.scheduled.v1`
- `lesson.rescheduled.v1`
- `lesson.cancelled.v1`
- `message.sent.v1`
- `message.read.v1`
- `assignment.published.v1`
- `assignment.submitted.v1`
- `assignment.graded.v1`
- `assignment.revision_requested.v1`
- `notification.created.v1`
- `file.upload_completed.v1`

Message body and file contents are not placed on the common event bus.

