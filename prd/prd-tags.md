# PRD: Tags — InventoryManagement

> **Status:** Draft v1.0
> **Scope:** Tags CRUD (API + UI), flat list with inline editing, cascade delete with user confirmation, tag color badges.

---

## 0. Cross-PRD Consistency Report

### PRDs Reviewed
- `prd-overarching-architecture.md` — High-level data model, API conventions, frontend views, tags scope (definitions only).
- `prd-database-schema.md` — Canonical `tags` and `definition_tags` table schemas.
- `prd-backend-architecture.md` — Go layering (handler/service/db), chi router, error mapping, payload validation.
- `prd-frontend-architecture.md` — CSS Modules, TanStack Query, React Router v6, Radix UI, mobile/desktop layouts.
- `prd-locations.md` — Reference for CRUD patterns, form validation, deletion flow (differs: Locations uses RESTRICT, Tags uses CASCADE).
- `prd-project-setup.md` — Dev tooling (no direct impact).

### Conflicts & Resolutions

| # | Conflict | PRDs Involved | Resolution |
|---|---|---|---|
| 1 | Overarching PRD §4.2 and database PRD §6 originally stated tags cannot be deleted if in use (409). This conflicted with the `definition_tags.tag_id ON DELETE CASCADE` FK in the schema. | prd-overarching-architecture.md, prd-database-schema.md | **Updated both PRDs.** Tags use **cascade delete** with user confirmation. Unlike locations (RESTRICT), the user is warned how many definitions are linked and may proceed. The schema already uses `ON DELETE CASCADE` for `tag_id` — no migration change needed. |
| 2 | OQ-2 in overarching PRD asked whether tags should have descriptions. | prd-overarching-architecture.md | **Resolved.** Tags use name + color (hex) only. No description field. Updated overarching PRD. |

### Confirmed Alignments
- Data model: Uses `tags` table exactly as defined in `prd-database-schema.md` TR-3 (name, color — no description).
- FK cascade: `definition_tags.tag_id` uses `ON DELETE CASCADE` — consistent with the confirmed cascade-delete design.
- API patterns: `/api/v1/` prefix, JSON bodies, UUID IDs, `{"error":"...","code":"..."}` error format per overarching §5.
- Error mapping: `ErrNotFound` → 404, `ErrConflict` → 409 (for duplicate name), `ErrInvalidInput` → 400 per `prd-backend-architecture.md` TR-2.
- UI: Radix UI primitives + CSS Modules + TanStack Query per `prd-frontend-architecture.md`.
- Mobile-first: Inline forms, 44x44px tap targets, bottom nav→sidebar layout.
- Scope: Tags apply to item definitions only (not instances, not locations) — per overarching PRD §4.2 and OQ-4.
- Scope does not contradict any non-goal in overarching PRD.

---

## 1. Overview & Problem Statement

Tags are lightweight, user-defined labels that categorize item definitions. They enable filtering, grouping, and at-a-glance visual identification of related inventory items. Unlike locations (hierarchical, hard-deletion-guarded), tags are a **flat namespace** with a **cascade-delete** policy — users are warned of linked definitions but may delete a tag, which silently removes the association.

### Core Deliverables
1. REST API: Create, read, update, delete, list tags with linked-definition counts.
2. Name uniqueness enforcement (globally, across all tags).
3. Cascade-delete with linked-definition count in response.
4. Frontend: flat list view with inline create/edit forms and color badge display.
5. Delete confirmation dialog when tag is in use.

---

## 2. Goals & Measurable Success Metrics

| Goal | Metric |
|------|--------|
| Full CRUD | All tag operations complete in < 100ms p95 |
| Name uniqueness | 100% of duplicate-name create/update attempts rejected with 409 |
| Cascade safety | Delete always succeeds; association count returned to client |
| Visual clarity | Tag badges are distinguishable by color on both mobile and desktop |
| Mobile usability | Inline forms usable on 375px viewport |

---

## 3. Critical Risks & Technical Assumptions

| Risk | Mitigation |
|------|------------|
| Duplicate tag name race condition | SQLite `UNIQUE` constraint on `name` column + transaction wrapping on create/update. |
| Accidental tag deletion | Frontend shows confirmation dialog with linked-definitions count before issuing DELETE. Service returns count in response for audit trail. |
| Color validation (arbitrary strings) | Accept any non-empty string up to 10 chars. Frontend validates hex format (`#XXXXXX`); backend is lenient for future flexibility. |
| Many definitions linked to one tag | `linked_definitions_count` computed via a lightweight `COUNT(*)` subquery. Fast even with thousands of definitions. |

