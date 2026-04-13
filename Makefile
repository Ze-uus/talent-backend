BINARY_NAME  = scaloo
MAIN_PATH    = ./cmd
DC           = docker compose
GOOSE        = goose
DB_URL       = $(shell grep '^DATABASE_URL=' .env.local | cut -d '=' -f2-)
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: all setup dev ci build run run-watch clean \
        test test-algo test-jobs test-services test-coverage test-integration test-load test-load-tracking \
        lint vet tidy \
        docker-up docker-up-build docker-down docker-down-volumes docker-build docker-logs \
        migrate-up migrate-down migrate-reset migrate-status migrate-create \
        docs-update docs-serve \
        install-tools sqlc-generate help

# ─── Primary targets ──────────────────────────────────────────────────────────

# Full check: lint → test → migrate → build → spin containers
all: tidy vet lint test migrate-up build docker-up-build
	@echo "✓ Scaloo ready — http://localhost:8080/v1/docs"

# First-time setup from zero
setup: install-tools tidy vet lint migrate-up test
	@echo "✓ Setup complete. Run 'make dev' to start."

# Daily dev cycle (assumes docker already up)
dev: docker-up migrate-up run

# CI-equivalent gate — what the pipeline runs
ci: tidy vet lint test-coverage build
	@echo "✓ CI checks passed"

# ─── Application ──────────────────────────────────────────────────────────────

build:
	@echo "→ Building $(BINARY_NAME) $(VERSION)..."
	CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(VERSION)" -o bin/$(BINARY_NAME) $(MAIN_PATH)

run:
	@echo "→ Running $(BINARY_NAME)..."
	go run $(MAIN_PATH)

run-watch:
	@echo "→ Live reload (requires air)..."
	air

# ─── Tests ────────────────────────────────────────────────────────────────────

test:
	@echo "→ Unit tests..."
	go test ./... -v -count=1 -race -timeout=60s

test-algo:
	@echo "→ Algo tests..."
	go test ./tests/algo/... -v -count=1 -race

test-jobs:
	@echo "→ Job tests..."
	go test ./internal/jobs/... -v -count=1 -race

test-services:
	@echo "→ Service tests..."
	go test ./internal/src/... -v -count=1 -race

test-coverage:
	@echo "→ Coverage report..."
	go test ./... -coverprofile=coverage.out -covermode=atomic -race
	go tool cover -html=coverage.out -o coverage.html
	@echo "→ Open coverage.html"

test-integration:
	@echo "→ Integration tests (requires live DB)..."
	go test ./... -tags=integration -v -count=1 -timeout=120s

test-load:
	@echo "→ Load tests (requires k6)..."
	k6 run ./tests/load/health.js

test-load-tracking:
	@echo "→ Load tests: tracking endpoint (requires k6)..."
	k6 run ./tests/load/tracking.js

# ─── Code quality ─────────────────────────────────────────────────────────────

lint:
	@echo "→ Linting..."
	golangci-lint run ./... --timeout=5m

vet:
	@echo "→ go vet..."
	go vet ./...

tidy:
	@echo "→ Tidying modules..."
	go mod tidy
	go mod verify

# ─── Docker ───────────────────────────────────────────────────────────────────

docker-up:
	@echo "→ Starting containers..."
	$(DC) up -d

docker-up-build:
	@echo "→ Building and starting containers..."
	$(DC) up -d --build

docker-down:
	$(DC) down

docker-down-volumes:
	@echo "→ Removing containers + volumes (DESTRUCTIVE)..."
	$(DC) down -v

docker-build:
	$(DC) build

docker-logs:
	$(DC) logs -f

docker-logs-app:
	$(DC) logs -f app

# ─── Migrations ───────────────────────────────────────────────────────────────

migrate-up:
	@echo "→ Applying migrations..."
	$(GOOSE) -dir ./migrations postgres "$(DB_URL)" up

migrate-down:
	@echo "→ Rolling back last migration..."
	$(GOOSE) -dir ./migrations postgres "$(DB_URL)" down

migrate-status:
	$(GOOSE) -dir ./migrations postgres "$(DB_URL)" status

migrate-reset:
	@echo "→ Resetting all migrations (DESTRUCTIVE)..."
	$(GOOSE) -dir ./migrations postgres "$(DB_URL)" reset

migrate-create:
	@read -p "Migration name: " name; \
	$(GOOSE) -dir ./migrations create $$name sql

# ─── Docs ─────────────────────────────────────────────────────────────────────

docs-update:
	@echo "→ Dumping OpenAPI spec (server must be on :8080)..."
	curl -sf http://localhost:8080/v1/openapi.json | jq . > docs/openapi.json
	curl -sf http://localhost:8080/v1/openapi.yaml > docs/openapi.yaml
	@echo "→ Spec written to docs/"

docs-serve:
	@echo "→ Docs: http://localhost:8080/v1/docs"
	@echo "→ Spec: http://localhost:8080/v1/openapi.json"

# ─── Utilities ────────────────────────────────────────────────────────────────

install-tools:
	@echo "→ Installing dev tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/cosmtrek/air@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

sqlc-generate:
	@echo "→ Generating sqlc queries..."
	sqlc generate

clean:
	rm -rf bin/ coverage.out coverage.html

# ─── Help ─────────────────────────────────────────────────────────────────────

help:
	@echo ""
	@echo "Scaloo — make targets"
	@echo ""
	@echo "  Primary"
	@echo "    make all              lint + test + migrate + build + docker (full cycle)"
	@echo "    make setup            first-time setup from clone"
	@echo "    make dev              docker-up + migrate + run (daily dev)"
	@echo "    make ci               lint + test-coverage + build (pipeline gate)"
	@echo ""
	@echo "  Application"
	@echo "    make run              run server"
	@echo "    make run-watch        run with live reload (air)"
	@echo "    make build            build binary → bin/"
	@echo ""
	@echo "  Tests"
	@echo "    make test             all unit tests"
	@echo "    make test-algo        algo layer only"
	@echo "    make test-jobs        jobs layer only"
	@echo "    make test-services    service layer only"
	@echo "    make test-coverage    coverage HTML report"
	@echo "    make test-integration integration tests (needs live DB)"
	@echo "    make test-load        k6 health load test"
	@echo "    make test-load-tracking  k6 tracking endpoint load test"
	@echo ""
	@echo "  Code quality"
	@echo "    make lint             golangci-lint"
	@echo "    make vet              go vet"
	@echo "    make tidy             go mod tidy + verify"
	@echo ""
	@echo "  Docker"
	@echo "    make docker-up        start containers"
	@echo "    make docker-down      stop containers"
	@echo "    make docker-build     rebuild images"
	@echo "    make docker-logs      tail all logs"
	@echo ""
	@echo "  Migrations"
	@echo "    make migrate-up       apply pending migrations"
	@echo "    make migrate-down     roll back last migration"
	@echo "    make migrate-status   show migration state"
	@echo "    make migrate-create   scaffold new migration file"
	@echo ""
	@echo "  Docs"
	@echo "    make docs-update      dump live OpenAPI spec to docs/"
	@echo ""