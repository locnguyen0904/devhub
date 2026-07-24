# Load .env if present (git-ignored). Real secrets live here; the ?= defaults
# below cover CI, where no .env exists. .env values win because -include runs first.
-include .env

DATABASE_URL         ?= postgres://devhub:devhub@localhost:5432/devhub?sslmode=disable
REDIS_URL            ?= redis://localhost:6379/0
# Dev/CI placeholders only. Production must set real values; these let `make
# openapi` build the spec in CI without real credentials (it never uses them).
JWT_SECRET           ?= dev-insecure-secret-do-not-use-in-production
GITHUB_CLIENT_ID     ?= dev-client-id
GITHUB_CLIENT_SECRET ?= dev-client-secret

# The golang-migrate CLI loads database drivers behind build tags, and `go tool`
# cannot pass tags — so run it through `go run -tags`. The version still comes
# from go.mod, so dev machines and CI use the same one.
MIGRATE := go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate

export DATABASE_URL REDIS_URL JWT_SECRET GITHUB_CLIENT_ID GITHUB_CLIENT_SECRET
export APP_ENV PORT

.PHONY: help dev dev-down dev-logs migrate-up migrate-down migrate-new sqlc openapi api test lint fe-install fe-dev fe-lint fe-typecheck fe-contrast verify

help: ## List available targets
	@grep -hE '^[a-z-]+:.*?## ' $(MAKEFILE_LIST) | sort | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

# ---- local infrastructure ----

dev: ## Start Postgres, Redis, MinIO and wait until healthy
	docker compose -f deploy/docker-compose.yml up -d --wait

dev-down: ## Stop local infrastructure (keep data)
	docker compose -f deploy/docker-compose.yml down

dev-logs: ## Tail infrastructure logs
	docker compose -f deploy/docker-compose.yml logs -f

# ---- database ----

migrate-up: ## Apply migrations
	cd backend && $(MIGRATE) -path db/migrations -database "$(DATABASE_URL)" up

migrate-down: ## Roll back one migration
	cd backend && $(MIGRATE) -path db/migrations -database "$(DATABASE_URL)" down 1

migrate-new: ## Create a new migration pair: make migrate-new name=add_something
	cd backend && $(MIGRATE) create -ext sql -dir db/migrations -seq $(name)

sqlc: ## Generate Go code from db/queries
	cd backend && go tool sqlc generate

# ---- backend ----

api: ## Run the API on :8080
	cd backend && go run ./cmd/api

openapi: ## Generate openapi.yaml then the TypeScript types
	cd backend && go run ./cmd/api openapi > ../docs/openapi.yaml
	cd frontend && pnpm exec openapi-typescript ../docs/openapi.yaml -o src/shared/types/api.ts

test: ## Run all Go tests
	cd backend && go test ./...

lint: ## golangci-lint
	cd backend && go tool golangci-lint run

# ---- frontend ----

fe-install: ## Install frontend dependencies
	cd frontend && pnpm install

fe-dev: ## Run the frontend on :5173
	cd frontend && pnpm dev

fe-typecheck: ## tsc --noEmit
	cd frontend && pnpm typecheck

fe-lint: ## eslint
	cd frontend && pnpm lint

fe-contrast: ## Check WCAG contrast of design tokens
	cd frontend && pnpm contrast

# ---- aggregate ----

verify: lint test fe-typecheck fe-lint fe-contrast ## Run every verification command from CLAUDE.md §6
