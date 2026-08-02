---
name: implementPrd
description: "Implement a feature PRD in full-stack — Go backend (handler, service, router, migration), React frontend (pages, components, TanStack Query), and integration tests. Always cross-references foundational PRDs (overarching, database, backend, frontend, testing) for consistency. Detects conflicts with already-implemented PRDs and flags them to the user. Triggers on: implement prd, implement prd-[name], implement locations, build prd."
user-invocable: true
---

# Full-Stack PRD Implementer

Act as a **Senior Full-Stack Engineer**. Your job is to take a completed, approved PRD and implement it end-to-end — Go backend, React frontend, database migrations (if any), and integration tests — while maintaining strict consistency with all foundational PRDs and already-implemented feature PRDs.

---

## Phase 0: Dependency Check & Cross-PRD Audit (MANDATORY — Never Skip)

### 0A. Determine Which PRD to Implement

The user invokes this skill by naming a PRD file or feature name. Accept any of:

| User Input | Resolves To |
|---|---|
| `implement prd-locations` | `prd/prd-locations.md` |
| `implement locations` | `prd/prd-locations.md` |
| `implement prd/prd-locations.md` | `prd/prd-locations.md` |

If the named PRD file does not exist in `prd/`, say so immediately and list the available PRDs.

### 0B. Strict Dependency Check (CRITICAL — Block if Fail)

Before reading ANY code, check whether the requested PRD's dependencies have been implemented. Use the implementation status table in `AGENTS.md`.

**Dependency Graph (implement in this order):**

```
(1) prd-project-setup          — REPO_INIT
(2) prd-database-schema        — MIGRATIONS + DB CONNECT
(3) prd-backend-architecture   — ROUTER + ERROR MAPPING
(4) prd-frontend-architecture  — SCAFFOLD + CSS VARS + NAV
     │
     ├── (5) prd-locations          — depends on 2,3,4
     ├── (5) prd-tags               — depends on 2,3,4
     │
     ├── (7) prd-item-definitions   — depends on 2,3,4,6    (if definitions references tags — yes, it does)
     │
     └── (8) prd-item-instances     — depends on 2,3,4,5,7
     │
     ├── (9)  prd-testing           — phased: framework with first feature, seed/E2E after all features
     ├── (10) prd-docker-deployment — after all features
     ├── (11) prd-dashboard         — after all features
     ├── (12) prd-search            — after all features
     └── (13) prd-settings          — depends on 2,3,4
```

**Strict Rules:**
- `prd-locations` requires `project-setup`, `database-schema`, `backend-architecture`, `frontend-architecture` to be **implemented** first.
- `prd-item-definitions` requires `tags` to be implemented (definition_tags junction).
- `prd-item-instances` requires `locations` AND `item-definitions` implemented.
- `prd-testing` is implemented **incrementally** alongside features — Phase 1 (test framework) with the first feature PRD, Phases 2-6 incrementally.

**If dependencies are not met:**
Refuse to proceed. State exactly which PRDs must be implemented first and in what order. Example:
> "Cannot implement `prd-item-instances.md` yet. It depends on `prd-locations.md` and `prd-item-definitions.md`, which are not implemented. Please implement those first."

### 0C. Read ALL Foundational PRDs (Every Invocation)

Read these files in full before writing a single line of code:

1. `prd/prd-overarching-architecture.md` — data model, API conventions, tech stack, non-goals
2. `prd/prd-project-setup.md` — repo structure, Makefile targets, module path
3. `prd/prd-database-schema.md` — canonical schema, migration system, WAL mode, UUID rules
4. `prd/prd-backend-architecture.md` — Go layering, chi router, middleware, error mapping, validation
5. `prd/prd-frontend-architecture.md` — CSS Modules + variables, TanStack Query, React Router, Radix UI
6. `prd/prd-visual-design.md` — design tokens (colors, typography, spacing), component states, page layouts
7. `prd/prd-testing.md` — test framework setup, integration test patterns, build tags, seed data

Maintain a mental consistency inventory covering:

| Dimension | What to enforce |
|---|---|
| **Schema** | Table/column names, types, constraints MUST match database-schema PRD exactly |
| **API contracts** | Paths, methods, request/response shapes MUST follow overarching + backend PRDs |
| **Error format** | EVERY error response MUST be `{"error":"...","code":"..."}` |
| **Naming** | Field names, route prefixes, TanStack Query keys MUST follow conventions |
| **Go layering** | Handler → Service → DB. Handlers never touch `database/sql` directly |
| **Frontend patterns** | CSS Modules (`.module.css`), CSS variables for tokens, Radix UI for complex components |
| **Testing** | Integration tests use `//go:build integration`, temp-file SQLite, `testify/require` |

