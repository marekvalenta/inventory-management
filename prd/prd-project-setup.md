# PRD: Project Setup — InventoryManagement

> **Status:** Draft v1.0
> **Last Updated:** 2026-07-31
> **Scope:** One-time foundational scaffold. Establishes the repo structure, Go module, frontend scaffold, developer tooling, and AGENTS.md build/run documentation. All subsequent PRDs depend on this being completed first.
> **Backlog Position:** #1 — Must be completed before any other PRD.

---

## Cross-PRD Consistency Report

### PRDs Reviewed
- `prd-overarching-architecture.md` — Top-level tech stack, data model, API conventions, Makefile targets, testing strategy, deployment architecture

### Conflicts & Resolutions

| # | Conflict | PRDs Involved | Resolution |
|---|---|---|---|
| 1 | Overarching PRD references `go test ./api/...` (implying a top-level `api/` directory), but layout Option B places handlers under `internal/handler/` | prd-overarching-architecture.md | Use Option B (`internal/` only). Update test commands to `go test ./internal/handler/...`. The overarching PRD's `./api/...` reference was a placeholder. This PRD supersedes it for test path conventions. |
| 2 | Overarching PRD shows `make dev` using `npx concurrently` with `go run ./cmd/server/...`. With Air chosen, Go is started via `air` not `go run`. | prd-overarching-architecture.md | Replace `go run ./cmd/server/...` with `air` in the `make dev` target. Air invokes `go build + run` internally; semantics are identical. |

### Confirmed Alignments
- Module path `github.com/marekvalenta/inventory-management` aligns with all future `import` paths
- `cmd/server/main.go` entrypoint matches overarching PRD's `go run ./cmd/server/...` reference
- `frontend/` directory name matches overarching PRD's `npm run dev --prefix frontend`
- Makefile target names (`make dev`, `make test`, `make test-fast`, `make build`, `make docker`, `make help`) are identical to overarching PRD §11.1
- Environment variables (`APP_PORT`, `APP_NAME`, `DATA_DIR`, `LOG_LEVEL`) match overarching PRD §7.3
- `modernc.org/sqlite` (CGO-free) driver confirmed — consistent with overarching PRD §3.1

---

## 1. Overview & Problem Statement

This PRD specifies the one-time scaffolding of the entire project repository. It is not a feature — it is the foundation every subsequent feature PRD builds on. After this PRD is implemented, a developer (or AI agent) should be able to clone the repo, run a single command, and have a working local development environment with both the Go API and React frontend running with hot-reload.

### Core Deliverables
1. Go module initialized with correct module path
2. Standard Go directory structure (`cmd/`, `internal/`, `migrations/`, `e2e/`)
3. Vite/React/TypeScript frontend scaffold in `frontend/` with core dependencies pre-installed
4. POSIX Makefile with all standard targets
5. `make.ps1` PowerShell wrapper for running Makefile targets on Windows
6. Air hot-reload configuration (`.air.toml`)
7. `.gitignore`, `.editorconfig`, `.gitattributes`
8. `README.md` (brief, user-facing)
9. `AGENTS.md` updated with a **Build & Run Reference** section for AI agents

---

## 2. Goals & Measurable Success Metrics

| Goal | Metric |
|---|---|
| Working dev environment | `.\make.ps1 dev` starts both servers with no errors |
| Frontend compiles | `npm run build` in `frontend/` exits 0 |
| Go compiles | `go build ./cmd/server/` exits 0 |
| Air hot-reload works | Saving a `.go` file triggers automatic backend restart within 3 seconds |
| Frontend proxy works | `http://localhost:5173/api/v1/health` proxies to Go backend without CORS errors |
| Type checks pass | `npx tsc --noEmit` in `frontend/` exits 0 |
| Windows usability | `.\make.ps1 help` runs successfully in PowerShell without extra tools beyond Git for Windows |

---

## 3. Critical Risks & Technical Assumptions

| Risk | Mitigation |
|---|---|
| Air not installed on developer machine | `make dev` runs `check-air` first and prints clear install instruction if missing |
| `concurrently` not found | `npx concurrently` auto-installs via npx on first run. No pre-install needed. |
| Git Bash not on PATH when PowerShell runs `make.ps1` | `make.ps1` probes known Git Bash paths and prints a clear error with download URL if not found |
| Go not installed | Developer must install Go 1.22+ manually. Not auto-installed. |
| Node.js not installed | Node 20+ must be pre-installed. Not auto-installed. |
| `npm install` not run in `frontend/` | Agent must run `npm install --prefix frontend` after scaffold. `make dev` does NOT auto-run npm install. |
| `go mod tidy` not run | `go.sum` will be missing. Agent must run `go mod tidy` after adding dependencies. |
| CRLF line endings in Makefile on Windows | `make` will fail with cryptic errors. `.gitattributes` and `.editorconfig` enforce LF. Implementing agent must verify Makefile uses LF. |

