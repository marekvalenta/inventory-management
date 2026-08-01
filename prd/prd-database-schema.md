# PRD: Database Schema & Migration System — InventoryManagement

> **Status:** Draft v1.0
> **Scope:** Full SQLite database schema, connection pooling, migration tool selection, and startup seeding.

---

## 0. Cross-PRD Consistency Report

### PRDs Reviewed
- `prd-overarching-architecture.md` — Top-level architecture, data model, SQLite usage
- `prd-project-setup.md` — Repo structure, dev tools, and execution context

### Conflicts & Resolutions
| # | Conflict | PRDs Involved | Resolution |
|---|---|---|---|
| 1 | `prd-overarching-architecture.md` mentions `golang-migrate`, which is heavier than necessary for a purely embedded SQLite app. | prd-overarching-architecture.md | Use `pressly/goose` instead of `golang-migrate` for its lighter footprint and superior `go:embed` support, aligning with the low-resource NAS goal. |

### Confirmed Alignments
- Data model aligns with: `prd-overarching-architecture.md` (Locations, Definitions, Instances, Tags).
- API patterns follow: Not directly applicable to DB schema, but ID types (UUID) and timestamps match.
- Scope does not contradict any stated non-goal in: `prd-overarching-architecture.md` (e.g., no multi-user auth schemas, no complex nested tags).

---

## 1. Overview & Problem Statement

This PRD defines the foundational database layer for InventoryManagement. It translates the conceptual data model from the overarching architecture into concrete SQLite tables, constraints, and relationships. It also establishes how migrations are managed and how the application securely connects to the database in a low-resource environment (NAS).

### Core Deliverables
1. Complete SQLite schema (Locations, Tags, Item Definitions, Item Instances, Fields, Values, Settings).
2. Selection and integration strategy for the migration runner (`pressly/goose`).
3. Connection tuning for WAL mode and Foreign Key enforcement.
4. Auto-seeding of a root location on first application boot.

---

## 2. Goals & Measurable Success Metrics

| Goal | Metric |
|---|---|
| Reliable Constraints | 100% of foreign key violations are rejected by SQLite. |
| Zero Data Loss | Database migrations apply safely via `go:embed` on container boot. |
| Performance | SQLite WAL mode enabled, with Go connection pool constrained to max 4 connections to prevent memory/locking issues on NAS. |
| Immediate Usability | App launches with at least one "Root Location" pre-seeded so the user is not met with a completely empty state. |

---

## 3. Critical Risks & Technical Assumptions

| Risk | Mitigation |
|---|---|
| Foreign keys silently ignored | Enforce `PRAGMA foreign_keys = ON` in the SQLite DSN connection string AND via an explicit connection hook in Go. |
| Locking errors on concurrent writes | Go connection pool explicitly tuned to max 4 connections (`SetMaxOpenConns(4)`), WAL mode enabled. |
| UUID generation in SQLite | SQLite lacks native UUID v4 functions. UUIDs will be strictly generated in the Go application tier (`google/uuid`) and passed into inserts. |
| Corrupt migrations | `pressly/goose` will be used with `go:embed` so the binary and migrations are always perfectly in sync. |

---

## 4. User Stories & Acceptance Criteria

### US-001: Automatic Database Migration on Startup
**Description:** As a user deploying the container, I want the database to automatically create or update its tables so I don't have to run manual SQL scripts.

**Acceptance Criteria:**
- [ ] Go application uses `pressly/goose` to run embedded `.sql` migrations on startup.
- [ ] If no database exists, it creates `inventory.db` and runs all migrations.
- [ ] If a migration fails, the app logs a fatal error and exits (container fails health check).
- [ ] Typecheck / build / test suite passes.

### US-002: Safe SQLite Connection Tuning
**Description:** As an application, I need the database connection to be tuned for SQLite WAL mode and foreign key safety so data remains consistent.

**Acceptance Criteria:**
- [ ] Go database connection string includes WAL mode and Foreign Key pragmas.
- [ ] Go `db.SetMaxOpenConns(4)` is applied.
- [ ] Go `db.SetMaxIdleConns(4)` is applied.
- [ ] Code explicitly verifies foreign keys are active immediately after connecting.

### US-003: Auto-Seeding the Root Location
**Description:** As a first-time user, I want the application to start with a default "Root" location so I can immediately begin adding items without configuring locations first.

**Acceptance Criteria:**
- [ ] On startup, after migrations, the app checks if the `locations` table is empty.
- [ ] If empty, it inserts a default Location (e.g., "Home" or "Root") and stores its UUID in `settings.root_location_id`.
- [ ] If empty, it inserts a default Settings row `(id=1, app_name="Inventory", theme="system", root_location_id=<uuid-of-root>)`.

