COMPOSE ?= docker compose
GO_PKGS ?= ./cmd/... ./internal/...
DEV_COMPOSE ?= $(COMPOSE) -f docker-compose.dev.yml
DEV_LOG_DIR ?= .devlogs

ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: all dev server web check-env dev-logs dev-docker dev-docker-down \
        test test-race bench accuracy lint quality \
        build up down logs sh clean \
        build-docker run-docker compose-up compose-down \
        extension release-build

all: build

dev: check-env
	$(MAKE) -j2 server web

check-env:
	@test -f .env || { echo "missing .env: run 'cp .env.example .env' first"; exit 1; }

server: check-env
	@mkdir -p $(DEV_LOG_DIR) data internal/web/dist
	@touch internal/web/dist/.gitkeep
	CGO_ENABLED=0 air 2>&1 | tee $(DEV_LOG_DIR)/server.log

web: frontend/node_modules
	@mkdir -p $(DEV_LOG_DIR)
	cd frontend && npm run dev 2>&1 | tee ../$(DEV_LOG_DIR)/web.log

frontend/node_modules:
	cd frontend && npm ci

dev-logs:
	@ls -la $(DEV_LOG_DIR) 2>/dev/null || echo "no dev logs yet; run 'make dev' first"

dev-docker: check-env
	$(DEV_COMPOSE) up --build

dev-docker-down:
	$(DEV_COMPOSE) down

test:
	CGO_ENABLED=0 go test -count=1 $(GO_PKGS)

test-race:
	go test -race -count=1 $(GO_PKGS)

bench:
	CGO_ENABLED=0 go test -bench=. -benchmem ./internal/anonymizer

accuracy:
	CGO_ENABLED=0 go run ./cmd/pii-benchmark -min-precision 0.90 -min-recall 0.90

lint:
	CGO_ENABLED=0 go vet $(GO_PKGS)
	@if [ -d frontend/node_modules ]; then cd frontend && npx tsc --noEmit; fi

quality: lint test accuracy

build:
	$(COMPOSE) build

up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f hadesgate

sh:
	$(COMPOSE) exec hadesgate sh

clean:
	$(COMPOSE) down -v

extension:
	go run ./cmd/package-extension

release-build:
	go run ./cmd/build-release

# Legacy aliases
build-docker:
	docker build -t hadesgate:latest .

run-docker:
	docker run --rm -p 8080:8080 --env-file .env --name hadesgate hadesgate:latest

compose-up: up

compose-down: down
