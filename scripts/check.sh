#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

(cd services/core && go test ./...)
(cd services/core && go vet ./...)
(cd services/core && go test -race ./...)

if command -v pnpm >/dev/null 2>&1; then
  (cd apps/web && pnpm lint && pnpm typecheck && pnpm build)
fi