---

## 4. User Stories & Acceptance Criteria

### US-001: Developer can start the full dev environment on Windows
**Description:** As a developer on Windows, I want to start both the Go backend and React frontend with a single command so that I can begin development immediately.

**Acceptance Criteria:**
- [ ] `.\make.ps1 dev` runs successfully in PowerShell 5.1+
- [ ] `make dev` runs successfully in Git Bash
- [ ] Go backend starts on `http://localhost:8080`
- [ ] React Vite dev server starts on `http://localhost:5173`
- [ ] Saving a `.go` file causes the Go server to restart automatically within 3 seconds
- [ ] Saving a `.tsx` or `.ts` file causes the browser to HMR without full page reload
- [ ] `Ctrl+C` kills both processes cleanly

### US-002: Developer can build the production binary
**Description:** As a developer, I want to build the production Go binary (with embedded frontend) so that it can be packaged into Docker.

**Acceptance Criteria:**
- [ ] `.\make.ps1 build` succeeds
- [ ] React frontend is built first (`npm run build --prefix frontend`)
- [ ] Go binary is compiled and placed at `./bin/server`
- [ ] `go build ./cmd/server/` exits 0

### US-003: AI agent can find and execute build/test commands reliably
**Description:** As an AI agent, I want a clear, authoritative reference for all build and test commands in `AGENTS.md`.

**Acceptance Criteria:**
- [ ] `AGENTS.md` contains a **Build & Run Reference** section
- [ ] The section documents all Makefile targets with expected exit codes
- [ ] Explicitly states: AI agents must run `make test-fast` after any Go change and verify exit code 0

### US-004: Developer understands prerequisites and startup
**Description:** As a developer, I want the README to clearly state prerequisites and startup commands.

**Acceptance Criteria:**
- [ ] `README.md` lists prerequisites with minimum versions
- [ ] `README.md` shows both Windows and Git Bash startup commands
- [ ] `README.md` contains Docker deployment section
- [ ] `README.md` is under 100 lines

---

## 5. Functional & Technical Requirements

### 5.1 Repository Directory Structure

```
inventory-management/
├── cmd/
│   └── server/
│       └── main.go              ← Go entrypoint
├── internal/
│   ├── config/                  ← env var config loading
│   ├── db/                      ← DB connection + migration runner
│   ├── handler/                 ← HTTP handlers
│   ├── router/                  ← chi router setup
│   └── service/                 ← business logic services
├── frontend/
│   ├── src/
│   │   ├── main.tsx
│   │   ├── App.tsx              ← placeholder
│   │   └── vite-env.d.ts
│   ├── public/
│   ├── package.json
│   ├── tsconfig.json
│   ├── tsconfig.node.json
│   └── vite.config.ts           ← includes /api proxy config
├── migrations/                  ← SQL migration files (empty in this PRD)
├── e2e/                         ← Playwright tests (empty in this PRD)
├── tmp/                         ← Air build output (git-ignored)
├── bin/                         ← Production binary output (git-ignored)
├── .air.toml
├── .editorconfig
├── .gitattributes
├── .gitignore
├── go.mod
├── go.sum
├── Makefile
├── make.ps1                     ← Windows PowerShell wrapper
├── README.md
└── AGENTS.md
```

**FR-1:** All empty placeholder directories must contain a `.gitkeep` file.

**FR-2:** All Go placeholder `*.go` files must be valid compilable packages with the correct `package` name (e.g., `package db`, `package service`) and a comment `// TODO: Implemented in PRD #X`. Do not leave them completely empty.

### 5.2 Go Module

**FR-3:** `go.mod` content:
```
module github.com/marekvalenta/inventory-management

go 1.22
```

**FR-4:** Initial Go dependencies (run `go get` then `go mod tidy`):
```
github.com/go-chi/chi/v5       ← HTTP router
modernc.org/sqlite             ← CGO-free SQLite driver
```

**FR-5:** `cmd/server/main.go` must be a minimal valid program that:
- Reads `APP_PORT` from environment (default `"8080"`)
- Starts HTTP server on that port
- Logs `"inventory-management server starting on :PORT"` to stdout
- Returns HTTP 200 with `{"status":"ok"}` at `GET /api/v1/health`

