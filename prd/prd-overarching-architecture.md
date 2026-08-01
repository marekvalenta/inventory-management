# PRD: Overarching Architecture — InventoryManagement

> **Status:** Living Document — v2.0  
> **Last Updated:** 2026-08-01  
> **Scope:** Top-level architectural decisions. All feature PRDs must reference and align with this document. Implementation details live in specialized PRDs — this document defines *what* and *why*, not *how*.

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
| HTTP framework | `net/http` + `chi` router | Lightweight, idiomatic, composable middleware |
| Database driver | `modernc.org/sqlite` (pure Go, CGO-free) | No CGO required, simpler Docker build |
| Migrations | `pressly/goose` with `go:embed` | Lightweight, embeds SQL in binary |
| Query | `database/sql` or `sqlx` | Explicit SQL preferred for performance visibility |
| Static file serving | Go serves compiled React build via `embed.FS` | Single binary, no Nginx needed |
| Config | Environment variables only | Aligns with Docker/compose idiom |

See `prd-backend-architecture.md` for detailed Go project layout, router setup, middleware, and error handling.

### 3.2 Frontend — React

| Decision | Choice | Rationale |
|---|---|---|
| Framework | React (Vite build toolchain) | Fast build, small bundle |
| Language | TypeScript | Type safety, better maintainability |
| Styling | CSS Modules + CSS variables | No heavy CSS framework, mobile-first |
| State management | TanStack Query for server state | No Redux overhead; REST cache layer |
| Routing | React Router v6 | Standard SPA routing |
| Bundle output | Static files embedded into Go binary via `go:embed` | Single container, no separate web server |

See `prd-frontend-architecture.md` for routing strategy, component architecture, design system, and UI toolkits.

### 3.3 Database — SQLite

| Decision | Choice |
|---|---|
| Engine | SQLite (single file, WAL mode) |
| Location | `/data/inventory.db` inside container, mounted from NAS volume |
| Migrations | Versioned SQL migrations run at application startup via `pressly/goose` |
| Backup | User responsibility — NAS volume backup; future: export endpoint |

See `prd-database-schema.md` for the complete schema, migration system, connection tuning, and seeding.

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

The result is a **single, minimal Docker image** (~20-40 MB estimated) built via a multi-stage Dockerfile. See `prd-docker-deployment.md` for the full build spec, docker-compose, health checks, and NAS deployment guide.

**User-facing `docker-compose.yml` example:**
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

The canonical SQL schema, constraints, and migration files live in `prd-database-schema.md`. This section defines the conceptual model only.

### 4.1 Conceptual Model

```
WORLD
 +-- Location (definition, hierarchical)
      +-- Sub-Location
      |    +-- Sub-Sub-Location...
      +-- Item Instance (placed here)
           +-- Item Instance (contained within parent item)
```

Four primary entity types:

| Entity | Nature | Description |
|---|---|---|
| **Location** | Definition only | A physical or logical place. No instances. Locations contain Locations and Item Instances. Hierarchical via self-referencing parent. |
| **Item Definition** | Class / Schema | Defines what a thing *is*. Holds the field schema (name, type, required), unit, optional parent definition for inheritance, and can have multiple tags. |
| **Item Instance** | Record | A physical occurrence of an Item Definition. Has a quantity, location/parent, and instance-specific field values. |
| **Settings** | Singleton | Application-level UI preferences (app name, theme, display options). Single-row table with id=1 constraint. |

### 4.2 Key Design Rules

- **Locations are definitions only** — a Room is just a Room. No "instances" of a location exist.
- **Definition inheritance** — a child definition inherits the field schema from its parent and may add new fields.
- **Smart quantity merging** — identical items (same definition, same field values, same parent) share one instance record with a merged `quantity`. When any field value diverges, they become separate instances.
- **Container nesting** — definitions can be marked `is_container`, allowing their instances to contain other instances (item-in-item). Only container instances can act as parents.
- **Move/split** — instances can be partially moved with transaction safety, auto-merging at target. Full logic in `prd-item-instances.md`.
- **Tags** — flexible many-to-many labeling of item definitions (e.g. "Fasteners", "Fragile"). Defined in `prd-tags.md`.
- **Deletion guards** — locations, definitions, and container-like instances with children cannot be deleted (hard block, FK `ON DELETE RESTRICT`, HTTP 409). Tags used by definitions can be deleted with user confirmation; associated definition_tags rows cascade via `ON DELETE CASCADE` — the user is warned how many definitions will be affected before proceeding.

### 4.3 Entity Relationship Summary

```
tags ----< definition_tags >---- item_definitions >---- item_definitions (self-ref/inheritance)
                       |
               definition_fields

item_definitions ----< item_instances >---- item_instances (item-in-item, self-ref)
                              |                      |
                         locations            instance_field_values
                       (self-ref tree)

settings (singleton row)
```

---

## 5. REST API Design

### 5.1 Conventions

- **Base path**: `/api/v1/`
- **Format**: JSON request/response bodies
- **IDs**: UUIDs (string), generated by the Go application layer
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

This is computed server-side.

---

## 7. Deployment Architecture

### 7.1 Configuration (Environment Variables)

| Variable | Default | Description |
|---|---|---|
| `APP_PORT` | `8080` | HTTP port the server listens on |
| `APP_NAME` | `Inventory` | Display name shown in the UI header |
| `DATA_DIR` | `/data` | Directory for SQLite DB and future assets |
| `LOG_LEVEL` | `info` | Logging verbosity (`debug`, `info`, `warn`, `error`) |

UI-level settings (theme, display preferences) are stored in a `settings` SQLite table and managed via the Settings page.

### 7.2 Volume Structure

