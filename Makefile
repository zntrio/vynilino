# ── Vynilino ─────────────────────────────────────────────────────────────────
#
# Usage:  make [target]
#
# Run `make help` to list all available targets.
# ─────────────────────────────────────────────────────────────────────────────

.DEFAULT_GOAL := help
.PHONY: help build run test lint clean ui-install ui-build ui-dev generate-graphql generate-sql release

# ── Colors / helpers ─────────────────────────────────────────────────────────
BOLD   := $(shell tput bold 2>/dev/null)
RESET  := $(shell tput sgr0 2>/dev/null)
CYAN   := $(shell tput setaf 6 2>/dev/null)
GREEN  := $(shell tput setaf 2 2>/dev/null)
YELLOW := $(shell tput setaf 3 2>/dev/null)

BIN       := bin/vynilino
GO_FILES  := $(shell find . -name '*.go' -not -path './internal/adapter/storage/sqlite/sqlcdb/*' -not -path './internal/adapter/graphql/graph/generated.go' -not -path './internal/adapter/graphql/graph/models_gen.go')

## help: Show this help message
help:
	@echo ""
	@echo "$(BOLD)Vynilino$(RESET) — self-hosted vinyl record manager"
	@echo ""
	@echo "$(BOLD)Usage:$(RESET)  make $(CYAN)<target>$(RESET)"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed -E 's/^## //' | awk -F: '{ printf "  $(CYAN)%-22s$(RESET) %s\n", $$1, $$2 }'
	@echo ""

# ── Build ────────────────────────────────────────────────────────────────────

## build: Build the full binary (UI + Go)
build: ui-build
	@echo "$(GREEN)▸ Building Go binary…$(RESET)"
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BIN) ./cmd/vynilino
	@echo "$(GREEN)✓ $(BIN)$(RESET)"

## run: Start the server locally (development mode)
run: build
	VYNILINO_ENV=development VYNILINO_TOKEN_KEY=$$(openssl rand -hex 32) ./$(BIN) serve

# ── UI ───────────────────────────────────────────────────────────────────────

## ui-install: Install frontend dependencies
ui-install:
	@echo "$(GREEN)▸ Installing UI dependencies…$(RESET)"
	cd ui && npm ci

## ui-build: Build frontend assets into web/dist/
ui-build: ui-install
	@echo "$(GREEN)▸ Building UI…$(RESET)"
	cd ui && npm run build

## ui-dev: Start Vite dev server on port 5173
ui-dev:
	cd ui && npm run dev

# ── Quality ──────────────────────────────────────────────────────────────────

## test: Run all Go tests
test:
	@echo "$(GREEN)▸ Running tests…$(RESET)"
	go test ./...

## lint: Run go vet + golangci-lint
lint:
	@echo "$(GREEN)▸ Running linters…$(RESET)"
	go vet ./...
	golangci-lint run ./...

## check: Run lint + tests (CI shortcut)
check: lint test

# ── Code generation ──────────────────────────────────────────────────────────

## generate-graphql: Regenerate gqlgen GraphQL code
generate-graphql:
	@echo "$(GREEN)▸ Generating GraphQL code…$(RESET)"
	go run github.com/99designs/gqlgen generate

## generate-sql: Regenerate sqlc database queries
generate-sql:
	@echo "$(GREEN)▸ Generating SQL code…$(RESET)"
	sqlc generate

## generate: Run all code generators
generate: generate-graphql generate-sql

# ── Release ──────────────────────────────────────────────────────────────────

VERSION ?=

## release: Tag and push a release (VERSION=x.y.z required)
release: check
	@if [ -z "$(VERSION)" ]; then \
		echo "$(YELLOW)Usage:$(RESET) make release VERSION=x.y.z"; \
		exit 1; \
	fi
	@if ! echo "$(VERSION)" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$'; then \
		echo "$(YELLOW)Error:$(RESET) VERSION must be semver (e.g. 1.2.3), got '$(VERSION)'"; \
		exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "$(YELLOW)Error:$(RESET) working tree is dirty — commit or stash changes first"; \
		exit 1; \
	fi
	@echo "$(GREEN)▸ Tagging v$(VERSION)…$(RESET)"
	git tag -S -a "v$(VERSION)" -m "Release v$(VERSION)"
	@echo "$(GREEN)▸ Pushing tag to origin…$(RESET)"
	git push origin "v$(VERSION)"
	@echo "$(GREEN)✓ Release v$(VERSION) triggered — check GitHub Actions for progress$(RESET)"

# ── Cleanup ──────────────────────────────────────────────────────────────────

## clean: Remove build artifacts and node_modules
clean:
	@echo "$(YELLOW)▸ Cleaning…$(RESET)"
	rm -rf bin/ ui/dist/ ui/node_modules/
	@echo "$(GREEN)✓ Clean$(RESET)"