### 5.3 Frontend Scaffold

**FR-6:** Initialize Vite project (using `npx -y` to ensure it runs non-interactively without prompting for confirmation):
```bash
npx -y create-vite@latest frontend --template react-ts
```

**FR-7:** Install core dependencies immediately after scaffold:
```bash
npm install --prefix frontend react-router-dom @tanstack/react-query
```

**FR-8:** `frontend/vite.config.ts` must include `/api` proxy:
```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
```

**FR-9:** `frontend/src/App.tsx` must be a placeholder rendering `<h1>InventoryManagement — Setup Complete</h1>`.

**TR-1:** TypeScript strict mode must be enabled in `tsconfig.json` (`"strict": true`).

### 5.4 Makefile

**FR-10:** Makefile targets:

```makefile
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
```

**TR-2:** Makefile MUST use tabs (not spaces) for recipe indentation.

### 5.5 PowerShell Wrapper (`make.ps1`)

**FR-11:** `make.ps1` content:

```powershell
<#
.SYNOPSIS
    Windows wrapper for Makefile targets via Git Bash.
.EXAMPLE
    .\make.ps1 dev
    .\make.ps1 build
    .\make.ps1 help
#>
param(
    [Parameter(Position = 0)]
    [string]$Target = "help",

    [Parameter(ValueFromRemainingArguments)]
    [string[]]$ExtraArgs
)

$gitBashPaths = @(
    "$env:ProgramFiles\Git\bin\bash.exe",
    "${env:ProgramFiles(x86)}\Git\bin\bash.exe",
    "$env:LocalAppData\Programs\Git\bin\bash.exe"
)

$gitBash = $gitBashPaths | Where-Object { Test-Path $_ } | Select-Object -First 1

if (-not $gitBash) {
    Write-Error @"
Git Bash not found. Install Git for Windows:
  https://git-scm.com/download/win
Then re-run: .\make.ps1 $Target
"@
    exit 1
}

$makeArgs = (@($Target) + $ExtraArgs) -join " "
Write-Host "Running: make $makeArgs" -ForegroundColor Cyan
& $gitBash -c "make $makeArgs"
exit $LASTEXITCODE
```

**FR-12:** `make.ps1` must propagate the exit code from `make`.

### 5.6 Air Configuration (`.air.toml`)

**FR-13:**
```toml
root = "."
tmp_dir = "tmp"

[build]
  cmd = "go build -o ./tmp/server ./cmd/server"
  bin = "./tmp/server"
  delay = 500
  exclude_dir = ["assets", "tmp", "bin", "vendor", "frontend", "e2e", "migrations"]
  include_ext = ["go"]
  exclude_regex = ["_test\\.go"]

[log]
  time = false

[color]
  main = "magenta"
  watcher = "cyan"
  build = "yellow"
  runner = "green"

[misc]
  clean_on_exit = true
```

### 5.7 `.gitignore`

**FR-14:**
```
# Go
/bin/
/tmp/
*.exe
*.test
*.out
go.work
go.work.sum

# Node / Frontend
frontend/node_modules/
frontend/dist/
frontend/.vite/

# OS
.DS_Store
Thumbs.db
desktop.ini

# SQLite (local dev DB)
*.db
*.db-shm
*.db-wal
```

### 5.8 `.editorconfig`

**FR-15:**
```ini
root = true

[*]
charset = utf-8
end_of_line = lf
insert_final_newline = true
trim_trailing_whitespace = true

[*.go]
indent_style = tab
indent_size = 4

[*.{ts,tsx,js,jsx,json,css,html,md,yaml,yml,toml}]
indent_style = space
indent_size = 2

[Makefile]
indent_style = tab
indent_size = 4
```

### 5.9 `.gitattributes`

**FR-16:**
```
* text=auto
*.go text eol=lf
Makefile text eol=lf
*.sh text eol=lf
*.ps1 text eol=crlf
```

### 5.10 README.md

**FR-17:** See Section 8.1 for full content.

### 5.11 AGENTS.md — Build & Run Reference

**FR-18:** Append the Build & Run Reference section to `AGENTS.md`. See Section 8.2 for full content.

### 5.12 AI Agent Execution Plan

