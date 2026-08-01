# PRD: Backend Architecture — InventoryManagement

> **Status:** Draft v1.0
> **Scope:** Go project layout, router setup, middleware stack, error handling strategy, config management, and SPA embedding.

---

## 0. Cross-PRD Consistency Report

### PRDs Reviewed
- `prd-overarching-architecture.md` — Top-level architecture, module path, deployment model.
- `prd-project-setup.md` — Directory structure (`internal/`), router selection (`chi`), module init.
- `prd-database-schema.md` — Schema usage, UUID ID generation.

### Conflicts & Resolutions
| # | Conflict | PRDs Involved | Resolution |
|---|---|---|---|
| - | No conflicts detected. This PRD is consistent with all prior PRDs. | - | - |

### Confirmed Alignments
- Uses `chi` router as established in project setup.
- Error responses follow `{ "error": "...", "code": "..." }` JSON structure from overarching PRD.
- Standardized project layout using `internal/` pattern.

---

## 1. Overview & Problem Statement

This PRD defines the backend core architecture for the InventoryManagement Go API. It establishes how HTTP requests are routed, validated, and processed before hitting the business logic. It also standardizes the error handling approach, configuration management, and how the React SPA is served seamlessly from the compiled Go binary.

### Core Deliverables
1. Strongly typed configuration injection.
2. `chi` router setup with essential middleware.
3. Centralized domain-to-HTTP error mapping.
4. Declarative payload validation using struct tags.
5. `go:embed` filesystem serving with SPA fallback.

---

## 2. Goals & Measurable Success Metrics

| Goal | Metric |
|---|---|
| Type-safe Config | 100% of env vars are loaded into a typed struct at startup. App fails fast if missing. |
| Robust Validation | Invalid JSON payloads are rejected with detailed 400 Bad Request errors before reaching services. |
| SPA Support | Any `GET` request not matching `/api/*` or a real static file serves `index.html`. |
| Clean Handlers | Handlers return errors up the chain; they don't write `500 Internal Server Error` manually. |

---

## 3. Critical Risks & Technical Assumptions

| Risk | Mitigation |
|---|---|
| SPA routing shadows API | Ensure `/api/*` routes are mounted first and explicitly isolated from the SPA wildcard handler. |
| Memory leaks in requests | Apply a global Timeout middleware so hung requests are aborted automatically. |
| Validation overhead | `go-playground/validator` is highly optimized and caches reflection data, meaning the overhead is negligible for NAS deployment. |

---

## 4. User Stories & Acceptance Criteria

### US-001: Configuration Management
**Description:** As a developer, I want all environment variables parsed into a typed struct on startup so that I have autocompletion and safety when using config values in the codebase.

**Acceptance Criteria:**
- [ ] `internal/config/config.go` defines a `Config` struct (Port, AppName, DataDir, LogLevel).
- [ ] Values are read from environment, with sensible defaults if missing.
- [ ] Startup panics immediately if a strictly required, non-defaultable variable is missing or malformed.

### US-002: Router & Middleware
**Description:** As a backend system, I need a robust routing layer to handle incoming requests securely and observably.

**Acceptance Criteria:**
- [ ] Uses `github.com/go-chi/chi/v5`.
- [ ] Global middleware: Request Logger (logs path/duration) and Panic Recovery.
- [ ] Global middleware: Timeout (e.g., 30s) to prevent stalled connections.
- [ ] API routes are explicitly grouped under `/api/v1/`.

### US-003: Centralized Error Handling
**Description:** As a developer, I want to return standard domain errors from my services and have the HTTP layer automatically translate them to standard JSON responses.

**Acceptance Criteria:**
- [ ] Domain errors (e.g., `ErrNotFound`, `ErrInvalidInput`) are defined.
- [ ] A helper function or middleware in the handler layer maps these domain errors to standard HTTP status codes (404, 400, etc.).
- [ ] Error response JSON exactly matches `{ "error": "...", "code": "..." }`.

### US-004: Payload Validation
**Description:** As a backend system, I want to reject bad JSON payloads before they reach the business logic to prevent database corruption.

**Acceptance Criteria:**
- [ ] Integrates `github.com/go-playground/validator/v10`.
- [ ] All request structs use `validate` tags (e.g., `validate:"required"`).
- [ ] Handlers run validation on the decoded struct; failures result in a 400 Bad Request with a clear message.

### US-005: SPA Embedding
**Description:** As a user, I want the React app served directly by the Go binary so I don't need a separate Nginx container.

**Acceptance Criteria:**
- [ ] Go binary uses `//go:embed all:dist` (or similar) to embed the frontend build.
- [ ] `chi` router serves these static files at the root `/`.
- [ ] Fallback: If a `GET` request doesn't match an API route or a static file, it serves `index.html`.

---

## 5. Functional & Technical Requirements

### TR-1: Project Layout (Go Layering)
The application will follow a strict layer separation:
1. **Config Layer (`internal/config`)**: Pure config parsing.
2. **Router Layer (`internal/router`)**: Wires handlers to paths and injects dependencies.
3. **Handler Layer (`internal/handler`)**: Parses HTTP requests, validates JSON, calls services, formats HTTP responses.
4. **Service Layer (`internal/service`)**: Pure business logic. Knows nothing about HTTP. Returns domain errors.
5. **Database Layer (`internal/db`)**: SQLite execution.

### TR-2: Error Mapping Implementation
Define standard errors in the service layer:
```go
var (
    ErrNotFound      = errors.New("not found")
    ErrInvalidInput  = errors.New("invalid input")
    ErrConflict      = errors.New("conflict")
)
```
The handler layer will use an `appErr` wrapper or `RespondWithError(w, err)` function to unwrap and match these to `http.StatusNotFound`, `http.StatusBadRequest`, and `http.StatusConflict`.

### TR-3: SPA Fallback Logic
The fallback router must distinguish between a missing API route and a frontend route:
- Any `GET` request starting with `/api/` that misses a route -> `404 Not Found` (JSON).
- Any other `GET` request that misses a route -> Serve `index.html` (HTML).

*(Note: Serving `index.html` for everything non-API is highly recommended as it requires zero backend maintenance whenever new routes are added to the React frontend).*

---

## 6. Edge Cases & Failure Modes

| Scenario | System Behavior |
|---|---|
| User navigates to `/dashboard` directly in browser | Go router finds no static file named `dashboard`, falls back to serving `index.html`. React Router takes over and renders the dashboard. |
| App receives JSON with missing required fields | Validator catches it. Handler returns `400 Bad Request` with `{"error": "Field 'name' is required", "code": "validation_failed"}`. |
| External API tries to hit the server | Request processes normally. CORS is not enabled, so browser-based cross-origin requests will fail, but programmatic (cURL/Postman) requests will succeed. |

---

## 7. Non-Goals & Scope Boundaries

- **CORS Configuration**: Explicitly out of scope for v1. The frontend proxy in dev and the embedded server in prod eliminate the need for CORS. Deferred to future iterations if a decoupled external client is added.
- **Complex Authentication**: A stub auth middleware will be added for future expansion, but v1 will not enforce JWTs or sessions.
- **Service implementations**: This PRD covers the *architecture* of how a handler talks to a service, not the actual `Location` or `Item` logic itself.

---

## 8. Open Questions & Deferred Items

| # | Question | Status |
|---|---|---|
| OQ-1 | Should we implement structured JSON logging (e.g. `slog`) instead of standard standard library logging? | Recommended to use `log/slog` which is standard in Go 1.21+ to keep things lightweight. |
| OQ-2 | Should CORS be added later? | Deferred. Will be added only when explicitly needed by an external client. |
