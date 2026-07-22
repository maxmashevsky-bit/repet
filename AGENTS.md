# Agent Operating Guide

This repository is the production-oriented education platform monorepo for «Репет».

## Safety Rules

- Work only inside this repository.
- Do not create production credentials.
- Do not deploy, push, open pull requests, or change GitHub settings.
- Do not run install scripts from unknown repositories.
- Do not copy code from reference repositories until license review is complete.
- Keep `.references/` uncommitted.

## Engineering Rules

- Target backend architecture is V2 microservices with page-BFF boundaries and domain services.
- `services/core` is a compatibility core kept runnable while the strangler migration proceeds.
- Frontend is Next.js, React, and TypeScript.
- Prefer explicit constructors and dependency injection.
- Security checks are deny-by-default and resource-specific.
- Server-owned fields such as `role`, `owner_id`, `organization_id`, `created_by`, and `permissions` are never accepted from clients.
- The product UI brand is only «Репет».
- Top-level product navigation is only `Главное`, `Сообщения`, `Календарь`, `Задания`.
- There is no standalone `Материалы` route; files live inside lessons, messages, or assignments.

## Verification

Run the strongest available local checks before handoff:

```sh
make test
make lint
```