---

## 4. User Stories & Acceptance Criteria

### US-001: Create a Tag
**Description:** As a user, I want to create a new tag with a name and optional color so I can categorize my item definitions.

**Acceptance Criteria:**
- [ ] `POST /api/v1/tags` with `{ "name": "...", "color": "..." }` creates a tag.
- [ ] `name` is required (2–100 chars), globally unique.
- [ ] `color` is optional (hex string, e.g. `"#FF5733"`). Max 10 chars.
- [ ] Returns `201 Created` with the new tag JSON including generated UUID, timestamps, and `linked_definitions_count: 0`.
- [ ] Duplicate name returns `409 Conflict` with `{"error": "Tag 'Fasteners' already exists", "code": "duplicate_name"}`.
- [ ] Invalid input returns `400 Bad Request` with field-level error messages.
- [ ] Typecheck / build / test suite passes.

### US-002: List All Tags
**Description:** As a user, I want to see all my tags in a flat, alphabetically sorted list so I can browse and manage them.

**Acceptance Criteria:**
- [ ] `GET /api/v1/tags` returns a flat array of all tags sorted by `name` ascending.
- [ ] Each tag includes `id`, `name`, `color`, `linked_definitions_count`, `created_at`.
- [ ] No pagination (assumes < 50 tags).
- [ ] Empty list returns `[]` with 200 OK.
- [ ] Typecheck / build / test suite passes.

### US-003: View a Single Tag
**Description:** As a user, I want to view a single tag's details including how many definitions use it.

**Acceptance Criteria:**
- [ ] `GET /api/v1/tags/:id` returns the tag JSON including `linked_definitions_count`.
- [ ] Non-existent tag returns `404 Not Found`.
- [ ] Typecheck / build / test suite passes.

### US-004: Update a Tag
**Description:** As a user, I want to rename a tag or change its color.

**Acceptance Criteria:**
- [ ] `PUT /api/v1/tags/:id` updates `name` and/or `color` (partial updates — only send changed fields).
- [ ] Renaming to an already-used name returns `409 Conflict`.
- [ ] Non-existent tag returns `404 Not Found`.
- [ ] `updated_at` is set on change.
- [ ] Typecheck / build / test suite passes.

### US-005: Delete a Tag (Cascade)
**Description:** As a user, I want to delete a tag, understanding that its associations with item definitions will be removed.

**Acceptance Criteria:**
- [ ] `DELETE /api/v1/tags/:id` always deletes the tag. All rows in `definition_tags` for this tag cascade-delete via FK.
- [ ] Response includes `linked_definitions_count` (the number of definition-tag associations removed).
- [ ] If no definitions were linked, `linked_definitions_count` is 0.
- [ ] Non-existent tag returns `404 Not Found`.
- [ ] Typecheck / build / test suite passes.

### US-006: Tags List View (UI)
**Description:** As a user, I want to see all my tags displayed as colored badges in a clean list.

**Acceptance Criteria:**
- [ ] Route `/tags` renders a responsive list of tags.
- [ ] Each tag is displayed as a colored badge/pill: a small color circle/swatch + the tag name.
- [ ] Each tag row has an edit (pencil) and delete (trash) action button.
- [ ] An "Add Tag" button is prominently placed at the top of the list.
- [ ] Empty state: "No tags yet — add your first tag" with a call-to-action button.
- [ ] Mobile: full-width list, 44x44px action buttons.
- [ ] Desktop: centered max-width list (~600px).
- [ ] **[UI]** Verified in browser on both 375px and 1920px viewports.

### US-007: Inline Create/Edit Tag Form (UI)
**Description:** As a user, I want to add or edit tags inline without navigating to a separate page.

**Acceptance Criteria:**
- [ ] "Add Tag" expands an inline form at the top of the list with: Name (text input), Color (text input with live preview swatch, accepting hex codes like `#FF5733`), and Save/Cancel buttons.
- [ ] Clicking edit on a tag replaces it with an inline form pre-filled with current values.
- [ ] Inline validation errors appear under each field on blur/submit.
- [ ] Saving successfully collapses the form back to the normal list view and invalidates the `['tags']` TanStack Query cache.
- [ ] Cancel discards changes and collapses the form.
- [ ] The color input shows a small square swatch next to the text field that updates live as the user types a valid hex code.
- [ ] **[UI]** Verified in browser.

