# ADR 0001: Modular Monolith

## Status

Accepted.

## Decision

Use a Go modular monolith for the primary backend.

## Consequences

This keeps transactional workflows simple for Phase 1 while preserving module boundaries for auth, users, relations, lessons, audit, files, messenger, video, whiteboards, notifications, and admin.

