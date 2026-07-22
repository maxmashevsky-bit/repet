# Runbook: Event Replay

Replay only idempotent handlers. Store processed event IDs per consumer. Never replay events containing message bodies or file contents onto a broad bus.

