# PRD: Overarching Architecture — InventoryManagement

> **Status:** Living Document — v1.1  
> **Last Updated:** 2026-07-31  
> **Scope:** Top-level architectural decisions. All feature PRDs must reference and align with this document. Update this PRD when foundational decisions change.

---

## 1. Overview & Problem Statement

InventoryManagement is a **self-hosted, Docker-based inventory management application** designed to run on a Ugreen NAS (or any low-resource Docker host). It allows a single user to track physical items across hierarchical locations, using a definition/instance model that cleanly separates *what a thing is* from *where and how many of it exist*.

### Core Problems Solved
- "Where is X, and how many do I have?" answered quickly from any device (mobile + desktop).
- Items are tracked in a structured, hierarchical way — not just flat lists.
- Lightweight enough to run on NAS hardware with minimal resource overhead.

### Design Philosophy
> **Start lean. Design for extension.** Every v1 decision must not block future features (photos, barcode scanning, bulk import, multi-user). The codebase must be extensible without requiring rewrites.

---

## 2. Goals & Measurable Success Metrics

| Goal | Metric |
|---|---|
| Lightweight runtime | Container idle memory < 50 MB RAM |
| Fast API responses | p95 latency < 200ms on NAS hardware |
| Mobile usability | Full CRUD usable on a 375px-wide viewport |
| Single-command deployment | `docker compose up -d` brings up a working app |
| Reliable data | Zero data loss on container restart (SQLite on mounted volume) |
| Extensibility | Adding a new item field requires schema migration only, no API restructuring |

---

## 3. Tech Stack

### 3.1 Backend — Go

| Decision | Choice | Rationale |
|---|---|---|
| Language | Go | Low memory footprint, single static binary, fast startup |
| HTTP framework | `net/http` + lightweight router (e.g. `chi` or `gorilla/mux`) | No heavy framework overhead |
| Database driver | `modernc.org/sqlite` (pure Go, CGO-free) | No CGO required, simpler Docker build |
| ORM / Query | `sqlx` or raw `database/sql` with migrations via `golang-migrate` | Explicit SQL preferred for performance visibility |
| Static file serving | Go serves compiled React build from embedded `embed.FS` | Single binary, no Nginx needed |
| Config | Environment variables only | Aligns with Docker/compose idiom |

### 3.2 Frontend — React

| Decision | Choice | Rationale |
|---|---|---|
| Framework | React (Vite build toolchain) | Fast build, small bundle |
| Language | TypeScript | Type safety, better maintainability |
| Styling | Vanilla CSS + CSS variables | No heavy CSS framework, mobile-first |
| State management | React Query (TanStack Query) for server state | No Redux overhead; REST cache layer |
| Routing | React Router v6 | Standard SPA routing |
| Bundle output | Static files embedded into Go binary via `go:embed` | Single container, no separate web server |

### 3.3 Database — SQLite

| Decision | Choice |
|---|---|
| Engine | SQLite (single file) |
| Location | `/data/inventory.db` inside container, mounted from NAS volume |
| Migrations | Versioned SQL migrations run at application startup |
| Backup | User responsibility — NAS volume backup; future: export endpoint |

### 3.4 Deployment — Docker

```
+-------------------------------------------------+
|             Single Docker Container             |
|                                                 |
|  +------------------------------------------+  |
|  |   Go Binary                              |  |
|  |   +-- REST API  (/api/v1/...)            |  |
|  |   +-- Static file server (React SPA)     |  |
|  |   +-- DB migration runner (on startup)   |  |
|  +------------------------------------------+  |
|                                                 |
|  Volume mounts:                                 |
|    /data  ->  NAS volume (SQLite DB)            |
+-------------------------------------------------+
```

**`docker-compose.yml` example (user-facing):**
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

No external services (no separate DB container, no reverse proxy required for basic use).

---

## 4. Core Data Model

This is the most critical architectural decision in the system. All feature PRDs must conform to this model or explicitly propose a migration.

### 4.1 Conceptual Model

```
WORLD
 +-- Location (definition, hierarchical)
      +-- Sub-Location
      |    +-- Sub-Sub-Location...
      +-- Item Instance (placed here)
           +-- Item Instance (contained within parent item)
```

