# ADR 0002: Identity Provider Strategy

## Status

Accepted for Phase 1.

## Decision

Production should use a reviewed identity provider such as Ory Kratos. Phase 1 uses a development-only provider behind an `IdentityProvider` interface.

## Consequences

The MVP is runnable without deploying Kratos. The development provider refuses to start in production, preventing accidental public use.

