#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
if [ "$#" -ne 2 ] || [ "$2" != "--confirm-replace-data" ]; then
  echo 'Usage: sh scripts/server-restore.sh backups/TIMESTAMP --confirm-replace-data' >&2
  echo 'Destructive: replaces the configured database and private files. Back up first.' >&2
  exit 1
fi
src="$1"
test -s "$src/database.dump"
test -s "$src/objects.tar.gz"
env_file="${REPET_ENV_FILE:-.env.production}"
compose_file="${REPET_COMPOSE_FILE:-compose.prod.yml}"
dc() { docker compose --env-file "$env_file" -f "$compose_file" "$@"; }
dc config --quiet
dc stop web api
dc up -d --wait postgres pdf-extractor
dc exec -T postgres pg_restore -U repet -d repet --clean --if-exists --exit-on-error --no-owner --no-acl < "$src/database.dump"
# Only the application-owned object directory is replaced; do not touch other volumes.
dc run --rm --no-deps -T --entrypoint sh api -c 'find /private/objects -mindepth 1 -maxdepth 1 -exec rm -rf {} \;'
dc run --rm --no-deps -T --entrypoint tar api -C /private -xzf - < "$src/objects.tar.gz"
dc up -d --wait api web
printf 'Restore completed. Check login, lessons and attachments before reopening access.\n'
