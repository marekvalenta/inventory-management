# PRD: Settings — InventoryManagement

> **Status:** Done v1.0
> **Scope:** Settings page (route `/settings`) with theme selector; typed-struct Settings service + handler; migration to drop `app_name`; app-wide rename to "Itema".

---

## 0. Cross-PRD Consistency Report

### PRDs Reviewed
- `prd-overarching-architecture.md` — Settings endpoint group (§5.2), Settings view (§6.1), env vars (§7.1), data model singleton (§4.1), non-goals (§11).
- `prd-database-schema.md` — `settings` table with `app_name`, `theme`, `root_location_id`, auto-seed of settings row on first boot.
- `prd-backend-architecture.md` — Go layering, chi router, error mapping, payload validation, SPA embedding.
- `prd-frontend-architecture.md` — React Router v6, TanStack Query, CSS Modules, Radix UI, mobile/desktop layouts, OQ-1 theme toggle deferred to this PRD.
- `prd-visual-design.md` — Golden Amber tokens, mobile bottom nav (4th tab: Settings), desktop sidebar (Settings below separator), button/card/form patterns, no settings page layout defined.
- `prd-locations.md` — Location CRUD, uses `root_location_id` from settings for root detection.
- `prd-dashboard.md` — Dashboard route `/`, stat cards, navigation to root location.
- `prd-search.md` — Header search bar visible on all pages except dashboard; Settings page inherits header search bar.
- `prd-item-instances.md` — Instance CRUD, move/split, references root location.
- All other PRDs (tags, definitions, testing, docker) — no direct settings impact beyond app name references.

### Conflicts & Resolutions

| # | Conflict | PRDs Involved | Resolution |
|---|---|---|---|
| 1 | **Dual source of truth for `app_name`**: Config env var `APP_NAME` vs settings DB column `app_name`, both defaulting to "Inventory". | `prd-overarching-architecture.md` §7.1, `prd-database-schema.md`, `internal/config/config.go`, `internal/db/seed.go` | **Drop `app_name` entirely.** Remove `APP_NAME` env var from config. Remove `app_name` column from settings table via new migration. Hardcode the app name to "Itema" in the frontend header/sidebar. The app name is not user-configurable in v1. |
| 2 | **No settings page layout in visual design PRD**: Visual design §4 shows Settings as a nav tab but §5 provides no page layout. | `prd-visual-design.md` §4, §5 | **This PRD defines the Settings page layout** as a simple form page using the existing card + form input patterns from visual PRD §6.1–6.3. |
| 3 | **`theme` column exists but no API or UI to manage it**: DB has `theme` column (default 'system') but visual PRD says v1 is dark-only. Frontend architecture OQ-1 defers theme toggle to Settings PRD. | `prd-database-schema.md`, `prd-visual-design.md`, `prd-frontend-architecture.md` OQ-1 | **Show theme in a dropdown, only "Dark" option in v1.** The column value is changed from "system" to "dark". The dropdown renders the available theme but only lists "Dark" as an option. When light mode is added in v2, the dropdown gains a "Light" option and the theme actually switches. The struct + API contract is designed for easy future extension. |
| 4 | **No `/api/v1/settings` endpoint exists**: Listed in overarching §5.2 but not implemented. No handler, service, or frontend page. | `prd-overarching-architecture.md` §5.2 | **Create GET + PUT endpoints.** `GET /api/v1/settings` returns the full typed settings struct. `PUT /api/v1/settings` replaces it. Singular, not plural — `settings` is a singleton resource. |
| 5 | **App name "Inventory" is referenced across all PRDs and code**: Overarching architecture, env var, seed data, Docker compose example, README, all PRD docs. | All PRDs, `internal/config/config.go`, `internal/db/seed.go` | **Rename application to "Itema".** Update all PRDs that reference the app name. Update the hardcoded header/sidebar display text. Remove `APP_NAME` env var. Update Docker compose example. The app is always called "Itema". |

