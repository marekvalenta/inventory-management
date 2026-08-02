# PRD: Testing — InventoryManagement

> **Status:** Draft v1.0
> **Scope:** Test framework setup, integration test patterns, Playwright E2E specification, local dev run workflow, seed data flag. Unit tests limited to critical pure logic only.

---

## 0. Cross-PRD Consistency Report

### PRDs Reviewed
- `prd-overarching-architecture.md` — Testing strategy outline (§9), Makefile targets, Docker deployment model, dev URLs.
- `prd-project-setup.md` — Repo scaffold, Makefile, Air config, Vite proxy, PowerShell wrapper, AGENTS.md reference.
- `prd-database-schema.md` — Migration system (`pressly/goose`), WAL mode, `go:embed`, seed logic, in-memory SQLite constraint.
- `prd-backend-architecture.md` — Go layering, chi router, error mapping (`{error, code}`), `RespondWithError` helper.
- `prd-frontend-architecture.md` — TanStack Query keys, React Router routes, CSS Modules, Radix UI.
- `prd-visual-design.md` — Golden Amber tokens, component states (loading/empty/error), page layouts.
- `prd-locations.md` — Full CRUD test surface, breadcrumb CTE, cycle detection, deletion guard.
- `prd-tags.md` — CRUD test surface, inline editing, cascade delete.
- `prd-item-definitions.md` — CRUD test surface, field resolution, inheritance, overrides, tag assignment.
- `prd-item-instances.md` — CRUD test surface, move/split transaction, auto-merge, breadcrumb, container nesting.

### Conflicts & Resolutions

| # | Conflict | PRDs Involved | Resolution |
|---|---|---|---|
| 1 | Overarching §9 says "in-memory SQLite" for integration tests. Database PRD uses `_journal_mode=WAL` and `_busy_timeout` pragmas designed for file-based SQLite, not `:memory:`. | prd-overarching-architecture.md, prd-database-schema.md | Use **temp file on disk** (not `:memory:`) for integration tests. Temp file supports WAL mode and goose migration semantics identically to production. Clean up temp file after test suite. Overarching PRD wording updated. |
| 2 | `make test-api` in project-setup targets `go test ./internal/handler/...` — but integration tests also need the service and db packages. | prd-project-setup.md | Expand `test-api` to run `go test ./internal/...` (all packages under `internal/`). Distinguish from `test-unit` by using build tags: unit tests skip DB, integration tests include it. Add `//go:build integration` tag to all integration test files. |
| 3 | Overarching PRD §11 lists "No OpenAPI/Swagger spec" as a non-goal — but tests shouldn't need it. | prd-overarching-architecture.md | No conflict. Confirmed. |

### Confirmed Alignments
- Makefile targets (`make test`, `make test-fast`, `make test-api`, `make test-unit`, `make test-e2e`) are consistent with project-setup PRD.
- `make dev` starts both servers via Air + Vite — consistent.
- Error format `{"error":"...","code":"..."}` used across all feature PRDs — integration tests assert against this.
- TanStack Query key patterns from frontend PRD — E2E tests don't need to know them, but Playwright assertions may target rendered UI elements.
- API routes `/api/v1/...` — consistent.
- JSON request/response, UUID IDs — consistent.
- No schema changes needed — testing PRD is infrastructure, not a feature.
- Scope does not contradict any non-goal in overarching PRD.

---

## 1. Overview & Problem Statement

This PRD defines how the InventoryManagement application is tested and how developers (including AI agents) run it locally for development and investigation. The testing strategy prioritizes high-level integration and E2E tests — the layers that catch real bugs with minimum test-writing investment. Unit tests are limited to critical pure logic only (cycle detection, breadcrumb resolution).

### Core Deliverables
1. Integration test framework: temp-file SQLite, goose migration runner, test helpers, per-test seed patterns.
2. Minimal unit test targets: cycle detection, breadcrumb resolution, field type validation — pure functions only.
3. Playwright E2E specification: 3-5 key user journeys defined at the flow level.
4. `--seed` CLI flag: populates the database with representative demo data for exploration.
5. Local dev run documentation: both `make dev` (hot-reload) and `make build && ./bin/server` (single binary).
6. AI agent investigation workflow: documented steps for running the app, navigating, and iterating.

