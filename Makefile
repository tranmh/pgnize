.PHONY: help build build-go build-web dev test test-go test-int e2e-api e2e-ui lint tidy migrate migrate-down seed up down

GO ?= go
TEST_DATABASE_URL ?= postgres://pgnize:pgnize@localhost:5432/pgnize_test?sslmode=disable

# Load .env (gitignored) into recipe environments so `make dev`/`make migrate` pick up
# AUTH_SECRET, GEMINI_API_KEY, etc. without a separate `source`. Real exported vars win.
ifneq (,$(wildcard .env))
include .env
export
endif

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-14s\033[0m %s\n",$$1,$$2}'

build: build-go build-web ## Build everything

build-go: ## Build Go binaries
	$(GO) build -o bin/api ./cmd/api

build-web: ## Build the Next.js frontend
	cd web && npm run build

dev: ## Run infra (db, minio) + api + web for local dev
	docker compose up -d db minio
	$(GO) run ./cmd/api & cd web && npm run dev

tidy: ## go mod tidy across the workspace
	cd chesskit && $(GO) mod tidy
	$(GO) mod tidy

test: test-go ## Run Go unit tests (no DB)

test-go: ## Run Go unit tests
	cd chesskit && $(GO) test ./...
	$(GO) test ./internal/... ./cmd/...

test-int: ## Run Go integration tests (needs Postgres at TEST_DATABASE_URL)
	# -p 1: all integration packages share the single TEST_DATABASE_URL and TRUNCATE it
	# between tests, so they must run serially — parallel packages race on the shared DB.
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" RUN_INTEGRATION=1 $(GO) test -tags=integration -p 1 ./internal/... -count=1

e2e-api: ## Playwright API project (no browser)
	cd e2e && npm run test:api

e2e-ui: ## Playwright UI project (chromium)
	cd e2e && npm run test:ui

lint: ## Lint Go + web
	$(GO) vet ./... ; cd chesskit && $(GO) vet ./...
	cd web && npm run lint

migrate: ## Apply DB migrations
	$(GO) run ./cmd/api -migrate-only

migrate-down: ## Roll back one migration
	$(GO) run ./cmd/api -migrate-down

# Seeding is CLI-only (no server), so AUTH_SECRET is irrelevant — supply a
# placeholder unless one is already exported (a real env value wins via ?=).
seed: AUTH_SECRET ?= seed-placeholder-secret-not-used-by-server
seed: ## Seed the demo user + 100 sample games across 3 players
	AUTH_SECRET="$(AUTH_SECRET)" $(GO) run ./cmd/api -seed
	AUTH_SECRET="$(AUTH_SECRET)" $(GO) run ./cmd/seedgames -n 100

up: ## docker compose up (dev profile)
	docker compose up -d db minio

down: ## docker compose down
	docker compose down
