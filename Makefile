.PHONY: dev build test test-fast test-unit test-api test-e2e docker help check-air

BINARY_NAME := server
FRONTEND_DIR := frontend
CMD_PATH := ./cmd/server

## dev: Start Go (Air hot-reload) + React (Vite HMR) dev servers
dev: check-air
	npx --yes concurrently \
	  --names "API,UI" \
	  --prefix-colors "cyan,magenta" \
	  "air" \
	  "npm run dev --prefix $(FRONTEND_DIR)"

## build: Build React frontend then compile Go binary
build:
	@echo "-> Building frontend..."
	npm run build --prefix $(FRONTEND_DIR)
	@echo "-> Copying frontend build for embed..."
	rm -rf cmd/server/static/*
	cp -r frontend/dist/* cmd/server/static/
	@echo "-> Building Go binary..."
	mkdir -p bin
	go build -o bin/$(BINARY_NAME) $(CMD_PATH)
	@echo "Binary at bin/$(BINARY_NAME)"

## test: Run full test suite (unit + API integration + E2E)
test: test-unit test-api test-e2e

## test-fast: Run only Go integration tests (primary AI agent validation)
test-fast: test-api

## test-unit: Run Go unit tests
test-unit:
	go test ./internal/...

## test-api: Run Go API integration tests against in-memory SQLite
test-api:
	go test ./internal/handler/...

## test-e2e: Run Playwright E2E tests
test-e2e:
	npx playwright test

## docker: Build production Docker image
docker: build
	docker build -t inventory-management:latest .

## help: List all targets
help:
	@grep -E "^## [a-z]" Makefile | sed "s/## /  /" | column -t -s ":"

## check-air: Verify Air is installed
check-air:
	@which air > /dev/null 2>&1 || \
	  (echo "ERROR: air not found. Install with:" && \
	   echo "  go install github.com/air-verse/air@latest" && \
	   echo "Then ensure Go bin directory is on PATH." && exit 1)