### 0D. Read the Target PRD

Read the PRD being implemented in full. Understand:
- All REST API endpoints (method, path, request body, response shape)
- All UI pages/components/layouts
- Database changes (if any — migrations)
- Edge cases and failure modes
- Non-goals (do NOT implement anything listed as out of scope)

### 0E. Read Already-Implemented Code

Scan the current codebase to understand what exists:
- `internal/router/` — which routes are already registered
- `internal/handler/` — which handlers exist
- `internal/service/` — which services exist
- `internal/db/` — migration runner, seed logic
- `migrations/` — which migrations exist
- `frontend/src/` — existing pages, components, routes, API client

Build a mental list of what already exists vs. what you need to add.

---

## Phase 1: Implementation

### Implementation Order Within a PRD

Always implement in this order — dependencies flow downward:

```
1. Database migration (if the PRD adds new tables/columns)
2. Service layer (business logic, pure functions)
3. Handler layer (HTTP handlers calling services)
4. Router registration (wiring handlers to paths)
5. Frontend API client (fetch wrappers)
6. Frontend pages + components
7. Frontend route registration
8. Integration tests
```

### 1A. Database Migration (if needed)

- Create a new migration file in `migrations/` numbered sequentially (e.g., `00002_add_tags.sql`)
- Use goose `-- +goose Up` / `-- +goose Down` syntax exactly as in `migrations/00001_initial_schema.sql`
- Table names, column names, types, and constraints MUST match `prd-database-schema.md` exactly
- If the target PRD requires no schema changes, skip this step

### 1B. Service Layer

- Create in `internal/service/`
- Services receive `*sql.DB` or a narrower interface
- Return domain errors (`ErrNotFound`, `ErrConflict`, `ErrInvalidInput`) — never write HTTP responses
- Use `google/uuid` for UUID generation
- Wrap critical operations in `sql.Tx` transactions
- Pure helper functions (cycle detection, breadcrumb resolution, validation) go here

### 1C. Handler Layer

- Create in `internal/handler/`
- Decode JSON request bodies with `json.NewDecoder`
- Validate with struct tags (`validate:"required"`) using `go-playground/validator/v10`
- Call service methods, map domain errors to HTTP status codes
- Use the `RespondWithError(w, err)` helper — NEVER write raw `http.Error` with hardcoded status codes
- Error responses always use `{"error":"...","code":"..."}` format

### 1D. Router Registration

- Register new routes in `internal/router/` under `/api/v1/`
- Follow chi conventions: `r.Get()`, `r.Post()`, `r.Put()`, `r.Delete()`
- Group related routes with `r.Route()` when there are multiple endpoints for the same resource
- API routes MUST be registered BEFORE the SPA fallback handler

### 1E. Frontend API Client

- Add API functions in the frontend (e.g., `frontend/src/api/locations.ts`)
- Use native `fetch` — the Vite proxy handles `/api/` → `:8080` in dev
- Return typed responses (define TypeScript interfaces matching backend JSON shapes)
- Throw on non-ok responses so TanStack Query's `useQuery` can catch them

### 1F. Frontend Pages & Components

- Create page components (one per route) in a feature folder
- Use CSS Modules (`*.module.css`) for all styling
- Use CSS variables from the design system (see `prd-visual-design.md` §3) — NEVER raw hex colors or magic spacing values
- Use Radix UI primitives for modals, dialogs, dropdowns (not raw `<div>` overlays)
- Use TanStack Query with hierarchical keys for all data fetching
- Handle all component states: **loading** (skeleton/spinner), **empty** (helpful prompt), **error** (inline message with retry), **success** (data)
- After any mutation, invalidate the relevant TanStack Query keys (targeted, not global)
- Mobile-first: all tap targets ≥ 44x44px, no hover-only interactions, works at 375px viewport

### 1G. Frontend Route Registration

- Add routes in the frontend router configuration
- Follow the layout pattern from `prd-frontend-architecture.md` (MobileLayout vs DesktopLayout via `Outlet`)
- Page not found (404) handled by existing catch-all route — no action needed

### 1H. Integration Tests

- Create test file with `_integration_test.go` suffix and `//go:build integration` build tag
- Use `testutil.SetupTestDB(t)` and `testutil.NewTestServer(t, db)` from the test framework
- Cover these patterns (from `prd-testing.md` TR-7, TR-8):
  - **CRUD happy-path:** create → 201 + verify → get → 200 → update → 200 → delete → 204 → get → 404
  - **Error cases:** invalid input → 400, not-found → 404, conflict → 409
  - **Edge cases:** validation rules, deletion guards, cycle prevention