To execute this PRD flawlessly, follow these exact steps in order:
1. **Initialize Go Module:** `go mod init github.com/marekvalenta/inventory-management`
2. **Install Go Dependencies:** `go get github.com/go-chi/chi/v5` and `go get modernc.org/sqlite`, then run `go mod tidy`.
3. **Create Directories:** Use mkdir (or create files within them) for `cmd/server`, `internal/config`, `internal/db`, `internal/handler`, `internal/router`, `internal/service`, `migrations`, `e2e`.
4. **Create Go Files & Gitkeeps:** Write `main.go` and the `internal/` package placeholders (making sure to include `package config`, `package db`, etc., so they are valid). Place `.gitkeep` files in `migrations` and `e2e`.
5. **Scaffold Frontend:** Run `npx -y create-vite@latest frontend --template react-ts`
6. **Install Frontend Dependencies:** Run `npm install --prefix frontend react-router-dom @tanstack/react-query`
7. **Update Frontend Files:** Write `frontend/vite.config.ts`, update `frontend/tsconfig.json` (add `"strict": true`), and write the `frontend/src/App.tsx` placeholder.
8. **Create Root Config Files:** Write `.air.toml`, `.gitignore`, `.editorconfig`, `.gitattributes`, `README.md`, and `Makefile`. *Important: when writing the `Makefile`, you MUST preserve tabs. Do not convert them to spaces.*
9. **Create Windows Script:** Write `make.ps1` in the project root.
10. **Verify & Complete:** If not already updated, ensure `AGENTS.md` has the new Build & Run Reference appended. Test `.\make.ps1 dev` (or `make dev` in Git Bash) to ensure servers start.

---

## 6. Edge Cases & Failure Modes

| Scenario | Expected Behaviour |
|---|---|
| `air` not installed when running `make dev` | `check-air` target prints install instruction and exits code 1 |
| Git Bash not found when running `make.ps1` | Script prints error with download URL and exits code 1 |
| Port 8080 already in use | Go server fails to start; Air shows port error in terminal. No auto-retry. |
| Port 5173 already in use | Vite fails with clear error. No auto-retry. |
| `go mod tidy` not run after initial `go.mod` | `go build` fails with "missing go.sum entry". Agent must run `go mod tidy`. |
| `npm install` not run in `frontend/` | `npm run dev` fails. Agent must run `npm install --prefix frontend`. |
| Makefile saved with CRLF on Windows | `make` fails with cryptic errors. `.gitattributes` enforces LF for Makefile. |
| Developer on macOS or Linux | `make dev` works natively. `make.ps1` is Windows-only and ignored. |
| WSL2 with project on Windows filesystem (`/mnt/c/`) | Linux-side `npm` may not see file changes made through Linux tools due to 9p cache. Production build MUST run via Windows PowerShell. Vite dev server (`npm run dev`) works correctly for HMR when accessed from Windows browser. |

---

## 7. Non-Goals & Scope Boundaries

