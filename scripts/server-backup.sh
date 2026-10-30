#!/bin/sh
set -eu
umask 077
cd "$(dirname "$0")/.."
env_file="${REPET_ENV_FILE:-.env.production}"
compose_file="${REPET_COMPOSE_FILE:-compose.prod.yml}"
dc() { docker compose --env-file "$env_file" -f "$compose_file" "$@"; }
dest="backups/$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "$dest"
dc config --quiet
running="$(dc ps --status running --services)"
case "$running" in *api*) ;; *) echo 'API must be running before backup' >&2; exit 1;; esac
trap 'dc up -d api web >/dev/null' 0
dc stop web api
dc exec -T postgres pg_dump -U repet -d repet --format=custom --no-owner --no-acl > "$dest/database.dump"
dc run --rm --no-deps -T --entrypoint tar api -C /private -czf - . > "$dest/objects.tar.gz"
test -s "$dest/database.dump"
test -s "$dest/objects.tar.gz"
printf 'Database and private files; keep secret configuration separately.\n' > "$dest/README.txt"
printf 'Backup: %s\n' "$dest"