---

## 2. Goals & Measurable Success Metrics

| Goal | Metric |
|---|---|
| Integration test speed | Full `make test-api` completes in < 30s |
| Integration test isolation | Each test file uses a clean temp-file SQLite; zero cross-test state leakage |
| Integration test reliability | 100% pass rate on clean checkout — no flaky tests |
| E2E smoke test | `make test-e2e` completes in < 2min |
| Seed data utility | `./bin/server --seed` populates 3+ locations, 5+ definitions, 10+ instances in < 2s |
| Local dev accessibility | `make dev` starts both servers on first attempt after `npm install` |
| AI agent autonomy | Agent can run `make dev`, open browser at localhost:5173, and verify UI state without human intervention |

---

## 3. Critical Risks & Technical Assumptions

| Risk | Mitigation |
|---|---|
| Integration tests leak state between test functions | Each test file calls `setupTestDB(t)` which creates a unique temp file. `t.Cleanup()` removes it. |
| Goose migrations fail on temp-file SQLite | Temp file uses same DSN pragmas as production (`_journal_mode=WAL`, `_foreign_keys=ON`). Verified during framework setup. |
| `--seed` flag conflicts with existing data | Seed only runs if the locations table is empty (no migration run). If DB already has data, `--seed` is a no-op with a log message. |
| Playwright tests depend on specific seed data | E2E tests use a dedicated seed `E2E_SEED=true` that creates deterministic data with known IDs. Separate from the demo seed. |
| Integration test tags (`//go:build integration`) confuse IDE/test runners | All integration test files end in `_integration_test.go` and carry the build tag. Unit test files end in `_test.go` without the tag. `go test ./...` picks up both; `go test -tags=integration ./internal/...` runs integration only. |
| Browser not available in CI/headless AI agent environment | Playwright runs headless by default. E2E tests are optional — `make test-fast` skips them. |

---

## 4. User Stories & Acceptance Criteria

### US-001: Integration Test Framework Setup
**Description:** As a developer, I want a reusable test framework that sets up a clean SQLite database with all migrations applied, so I can write handler-level integration tests with minimal boilerplate.

**Acceptance Criteria:**
- [ ] Test helper `setupTestDB(t *testing.T) *sql.DB` creates a unique temp file via `os.CreateTemp`, opens it with WAL + FK pragmas, runs goose `Up()`, and registers cleanup via `t.Cleanup()`.
- [ ] Test helper `newTestServer(t *testing.T) *httptest.Server` wires the full chi router with real handlers and the test DB.
- [ ] Test helper `seedTestData(t *testing.T, db *sql.DB)` inserts common base data (root location "Home", a few tags, 1-2 definitions) and returns their IDs.
- [ ] All integration test files use `//go:build integration` tag and `_integration_test.go` suffix.
- [ ] `make test-api` runs `go test -tags=integration ./internal/...` and exits 0.
- [ ] Typecheck / build passes.

### US-002: Per-Feature Integration Test Patterns
**Description:** As a developer implementing a feature PRD, I want documented test patterns so I can write integration tests that follow the same conventions as all other features.

**Acceptance Criteria:**
- [ ] Pattern documented for CRUD handler tests: create → assert 201 + JSON shape → get → assert 200 + match → update → assert 200 → delete → assert 204 → get → assert 404.
- [ ] Pattern documented for error cases: invalid input → 400 with `code` field, not-found → 404, conflict → 409.
- [ ] Pattern documented for transactional operations (move/split): verify atomicity — invalid operation leaves DB unchanged.
- [ ] Pattern documented for auto-merge: create identical instance → quantity incremented, no new row.
- [ ] A single complete example test file (`locations_integration_test.go`) is included in this PRD's implementation as the canonical reference.
- [ ] Typecheck / build / test suite passes.

### US-003: Minimal Unit Tests
**Description:** As a developer, I want unit tests for the few pure-logic functions that are complex enough to warrant isolated testing.