Three primary entity types:

| Entity | Nature | Description |
|---|---|---|
| **Location** | Definition only | A physical or logical place. No instances. Locations contain Locations and Item Instances. |
| **Item Definition** | Class / Schema | Defines what a thing *is*. Holds the field schema, unit, optional parent definition, and can have multiple tags. |
| **Item Instance** | Record | A physical occurrence of an Item Definition. Has a quantity, location/parent, and instance-specific field values. |

---

### 4.2 Location Entity

```sql
CREATE TABLE locations (
    id          TEXT PRIMARY KEY,          -- UUID
    name        TEXT NOT NULL,
    description TEXT,
    parent_id   TEXT REFERENCES locations(id) ON DELETE RESTRICT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

- **Hierarchical**: Unlimited depth via self-referencing `parent_id`.
- **Definitions only**: No "instances" of a location exist. A Room is just a Room.
- **Deletion rule**: Block delete if location contains sub-locations or item instances. Prompt: *"This location contains X items and Y sub-locations. Delete all?"*

---

### 4.3 Item Definition Entity

```sql
CREATE TABLE item_definitions (
    id              TEXT PRIMARY KEY,       -- UUID
    name            TEXT NOT NULL,
    description     TEXT,
    parent_def_id   TEXT REFERENCES item_definitions(id),  -- inheritance
    unit            TEXT,                   -- e.g. "pcs", "litres", "kg"
    -- Future extensibility hooks (nullable, unused in v1):
    -- image_path   TEXT,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE definition_fields (
    id              TEXT PRIMARY KEY,
    definition_id   TEXT NOT NULL REFERENCES item_definitions(id) ON DELETE CASCADE,
    field_name      TEXT NOT NULL,
    field_type      TEXT NOT NULL,          -- "text" | "number" | "date" | "boolean"
    is_required     BOOLEAN DEFAULT FALSE,
    display_order   INTEGER DEFAULT 0
);
```

- **Tags**: Flexible labeling (e.g. "Fasteners", "Fragile"). See Section 4.5.
- **Inheritance**: A definition may inherit from a parent definition, receiving its field schema. Child definitions may add fields. Instances of a child definition carry all ancestor fields.
- **Field schema**: The set of `definition_fields` rows is the schema that all instances of this definition must satisfy.

---

### 4.4 Item Instance Entity — Smart Quantity Model

> **Key rule**: Identical items (same definition, same field values, same parent) share **one instance record** with a `quantity` field. The moment any field value diverges, they become separate instances.

```sql
CREATE TABLE item_instances (
    id                  TEXT PRIMARY KEY,   -- UUID
    definition_id       TEXT NOT NULL REFERENCES item_definitions(id) ON DELETE RESTRICT,
    quantity            INTEGER NOT NULL DEFAULT 1 CHECK (quantity > 0),
    -- Location: exactly one of these is non-null
    location_id         TEXT REFERENCES locations(id),
    parent_instance_id  TEXT REFERENCES item_instances(id),
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_single_parent CHECK (
        (location_id IS NULL) != (parent_instance_id IS NULL)
    )
);

CREATE TABLE instance_field_values (
    id          TEXT PRIMARY KEY,
    instance_id TEXT NOT NULL REFERENCES item_instances(id) ON DELETE CASCADE,
    field_id    TEXT NOT NULL REFERENCES definition_fields(id),
    value       TEXT,                       -- stored as text, parsed by field_type
    UNIQUE (instance_id, field_id)
);
```

#### Smart Move Logic (Partial Transfer)

When moving **M of N** items (M < N) to a new location/parent:
1. Reduce source instance `quantity` by M: `quantity = N - M`.
2. Check if an identical instance (same definition + same field values) exists at the destination.
   - **Yes** → increment its `quantity` by M.
   - **No** → create a new instance at destination with `quantity = M`, copying field values.

When moving **all N** items: simply update `location_id` / `parent_instance_id`.

This logic lives in a backend service layer (`InstanceMoveService`) to keep it transactional.

---

### 4.5 Tags Entity

```sql
CREATE TABLE tags (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    color       TEXT,                       -- optional UI color hint
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE definition_tags (
    definition_id TEXT NOT NULL REFERENCES item_definitions(id) ON DELETE CASCADE,
    tag_id        TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (definition_id, tag_id)
);
```

Many-to-many relationship allowing item definitions to be tagged with multiple labels.

---

### 4.6 Entity Relationship Summary

```
tags ----< definition_tags >---- item_definitions >---- item_definitions (self-ref/inheritance)
                       |
               definition_fields

item_definitions ----< item_instances >---- item_instances (item-in-item, self-ref)
                              |                      |
                         locations            instance_field_values
                       (self-ref tree)
```

---

## 5. REST API Design

### 5.1 Conventions

- **Base path**: `/api/v1/`
- **Format**: JSON request/response bodies
- **IDs**: UUIDs (string)
- **Dates**: ISO 8601 strings
- **Errors**: Structured JSON `{ "error": "...", "code": "..." }`
- **Pagination**: Cursor-based, optional on list endpoints

### 5.2 Core Endpoint Groups

| Group | Prefix | Key Operations |
|---|---|---|
| Locations | `/api/v1/locations` | CRUD, get tree, get contents |
| Item Definitions | `/api/v1/definitions` | CRUD, get with fields, get instances |
| Item Instances | `/api/v1/instances` | CRUD, move (with split logic), get by location |
| Tags | `/api/v1/tags` | CRUD |
| Search | `/api/v1/search` | v1: name search; future: full-text + filters |
| Settings | `/api/v1/settings` | Read/write UI settings (stored in SQLite) |
| Health | `/api/v1/health` | Liveness probe for Docker healthcheck |

### 5.3 API Versioning

All routes are under `/api/v1/`. Future breaking changes go to `/api/v2/`. Old version retained for at least one release cycle.

---

## 6. Frontend Architecture

### 6.1 Key Views (v1)

| View | Purpose |
|---|---|
| **Dashboard** | Total item counts, recent activity, quick search bar |
| **Location Tree** | Hierarchical browser of all locations and their contents |
| **Location Detail** | Contents of a specific location (sub-locations + instances) |
| **Item Definition List** | Browse/search all definitions, filterable by tags |
| **Item Definition Detail** | Definition info, field schema, all instances + quantities per location, total quantity |
| **Item Instance Detail** | Specific instance, field values, parent chain (breadcrumb to root location) |
| **Settings** | App name and UI preferences |

### 6.2 Mobile-First Constraints

- All primary actions (add, move, search, view) must be reachable with a single thumb on mobile.
- No hover-only interactions — all hover states must have a tap equivalent.
- Bottom navigation bar on mobile; sidebar on desktop.
- Minimum tap target: 44x44px.

### 6.3 Location Breadcrumb

Every item instance displays a **root location breadcrumb** — the chain of parents (location or item) resolved up to the nearest `Location` node:

```
Livingroom > Chest of Drawers > Screws (x5)
```

This is computed server-side via recursive CTE on the SQLite schema.

---

## 7. Deployment Architecture

### 7.1 Container Build (Multi-Stage Dockerfile)

```
Stage 1: node:alpine   --> builds React --> /app/dist
Stage 2: golang:alpine --> builds Go binary with go:embed of /app/dist
Stage 3: scratch/alpine --> minimal runtime image with the single binary
```

The result is a **single, minimal Docker image** (~20-40 MB estimated).

### 7.2 Volume Structure

```
/data/
  inventory.db        <- SQLite database
  (future) images/    <- Photo attachments (reserved path, not used in v1)
```

### 7.3 Configuration (Environment Variables)

| Variable | Default | Description |
|---|---|---|
| `APP_PORT` | `8080` | HTTP port the server listens on |
| `APP_NAME` | `Inventory` | Display name shown in the UI header |
| `DATA_DIR` | `/data` | Directory for SQLite DB and future assets |
| `LOG_LEVEL` | `info` | Logging verbosity (`debug`, `info`, `warn`, `error`) |

UI-level settings (theme, display preferences) are stored in a `settings` SQLite table and managed via the Settings page.

---

## 8. Critical Risks & Technical Assumptions

| Risk | Mitigation |
|---|---|
| SQLite write contention (future multi-user) | WAL mode enabled by default; single-user in v1 makes this a non-issue |
| Recursive tree queries on large hierarchies | Use SQLite recursive CTEs; add depth limit guard at API layer |
| Instance split logic bugs causing data loss | Wrap all move operations in a database transaction; rollback on any error |
| CGO dependency breaking Docker build | Use `modernc.org/sqlite` (pure Go, CGO-free) |
| React bundle size on low-bandwidth LAN | Code splitting by route; target < 300 KB gzipped initial bundle |
| Schema migration failures on startup | Migrations are idempotent and versioned; failed migration = container exits with clear error log |

---

## 9. Non-Goals (v1 Scope Boundaries)

The following are **explicitly out of scope** for v1 and must not be designed in ways that block them later:

- No user authentication / multi-user access (future: simple auth or reverse proxy SSO)
- No barcode / QR code scanning
- No photo / image attachments (schema hook reserved in volume path)
- No bulk CSV/JSON import-export
- No reporting, analytics, or dashboards beyond basic totals
- No notifications or alerts (e.g. low stock)
- No external service integrations
- No OpenAPI/Swagger spec (clean REST API is in scope; auto-generated docs are not)
- No offline / PWA support

---

## 10. Extensibility Commitments

Even though the above are non-goals for v1, the architecture must not block them:

| Future Feature | Architectural Hook Required Now |
|---|---|
| Photo attachments | `/data/images/` volume path reserved; `item_definitions` schema designed to accept `image_path` cleanly via migration |
| Barcode scanning | Item definition has a nullable `barcode` field slot available (unused in v1) |
| Bulk import | Import service will reuse the same `InstanceMoveService` and definition upsert logic |
| Multi-user | Auth middleware slot in Go router already stubbed (passthrough in v1) |
| Full-text search | SQLite FTS5 extension can be added without breaking the REST API contract |
| External API access | `/api/v1/` versioned from day one |

---

## 11. Developer Experience & Local Setup

The project must be runnable locally **without Docker** for fast development iteration. A `Makefile` is the task runner of choice — it is the Go community standard, familiar to AI agents, and works on Linux (NAS), macOS, and Windows (via Git Bash, WSL, or `choco install make`).

### 11.1 Makefile Targets

| Target | Command | Description |
|---|---|---|
| `make dev` | Starts Go backend + React frontend together | Primary local dev command |
| `make test` | Runs full test suite (unit + integration + E2E) | Used by developer and AI agent |
| `make test-fast` | Runs only Go integration tests (no browser) | Fast feedback loop for AI agent |
| `make build` | Compiles React then embeds into Go binary | Produces production binary |
| `make docker` | Builds Docker image | For NAS deployment |
| `make help` | Lists all targets with descriptions | Discoverability |

### 11.2 Local Dev Script (`make dev`)

Runs both servers concurrently in a single terminal using [`concurrently`](https://www.npmjs.com/package/concurrently) (npm package, dev-dependency):

```makefile
dev:
	@echo "Starting dev servers..."
	npx concurrently \
	  --names "API,UI" \
	  --prefix-colors "cyan,magenta" \
	  "go run ./cmd/server/..." \
	  "npm run dev --prefix frontend"
```

- Go backend starts on `http://localhost:8080` (API + static fallback)
- React Vite dev server starts on `http://localhost:5173` with proxy to Go for `/api/*`
- Both processes share one terminal; `Ctrl+C` kills both

### 11.3 Prerequisites (documented in README)

| Tool | Min Version | Purpose |
|---|---|---|
| Go | 1.22+ | Backend |
| Node.js | 20+ | Frontend build & dev |
| make | any | Task runner |
| Docker | 24+ | Production build only |

---

## 12. Testing Strategy

> **Principle**: Minimum test investment, maximum AI agent utility. Tests must give fast, unambiguous pass/fail signals so an agent can iterate without burning tokens on manual inspection.

The detailed test plan (specific flows, seed data strategy, assertion patterns) lives in `prd/prd-testing.md`. This section captures only the architectural testing decisions.

### 12.1 Layered Testing Pyramid

```
         [E2E — Playwright]          <- Slowest. Catches UI regressions.
        /                  \
  [API Integration — Go tests]       <- Medium. Covers all business logic.
      /                      \
[Go unit tests (service layer)]      <- Fastest. Pure functions, no I/O.
```

Agents run layers **bottom-up**, stopping as soon as a layer fails. This gives fastest possible feedback.

### 12.2 Layer Definitions

#### Layer 1 — Go Unit Tests (`make test-unit`)
- Scope: Pure service-layer functions (e.g. `InstanceMoveService` split logic, field inheritance resolution)
- No database, no HTTP — pure logic only
- Tool: `go test ./internal/...`
- Target: < 5 seconds total

#### Layer 2 — Go Integration Tests (`make test-api`)
- Scope: Full HTTP handler tests against a real SQLite **in-memory** database
- Spins up the real Go HTTP server, fires real HTTP requests, asserts JSON responses
- No browser, no frontend — pure API surface
- Tool: `go test ./api/...` using `net/http/httptest`
- Target: < 30 seconds total
- **This is the primary AI agent validation layer** — `make test-fast` runs only this

#### Layer 3 — Playwright E2E Tests (`make test-e2e`)
- Scope: Critical UI flows in a headless browser against the full running app
- Requires `make dev` stack to be running first, OR a dedicated test-mode startup
- Tool: Playwright (TypeScript, lives in `e2e/` directory)
- Specific flows to be defined in `prd-testing.md`
- Target: < 2 minutes total

### 12.3 AI Agent Test Workflow

An AI agent validating its own changes follows this sequence:

```
1. make test-fast        # Go integration tests (~30s). If FAIL → fix and retry.
2. make test-e2e         # Playwright smoke tests (~2min). If FAIL → fix and retry.
3. Report result to user.
```

All test output must be:
- **Machine-readable**: exit code 0 = pass, non-zero = fail
- **Concise**: Failures print the specific assertion and line, not megabytes of logs
- **Self-contained**: No manual test data setup required — each test seeds its own data

### 12.4 Test Architecture Constraints

| Constraint | Rule |
|---|---|
| No shared state | Each test creates and tears down its own data |
| Real SQLite | Integration tests use real SQLite (in-memory mode) — no mocks |
| No network calls | Tests must never call external services |
| Idempotent | Running tests multiple times produces the same result |
| Fast feedback first | `make test-fast` must complete in < 30s — never add slow tests here |

### 12.5 CI / Automation

- v1: **Local only** — developer (or AI agent) runs `make test` manually
- Future: GitHub Actions running `make test` on PR (out of scope for v1)

---

## 13. Open Questions & Deferred Items

| # | Question | Status |
|---|---|---|
| OQ-1 | Conflict resolution if NAS volume is simultaneously accessed by two container instances | Deferred — single instance assumed in v1 |
| OQ-2 | Should tags have descriptions in addition to colors? | Deferred to tags feature PRD |
| OQ-3 | Exact field types supported (e.g. enum dropdowns, date pickers)? | Deferred to Item Definition feature PRD |
| OQ-4 | Should instances support instance-specific tags? | Deferred. For now, tags only apply to definitions. |
| OQ-5 | Maximum supported tree depth before performance degrades on NAS hardware? | Validate with benchmark |

---

## 14. Related PRDs

> This section is updated as new feature PRDs are created.

| PRD File | Feature | Status |
|---|---|---|
| `prd-overarching-architecture.md` | This document | Active |
| *(future)* `prd-testing.md` | Full test plan: flows, seed data, Playwright scripts | Planned |
| *(future)* `prd-item-definitions.md` | Item Definition CRUD, fields, inheritance, tags | Planned |
| *(future)* `prd-item-instances.md` | Instance CRUD, smart move/split logic | Planned |
| *(future)* `prd-locations.md` | Location tree CRUD, breadcrumb | Planned |
| *(future)* `prd-dashboard.md` | Dashboard, aggregated counts | Planned |
| *(future)* `prd-search.md` | Search and filtering | Planned |