```
/data/
  inventory.db        <- SQLite database
  (future) images/    <- Photo attachments (reserved path, not used in v1)
```

---

## 8. Developer Experience

Local development uses a **Makefile** as the task runner with `air` for Go hot-reload and Vite HMR for the frontend. Both run concurrently via a single `make dev` command. A PowerShell wrapper (`make.ps1`) enables the same workflow on Windows.

See `prd-project-setup.md` for the full repo structure, Makefile targets, Air configuration, and first-time setup instructions.

---

## 9. Testing Strategy

> **Principle**: Minimum test investment, maximum AI agent utility. Tests must give fast, unambiguous pass/fail signals.

The testing pyramid has three layers: **Go unit tests** (pure logic, <5s), **Go API integration tests** (HTTP handlers vs in-memory SQLite, <30s — the primary AI agent validation layer), and **Playwright E2E tests** (browser flows, <2min).

Agents run layers bottom-up, stopping on first failure. `make test-fast` runs the integration tests only.

See `prd-testing.md` for the full test plan: specific flows, seed data strategy, assertion patterns, and Playwright script specifications.

---

## 10. Critical Risks & Technical Assumptions

| Risk | Mitigation |
|---|---|
| SQLite write contention (future multi-user) | WAL mode enabled by default; single-user in v1 makes this a non-issue |
| Recursive tree queries on large hierarchies | Use SQLite recursive CTEs; add depth limit guard at API layer |
| Instance split logic bugs causing data loss | Wrap all move operations in a database transaction; rollback on any error |
| CGO dependency breaking Docker build | Use `modernc.org/sqlite` (pure Go, CGO-free) |
| React bundle size on low-bandwidth LAN | Code splitting by route; target < 300 KB gzipped initial bundle |
| Schema migration failures on startup | Migrations are idempotent and versioned; failed migration = container exits with clear error log |

---

## 11. Non-Goals (v1 Scope Boundaries)

The following are **explicitly out of scope** for v1 and must not be designed in ways that block them later:

- No user authentication / multi-user access
- No barcode / QR code scanning
- No photo / image attachments (schema hook reserved in volume path)
- No bulk CSV/JSON import-export
- No reporting, analytics, or dashboards beyond basic totals
- No notifications or alerts (e.g. low stock)
- No external service integrations
- No OpenAPI/Swagger spec
- No offline / PWA support

---

## 12. Extensibility Commitments

Even though the above are non-goals for v1, the architecture must not block them:

| Future Feature | Architectural Hook Required Now |
|---|---|
| Photo attachments | `/data/images/` volume path reserved; schema designed for clean `image_path` migration |
| Barcode scanning | Item definition has a nullable `barcode` field slot available (unused in v1) |
| Bulk import | Import service will reuse the same move/upsert logic |
| Multi-user | Auth middleware slot planned in Go router (passthrough stub in v1) |
| Full-text search | SQLite FTS5 extension can be added without breaking the REST API contract |
| External API access | `/api/v1/` versioned from day one |

---

## 13. Open Questions & Deferred Items

| # | Question | Status |
|---|---|---|
| OQ-1 | Conflict resolution if NAS volume is simultaneously accessed by two container instances | Deferred — single instance assumed in v1 |
| OQ-2 | Should tags have descriptions in addition to colors? | Resolved — Tags use name + color (hex) only. No description field. |
| OQ-3 | Exact field types supported (e.g. enum dropdowns, date pickers)? | Resolved — v1 supports text, number, boolean, date, enum. See `prd-item-definitions.md`. |
| OQ-4 | Should instances support instance-specific tags? | Deferred. For now, tags only apply to definitions. |
| OQ-5 | Maximum supported tree depth before performance degrades on NAS hardware? | Validate with benchmark |

---

## 14. Related PRDs

> This section tracks all PRDs in the project. The canonical backlog is in `AGENTS.md`.

| # | PRD File | Topic | Status |
|---|---|---|---|
| 0 | `prd-overarching-architecture.md` | This document — high-level architecture, tech stack, data model | ✅ Done |
| 1 | `prd-project-setup.md` | Repo structure, Go module, Vite init, Makefile, dev workflow | ✅ Done |
| 2 | `prd-database-schema.md` | Full SQLite schema, migration system, WAL mode, startup runner | ✅ Done |
| 3 | `prd-backend-architecture.md` | Go project layout, router, middleware, error handling, config, embed | ✅ Done |
| 4 | `prd-frontend-architecture.md` | React/Vite/TS scaffold, TanStack Query, routing, CSS design system, nav | ✅ Done |
| 5 | `prd-locations.md` | Locations CRUD — API + UI, tree browser, deletion guard | ✅ Done |
| 6 | `prd-tags.md` | Tags CRUD — API + UI, deletion guard | ✅ Done |
| 7 | `prd-item-definitions.md` | Definitions CRUD — API + UI, field schema, inheritance, tags | ✅ Done |
| 8 | `prd-item-instances.md` | Instances CRUD — API + UI, smart move/split logic, breadcrumb | ✅ Done |
| 9 | `prd-dashboard.md` | Dashboard — totals, recent activity, quick search bar | 🔲 Planned |
| 10 | `prd-search.md` | Name-based search v1 — API + UI, extensible for filters later | 🔲 Planned |
| 11 | `prd-settings.md` | Settings page — UI + backend, app name, display prefs in SQLite | 🔲 Planned |
| 12 | `prd-testing.md` | Full test plan — flows, seed data, Go integration tests, Playwright E2E | 🔲 Planned |
| 13 | `prd-docker-deployment.md` | Multi-stage Dockerfile, docker-compose, health check, NAS deploy guide | 🔲 Planned |