**Acceptance Criteria:**
- [ ] Unit tests exist for **cycle detection** — given a parent_id and a tree, verify descendant check is correct.
- [ ] Unit tests exist for **breadcrumb resolution** — given an instance chain + location chain, verify merged output.
- [ ] Unit tests exist for **field type validation** — verify number/boolean/enum/default_value validation rules in isolation.
- [ ] Unit tests are in regular `_test.go` files (no build tag), run via `make test-unit` or `go test ./internal/...`.
- [ ] Unit tests complete in < 5s total.
- [ ] Typecheck / build / test suite passes.

### US-004: `--seed` CLI Flag for Demo Data
**Description:** As a developer (or AI agent), I want to start the app with representative demo data so I can explore the UI without manually creating locations, definitions, and instances.

**Acceptance Criteria:**
- [ ] Running `./bin/server --seed` or `go run ./cmd/server --seed` seeds the database and starts the server.
- [ ] Seed data includes:
  - 4+ locations in a realistic hierarchy (e.g., Home → Living Room, Workshop, Garage; Workshop → Tool Cabinet)
  - 5+ tags with distinct colors (Fasteners, Hardware, Tools, Electronics, Office)
  - 5+ item definitions spanning parent-child inheritance (e.g., Fastener → Screw → M3 Screw; Tool; Container → Toolbox)
  - 15+ item instances distributed across locations/containers with varied quantities and field values
- [ ] Seed only runs if `locations` table is empty. If data exists, logs `"Seed skipped: database already contains data"` and starts normally.
- [ ] Seed runs after migrations on startup.
- [ ] Seed data uses deterministic UUIDs (hardcoded in a Go constants map) so AI agents can reference specific IDs.
- [ ] Typecheck / build / test suite passes.

### US-005: Playwright E2E Test Suite
**Description:** As a developer, I want browser-based end-to-end tests that cover the most critical user journeys.

**Acceptance Criteria:**
- [ ] Playwright configured with `playwright.config.ts` in `e2e/` directory.
- [ ] E2E tests start the built binary (`./bin/server` with E2E seed) as a subprocess, run tests against it, then kill it.
- [ ] **E2E-001: Locations CRUD Journey**
  - Navigate to locations page → tree shows "Home" → expand → click "Living Room" → detail page loads → add sub-location "Bookshelf" → verify it appears → delete → verify 409 if children exist → verify 204 if empty.
- [ ] **E2E-002: Definitions + Tags Journey**
  - Navigate to definitions → list shows seeded definitions → create new "Bolt" with fields (Material: enum, Length: number) and tag "Fasteners" → verify detail page shows fields and tag → navigate to tags page → edit a tag color → verify badge updates.
- [ ] **E2E-003: Instance Create + Move Journey**
  - Navigate to a location → add instance of "M3 Screw" with quantity 10 → verify instance appears in contents → open instance detail → move 3 to "Workshop" → verify source quantity becomes 7 → navigate to Workshop → verify 3 screws arrived.
- [ ] **E2E-004: Breadcrumb Navigation Journey**
  - Create a container instance "Toolbox" in Workshop → add instance inside it → open the nested instance → verify breadcrumb shows "Home > Workshop > Toolbox > [item]" → click breadcrumb segments to navigate back up the chain.
- [ ] **E2E-005: Error Handling Journey**
  - Attempt to delete a location with items → verify 409 error toast → attempt to create tag with duplicate name → verify inline error → attempt to move more items than available → verify error message.
- [ ] All E2E tests pass reliably (no flakiness over 3 consecutive runs).
- [ ] `make test-e2e` runs the suite and exits 0 on success.

### US-006: Local Dev Run Documentation
**Description:** As a developer (or AI agent), I want clear, authoritative instructions for running the application locally with and without Docker.

**Acceptance Criteria:**
- [ ] `AGENTS.md` Build & Run Reference is confirmed accurate for all targets (already documented in project-setup PRD — verify, not rewrite).
- [ ] Testing PRD documents both run modes:
  - **Dev mode (hot-reload):** `make dev` → Go API at :8080, React UI at :5173. Saving any file triggers hot-reload.
  - **Single binary mode:** `make build && ./bin/server` → single server at :8080 serving both API and React SPA.
