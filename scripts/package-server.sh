#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
out="${1:-tmp/server-bundle.tar.gz}"
case "$out" in /*) echo "Use a repository-relative output path" >&2; exit 1;; esac
mkdir -p "$(dirname "$out")"
COPYFILE_DISABLE=1 tar --exclude=node_modules --exclude=.next --exclude=.DS_Store --exclude=.env --exclude='.env.local' \
    --exclude='.env.production' --exclude='*/.env*' --exclude=.data --exclude=bin --exclude=coverage \
    --exclude=tmp --exclude=secrets --exclude=backups --exclude=.references \
    --exclude=design/screens \
    --exclude='*.dump' --exclude='*.sqlite*' --exclude='*.db' --exclude='*.xlsx' \
    --exclude='*.log' --exclude='*.tsbuildinfo' --exclude='*.key' --exclude='*.pem' \
    -czf "$out" Dockerfile .dockerignore .env.production.example compose.prod.yml \
    deploy apps/web services/core packages infrastructure design scripts docs \
    README.md THIRD_PARTY_NOTICES.md AGENTS.md Makefile package.json .gitignore
printf 'Server package: %s\n' "$out"