### US-008: Delete Tag with Confirmation (UI)
**Description:** As a user, I want to be warned before deleting a tag that is in use by item definitions.

**Acceptance Criteria:**
- [ ] Clicking delete on a tag opens a confirmation dialog (Radix Dialog/AlertDialog).
- [ ] If `linked_definitions_count > 0`, the dialog says: `"Tag 'Fasteners' is used by 5 item definitions. Deleting it will remove this tag from all of them."` with Cancel and "Delete Anyway" buttons.
- [ ] If `linked_definitions_count === 0`, the dialog simply asks: `"Delete tag 'Office'?"` with Cancel and Delete buttons.
- [ ] On confirm: tag is deleted, removed from the list, query cache invalidated, success toast shown.
- [ ] On error: error toast with the backend error message.
- [ ] **[UI]** Verified in browser.

---

## 5. Functional & Technical Requirements

### 5.1 Database Dependencies

The `tags` and `definition_tags` tables are defined in the initial migration of `prd-database-schema.md`. No schema changes are needed — the `ON DELETE CASCADE` on `definition_tags.tag_id` is already correct for the cascade-delete design.

**FR-1:** `linked_definitions_count` is computed at query time via a subquery or LEFT JOIN, not stored as a column:

```sql
SELECT t.*,
  (SELECT COUNT(*) FROM definition_tags dt WHERE dt.tag_id = t.id) AS linked_definitions_count
FROM tags t
ORDER BY t.name ASC;
```

### 5.2 REST API Endpoints

All endpoints under `/api/v1/tags`.

| Method | Path | Description | Request Body | Response |
|--------|------|-------------|--------------|----------|
| `GET` | `/tags` | List all tags (sorted by name) | — | `Tag[]` |
| `GET` | `/tags/:id` | Single tag | — | `Tag` |
| `POST` | `/tags` | Create tag | `{ name, color? }` | `Tag` (201) |
| `PUT` | `/tags/:id` | Update tag (partial) | `{ name?, color? }` | `Tag` |
| `DELETE` | `/tags/:id` | Delete tag (cascade) | — | `{ deleted: true, linked_definitions_count: N }` |

**FR-2:** Tag response shape:

```json
{
  "id": "uuid",
  "name": "Fasteners",
  "color": "#FF5733",
  "linked_definitions_count": 5,
  "created_at": "2026-08-01T12:00:00Z",
  "updated_at": "2026-08-01T12:00:00Z"
}
```

**FR-3:** `POST /tags` validates:
- `name`: required, 2–100 characters, globally unique.
- `color`: optional, max 10 characters. No format validation on backend (lenient).

**FR-4:** `PUT /tags/:id` supports partial updates. Only provided fields are changed. Same validation as POST. If `name` is unchanged (same value), uniqueness check passes. If `name` is changed to an existing tag's name, return 409.

**FR-5:** `DELETE /tags/:id` must:
1. Count linked definitions (`SELECT COUNT(*) FROM definition_tags WHERE tag_id = ?`).
2. Delete the tag row (cascade handles junction table).
3. Return the count in the response body.

**FR-6:** `GET /tags` returns all tags. No pagination. No filtering (v1). Sorted by `name ASC`.

### 5.3 Service Layer

**FR-7:** `TagService` (in `internal/service/`) implements:
- `Create(ctx, name, color) (*Tag, error)` — validates, checks uniqueness, generates UUID, inserts.
- `GetAll(ctx) ([]Tag, error)` — selects all with linked_definitions_count, sorted by name.
- `GetByID(ctx, id) (*Tag, error)` — selects one by ID with linked_definitions_count.
- `Update(ctx, id, name, color) (*Tag, error)` — partial update with uniqueness check.
- `Delete(ctx, id) (int, error)` — returns linked_definitions_count before deleting.

**FR-8:** Duplicate name error uses `errors.Is(err, ErrConflict)` → handler maps to HTTP 409.

### 5.4 Handler Layer

**FR-9:** Handlers in `internal/handler/tag.go` follow the standard pattern from `prd-backend-architecture.md`:
- Decode JSON → validate struct tags → call service → format response.
- Use `RespondWithError(w, err)` for consistent error JSON.
- Use `google/uuid` for ID generation.

### 5.5 Router Registration

**FR-10:** Tag routes registered in `internal/router/` under `r.Route("/api/v1/tags", ...)`:

```go
r.Get("/", tagHandler.List)
r.Get("/{id}", tagHandler.Get)
r.Post("/", tagHandler.Create)
r.Put("/{id}", tagHandler.Update)
r.Delete("/{id}", tagHandler.Delete)
```

### 5.6 Frontend

**FR-11:** Single route: `/tags`. Renders `TagsPage` component.

**FR-12:** Use TanStack Query with keys:
- `['tags']` — list of all tags (sole query key; no nested keys needed since tags are flat).

**FR-13:** On any successful mutation, **invalidate the entire `['tags']` cache** (simpler than targeted invalidation for a flat list of < 50 items). Since tags are not hierarchical and the list is small, full invalidation is performant and correct.

**FR-14:** Tag badge component renders:
- A small colored circle/swatch (16x16px, `border-radius: 50%`, `background-color: tag.color || '#CCCCCC'`).
- The tag name beside it.
- Edit (pencil icon) and delete (trash icon) action buttons on the right.

**FR-15:** Inline form state is managed with local React state (`useState`). No separate route or modal needed. Only one form can be open at a time (either create-new OR editing one tag).

**FR-16:** Color input includes a live preview swatch. As the user types a valid hex code (matches `/^#[0-9A-Fa-f]{6}$/`), the swatch updates. Client-side validation only; server is lenient.

**FR-17:** Forms use HTML5 validation as first line + controlled component validation before submit.

**FR-18:** `linked_definitions_count` is shown as a small badge next to the tag (e.g., "5 definitions") when > 0, or omitted when 0.

---

## 6. Edge Cases & Failure Modes

| Scenario | System Behavior |
|----------|----------------|
| User creates a tag with a duplicate name | Server returns `409 Conflict` with `"code": "duplicate_name"`. Frontend shows inline error under name field. |
| User renames a tag to its current name | Server detects name unchanged → no uniqueness check → success (no-op for name field). |
| User submits empty or whitespace-only name | Server rejects with `400 Bad Request` (`"code": "validation_failed"`, field: `name`). |
| User deletes a tag that was just unlinked from the last definition | Race condition possible: count read before delete shows N, but cascade removes 0 rows. Harmless — response just reports N (stale count). |
| User enters invalid hex color (e.g., `"red"` or `"abc"`) | Backend accepts any string up to 10 chars (lenient). Frontend only renders swatch if hex format matches; otherwise shows default grey. |
| User creates a tag with no color | `color` stored as `NULL`. Frontend renders default grey swatch. |
| User navigates directly to `/tags` by URL | React Router renders `TagsPage`. TanStack Query fetches `['tags']`. |
| Network fails during tag save | TanStack Query's `onError` fires. Inline form remains open with the typed values preserved. Error toast displays. |
| User tries to edit or delete a non-existent tag (stale UI state) | Server returns `404 Not Found`. Frontend shows error toast and refreshes the list. |
| User deletes a tag while another user (future multi-user) is viewing it | Currently single-user, non-issue. Tag disappears from list on next invalidation. |

---

## 7. Non-Goals & Scope Boundaries

- **Tag descriptions:** Deliberately excluded — tags are name + color only.
- **Tag hierarchy / nesting:** Tags are flat. No parent tags, no sub-tags.
- **Instance-level tags:** Tags apply to item definitions only (per overarching PRD §4.2).
- **Location-level tags:** Tags apply to item definitions only.
- **Tag filtering on definitions page:** This is defined in PRD #7 (Item Definitions) — the filtering UI uses tags created here.
- **Bulk tag operations:** No multi-select, batch delete, or batch create.
- **Drag-and-drop reordering:** Alphabetical sort only.
- **Full-text search on tags:** Deferred to PRD #10 (Search).
- **Tag usage analytics:** No reporting on most-used tags, etc.

---

## 8. Open Questions & Deferred Items

| # | Question | Status |
|---|----------|--------|
| OQ-1 | Should there be an `updated_at` column on the `tags` table? | Resolved — `updated_at` added to the initial migration in `prd-database-schema.md`. Consistent with locations and item_definitions. |
| OQ-2 | Should the color field support a color picker widget in addition to text input? | Deferred — v1 uses text input with hex validation. A color picker can be added later. |
| OQ-3 | Should deleting a tag have a soft-delete option (undo)? | Deferred — v1 is hard delete only. Undo would require confirmation/timeout UX not warranted for < 50 tags. |