- [ ] Both modes documented with exact URLs, expected behavior, and troubleshooting for common issues (port conflicts, missing npm install, air not installed).
- [ ] Seed data mode documented: `make build && ./bin/server --seed` for exploration with demo data.

### US-007: AI Agent Investigation Workflow
**Description:** As an AI agent, I want a documented workflow for running the app, investigating issues in the browser, and iterating on fixes.

**Acceptance Criteria:**
- [ ] `AGENTS.md` updated with an **AI Agent Investigation Workflow** section:
  ```
  ## AI Agent Investigation Workflow
  
  1. Start the app: `make dev` (or `make build && ./bin/server --seed` for single binary with demo data)
  2. Open browser: http://localhost:5173 (dev mode) or http://localhost:8080 (single binary)
  3. Navigate through the UI to reproduce the reported issue.
  4. Use browser DevTools (Network tab, Console) to inspect API calls and errors.
  5. Make code changes. In dev mode, Go restarts via Air and React HMRs.
  6. Verify fix by navigating through the same flow again.
  7. Run `make test-fast` to confirm no regressions.
  8. Run `npx tsc --noEmit --project frontend/tsconfig.json` for TypeScript checks.
  ```
- [ ] Agent knows to check `http://localhost:8080/api/v1/health` as a smoke test.
- [ ] Agent knows URLs for every page: `/locations`, `/locations/:id`, `/definitions`, `/definitions/:id`, `/tags`, `/instances/:id`.

---

## 5. Functional & Technical Requirements

### 5.1 Integration Test Framework

**TR-1:** Test DB setup helper lives in `internal/db/testutil.go` (or a `internal/testutil/` package):

```go
//go:build integration

package testutil

import (
    "database/sql"
    "os"
    "testing"

    "github.com/marekvalenta/inventory-management/internal/db"
    _ "modernc.org/sqlite"
)

func SetupTestDB(t *testing.T) *sql.DB {
    t.Helper()

    f, err := os.CreateTemp("", "inventory-test-*.db")
    if err != nil {
        t.Fatalf("create temp db: %v", err)
    }
    t.Cleanup(func() {
        f.Close()
        os.Remove(f.Name())
    })

    dsn := f.Name() + "?_journal_mode=WAL&_foreign_keys=ON"
    database, err := sql.Open("sqlite", dsn)
    if err != nil {
        t.Fatalf("open test db: %v", err)
    }
    database.SetMaxOpenConns(1)

    if err := db.RunMigrations(database); err != nil {
        t.Fatalf("run migrations: %v", err)
    }

    return database
}
```

**TR-2:** `db.RunMigrations(db *sql.DB) error` — extracted from the startup code so both main and tests can call it. Uses `go:embed` migrations via goose.

**TR-3:** Test HTTP server helper:

```go
func NewTestServer(t *testing.T, db *sql.DB) *httptest.Server {
    t.Helper()
    // Construct the full chi router using the same router.New(db, ...) function
    // from internal/router/.
    r := router.New(db, &config.Config{...})
    return httptest.NewServer(r)
}
```

**TR-4:** `make test-api` updated to include the integration build tag:

```makefile
test-api:
	go test -tags=integration ./internal/...
```

**TR-5:** `make test-unit` runs tests WITHOUT the integration tag (excludes integration tests):

```makefile
test-unit:
	go test ./internal/...
```

Since integration tests have `//go:build integration`, they are excluded from regular `go test`. Unit tests (no build tag) run in both modes. This is acceptable because unit tests are fast and few.

**TR-6:** All integration test files follow the naming convention `*_integration_test.go` and start with `//go:build integration`.

### 5.2 Integration Test Patterns

**TR-7:** **CRUD happy-path pattern** (reference for all feature PRDs):

