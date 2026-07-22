#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

(cd services/core && go test ./...)
(cd services/core && go vet ./...)

if command -v pnpm >/dev/null 2>&1; then
  (cd apps/web && pnpm typecheck)
fi

