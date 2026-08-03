# InventoryManagement - Agent Instructions (`AGENTS.md`)

> **CURRENT PHASE: IMPLEMENTATION** — Foundational PRDs are being implemented in numbered order. Features are built incrementally. When a PRD references another PRD's schema/API/component and a conflict is found, update the source PRD — do NOT create a separate "fix" or migration.

**CRITICAL:** Never start changing the code without my confirmation, unless it is a trivial change. This is to prevent you from making incorrect changes to the codebase.

## 🚀 Workspace & Project Overview

- **Project Name:** InventoryManagement
- **Phase:** Implementation — foundational PRDs are being implemented. PRDs 1–14 are now implemented.
- **PRD Directory:** `prd/` (contains all feature specs: `prd/prd-[feature-name].md`)
- **Custom Skills Directory:** `.agents/skills/`

## 📋 Core Workflows & Guidelines

### 1. Code Quality & Architecture
- Maintain clean, modular code with clear separation of concerns.
- Always include explicit input validation and error handling for failure modes.
- Avoid dummy fallbacks or silent error swallowing—surface errors cleanly.

### 2. Verification & Testing
- Never declare success without running build/test commands to verify changes.
- Ensure all type checks and test suites pass cleanly before completing a task.

---

## 📚 PRD Backlog & Status

> All PRDs live in `prd/`. Use `/prd` skill to create them. Implement in the order listed — foundational PRDs (1–4) must be done before feature PRDs (5–13).

| # | PRD File | Topic | Status |
|---|---|---|---|
| 0 | `prd-overarching-architecture.md` | High-level architecture, tech stack, data model, testing strategy | ✅ Done |
| 1 | `prd-project-setup.md` | Repo structure, Go module, Vite init, Makefile, README | ✅ Implemented |
| 2 | `prd-database-schema.md` | Full SQLite schema, migration system, WAL mode, startup runner | ✅ Implemented |
| 3 | `prd-backend-architecture.md` | Go project layout, router, middleware, error handling, config, embed | ✅ Implemented |
| 4 | `prd-frontend-architecture.md` | React/Vite/TS scaffold, TanStack Query, routing, CSS design system, nav | ✅ Implemented |
| — | `prd-visual-design.md` | Complete visual design system — colors, typography, spacing, key page layouts, component states | ✅ Done |
| 5 | `prd-locations.md` | Locations CRUD — API + UI, unified browse tree with stacks, location detail showing stacks, deletion guard | ✅ Implemented |
| 6 | `prd-tags.md` | Tags CRUD — API + UI, deletion guard | ✅ Implemented |
| 7 | `prd-item-definitions.md` | Definitions CRUD — API + UI, field schema, inheritance, tags | ✅ Implemented |
| 8 | `prd-item-instances.md` | Instances CRUD — API + UI, smart move/split logic, breadcrumb | ✅ Implemented |
| 9 | `prd-testing.md` | Full test plan — flows, seed data, Go integration tests, Playwright E2E | ✅ Done |
| 10 | `prd-docker-deployment.md` | Multi-stage Dockerfile, docker-compose, health check, NAS deploy guide | ✅ Done |
| 11 | `prd-dashboard.md` | Dashboard — totals, location breakdown, search bar, onboarding | ✅ Implemented |
| 12 | `prd-search.md` | Name-based search v1 — API + UI, extensible for filters later | ✅ Implemented |
| 13 | `prd-settings.md` | Settings page — UI + backend, theme selector, typed-struct extensibility, app renamed to Itema | ✅ Implemented |
| 14 | `prd-item-stacks.md` | Item Stacks — UI-level grouping, stack detail page, stack move/delete, browse tree + search integration | ✅ Implemented |

### Status Key
- ✅ Implemented — PRD written, approved, and fully implemented (all code + tests)
- ✅ Done — PRD written and approved
- 🔄 In Progress — PRD being written or feature being implemented
- 🔲 Planned — Not yet started
- ⏸️ Deferred — Intentionally postponed

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