```go
//go:build integration

package handler_test

func TestLocationsCRUD(t *testing.T) {
    db := testutil.SetupTestDB(t)
    srv := testutil.NewTestServer(t, db)
    client := srv.Client()

    // CREATE
    body := `{"name": "Workshop", "description": "My workshop"}`
    resp, err := client.Post(srv.URL+"/api/v1/locations", "application/json",
        strings.NewReader(body))
    require.NoError(t, err)
    require.Equal(t, http.StatusCreated, resp.StatusCode)

    var loc Location
    json.NewDecoder(resp.Body).Decode(&loc)
    resp.Body.Close()
    require.NotEmpty(t, loc.ID)
    require.Equal(t, "Workshop", loc.Name)

    // GET by ID
    resp, err = client.Get(srv.URL + "/api/v1/locations/" + loc.ID)
    require.NoError(t, err)
    require.Equal(t, http.StatusOK, resp.StatusCode)

    // UPDATE
    body = `{"name": "Main Workshop"}`
    req, _ := http.NewRequest("PUT", srv.URL+"/api/v1/locations/"+loc.ID,
        strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    resp, err = client.Do(req)
    require.NoError(t, err)
    require.Equal(t, http.StatusOK, resp.StatusCode)

    // DELETE
    req, _ = http.NewRequest("DELETE", srv.URL+"/api/v1/locations/"+loc.ID, nil)
    resp, err = client.Do(req)
    require.NoError(t, err)
    require.Equal(t, http.StatusNoContent, resp.StatusCode)

    // GET after delete -> 404
    resp, err = client.Get(srv.URL + "/api/v1/locations/" + loc.ID)
    require.NoError(t, err)
    require.Equal(t, http.StatusNotFound, resp.StatusCode)
}
```

Uses `testing` + `github.com/stretchr/testify/require` for assertions. No custom assertion library needed beyond testify.

**TR-8:** **Error case pattern:**

```go
func TestLocationsCreateValidation(t *testing.T) {
    db := testutil.SetupTestDB(t)
    srv := testutil.NewTestServer(t, db)
    client := srv.Client()

    // Missing required field
    resp, _ := client.Post(srv.URL+"/api/v1/locations", "application/json",
        strings.NewReader(`{}`))
    require.Equal(t, http.StatusBadRequest, resp.StatusCode)

    var errResp ErrorResponse
    json.NewDecoder(resp.Body).Decode(&errResp)
    require.NotEmpty(t, errResp.Error)
    require.NotEmpty(t, errResp.Code)

    // Name too short
    resp, _ = client.Post(srv.URL+"/api/v1/locations", "application/json",
        strings.NewReader(`{"name": "X"}`))
    require.Equal(t, http.StatusBadRequest, resp.StatusCode)

    // Non-existent parent
    resp, _ = client.Post(srv.URL+"/api/v1/locations", "application/json",
        strings.NewReader(`{"name": "Shelf", "parent_id": "nonexistent"}`))
    require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
```

**TR-9:** **Transactional operation pattern** (move/split):

```go
func TestInstanceMoveAtomicity(t *testing.T) {
    db := testutil.SetupTestDB(t)
    srv := testutil.NewTestServer(t, db)
    client := srv.Client()

    // Setup: create definition + instance
    // Attempt invalid move (quantity > available)
    // Verify instance quantity unchanged (transaction rolled back)
}
```

### 5.3 Unit Tests

**TR-10:** Unit test locations (pure logic, no DB, no HTTP):

| Package | Test File | What It Tests |
|---|---|---|
| `internal/service/` | `cycle_test.go` | `isDescendant(parentID, childID, tree)` — verify cycle detection logic |
| `internal/service/` | `breadcrumb_test.go` | `mergeBreadcrumb(locations, instances)` — verify correct ordering and kind assignment |
| `internal/service/` | `field_validation_test.go` | `validateFieldValue(fieldType, value, enumValues)` — verify type validation rules |

**TR-11:** Unit tests use table-driven patterns with `testify/require`. No DB connection, no HTTP server — pure function tests.

### 5.4 `--seed` CLI Flag

**TR-12:** Added to `cmd/server/main.go`:

```go
func main() {
    seedMode := flag.Bool("seed", false, "Seed database with demo data and start server")
    flag.Parse()

    cfg := config.Load()
    database := db.Connect(cfg.DataDir)
    db.RunMigrations(database)

    if *seedMode {
        db.SeedIfEmpty(database)
    } else {
        db.AutoSeed(database) // existing root location + settings seed
    }

    // ... start server
}
```

**TR-13:** `db.SeedIfEmpty(db *sql.DB)` lives in `internal/db/seed.go`:

```go
func SeedIfEmpty(db *sql.DB) {
    var count int
    db.QueryRow("SELECT COUNT(*) FROM locations").Scan(&count)
    if count > 1 { // > 1 because auto-seed already created root
        log.Println("Seed skipped: database already contains data")
        return
    }
    // Insert demo data...
}
```

**TR-14:** Seed data must use deterministic, hardcoded UUIDs stored as Go constants so AI agents and E2E tests can reference specific entities:

```go
const (
    SeedLocationHome      = "00000000-0000-0000-0000-000000000001"
    SeedLocationLiving    = "00000000-0000-0000-0000-000000000002"
    SeedLocationWorkshop  = "00000000-0000-0000-0000-000000000003"
    // ... etc
)
```

**TR-15:** Seed data content:

```
Locations:
  Home (root)
    Living Room
      Bookshelf
    Workshop
      Tool Cabinet
    Garage

Tags:
  Fasteners (#E8A838), Hardware (#6B8E5A), Tools (#4A7FB5),
  Electronics (#C2543D), Office (#8E7CC3)

Definitions:
  Fastener (parent, fields: Material: enum[Steel,Brass,Aluminum], is_container: false)
    Screw (child of Fastener, adds: Length: number, Thread: text)
      M3 Screw (child of Screw, overrides Length default to 12)
  Tool (fields: Brand: text, is_container: false)
  Storage Box (fields: Color: text, is_container: true)
  Cable (fields: Type: enum[USB-C,Micro-USB,Lightning], Length: number)

Instances (distributed across locations):
  Home/Living Room:
    - M3 Screw (x50, Material: Steel, Length: 12, Thread: 0.5mm)
    - Screw (x20, Material: Brass, Length: 20)
  Home/Workshop:
    - M3 Screw (x100, Material: Steel, Length: 12, Thread: 0.5mm)
    - Tool (x1, Brand: Bosch)
    - Storage Box (x2, Color: Red) [container with children]
      - M3 Screw (x30, inside Toolbox)
      - Cable (x5, Type: USB-C, Length: 1)
  Home/Workshop/Tool Cabinet:
    - Screw (x200, Material: Steel, Length: 30)
    - Tool (x1, Brand: Makita)
```

**TR-16:** `db.AutoSeed(db)` — existing root-location + settings seed logic unchanged. Called on normal startup (no `--seed` flag). `--seed` mode calls `SeedIfEmpty` which includes the root location.

### 5.5 Playwright E2E Setup

**TR-17:** `e2e/playwright.config.ts`:

```typescript
import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  timeout: 30000,
  retries: 0,
  use: {
    baseURL: 'http://localhost:8080',
    headless: true,
    viewport: { width: 1280, height: 720 },
  },
  webServer: {
    command: 'E2E_SEED=true ../bin/server',
    port: 8080,
    reuseExistingServer: false,
    timeout: 10000,
  },
});
```

**TR-18:** E2E tests installed via `npm install --save-dev @playwright/test` in `e2e/` directory (separate from frontend for clean dependency separation). Actually, since the e2e dir already exists in project root, put `package.json` and config there.

**TR-19:** E2E test structure:
```
e2e/
  package.json
  playwright.config.ts
  tests/
    locations.spec.ts        # E2E-001
    definitions-tags.spec.ts # E2E-002
    instances.spec.ts        # E2E-003
    breadcrumbs.spec.ts      # E2E-004
    error-handling.spec.ts   # E2E-005
  helpers/
    seed.ts                  # Helper to verify seed data is ready
    api.ts                   # Reusable API call helpers for setup/teardown
```

**TR-20:** E2E tests wait for seed data readiness:

```typescript
// helpers/seed.ts
export async function waitForSeed(page: Page) {
  await page.goto('/');
  await page.waitForSelector('text=Home'); // seeded root location
}
```

**TR-21:** `make test-e2e` runs: `npx playwright test --config e2e/playwright.config.ts` (or from within `e2e/` directory).

**TR-22:** E2E seed mode: when `E2E_SEED=true`, the binary uses a deterministic seed with `E2E_` prefix UUIDs (distinct from `--seed` demo data) so E2E tests can reference exact IDs:

```go
const (
    E2ELocationHome    = "e0000000-0000-0000-0000-000000000001"
    // ...
)
```

### 5.6 Local Dev Run Workflow

**TR-23:** Two documented run modes, both without Docker:

**Mode 1: Dev mode (hot-reload)**
```bash
make dev
```
- Go API: http://localhost:8080 (Air hot-reloads on `.go` changes)
- React UI: http://localhost:5173 (Vite HMR on `.tsx`/`.css` changes)
- API proxy: http://localhost:5173/api/... proxies to :8080
- DB location: `./inventory.db` (in project root, git-ignored)
- Best for: active development, rapid iteration

**Mode 2: Single binary mode**
```bash
make build
./bin/server --seed
```
- Single server at http://localhost:8080
- Serves both API and compiled React SPA
- DB location: `./data/inventory.db` (`DATA_DIR` env var, defaults to `./data`)
- `--seed` populates demo data on first run
- Best for: verifying production behavior, E2E testing, demo

**TR-24:** Troubleshooting in AGENTS.md:
- Port 8080 in use → `lsof -i :8080` and kill the process
- Port 5173 in use → Vite auto-increments to 5174; check terminal output
- `air: command not found` → `go install github.com/air-verse/air@latest`
- `npm run dev` fails → `npm install --prefix frontend`
- DB in wrong place → check `DATA_DIR` env var; defaults to `./`

### 5.7 AI Agent Investigation Workflow

**TR-25:** AGENTS.md updated with:

```markdown
## AI Agent Investigation Workflow

When investigating a bug or verifying a feature, follow these steps:

### 1. Start the Application

**Option A — Dev mode with hot-reload (for iteration):**
```bash
make dev
```
- Backend at http://localhost:8080, Frontend at http://localhost:5173
- Auto-reloads on code changes

**Option B — Single binary with demo data (for verification):**
```bash
make build && ./bin/server --seed
```
- Single server at http://localhost:8080
- Pre-populated with demo data

### 2. Verify the App is Running
```bash
curl http://localhost:8080/api/v1/health
# Expected: {"status":"ok"}
```

### 3. Explore in Browser
| Page | URL (dev mode) |
|---|---|
| Locations tree | http://localhost:5173/locations |
| Location detail | http://localhost:5173/locations/:id |
| Definitions list | http://localhost:5173/definitions |
| Definition detail | http://localhost:5173/definitions/:id |
| Tags | http://localhost:5173/tags |
| Instance detail | http://localhost:5173/instances/:id |

Use browser DevTools (F12) → Network tab to inspect API responses.

### 4. Iterate on a Fix
1. Make code changes in Go or TypeScript files.
2. In dev mode, servers reload automatically (Air for Go, HMR for React).
3. In binary mode, stop (`Ctrl+C`), rebuild (`make build`), restart.
4. Navigate through the UI to verify the fix.

### 5. Run Tests
```bash
make test-fast     # Go integration tests (~30s)
npx tsc --noEmit --project frontend/tsconfig.json  # TypeScript check
```

### 6. Commit
Never commit unless explicitly asked. Run `make test-fast` before reporting success.
```

---

## 6. Edge Cases & Failure Modes

| Scenario | System Behavior |
|---|---|
| Integration test DB temp file can't be created (permission denied) | `t.Fatalf` with clear error. Test fails immediately. |
| Goose migrations fail on temp-file DB (corrupt migration SQL) | `t.Fatalf` with goose error. Test fails. Developer fixes the migration. |
| Two integration test files run in parallel (Go default: parallel by file) | Each creates its own temp file with unique name. No collision. |
| Test file doesn't close response body (resource leak) | Go test's `-race` flag catches this. Document `defer resp.Body.Close()` pattern. |
| `--seed` flag combined with non-empty DB | `SeedIfEmpty` checks row count. Skips seed with log. No data overwritten. |
| `--seed` flag but `DATA_DIR` is unwritable | Server fails to start. Clear error logged. |
| Playwright can't find the built binary | `make test-e2e` requires `make build` first. Add `build` as dependency to the e2e Makefile target. |
| Browser crashes mid-E2E test | Playwright retries the specific test (configured retries: 0 for v1 — no auto-retry, clear failure output). |
| Test DB left on disk after test crash | `t.Cleanup()` runs even on panic/`t.Fatal`. Temp file removed. |
| Integration test timeout | Default Go test timeout is 10min. Set `-timeout 60s` in Makefile target for faster feedback. |
| `make test-api` runs unit tests too (no build tag exclusion) | Unit tests are fast (< 5s). Acceptable. Integration tests are the bulk (~25s). |

