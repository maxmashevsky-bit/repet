# Репет

«Репет» — ролевая образовательная платформа для ученика и преподавателя. Текущая версия реализует V2 product shell и совместимый Go API, который постепенно мигрируется из compatibility core в page-BFF и domain services по strangler pattern.

## Реализованный вертикальный срез

- регистрация и вход с выбором роли;
- server-owned роль через session cookie;
- разные главные экраны ученика и преподавателя;
- сообщения с сохранением через API;
- календарь `/calendar?view=day|week|month`;
- задания ученика и центр заданий преподавателя;
- уведомления, профиль и настройки;
- PostgreSQL migrations;
- request ID, security headers, CORS allowlist, structured logs, health/readiness/metrics.

## Быстрый запуск

```sh
cp .env.example .env
docker compose -p repet up -d postgres valkey mailpit
make migrate
make api
```

В другом терминале:

```sh
PATH=/Users/maksimmasevskij/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/bin:/Users/maksimmasevskij/.cache/codex-runtimes/codex-primary-runtime/dependencies/bin:$PATH make web
```

Откройте [http://127.0.0.1:3000/register](http://127.0.0.1:3000/register).

Флаг `-p repet` нужен, потому что Docker Compose не принимает имя проекта, автоматически полученное из кириллического имени папки.

## Структура

- `apps/web` — Next.js App Router, feature slices.
- `apps/api-gateway` — целевая граница gateway.
- `apps/bff/*` — целевые page-BFF границы.
- `services/core` — совместимый Go API для текущего runnable slice.
- `services/*` — целевые владельцы доменных данных.
- `packages/contracts` — OpenAPI/AsyncAPI контракты.
- `design` — локальные PNG-референсы, не runtime assets.
- `docs` — архитектура, security, runbooks и ADR.

## Проверки

```sh
make test
make vet
make race
cd apps/web && pnpm typecheck && pnpm exec next build
```
