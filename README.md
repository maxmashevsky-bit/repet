# Репет

«Репет» — приложение для занятий преподавателя и ученика. Рабочий API находится в `services/core` (Go/PostgreSQL), интерфейс — в `apps/web` (Next.js). Каталоги BFF и domain services описывают целевые границы миграции и пока не запускаются как отдельные сервисы.

## Запуск

Требуются Docker и Docker Compose. В корне репозитория:

```sh
git clone git@github.com:maxmashevsky-bit/repet.git
cd repet
docker compose up --build
```

Откройте http://127.0.0.1:3000/register. API автоматически применяет миграции к пустой базе. Письма с кодом восстановления пароля доступны в Mailpit: http://127.0.0.1:8025.

Compose слушает только loopback. Если стандартные порты заняты, можно задать `POSTGRES_PORT`, `API_PORT`, `WEB_PORT`, `SMTP_PORT` и `MAILPIT_PORT` перед командой. Остановка: `docker compose down` (данные PostgreSQL сохраняются в volume).

Для локальной разработки без контейнеров приложения нужны Go 1.25+, Node.js 22+ и pnpm 11.19.0:

```sh
cp .env.example .env
docker compose up -d postgres mailpit
(cd apps/web && pnpm install --frozen-lockfile --ignore-scripts)
set -a; . ./.env; set +a
make api
```

В другом терминале запустите `make web`. Next.js направляет `/api/v1/*` в Go API через `API_INTERNAL_URL`; по умолчанию это `http://127.0.0.1:8080`.

## Основной сценарий

1. Зарегистрируйтесь как преподаватель. На главной создайте приглашение по email ученика и передайте полученную ссылку.
2. Ученик регистрируется с тем же email, открывает ссылку и принимает приглашение.
3. Преподаватель создаёт занятие в календаре и задание в разделе «Задания». Ученик видит их, отправляет ответ, преподаватель выставляет баллы и комментарий.
4. Обе стороны видят уведомления и могут писать в созданном диалоге.

## Проверки

Для этих команд нужны Go 1.25+, Node.js 22+, pnpm 11.19.0 и установленные зависимости `apps/web`.

```sh
(cd apps/web && pnpm install --frozen-lockfile --ignore-scripts)
make test
make lint
make race
make typecheck
make web-build
```

Интеграционный тест с реальной PostgreSQL запускается при заданном `TEST_DATABASE_URL`:

```sh
cd services/core
TEST_DATABASE_URL='postgres://tutor:tutor@127.0.0.1:5432/tutor_platform?sslmode=disable' go test ./internal/store -run TestProductFlowPostgres -count=1
```

Контракт текущего API: `packages/contracts/openapi/core-compatibility.yaml`. Настройки развития архитектуры находятся в `docs/`.