---

## 7. Non-Goals & Scope Boundaries

- **Code coverage metrics:** No coverage thresholds or reporting in v1. Not generating `coverage.html` or enforcing percentage targets.
- **CI/CD pipeline:** No GitHub Actions, no automated test runners. Tests are run locally by developers and AI agents.
- **Performance/benchmark tests:** No `go test -bench` targets. Not measuring p95 latency in tests.
- **Fuzz testing:** No `go test -fuzz` targets.
- **Accessibility testing:** No axe-core or screen reader test automation.
- **Visual regression testing:** No screenshot comparison, no Percy/Chromatic.
- **API contract testing:** No OpenAPI spec validation, no Dredd/Pact.
- **Load/stress testing:** No k6, wrk, or similar.
- **Test data factories beyond `--seed`:** No faker/gofakeit. Seed data is hand-written.
- **Mocking framework:** No testify mocks, no gomock. Integration tests use real SQLite; unit tests use pure functions with table-driven inputs.
- **Frontend component tests:** No Jest/Vitest unit tests for React components. UI coverage comes from E2E only.
- **Test parallelization beyond Go defaults:** Go runs test files in parallel by default. No custom parallel configuration.

---

## 8. Open Questions & Deferred Items

| # | Question | Status |
|---|---|---|
| OQ-1 | Should `make dev` automatically seed on first run? | Deferred — current behavior: `make dev` starts with empty DB. User must `make build && ./bin/server --seed` or manually create entities. |
| OQ-2 | Should Playwright tests also run against dev mode (Air + Vite at :5173)? | Deferred — v1 E2E runs against the single binary at :8080. Dev mode testing is manual browser interaction. |
| OQ-3 | Should test helpers live in a shared `internal/testutil/` package or per-package? | `internal/testutil/` package for the framework (DB setup, HTTP server). Feature-specific test helpers defined within each handler test file. |
| OQ-4 | Should `make test` require `make build` first (for E2E)? | Resolved — yes, `make test` target depends on `test-unit test-api`. `test-e2e` is a separate target requiring `build` first. Running `make test` runs unit + integration only. Full `make test test-e2e` requires build. |
| OQ-5 | Should there be a `make seed` target that builds + seeds + starts? | Deferred — adding `--seed` to the binary is sufficient. Can add convenience target later if needed. |

---

## 9. Implementation Order

This PRD is framework infrastructure. It should be implemented *incrementally* as features are built:

1. **Phase 1 (with first feature PRD):** Integration test framework (`testutil` package, build tags, Makefile update, one example test file).
2. **Phase 2 (during feature development):** Per-feature integration tests added in each feature PRD using the documented patterns.
3. **Phase 3 (after all features):** Unit tests for pure logic (cycle, breadcrumb, field validation).
4. **Phase 4 (after all features):** `--seed` flag implementation.
5. **Phase 5 (after all features + frontend):** Playwright E2E setup + 5 test files.
6. **Phase 6 (final):** AGENTS.md AI Agent Investigation Workflow update.

---

## 10. Makefile Target Summary

Final Makefile targets after this PRD:

| Target | Command | Time | What It Tests |
|---|---|---|---|
| `test-unit` | `go test ./internal/...` | < 5s | Unit tests only (integration tests skipped via build tag) |
| `test-api` | `go test -tags=integration -timeout 60s ./internal/...` | < 30s | Integration tests (HTTP handlers vs temp-file SQLite) |
| `test-fast` | `test-api` | < 30s | Primary AI agent validation |
| `test-e2e` | `npx playwright test --config e2e/playwright.config.ts` | < 2min | Browser-based E2E flows |
| `test` | `test-unit test-api` | < 35s | Full Go test suite (no E2E) |

`test-e2e` is separate from `test` to keep the default suite fast (< 35s).
