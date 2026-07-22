# ADR 0003: Realtime and Collaboration Providers

## Status

Accepted.

## Decision

Use provider interfaces for realtime messaging, whiteboards, and video. Centrifugo, Excalidraw, and LiveKit remain candidates. Tinode is not linked into the Go binary because of GPL licensing.

## Consequences

Future integrations can run as separate services without contaminating the core backend licensing or authorization model.