This PRD does NOT include:
- Database schema or migration files (PRD #2)
- Real Go handler or router implementation (PRD #3)
- Real React pages, components, or routing (PRD #4)
- `Dockerfile` or `docker-compose.yml` (PRD #10)
- CI/CD configuration (explicitly deferred)
- Go test files beyond placeholder stubs (PRD #9)
- Playwright configuration (PRD #9)
- VSCode workspace settings (deferred)
- SQLite connection code (PRD #2)
- `go:embed` directives for frontend (PRD #3)

---

## 8. Specified File Contents

### 8.1 README.md (full content)

```markdown
# InventoryManagement

A self-hosted inventory management app. Track physical items across hierarchical locations.
Built with Go + SQLite backend and a React frontend. Runs in a single Docker container.

---

## Prerequisites

| Tool | Min Version | Install |
|---|---|---|
| Go | 1.22+ | https://go.dev/dl/ |
| Node.js | 20+ | https://nodejs.org/ |
| Git for Windows | any | https://git-scm.com/download/win |
| Air (Go hot-reload) | latest | `go install github.com/air-verse/air@latest` |

---

## Quick Start (Local Development)

**First-time setup** (run once after cloning):

```bash
# Install frontend dependencies
npm install --prefix frontend
```

**Start dev servers:**

```powershell
# Windows (PowerShell)
.\make.ps1 dev
```

```bash
# Git Bash / macOS / Linux
make dev
```

- Go API: http://localhost:8080
- React UI: http://localhost:5173

---

## Available Commands

| Windows (PowerShell) | Git Bash | Description |
|---|---|---|
| `.\make.ps1 dev` | `make dev` | Start dev servers with hot-reload |
| `.\make.ps1 build` | `make build` | Build production binary |
| `.\make.ps1 test-fast` | `make test-fast` | Run Go integration tests (~30s) |
| `.\make.ps1 test` | `make test` | Run full test suite |
| `.\make.ps1 docker` | `make docker` | Build Docker image |
| `.\make.ps1 help` | `make help` | List all targets |

---

## Docker Deployment

```yaml
services:
  inventory:
    image: inventory-management:latest
    container_name: inventory
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    environment:
      - APP_PORT=8080
      - APP_NAME=My Inventory
    restart: unless-stopped
```

```bash
docker compose up -d
```

Access at `http://your-nas-ip:8080`
```

### 8.2 AGENTS.md — Build & Run Reference (append to existing file)

```markdown
---

## 🔨 Build & Run Reference

> **Authoritative command reference for AI agents.** Use these commands exactly as specified. Never guess — check this section first.

### Local Dev URLs

| Service | URL | Started by |
|---|---|---|
| Go API backend | http://localhost:8080 | `air` via `make dev` |
| React frontend | http://localhost:5173 | `npm run dev` via `make dev` |
| API via frontend proxy | http://localhost:5173/api/... | Auto-proxied by Vite to :8080 |

### First-Time Setup (run once after cloning)

```bash
# Verify Go 1.22+
go version

# Verify Node 20+
node --version

# Install Air (Go hot-reload)
go install github.com/air-verse/air@latest

# Install frontend dependencies
npm install --prefix frontend
```

### Makefile Targets

| Target | Git Bash | PowerShell | Description | Exit 0 = |
|---|---|---|---|---|
| `dev` | `make dev` | `.\make.ps1 dev` | Start Go (Air) + React (Vite) dev servers | Both servers running |
| `build` | `make build` | `.\make.ps1 build` | Build React then Go binary at `bin/server` | Binary compiles |
| `test` | `make test` | `.\make.ps1 test` | Full suite: unit + API integration + E2E | All tests pass |
| `test-fast` | `make test-fast` | `.\make.ps1 test-fast` | Go API integration tests only (~30s) | API tests pass |
| `test-unit` | `make test-unit` | `.\make.ps1 test-unit` | Go unit tests (pure functions) | Unit tests pass |
| `test-api` | `make test-api` | `.\make.ps1 test-api` | HTTP handler tests vs in-memory SQLite | Handler tests pass |
| `test-e2e` | `make test-e2e` | `.\make.ps1 test-e2e` | Playwright browser tests | E2E tests pass |
| `docker` | `make docker` | `.\make.ps1 docker` | Build Docker image | Image builds |
| `help` | `make help` | `.\make.ps1 help` | List all targets | (informational) |

### AI Agent Test Workflow

**CRITICAL:** After any Go source code change, you MUST run:

```bash
make test-fast
```

- Exit code **0** → proceed
- Exit code **non-zero** → read output, fix, re-run. **Never report success before this passes.**

After any TypeScript/React change:

```bash
npx tsc --noEmit --project frontend/tsconfig.json
```

### Go Module Path

All Go imports use:
```
github.com/marekvalenta/inventory-management
```

Example: `import "github.com/marekvalenta/inventory-management/internal/service"`

### Directory Reference

| Directory | Contents |
|---|---|
| `cmd/server/` | Go entrypoint (`main.go`) |
| `internal/config/` | Environment variable loading |
| `internal/db/` | SQLite connection + migration runner |
| `internal/handler/` | HTTP request handlers |
| `internal/router/` | chi router setup |
| `internal/service/` | Business logic (InstanceMoveService etc.) |
| `frontend/` | React/Vite/TypeScript app |
| `migrations/` | Versioned SQL migration files |
| `e2e/` | Playwright end-to-end tests |
| `bin/` | Production binary (git-ignored) |
| `tmp/` | Air build cache (git-ignored) |
```

---

## 9. Open Questions & Deferred Items

| # | Question | Status |
|---|---|---|
| OQ-1 | Should `make dev` auto-run `npm install` if `node_modules` missing? Adds convenience but slows every start. | Deferred — manual first-time setup documented in README |
| OQ-2 | VSCode workspace settings (`.vscode/`) | Deferred — add in a future polish pass if needed |
| OQ-3 | Should `make.ps1` support env var overrides passed as additional args? | Current implementation passes `$ExtraArgs` through — already supported |
| OQ-4 | WSL2 9p filesystem cache prevents Linux `npm` from seeing file changes on `/mnt/c/` | Resolved — for production builds, run `npm run build` via PowerShell: `powershell.exe -Command "cd C:\Users\marek\Projects\InventoryManagement\frontend; npm run build"`. For dev, Vite HMR works correctly through the Windows browser at `localhost:5173`. The Vite dev server runs in a persistent tmux session (`tmux new-session -d -s vite -c frontend 'npm run dev'`). |