### Confirmed Alignments
- Data model: Settings remains a singleton table (`CHECK id=1`). `app_name` column dropped. `theme` column kept, default changed to `'dark'`. `root_location_id` kept for internal use only.
- API patterns: `GET /api/v1/settings` and `PUT /api/v1/settings` under `/api/v1/`, returns JSON, uses standard error format `{"error":"...","code":"..."}`.
- Go layering: `SettingsService` (service) → `SettingsHandler` (handler) → registered in `chi` router. Follows `prd-backend-architecture.md` TR-1.
- Navigation: Settings is the 4th tab in mobile bottom nav and desktop sidebar per visual PRD §4.
- UI: CSS Modules + CSS variables (Golden Amber tokens), TanStack Query, React Router v6, `<input>` + `<select>` form patterns from visual PRD §6.1–6.2, card pattern §6.3.
- Visibility: Settings page shows the header search bar (unlike dashboard which suppresses it) per `prd-search.md` conflict #8.
- Scope: Single-user app — no authentication, no permissions. Settings are writable by anyone who can access the UI.

---

## 1. Overview & Problem Statement

The Settings page provides a place for user-configurable application preferences. In v1, the only visible setting is the **theme selector**, which currently only supports "Dark" mode (the app's one and only theme). The architecture is designed so future settings — light mode toggle, compact mode, default view, items per page, etc. — can be added by extending a typed Go struct, adding a column to the settings table, and adding a form field to the Settings page. No refactoring required.

### Core Deliverables
1. **Rename application to "Itema"** — remove the configurable `app_name` column and `APP_NAME` env var. Hardcode the name everywhere.
2. **Migration `00003_drop_app_name.sql`** — drops the `app_name` column from the `settings` table. Updates existing `theme` values from 'system' to 'dark'.
3. **Settings service + handler** — `GET /api/v1/settings` and `PUT /api/v1/settings` with a typed `Settings` struct.
4. **Settings page** — route `/settings`, form card with theme dropdown. Follows visual PRD patterns but defines its own page layout (visual PRD §5 has no settings layout).
5. **Extensibility design** — typed struct + explicit columns pattern. Adding a setting in the future requires: (a) new column via migration, (b) add field to `Settings` struct, (c) add form input to Settings page.

---

## 2. Goals & Measurable Success Metrics

| Goal | Metric |
|---|---|
| Single source of truth | Settings data loads via one `GET /api/v1/settings` call returning a single typed struct |
| Fast response | Settings API responds in < 20ms (single-row SQLite read) |
| Consistent app naming | All UI text, PRDs, and code use "Itema" — zero references to "Inventory" remain |
| Clean migration | New migration drops `app_name` cleanly with proper `-- +goose Down` for rollback |
| Extensible | Adding a new setting requires: migration (1 file), struct field (1 line), handler (0 lines), form input (1 component change). No API contract changes. |
| No breaking API changes | Existing endpoints and frontend routes continue to work |
| Visual consistency | Settings page uses Golden Amber tokens, card pattern, consistent with all other pages |

---

## 3. Critical Risks & Technical Assumptions

| Risk | Mitigation |
|---|---|
| Migration `ALTER TABLE DROP COLUMN` may fail on old SQLite versions | `modernc.org/sqlite` (pure Go SQLite driver) supports `ALTER TABLE DROP COLUMN` since SQLite 3.35.0+. The Go driver version in use includes this. If unsupported, migration fails on startup with a clear error. |
| `theme` value "dark" has no visual effect in v1 | By design. The dropdown shows "Dark" as the only option. The `theme` value is stored and returned by the API but triggers no visual change in v1. It's a placeholder for v2 when light mode is added. |
| Hardcoded "Itema" name requires code changes if the user wants to rename | By design. Single-user NAS app — the app name is the brand. Future: if renaming is desired, a new setting can be added (e.g., `display_name`). |
| $2 | — |

---

## 4. User Stories & Acceptance Criteria

### US-001: Settings API Endpoints (GET + PUT)

**Description:** As a frontend, I want to read and update application settings via a single typed endpoint.

**Acceptance Criteria:**
- [ ] `GET /api/v1/settings` returns the full settings object:
  ```json
  { "theme": "dark" }
  ```
- [ ] `PUT /api/v1/settings` accepts and validates a settings object. Only `theme` is accepted in v1:
  ```json
  { "theme": "dark" }
  ```
- [ ] `theme` validation: only `"dark"` is accepted in v1. Any other value returns `400 Bad Request` with `{"error": "Invalid theme: 'light'. Valid themes: dark", "code": "invalid_input"}`.
- [ ] `theme` is required in PUT request. Missing `theme` returns `400 Bad Request`.
- [ ] Unknown fields in the PUT request body are accepted and ignored (forward-compatible for future settings additions).
- [ ] Response for both GET and PUT is the current settings object after the operation.
- [ ] `root_location_id` is NOT included in the JSON response — it's internal only.
- [ ] Response time < 20ms.
- [ ] Typecheck / build / test suite passes.

### US-002: Settings Page (UI)

**Description:** As a user, I want a Settings page accessible from the bottom nav (mobile) or sidebar (desktop) where I can see and change my preferences.

**Acceptance Criteria:**
- [ ] Route `/settings` renders the `SettingsPage` component.
- [ ] The page is reachable via the 4th tab in mobile bottom nav (`[=]` icon) and the "Settings" item in desktop sidebar.
- [ ] The active nav state highlights "Settings" when on `/settings`.
- [ ] The header search bar is visible on the Settings page (unlike the dashboard).
- [ ] Page layout: a centered form card with a header and form fields.
- [ ] The form card contains:
  - **Page title:** "Settings" (`--text-h1`, same as other pages per visual PRD §5.7 unified list page layout).
  - **Theme dropdown:** Label "Theme" (`--text-body-strong`), a `<select>` dropdown styled per visual PRD §6.2 form inputs, with one option: "Dark". Pre-filled with the current value from the API.
  - **Save button:** Primary variant (`--color-accent` bg), positioned below the form fields. Disabled when no change has been made. Enabled when the dropdown value differs from the current API value.
- [ ] **Loading state:** Skeleton form card (single card with two shimmer bars representing label + dropdown) per visual PRD §6.8.
- [ ] **Error state:** Inline error card per visual PRD §6.9 with retry button that refetches settings.
- [ ] **Save flow:**
  - Clicking Save fires `PUT /api/v1/settings` via TanStack Query `useMutation`.
  - On success: a brief success toast ("Settings saved") appears per visual PRD §6.6 toast pattern. The Save button becomes disabled again (current value matches saved value).
  - On error: inline error message below the Save button. The dropdown value is preserved.
- [ ] TanStack Query key: `['settings']` (single key for the entire settings object).
- [ ] `staleTime: 60_000` (1 minute) — settings rarely change.
- [ ] **[UI]** Verified in browser on both 375px and 1920px viewports.
- [ ] Typecheck / build / test suite passes.

### US-003: Database Migration — Drop `app_name`

**Description:** As the system, I need a migration that removes the `app_name` column from the settings table and updates existing `theme` values.

**Acceptance Criteria:**
- [ ] New migration file `migrations/00003_drop_app_name.sql`:
  ```sql
  -- +goose Up
  -- +goose StatementBegin
  UPDATE settings SET theme = 'dark' WHERE theme = 'system';
  ALTER TABLE settings DROP COLUMN app_name;
  -- +goose StatementEnd

  -- +goose Down
  -- +goose StatementBegin
  ALTER TABLE settings ADD COLUMN app_name TEXT NOT NULL DEFAULT 'Inventory';
  -- +goose StatementEnd
  ```
- [ ] Up migration: updates existing `theme='system'` rows to `'dark'`, then drops `app_name` column.
- [ ] Down migration: re-adds `app_name` column with default `'Inventory'` (rollback preserves data integrity).
- [ ] Migration runs on startup via goose. Idempotent — safe to run on a DB that already has the column dropped (goose tracks version).
- [ ] Typecheck / build / test suite passes.

### US-004: Config Cleanup — Remove `APP_NAME`

**Description:** As a developer, I want the `APP_NAME` env var and `Config.AppName` field removed since the app name is now hardcoded.

**Acceptance Criteria:**
- [ ] `Config.AppName` field removed from `internal/config/config.go`.
- [ ] `APP_NAME` env var reading removed from `Load()`.
- [ ] All references to `cfg.AppName` removed from the codebase.
- [ ] Docker compose example in overarching PRD updated to not include `APP_NAME`.
- [ ] Typecheck / build / test suite passes.

### US-005: App-Wide Rename to "Itema"

**Description:** As the application, I should display "Itema" as the name everywhere — header, sidebar, page title, favicon, and documentation.

**Acceptance Criteria:**
- [ ] Frontend header (mobile) displays "Itema" as the app name link.
- [ ] Frontend sidebar (desktop) displays "Itema" as the app/brand name.
- [ ] HTML `<title>` tag is "Itema" (in `index.html`).
- [ ] All PRDs updated: replace "Inventory" with "Itema" where it refers to the application name.
- [ ] `AGENTS.md` updated: project name changed to "Itema".
- [ ] Overarching architecture PRD §3 docker-compose example: remove `APP_NAME` line.
- [ ] `README.md` updated: project name changed to "Itema".
- [ ] Dashboard PRD: references to "Inventory" in header wireframes updated to "Itema".
- [ ] Visual design PRD: header and sidebar ASCII wireframes updated to show "Itema".
- [ ] Database schema PRD: seed code snippet updated to not include `app_name`.
- [ ] Typecheck / build / test suite passes.

---

## 5. Functional & Technical Requirements

### 5.1 Database Changes

**FR-1:** New migration `migrations/00003_drop_app_name.sql`:
```sql
-- +goose Up
-- +goose StatementBegin
UPDATE settings SET theme = 'dark' WHERE theme = 'system';
ALTER TABLE settings DROP COLUMN app_name;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE settings ADD COLUMN app_name TEXT NOT NULL DEFAULT 'Inventory';
-- +goose StatementEnd
```

**FR-2:** Updated settings table schema (after migration):
```sql
CREATE TABLE settings (
    id               INTEGER PRIMARY KEY CHECK (id = 1),
    theme            TEXT NOT NULL DEFAULT 'dark',
    root_location_id TEXT REFERENCES locations(id),
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**FR-3:** Seed code update (`internal/db/seed.go`):
```go
// Before:
"INSERT INTO settings (id, app_name, theme, root_location_id) VALUES (1, ?, ?, ?)",
"Inventory", "system", rootID,

// After:
"INSERT INTO settings (id, theme, root_location_id) VALUES (1, ?, ?)",
"dark", rootID,
```

**FR-4:** Test utilities update (`internal/testutil/db.go`): The `CREATE TABLE settings` in test schema and `SeedRootLocation` function must be updated to match the new schema (no `app_name` column, `theme` default `'dark'`).

### 5.2 REST API Endpoints

| Method | Path | Description | Request Body | Response |
|---|---|---|---|---|
| `GET` | `/settings` | Read settings | — | `Settings` object |
| `PUT` | `/settings` | Update settings | `Settings` object | Updated `Settings` object |

**FR-5:** `Settings` struct (Go service layer):
```go
type Settings struct {
    Theme string `json:"theme" validate:"required,oneof=dark"`
}
```

- `root_location_id` is NOT included in this struct. It remains internal to the locations service.
- Future settings are added as new fields on this struct with their own validation tags.
- `validate` tag: `oneof=dark` ensures only "dark" is accepted. When light mode is added, change to `oneof=dark light`.

**FR-6:** `GET /api/v1/settings` response:
```json
{
    "theme": "dark"
}
```

**FR-7:** `PUT /api/v1/settings` request and response:
```json
// Request
{
    "theme": "dark"
}

// Response (200 OK — same shape, reflects saved state)
{
    "theme": "dark"
}
```

**FR-8:** Validation errors:
- Missing `theme`: `400 Bad Request` — `{"error": "Field 'theme' is required", "code": "validation_failed"}`
- Invalid `theme`: `400 Bad Request` — `{"error": "Invalid theme: 'light'. Valid themes: dark", "code": "invalid_input"}`
- Malformed JSON: `400 Bad Request` — `{"error": "Invalid JSON: ...", "code": "invalid_json"}`

**FR-9:** Unknown fields in PUT request body are silently ignored. This ensures forward compatibility — a newer frontend can send additional fields that an older backend will simply skip, and vice versa.

### 5.3 Service Layer

**FR-10:** `SettingsService` in `internal/service/settings.go`:

```go
type SettingsService struct {
    db *sql.DB
}

type Settings struct {
    Theme string `json:"theme" validate:"required,oneof=dark"`
}

func (s *SettingsService) Get(ctx context.Context) (*Settings, error) {
    // SELECT theme FROM settings WHERE id = 1
    // Returns ErrNotFound if no settings row exists
}

func (s *SettingsService) Update(ctx context.Context, input *Settings) (*Settings, error) {
    // Validate input
    // UPDATE settings SET theme = ?, updated_at = CURRENT_TIMESTAMP WHERE id = 1
    // Returns the updated Settings
}
```

**FR-11:** Error handling:
- No settings row (impossible after seed, but defensive): `ErrNotFound` → handler maps to 500.
- Update returns the fresh `Settings` object after the write.

### 5.4 Handler Layer

**FR-12:** `SettingsHandler` in `internal/handler/settings.go`:

```go
type SettingsHandler struct {
    svc *service.SettingsService
}

func NewSettingsHandler(svc *service.SettingsService) *SettingsHandler

// GET /api/v1/settings → calls svc.Get(), returns 200 + Settings JSON
// PUT /api/v1/settings → decodes JSON, validates, calls svc.Update(), returns 200 + updated Settings JSON
func (h *SettingsHandler) RegisterRoutes(r chi.Router)
```

- Routes registered under `/api/v1/settings`:
  ```go
  r.Get("/", h.Get)
  r.Put("/", h.Update)
  ```

### 5.5 Router Registration

**FR-13:** Wire up in `internal/router/router.go` (both `New` and `NewTestRouter`):

```go
settingsSvc := service.NewSettingsService(db)
settingsHandler := handler.NewSettingsHandler(settingsSvc)
settingsHandler.RegisterRoutes(r)  // inside r.Route("/api/v1", ...)
```

### 5.6 Config Cleanup

**FR-14:** Remove from `internal/config/config.go`:
- Remove `AppName string` field from `Config` struct.
- Remove `APP_NAME` env var reading from `Load()`.

**FR-15:** Remove all `cfg.AppName` references from the codebase. If the app name is needed anywhere in Go (e.g., for an HTML meta tag), use the hardcoded string `"Itema"`.

### 5.7 Frontend

**FR-16:** Route: `/settings` → `SettingsPage` component.

**FR-17:** TanStack Query setup:
- Query key: `['settings']`
- `staleTime: 60_000` (60 seconds) — settings change rarely.
- `queryFn`: `GET /api/v1/settings`
- Mutation: `PUT /api/v1/settings` with `onSuccess` invalidating `['settings']`.

**FR-18:** `SettingsPage` component structure:

```
<SettingsPage>
  <Card>
    <h1>Settings</h1>
    <FormGroup>
      <Label>Theme</Label>
      <Select value={theme} onChange={...}>
        <option value="dark">Dark</option>
      </Select>
    </FormGroup>
    <Button type="submit" disabled={!isDirty}>
      Save
    </Button>
  </Card>
</SettingsPage>
```

**FR-19:** Visual design alignment:
- Page uses the unified list page layout per visual PRD §5.7: centered, max-width 720px on desktop.
- Card: `--color-bg-surface`, `--radius-md`, `--shadow-card`, padding `--space-xl`.
- Page title: `--text-h1` (Nunito 700), matching all other page headings.
- Form group: label `--text-body-strong`, margin-bottom `--space-sm`, select `margin-bottom: --space-xl`.
- Select/dropdown: styled per visual PRD §6.2 form inputs — bg `--color-bg-surface-alt`, border `--color-border`, `--radius-sm`, height 40px, padding `--space-md`.
- Save button: Primary variant (`--color-accent` bg, `--color-text-inverse` text), `--radius-md`, min height 40px, aligned to the right of the form area.
- Disabled Save button: `opacity: 0.4`, `cursor: not-allowed`.
- Loading state: skeleton card with shimmer per visual PRD §6.8.
- Error state: inline error card per visual PRD §6.9 with retry button.
- Success toast: per visual PRD §6.6, auto-dismiss 3 seconds.

**FR-20:** Header app name update:
- Mobile header (in `MobileLayout`): Change "Inventory" to "Itema". Text uses `--text-body-strong`, color `--color-text-primary`.
- Desktop sidebar (in `DesktopLayout`): Change "INV" / "Inventory" to "ITM" / "Itema". The sidebar logo area shows "ITM" in `--color-accent` with the full name "Itema" below it.
- The app name is clickable and navigates to route `/` (dashboard) in both layouts — this behavior already exists for dashboard PRD US-001.

**FR-21:** HTML `<title>` in `frontend/index.html`: Change to `<title>Itema</title>`.

**FR-22:** The Settings page inherits the persistent header search bar from `prd-search.md` US-001/US-002. On mobile, the magnifying glass icon is shown. On desktop, the inline search input is shown. The search bar is standard behavior for all pages except the dashboard.

### 5.8 Extensibility Design

**FR-23:** Adding a new setting in the future follows this 3-step pattern:

1. **Migration:** Add a new column to the `settings` table.
   ```sql
   ALTER TABLE settings ADD COLUMN compact_mode BOOLEAN NOT NULL DEFAULT 0;
   ```

2. **Service struct:** Add the field to the `Settings` struct.
   ```go
   type Settings struct {
       Theme       string `json:"theme" validate:"required,oneof=dark light"`
       CompactMode bool   `json:"compact_mode"`
   }
   ```
   Update `Get()` to `SELECT` the new column. Update `Update()` to `SET` the new column.

3. **Frontend form:** Add the corresponding form input to the `SettingsPage`.
   ```tsx
   <FormGroup>
     <Label>Compact Mode</Label>
     <Checkbox checked={settings.compact_mode} onChange={...} />
   </FormGroup>
   ```

No API contract changes — the single GET/PUT pattern accepts and returns the entire `Settings` struct. Unknown fields are ignored (FR-9). The frontend can add form inputs independently of the backend.

**FR-24:** Future candidates for settings (not in v1 scope):
- `theme`: add `"light"` option (needs light-mode CSS variables)
- `compact_mode`: toggle compact spacing density (per visual PRD OQ-3)
- `default_view`: which page to show on app open (dashboard, locations, etc.)
- `items_per_page`: pagination preference
- `confirm_deletes`: toggle delete confirmation dialogs

---

## 6. Edge Cases & Failure Modes

| Scenario | System Behavior |
|---|---|
| Settings row doesn't exist (corrupted DB, migration failed) | `GET` returns `500 Internal Server Error`. Handler logs the error. This should never happen due to auto-seed on first boot and single-row constraint. |
| PUT request with empty JSON body `{}` | `PUT` returns `400 Bad Request` — `"Field 'theme' is required"`. The settings row is not modified. |
| PUT request with extra unknown fields `{"theme":"dark","foo":"bar"}` | Unknown fields are ignored. `theme` is validated and updated. Returns `200 OK` with only `{"theme":"dark"}`. Forward-compatible. |
| SQLite write fails (disk full, permissions) | `PUT` returns `500 Internal Server Error`. Original settings row unchanged. Transaction rollback. |
| Two browser tabs both change settings concurrently | Last-write-wins. Since v1 is single-user with a single settings row, race conditions are negligible. The PUT is atomic. |
| User navigates to `/settings` directly (bookmark) | React Router renders `SettingsPage`. TanStack Query fetches `['settings']`. Loading state shows briefly, then form appears. |
| Settings API returns an unexpected theme value (e.g., "system" from old seed) | The migration (`00003`) updates `system` → `dark`. If somehow "system" persists, the dropdown shows it as an additional `<option>` with a warning. Defensive: the PUT endpoint rejects non-"dark" values on validation. |
| Migration `00003` runs on a DB that already had `app_name` manually dropped | Goose migration versioning prevents re-running completed migrations. No impact. |
| `modernc.org/sqlite` doesn't support `ALTER TABLE DROP COLUMN` | Migration fails at startup. App logs a fatal error and exits. The migration uses `StatementBegin/End` blocks for goose transactional safety. |

---

## 7. Non-Goals & Scope Boundaries

- **Light mode:** v1 is dark-only. The theme dropdown shows only "Dark". The CSS variable architecture supports light mode via token swaps (per visual PRD §9), but no CSS is written for it.
- **Multiple themes / custom themes:** No color picker, no user-uploaded themes, no preset theme gallery. The setting is a single string with controlled values.
- **Per-user settings:** v1 is single-user. All settings are global.
- **Root location management:** `root_location_id` is internal. Users cannot view or change which location is root from the Settings page.
- **Settings import/export:** No JSON export, no backup/restore of settings. Settings are part of the SQLite DB backup.
- **Settings history/audit log:** No tracking of who changed what setting or when (beyond the `updated_at` timestamp).
- **Notification preferences:** No email alerts, push notifications, or webhook settings.
- **Integration settings:** No API keys, webhook URLs, or third-party service configuration.
- **Advanced UI customization:** No font size controls, no custom CSS injection, no dashboard layout editor.
- **Env var for app name:** Removed entirely. App is always "Itema".

---

## 8. Open Questions & Deferred Items

| # | Question | Status |
|---|---|---|
| OQ-1 | Should the `theme` dropdown also show "System" as an option (follow OS preference)? | Deferred — v1 is dark-only. "System" adds complexity (media query detection) with no benefit since light mode doesn't exist. Revisit when light mode is added. |
| OQ-2 | Should settings be accessible via a dedicated icon in the mobile header? | Resolved — no. Settings is reached via the bottom nav tab `[=]`. It's one of only 4 permanent navigation destinations. Adding a header icon would be redundant. |
| OQ-3 | Should the initial `theme` value (after seed) be "dark" or "system"? | Resolved — "dark". The migration updates existing "system" values to "dark". New seeds use "dark". This matches the visual PRD v1 constraint. |
| OQ-4 | Should `root_location_id` be writable via the API (admin-only)? | Deferred — internal read-only in v1. If multi-location root management is needed later, a dedicated endpoint or flag could be added. For now, it's set once during seed and never changes. |
| OQ-5 | Should the Settings page have a "Reset to defaults" button? | Deferred — minimal v1. With only one setting (theme = dark), reset is identical to the current value. Add when there are more settings. |
