# ADR 0005: Strangler Migration to V2 Microservices

## Status

Accepted.

## Context

The first repository version implemented a Go modular monolith. The V2 product direction requires page-BFF boundaries and independently deployable domain services.

## Decision

Keep `services/core` runnable as a compatibility core while introducing service directories, contracts, and frontend page boundaries. Migrate identity/profile/relationships first, then lessons/schedule, messaging, assignments, notifications, files, realtime, and audit.

## Consequences

User journeys remain runnable during migration. The repository avoids a big-bang rewrite, but documentation must clearly mark which endpoints are compatibility paths and which contracts are target service contracts.