---

## 5. Functional & Technical Requirements

### TR-1: Migration Tooling
- We will use `github.com/pressly/goose/v3`.
- Migrations will be written in raw SQL in the `migrations/` folder.
- The `migrations` package in Go will use `//go:embed *.sql` to bundle these files.
- On startup, `goose.Up()` will be called.

### TR-2: Connection DSN & Pragmas
The SQLite connection string (DSN) must include:
- `_journal_mode=WAL`
- `_foreign_keys=ON`
- `_busy_timeout=5000` (Wait up to 5s if the DB is locked before returning `database is locked`).

Additionally, execute `PRAGMA foreign_keys = ON;` dynamically as a connection hook for absolute safety.

### TR-3: Schema Definitions

**Migrations/00001_initial_schema.sql**
```sql
-- +goose Up
-- +goose StatementBegin
CREATE TABLE locations (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT,
    parent_id   TEXT REFERENCES locations(id) ON DELETE RESTRICT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tags (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    color       TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE item_definitions (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    description     TEXT,
    parent_def_id   TEXT REFERENCES item_definitions(id),
    unit            TEXT,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE definition_tags (
    definition_id TEXT NOT NULL REFERENCES item_definitions(id) ON DELETE CASCADE,
    tag_id        TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (definition_id, tag_id)
);

CREATE TABLE definition_fields (
    id              TEXT PRIMARY KEY,
    definition_id   TEXT NOT NULL REFERENCES item_definitions(id) ON DELETE CASCADE,
    field_name      TEXT NOT NULL,
    field_type      TEXT NOT NULL, -- 'text', 'number', 'boolean', 'date'
    is_required     BOOLEAN NOT NULL DEFAULT 0,
    display_order   INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE item_instances (
    id                  TEXT PRIMARY KEY,
    definition_id       TEXT NOT NULL REFERENCES item_definitions(id) ON DELETE RESTRICT,
    quantity            INTEGER NOT NULL DEFAULT 1 CHECK (quantity > 0),
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
    value       TEXT,
    UNIQUE (instance_id, field_id)
);

CREATE TABLE settings (
    id               INTEGER PRIMARY KEY CHECK (id = 1),
    app_name         TEXT NOT NULL DEFAULT 'Inventory',
    theme            TEXT NOT NULL DEFAULT 'system',
    root_location_id TEXT REFERENCES locations(id),
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS instance_field_values;
DROP TABLE IF EXISTS item_instances;
DROP TABLE IF EXISTS definition_fields;
DROP TABLE IF EXISTS item_definitions;
DROP TABLE IF EXISTS definition_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS locations;
-- +goose StatementEnd
```

### TR-4: Application-Level UUIDs
Since SQLite lacks native UUID v4, Go must generate the IDs using `github.com/google/uuid` before performing an `INSERT`. All `id` columns in the schema are `TEXT` to support this.

---

## 6. Edge Cases & Failure Modes

| Failure Mode | System Behavior |
|---|---|
| User attempts to delete a Tag with linked definitions | Rejected by DB (`FOREIGN KEY` violation). Go returns HTTP 409 Conflict. |
| User attempts to delete a Location with items | Rejected by DB (`ON DELETE RESTRICT`). Go returns HTTP 409 Conflict. |
| Database file is unwriteable due to NAS permissions | Application fails to start, logging a fatal error before the HTTP server binds. |
| Migration fails midway | Goose rollback will trigger (`-- +goose Down`). If irrecoverable, app exits. |
| Concurrent writes overwhelm the DB | Go's `_busy_timeout=5000` pauses the request up to 5s. If it fails, Go returns HTTP 503. |

---

## 7. Non-Goals & Scope Boundaries
- We are **not** building a dynamic ORM or query builder. Queries will use standard `database/sql` (or `sqlx` if needed) to keep performance high and dependencies low.
- We are **not** using `golang-migrate` as originally listed in the overarching architecture.
- We are **not** building database export/backup logic into the application code for v1 (relying on NAS volume backups).

---

## 8. Open Questions & Deferred Items

| # | Question | Status |
|---|---|---|
| OQ-1 | Should we index `name` columns for faster substring search? | Deferred to Search PRD. If indexes are needed, they will be part of the initial schema, not a separate migration. |
| OQ-2 | Are there specific constraints needed for `parent_instance_id` to prevent infinite loops? | A database `CHECK` constraint can't easily prevent recursive loops. This must be handled by Go application logic in the Instance Move service. |