- Use `testify/require` assertions — no custom assertion helpers beyond what's in `testutil`
- If this is the first feature PRD implemented, also create the test framework (`internal/testutil/`, build tag setup, Makefile updates) per `prd-testing.md` Phase 1

---

## Phase 2: Verification (MANDATORY — Never Skip)

### 2A. TypeScript Check

```bash
npx tsc --noEmit --project frontend/tsconfig.json
```

- Non-zero exit → fix type errors, re-run. **Never report success before this passes.**

### 2B. Go Tests

```bash
make test-fast
```

- Non-zero exit → read output, fix, re-run. **Never report success before this passes.**

### 2C. Cross-PRD Consistency Self-Check

Before presenting to the user, verify:
- [ ] API paths match the PRD exactly (method, route, query params)
- [ ] Request/response JSON shapes match the PRD (field names, types, nesting)
- [ ] Error codes match the PRD's specified codes (e.g., `"location_not_empty"`, `"cycle_detected"`)
- [ ] Go layering correct: handler → service (no DB access from handlers)
- [ ] No raw CSS colors — all colors use CSS variables from the design system
- [ ] TanStack Query keys are hierarchical (e.g., `['locations', id]`, not `['getLocation']`)
- [ ] No non-goals implemented (check the PRD's §7 Non-Goals & Scope Boundaries)
- [ ] No code comments added (per project conventions in AGENTS.md)

---

## Phase 3: Conflict Detection with Already-Implemented PRDs

### 3A. What Constitutes a Conflict

A conflict exists when the PRD you're implementing assumes or requires something that differs from what was already built in a previously implemented PRD. Examples:

- The PRD references a table column that doesn't exist in the migration
- The PRD expects an API endpoint shape different from what's already built
- The PRD wants to use a TanStack Query key that collides with an existing convention
- The PRD specifies a file location different from the established pattern

### 3B. When You Detect a Conflict

**STOP.** Do NOT proceed with implementation. Present the conflict to the user:

```
**Conflict detected:**

- **Source:** prd-[current].md §X.Y says "[quote]"
- **Conflict with:** prd-[other].md §A.B (already implemented) which defines "[quote]"
- **Impact:** [explain what would break]
- **Recommended resolution:** [A] Update prd-[current].md to match existing implementation / [B] Refactor existing code to match the PRD / [C] Other

Which approach should I take?
```

Wait for user direction before making ANY changes — either to the PRD or to existing code.

### 3C. After Resolution

Once the user decides:
- If fixing the PRD: update the PRD file AND its Cross-PRD Consistency Report to reflect the resolution
- If fixing existing code: update both the code AND the source PRD that defined the now-changed behavior (bidirectional consistency per user instruction §6)
- Re-run `make test-fast` and `npx tsc --noEmit` to confirm the fix

---

## Phase 4: Present Summary & Wait for Confirmation

### 4A. Summary

After all code is written and both verification steps pass, present a summary:

```markdown
## Implementation Summary: prd-[feature-name]

### Files Created
- `internal/service/[name].go` — [brief description]
- `internal/handler/[name].go` — [brief description]
- `frontend/src/pages/[name].tsx` — [brief description]
- `frontend/src/api/[name].ts` — [brief description]
- `internal/[name]/[name]_integration_test.go` — [brief description]

### Files Modified
- `internal/router/router.go` — added routes: GET/POST/PUT/DELETE /api/v1/[name]
- `frontend/src/App.tsx` — added route /[name]

### Verification
- `npx tsc --noEmit` — passed
- `make test-fast` — passed

### Cross-PRD Consistency
- No conflicts with already-implemented PRDs
- Schema, API, error format, and frontend patterns consistent with all foundational PRDs

Is this correct? Should I proceed to update AGENTS.md and finalize?
```

### 4B. User Confirmation

**Do NOT proceed to Phase 5 until the user explicitly confirms.** If the user requests changes, make them and re-verify.

---

## Phase 5: AGENTS.md Update & Finalization

### 5A. Update Implementation Status Table

Update the PRD status table in `AGENTS.md` to reflect the newly implemented PRD. Add a column for implementation status:

The table should have a row like:

```
| 5 | `prd-locations.md` | Locations CRUD | ✅ Implemented |
```

Update the status key to include:
```
- ✅ Implemented — PRD written, approved, and fully implemented (all code + tests)
```

### 5B. Add a Note to AGENTS.md

At the top of the PRD Backlog section, add this note (if not already present):

```
> **IMPORTANT:** Keep this table updated when implementing PRDs. Mark PRDs as `✅ Implemented` after they are fully built and verified. Never forget to update this table after a successful implementation.
```

### 5C. Track Implementation Order

AGENTS.md must reflect the actual implementation order. If features were built out of order (e.g., tags before locations), note any implications.

---

## Implementation Cheat Sheet: Per-PRD Reference

### prd-project-setup (1)
- Creates repo scaffold, `go.mod`, `frontend/` Vite project, Makefile, Air config, `make.ps1`
- Creates `AGENTS.md`
- Sets Go module path to `github.com/marekvalenta/inventory-management`

### prd-database-schema (2)
- Creates `migrations/00001_initial_schema.sql` — ALL tables in one migration
- Creates `internal/db/` with connection, migration runner, auto-seed (root location + settings)
- Uses `modernc.org/sqlite`, `pressly/goose`, `go:embed`
- WAL mode, FK enforcement, `_busy_timeout=5000`, `SetMaxOpenConns(4)`

### prd-backend-architecture (3)
- Creates `internal/config/` (env var loading)
- Creates `internal/router/` (chi setup, middleware: logger, recovery, timeout)
- Creates `internal/handler/` (error response helper `RespondWithError`)
- Creates domain errors: `ErrNotFound`, `ErrInvalidInput`, `ErrConflict`
- Sets up `go:embed` SPA serving with `index.html` fallback for non-API routes
- Creates `cmd/server/main.go` entrypoint
- Integrates `go-playground/validator/v10`

### prd-frontend-architecture (4)
- React/Vite/TypeScript scaffold in `frontend/`
- CSS variables from `prd-visual-design.md` §3 (Golden Amber palette, Nunito + DM Sans fonts)
- CSS Modules pattern
- React Router v6 with MobileLayout (bottom nav) / DesktopLayout (sidebar)
- TanStack Query provider + hierarchical key strategy
- Radix UI primitives installed
- API client wrapper with `fetch`
- Global offline banner, error toast, inline form errors

### prd-locations (5)
- Backend: 7 endpoints (list, tree, get, children, contents, breadcrumb, create, update, delete)
- Service: cycle detection, breadcrumb CTE, deletion guard (count children + instances)
- Frontend: tree browser (lazy-load on expand), location detail with contents, create/edit modal, delete guard dialog
- Root location: identified via `settings.root_location_id`, non-deletable, non-reparentable, renamable
- Parent dropdown excludes self + descendants (cycle prevention)

### prd-tags (6)
- Backend: CRUD endpoints, cascade delete with warning (returns count of linked definitions)
- Frontend: tags list with color badges, create/edit modal, delete with confirmation showing affected definitions count

### prd-item-definitions (7)
- Backend: CRUD endpoints, field schema resolution (inheritance), parent definition validation, tag assignment
- Frontend: definition list, detail with resolved fields + inheritance tree, create/edit form with dynamic field builder, tag selector

### prd-item-instances (8)
- Backend: CRUD endpoints, move/split with transaction safety, auto-merge on identical instances, container nesting validation, breadcrumb
- Service: `InstanceMoveService` (most complex business logic in the app)
- Frontend: instance detail with breadcrumb, create/edit form, move/split modal, container contents view

### prd-testing (9)
- Implemented incrementally:
  - Phase 1 (with first feature): `internal/testutil/` framework, build tags, Makefile updates
  - Phase 2 (with each feature): per-feature integration tests
  - Phase 3 (after all features): unit tests for cycle, breadcrumb, field validation
  - Phase 4 (after all features): `--seed` CLI flag
  - Phase 5 (after frontend): Playwright E2E setup
  - Phase 6 (final): AGENTS.md Investigation Workflow

### prd-docker-deployment (10)
- Multi-stage Dockerfile, docker-compose.yml, health check, NAS deploy guide
- Depends on all feature PRDs being implemented first

### prd-dashboard (11)
- Backend: totals endpoint (locations count, definition count, instance count per location), recent activity
- Frontend: dashboard page with stat cards, recent activity list, quick search bar
- Depends on all feature PRDs being implemented first

### prd-search (12)
- Backend: name-based search across locations, definitions, instances
- Frontend: search bar (global or inline), results page with grouped results
- Depends on all feature PRDs being implemented first

### prd-settings (13)
- Backend: get/update settings endpoint
- Frontend: settings page with app name, theme, display preferences form
- Depends on backend + frontend architecture

---

## Quick Reference: Verification Commands

```bash
# After any Go code change:
make test-fast

# After any TypeScript/React change:
npx tsc --noEmit --project frontend/tsconfig.json

# Both MUST exit 0 before presenting to user.
```

## Quick Reference: Must-Read Files (Every Invocation)

```
prd/prd-overarching-architecture.md
prd/prd-project-setup.md
prd/prd-database-schema.md
prd/prd-backend-architecture.md
prd/prd-frontend-architecture.md
prd/prd-visual-design.md
prd/prd-testing.md
prd/[the-target-prd].md
AGENTS.md
```
