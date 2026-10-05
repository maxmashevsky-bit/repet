SHELL := /bin/sh

GO_SERVICE := ./services/core
NODE ?= node
PNPM ?= pnpm

.PHONY: api test lint vet race migrate web web-build typecheck staticcheck govulncheck gosec audit sbom license-check secret-scan container-scan security-checks

api:
	cd $(GO_SERVICE) && go run ./cmd/api

test:
	cd $(GO_SERVICE) && go test ./...

race:
	cd $(GO_SERVICE) && go test -race ./...

vet:
	cd $(GO_SERVICE) && go vet ./...

lint: vet
	cd $(GO_SERVICE) && test -z "$$(gofmt -l $$(find . -name '*.go'))"
	cd apps/web && $(PNPM) lint

migrate:
	cd $(GO_SERVICE) && go run ./cmd/api -migrate-only

web:
	cd apps/web && $(PNPM) dev

web-build:
	cd apps/web && $(PNPM) build

typecheck:
	cd apps/web && $(PNPM) typecheck

staticcheck:
	cd $(GO_SERVICE) && command -v staticcheck >/dev/null 2>&1 && staticcheck ./... || echo "staticcheck is not installed"

govulncheck:
	cd $(GO_SERVICE) && command -v govulncheck >/dev/null 2>&1 && govulncheck ./... || echo "govulncheck is not installed"

gosec:
	cd $(GO_SERVICE) && command -v gosec >/dev/null 2>&1 && gosec ./... || echo "gosec is not installed"

audit:
	cd apps/web && $(PNPM) audit --audit-level high

sbom:
	command -v syft >/dev/null 2>&1 && syft . -o spdx-json=sbom.spdx.json || echo "syft is not installed"

license-check:
	command -v go-licenses >/dev/null 2>&1 && cd $(GO_SERVICE) && go-licenses check ./... || echo "go-licenses is not installed"

secret-scan:
	command -v gitleaks >/dev/null 2>&1 && gitleaks detect --no-git || echo "gitleaks is not installed"

container-scan:
	command -v trivy >/dev/null 2>&1 && trivy fs . || echo "trivy is not installed"

security-checks: govulncheck gosec audit secret-scan sbom license-check container-scan
