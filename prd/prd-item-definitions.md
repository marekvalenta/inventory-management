# PRD: Item Definitions — InventoryManagement

> **Status:** Draft v1.0
> **Scope:** Item Definitions CRUD (API + UI), dynamic field schema with inheritance, enum field type, field overrides on children, tag assignment, instance summary per definition.

---

## 0. Cross-PRD Consistency Report

### PRDs Reviewed
- `prd-overarching-architecture.md` — Data model, definition/instance separation, inheritance concept, tags on definitions.
- `prd-database-schema.md` — Canonical `item_definitions`, `definition_fields`, `definition_tags` tables.
- `prd-backend-architecture.md` — Go layering, chi router, error mapping, payload validation.
- `prd-frontend-architecture.md` — CSS Modules, TanStack Query, React Router v6, Radix UI, mobile/desktop layouts.
- `prd-locations.md` — Reference for CRUD patterns, deletion guard flow.
- `prd-tags.md` — Tag data model and API for tag assignment on definitions.

### Conflicts & Resolutions

| # | Conflict | PRDs Involved | Resolution |
|---|---|---|---|
| 1 | `parent_def_id` on `item_definitions` has no explicit `ON DELETE` clause. Need to prevent deletion of parent definitions with inheriting children. | prd-database-schema.md | **Change `parent_def_id REFERENCES item_definitions(id)` → `ON DELETE RESTRICT`.** Matches the hard-block deletion pattern for definitions. |
| 2 | `definition_fields` lacks `default_value`, `is_child_editable`, and `enum_values` columns needed for inheritance model and enum type support. | prd-database-schema.md | **Add three columns to `definition_fields`** in the initial migration. No separate migration needed since no code exists yet. |
| 3 | `enum` field type is not listed in the schema comment (`'text', 'number', 'boolean', 'date'`). | prd-database-schema.md | **Add `'enum'` as a supported `field_type`.** Requires `enum_values TEXT` column (JSON array). |
| 4 | No table exists for child definition field overrides. | prd-database-schema.md | **Add `definition_field_overrides` table.** Stores child's overridden `default_value` for inherited fields with `is_child_editable = true`. |

### Confirmed Alignments
- `item_instances.definition_id` uses `ON DELETE RESTRICT` — correctly blocks deletion of definitions with existing instances.
- `definition_fields.definition_id` uses `ON DELETE CASCADE` — correct: deleting a definition removes its own fields.
- `definition_tags` uses `ON DELETE CASCADE` for both columns — consistent with cascade-delete design confirmed in Tags PRD.
- Both Tag and Location deletion guards reference item instances/definitions as blockers — no circular conflicts.
- Scope does not contradict any non-goal in overarching PRD.

---

## 1. Overview & Problem Statement

Item Definitions are the "class" layer of the inventory system. They define *what* a tracked item is — its name, unit of measurement, custom field schema, and classification tags. Definitions support **dynamic inheritance**: a child definition inherits the parent's full field schema and may add its own fields or (when permitted) override inherited defaults. This enables modeling like `Fastener → Screw → M3 Screw` where each level adds specificity.

### Core Deliverables
1. REST API for definitions CRUD including nested field and tag management.
2. Dynamic field resolution: inherited fields are computed at query time, not duplicated.
3. Child field override support via `definition_field_overrides`.
4. Enum field type with predefined value lists.
5. Tag assignment as part of definition create/update.
6. Instance summary on detail endpoint (counts by location).
7. Frontend: list view → detail page with field table, tag badges, and instance preview.

---

## 2. Goals & Measurable Success Metrics

| Goal | Metric |
|------|--------|
| Full CRUD | All definition operations < 200ms p95 |
| Field resolution | 100% of inherited fields correctly resolved through arbitrary-depth parent chain |
| Inheritance safety | 100% of parent-delete attempts with children blocked (409) |
| Enum integrity | Instance field values validated against definition's enum_values |
| Tag consistency | Definition detail always shows accurate tag list |
| Instance summary | Detail page loads instance counts in same request as definition data |

---

## 3. Critical Risks & Technical Assumptions

| Risk | Mitigation |
|------|------------|
| Deep inheritance chains causing N+1 field resolution | Resolve entire parent chain in a single recursive CTE or Go tree-walk with caching per request. |
| Schema change on existing data | Since no production data exists, include all new columns in the initial migration. Update `prd-database-schema.md` migration to include them. |
| Field type validation on instances | Instance creation must validate field values against definition field types (especially `enum` and `number`). This is enforced by the Instance service in PRD #8. |
| Deleting a parent definition after it's been used by instances | Instances reference their specific definition_id, not parent. Deleting a parent that has NO direct instances but has child definitions IS blocked if any child definition has instances. This is correct — the child can't exist without knowing its parent's fields. |
| Circular inheritance | Cycle detection at create/update time: before setting `parent_def_id`, walk up the parent chain to verify the target is not a descendant. |

---

## 4. User Stories & Acceptance Criteria

### US-001: Create a Definition with Fields and Tags
**Description:** As a user, I want to create a new item definition with custom fields and tags so I can define what properties my items have.

**Acceptance Criteria:**
- [ ] `POST /api/v1/definitions` with `{ "name", "description?", "parent_def_id?", "unit?", "is_container?", "fields": [...], "tag_ids": [...] }` creates a definition.
- [ ] `name`: required, 2–200 chars, globally unique.
- [ ] `description`: optional, max 2000 chars.
- [ ] `parent_def_id`: optional. If provided, must reference an existing definition. No cycle check needed for root-creation (parent must exist).
- [ ] `unit`: optional, free text, max 20 chars (e.g. `"pcs"`, `"kg"`, `"m"`).
- [ ] `is_container`: optional boolean (default `false`). If `true`, instances of this definition can contain other instances (item-in-item nesting).
- [ ] `fields`: array of field objects (see US-006 for field shape). At least 1 field recommended but not required.
- [ ] `tag_ids`: array of existing tag UUIDs.
- [ ] Returns `201 Created` with the full definition including resolved fields, tags, and `instances_summary: { total_instances: 0, total_quantity: 0, by_location: [] }`.
- [ ] Duplicate name returns `409 Conflict`.
- [ ] Invalid parent_def_id returns `400 Bad Request`.
- [ ] Typecheck / build / test suite passes.

### US-002: List All Definitions
**Description:** As a user, I want to browse all item definitions in a flat list so I can find what I need.

**Acceptance Criteria:**
- [ ] `GET /api/v1/definitions` returns a flat array sorted by `name` ascending.
- [ ] Each item includes `id`, `name`, `description`, `unit`, `parent_def_id`, `tags` (full tag objects), `total_instances`, `created_at`, `updated_at`.
- [ ] Fields are NOT included in the list response (too heavy — fetch detail for fields).
- [ ] No pagination in v1 (assumes < 100 definitions).
- [ ] Empty list returns `[]` with 200 OK.
- [ ] Typecheck / build / test suite passes.

### US-003: View Definition Detail
**Description:** As a user, I want to see a definition's full details including its resolved fields, tags, and instance summary.

**Acceptance Criteria:**
- [ ] `GET /api/v1/definitions/:id` returns the full definition with:
  - All own fields + inherited fields (resolved dynamically).
  - Each field includes `inherited_from_def_id` if inherited (null for own fields).
  - Full tag objects.
  - `instances_summary`: `{ total_instances, total_quantity, by_location: [{ location_id, location_name, instance_count, total_quantity }] }`.
  - `child_definition_count`: number of direct children inheriting from this definition.
- [ ] Non-existent definition returns `404 Not Found`.
- [ ] Typecheck / build / test suite passes.

### US-004: Update a Definition
**Description:** As a user, I want to edit a definition's metadata, fields, and tags.

**Acceptance Criteria:**
- [ ] `PUT /api/v1/definitions/:id` updates `name`, `description`, `parent_def_id`, `unit`, `is_container`, `fields`, and/or `tag_ids`.
- [ ] Partial updates allowed — only send changed properties.
- [ ] If `parent_def_id` is changed: cycle detection must verify the new parent is not a descendant of this definition.
- [ ] Renaming to an already-used name returns `409 Conflict`.
- [ ] `fields` array (if provided) **replaces** all own fields. Inherited fields are NOT included in the PUT body — only own fields. The server clears and re-inserts own `definition_fields` for this definition.
- [ ] `tag_ids` array (if provided) **replaces** all tag associations. The server clears and re-inserts `definition_tags` for this definition.
- [ ] `updated_at` auto-updated.
- [ ] Typecheck / build / test suite passes.

### US-005: Delete a Definition (Guarded)
**Description:** As a user, I want to delete a definition that is no longer needed.

**Acceptance Criteria:**
- [ ] `DELETE /api/v1/definitions/:id` succeeds only if:
  1. No item instances reference this definition (`ON DELETE RESTRICT` — returns 409).
  2. No child definitions inherit from this definition (`ON DELETE RESTRICT` on `parent_def_id` — returns 409).
- [ ] If blocked by instances, returns `409 Conflict` with `{ "error": "Cannot delete: definition has X item instances", "code": "definition_has_instances" }`.
- [ ] If blocked by child definitions, returns `409 Conflict` with `{ "error": "Cannot delete: X child definitions inherit from this definition", "code": "definition_has_children" }`.
- [ ] On success: returns `204 No Content`. Own fields cascade-delete via FK. Tag associations cascade-delete. Override rows cascade-delete.
- [ ] Non-existent definition returns `404 Not Found`.
- [ ] Typecheck / build / test suite passes.

### US-006: Field Inheritance & Resolution
**Description:** As a user, I want child definitions to automatically inherit their parent's fields, with the ability to override certain defaults when permitted.

**Acceptance Criteria:**
- [ ] `GET /api/v1/definitions/:id` returns a unified `fields` array containing:
  - Parent's fields (and grandparent's, recursively) marked with `inherited_from_def_id: "parent-uuid"`.
  - The definition's own fields marked with `inherited_from_def_id: null`.
- [ ] Own fields are appended AFTER inherited fields in `display_order`.
- [ ] If a parent field has `is_child_editable = true`, the child can store an override `default_value` in `definition_field_overrides`. The resolved field shows the child's override value.
- [ ] If a parent field has `is_child_editable = false`, the field is sealed — no override possible. The resolved field shows the parent's value.
- [ ] If a parent field has `is_child_editable = false` and `default_value IS NULL` and `is_required = true`, the field is "mandatory filler" — child instances MUST provide a value at creation time.
- [ ] Inherited fields in the response include the resolved `default_value` (own override > parent override > parent default).
- [ ] Typecheck / build / test suite passes.

### US-007: Manage Own Fields
**Description:** As a user, I want to add, edit, reorder, and remove custom fields on my definition.

**Acceptance Criteria:**
- [ ] Fields are managed via the definition's `PUT` endpoint — client sends the full array of own fields to replace.
- [ ] Each field in the request has: `field_name` (required, 1–100 chars, unique within the definition), `field_type` (required: `"text"`, `"number"`, `"boolean"`, `"date"`, `"enum"`), `enum_values` (required if type is `"enum"`, must be non-empty array of strings), `is_required` (boolean), `display_order` (integer), `default_value` (nullable string), `is_child_editable` (boolean, default `false`).
- [ ] `field_type = "enum"` requires `enum_values` to be a non-empty array. Server validates.
- [ ] `default_value` is validated against field_type: `"number"` must be numeric, `"boolean"` must be `"true"` or `"false"`, `"enum"` must be one of the `enum_values`.
- [ ] Duplicate `field_name` within the same definition returns `400 Bad Request`.
- [ ] `display_order` is used for sorting in the resolved fields list.
- [ ] Typecheck / build / test suite passes.

### US-008: Override Inherited Field Defaults
**Description:** As a user managing a child definition, I want to override the default value of inherited fields that permit it.

**Acceptance Criteria:**
- [ ] `PUT /api/v1/definitions/:id/overrides` accepts `{ "overrides": [ { "parent_field_id": "...", "default_value": "..." } ] }`.
- [ ] `parent_field_id` must reference a field on a direct or indirect parent that has `is_child_editable = true`.
- [ ] Override `default_value` is validated against the parent field's `field_type` and `enum_values`.
- [ ] Setting `default_value` to `null` clears the override (falls back to parent's default).
- [ ] Returns the updated list of current overrides.
- [ ] Non-existent parent_field_id returns `400 Bad Request`.
- [ ] Attempting to override a sealed field (`is_child_editable = false`) returns `400 Bad Request` with `"code": "field_sealed"`.
- [ ] Typecheck / build / test suite passes.

### US-009: Definition List View (UI)
**Description:** As a user, I want to browse all definitions in a searchable list with tag badges.

**Acceptance Criteria:**
- [ ] Route `/definitions` renders a responsive list of all definitions.
- [ ] Each row shows: name, unit (if set), tag badges (colored pills), total_instances count.
- [ ] "Add Definition" button at top navigates to the create form.
- [ ] Clicking a row navigates to `/definitions/:id` detail page.
- [ ] Empty state: "No definitions yet" with call-to-action button.
- [ ] Mobile: full-width list, tap rows to navigate.
- [ ] Desktop: centered list (~800px max-width).
- [ ] **[UI]** Verified in browser on both 375px and 1920px viewports.

### US-010: Definition Detail Page (UI)
**Description:** As a user, I want a comprehensive detail view with tabs/sections for fields, tags, and instances.

**Acceptance Criteria:**
- [ ] Route `/definitions/:id` renders the definition detail page.
- [ ] **Header section:** Name, description, unit, parent definition (linked, if set), child definition count.
- [ ] **Fields section (tab/panel):** Editable table of fields (US-011).
- [ ] **Tags section:** Tag badges with an "Add Tag" dropdown. Clicking a badge's × removes the tag association.
- [ ] **Instances section (tab/panel):** Summary showing total instances + total quantity, plus a table/list grouped by location: location name, instance count, total quantity.
- [ ] Edit button opens the definition metadata in a modal form (name, description, parent, unit).
- [ ] Delete button with confirmation dialog showing blocking counts.
- [ ] Parent chain breadcrumb at top: "Fastener > Screw > M3 Screw".
- [ ] Mobile: sections stacked vertically.
- [ ] Desktop: side-by-side or tabbed layout.
- [ ] **[UI]** Verified in browser.

### US-011: Field Management UI
**Description:** As a user, I want to add, edit, reorder, and remove fields on my definition in a visual table.

**Acceptance Criteria:**
- [ ] Fields are displayed in a table with columns: Name, Type, Required (toggle), Default Value, Child Editable (toggle, only on parent definitions), Actions.
- [ ] **Inherited fields** (from parent chain) are shown as locked/read-only rows with a visual indicator (e.g., faded background, lock icon, parent name tooltip).
- [ ] For inherited fields with `is_child_editable = true`, the "Default Value" cell is editable — changing it creates an override.
- [ ] **Own fields** are fully editable inline or via an edit modal.
- [ ] "Add Field" button adds a new empty row at the bottom.
- [ ] Delete field button with inline confirmation (no dialog for single field removal).
- [ ] Reorder via simple up/down buttons per row (drag-and-drop deferred to v2).
- [ ] All field changes are batched and sent on a "Save Fields" button (not auto-save per change).
- [ ] If the definition has no parent, the "Child Editable" column and inherited-field behavior are hidden/absent.
- [ ] **[UI]** Verified in browser.

### US-012: Delete Confirmation Dialog (UI)
**Description:** As a user, I want clear feedback when I attempt to delete a definition that is blocked.

**Acceptance Criteria:**
- [ ] Delete button triggers a confirmation dialog.
- [ ] Dialog shows counts of blocking entities: "This definition has X item instances and Y child definitions. You must remove them before deleting."
- [ ] If no blockers: "Delete definition '[name]'?" with Cancel/Delete buttons.
- [ ] If deletion succeeds: navigate back to `/definitions`, invalidate cache, show success toast.
- [ ] If deletion blocked (409): dialog closes, error toast with blocking counts.
- [ ] **[UI]** Verified in browser.

---

## 5. Functional & Technical Requirements

### 5.1 Schema Changes

The following changes must be applied to the initial migration in `prd-database-schema.md`:

**FR-1:** Add columns to `definition_fields`:
```sql
CREATE TABLE definition_fields (
    id              TEXT PRIMARY KEY,
    definition_id   TEXT NOT NULL REFERENCES item_definitions(id) ON DELETE CASCADE,
    field_name      TEXT NOT NULL,
    field_type      TEXT NOT NULL, -- 'text', 'number', 'boolean', 'date', 'enum'
    enum_values     TEXT,            -- NEW: JSON array of strings, required when field_type = 'enum'
    is_required     BOOLEAN NOT NULL DEFAULT 0,
    display_order   INTEGER NOT NULL DEFAULT 0,
    default_value   TEXT,            -- NEW: default value for instances
    is_child_editable BOOLEAN NOT NULL DEFAULT 0  -- NEW: can child definitions override default_value?
);
```

**FR-2:** Change `parent_def_id` FK to `ON DELETE RESTRICT`:
```sql
parent_def_id TEXT REFERENCES item_definitions(id) ON DELETE RESTRICT,
```

**FR-3:** Add `definition_field_overrides` table:
```sql
CREATE TABLE definition_field_overrides (
    definition_id    TEXT NOT NULL REFERENCES item_definitions(id) ON DELETE CASCADE,
    parent_field_id  TEXT NOT NULL REFERENCES definition_fields(id) ON DELETE CASCADE,
    default_value    TEXT,
    PRIMARY KEY (definition_id, parent_field_id)
);
```

### 5.2 REST API Endpoints

All endpoints under `/api/v1/definitions`.

| Method | Path | Description | Request Body | Response |
|--------|------|-------------|--------------|----------|
| `GET` | `/definitions` | List all definitions | — | `DefinitionSummary[]` |
| `GET` | `/definitions/:id` | Single definition (resolved) | — | `DefinitionDetail` |
| `POST` | `/definitions` | Create definition | `{ name, description?, parent_def_id?, unit?, is_container?, fields?, tag_ids? }` | `DefinitionDetail` (201) |
| `PUT` | `/definitions/:id` | Update definition | `{ name?, description?, parent_def_id?, unit?, is_container?, fields?, tag_ids? }` | `DefinitionDetail` |
| `DELETE` | `/definitions/:id` | Delete (guarded) | — | 204 or 409 |
| `PUT` | `/definitions/:id/overrides` | Update field overrides | `{ overrides: [{ parent_field_id, default_value }] }` | `{ overrides: [...] }` |

**FR-4:** `DefinitionSummary` (list response):
```json
{
  "id": "uuid",
  "name": "Screw",
  "description": "Various screws",
  "parent_def_id": "uuid or null",
  "parent_def_name": "Fastener or null",
  "unit": "pcs",
  "is_container": false,
  "total_instances": 42,
  "tags": [
    { "id": "uuid", "name": "Fasteners", "color": "#FF5733" }
  ],
  "created_at": "...",
  "updated_at": "..."
}
```

**FR-5:** `DefinitionDetail` (single-get + create/update response):
```json
{
  "id": "uuid",
  "name": "Screw",
  "description": "...",
  "parent_def_id": "uuid or null",
  "parent_def_name": "Fastener or null",
  "unit": "pcs",
  "is_container": false,
  "created_at": "...",
  "updated_at": "...",
  "fields": [
    {
      "id": "uuid",
      "field_name": "Material",
      "field_type": "enum",
      "enum_values": ["Steel", "Brass", "Aluminum"],
      "is_required": true,
      "display_order": 0,
      "default_value": "Steel",
      "is_child_editable": true,
      "inherited_from_def_id": null
    }
  ],
  "tags": [
    { "id": "uuid", "name": "Fasteners", "color": "#FF5733" }
  ],
  "instances_summary": {
    "total_instances": 42,
    "total_quantity": 150,
    "by_location": [
      {
        "location_id": "uuid",
        "location_name": "Workshop",
        "instance_count": 3,
        "total_quantity": 85
      }
    ],
    "by_parent_instance": [
      {
        "parent_instance_id": "uuid",
        "parent_instance_name": "Toolbox #3",
        "location_id": "uuid",
        "location_name": "Workshop",
        "instance_count": 4,
        "total_quantity": 20
      }
    ]
  },
  "child_definition_count": 2
}
```

**FR-6:** Field resolution logic (service layer):
1. Query definition's own fields from `definition_fields`.
2. Walk up the parent chain recursively.
3. For each inherited field, check `definition_field_overrides` for a child-specific `default_value`.
4. Merge: own fields first, then inherited fields appended in chain order (closest ancestor first).
5. Return unified array with `inherited_from_def_id` set for inherited fields.

**FR-7:** `POST /definitions` validates:
- `name`: required, 2–200 chars, globally unique.
- `description`: optional, max 2000 chars.
- `parent_def_id`: optional, must exist. Cycle check: parent must not be a descendant of this definition (trivially satisfied on create since this is a new definition).
- `unit`: optional, max 20 chars.
- `fields`: optional array. Each field validates per US-007 rules.
- `tag_ids`: optional array. Each must reference an existing tag.

**FR-8:** `PUT /definitions/:id` validates:
- Same as POST, plus: if `parent_def_id` changes, cycle detection must walk up the parent chain to verify target is not a descendant.
- If `fields` is provided, replaces all own fields. Inherited fields remain resolved dynamically and are not stored on this definition.

**FR-9:** `DELETE /definitions/:id` checks:
1. Count of child definitions (`SELECT COUNT(*) FROM item_definitions WHERE parent_def_id = ?`).
2. Count of item instances (`SELECT COUNT(*) FROM item_instances WHERE definition_id = ?`).
3. Both checks in a single transaction with the DELETE.

### 5.3 Service Layer

**FR-10:** `DefinitionService` (in `internal/service/`) implements:
- `Create(ctx, input) (*DefinitionDetail, error)` — validate, generate UUID, insert definition + fields + tags.
- `GetAll(ctx) ([]DefinitionSummary, error)` — list with tags joined, sorted by name.
- `GetByID(ctx, id) (*DefinitionDetail, error)` — single with resolved fields, tags, instances summary, child count.
- `Update(ctx, id, input) (*DefinitionDetail, error)` — partial update with cycle check, field replacement, tag replacement.
- `Delete(ctx, id) error` — guarded delete with blocker counts in error.
- `UpdateOverrides(ctx, id, overrides) ([]Override, error)` — validate override eligibility, upsert overrides.

**FR-11:** `ResolveFields(ctx, definitionID)` — internal helper:
- Recursively walks `parent_def_id` chain.
- Fetches own fields. Fetches parent fields + overrides.
- Returns unified sorted field list with `inherited_from_def_id`.

### 5.4 Handler Layer

**FR-12:** Handlers in `internal/handler/definition.go` follow the standard backend architecture pattern.

**FR-13:** Validation structs use `go-playground/validator/v10` tags. Custom validators for:
- Field type enum (`oneof=text number boolean date enum`)
- Enum values present when type is `enum` (custom validator)
- Default value matches type (custom validator)

### 5.5 Router Registration

**FR-14:** Definition routes registered under `r.Route("/api/v1/definitions", ...)`:

```go
r.Get("/", definitionHandler.List)
r.Get("/{id}", definitionHandler.Get)
r.Post("/", definitionHandler.Create)
r.Put("/{id}", definitionHandler.Update)
r.Delete("/{id}", definitionHandler.Delete)
r.Put("/{id}/overrides", definitionHandler.UpdateOverrides)
```

### 5.6 Frontend

**FR-15:** Routes:
- `/definitions` — `DefinitionListPage`
- `/definitions/:id` — `DefinitionDetailPage`
- `/definitions/new` — `DefinitionCreatePage` (or create via modal on list page)

**FR-16:** TanStack Query keys:
- `['definitions']` — flat list
- `['definitions', id]` — single definition detail
- `['definitions', id, 'overrides']` — field overrides for a child definition

**FR-17:** On any mutation, invalidate `['definitions']` and the specific `['definitions', id]` if applicable.

**FR-18:** Detail page uses tabs or sections:
- Fields tab: FieldTable component (editable table with inherited/own distinction)
- Tags tab: Tag assignment with search/dropdown
- Instances tab: InstanceSummary component (read-only grouped list)

**FR-19:** Field table renders:
- Own fields: full edit controls (inline text inputs, type dropdown, toggles, delete button)
- Inherited sealed fields: locked rows, greyed out, lock icon
- Inherited overridable fields: locked rows except default_value cell (editable)
- "Add Field" button: adds a row with empty inputs
- "Save Fields" button: batches all changes and sends PUT to the definition endpoint
- Reorder: up/down arrow buttons per row

**FR-20:** Tag assignment uses a multi-select pattern:
- Tags already assigned: shown as removable badge-chips with × button
- Add tag: dropdown/combobox (Radix Popover or Select) with search-as-you-type, showing unassigned tags
- Changes auto-saved (or batched with a save button)

**FR-21:** Parent definition dropdown in create/edit form must exclude:
- The definition itself (can't be own parent)
- All descendants of this definition (prevents cycles)
- Null option available to clear parent (make root-level)

**FR-22:** InstanceSummary component displays:
- A summary header: "42 instances (150 total items) across 5 locations"
- A table/list grouped by location with columns: Location Name, Instances, Total Quantity
- Each location row is clickable, navigating to that location's detail page
- Empty state: "No instances of this definition yet"

---

## 6. Edge Cases & Failure Modes

| Scenario | System Behavior |
|----------|----------------|
| User creates a definition with a non-existent tag ID | Server returns `400 Bad Request` with `"code": "tag_not_found"`. |
| User creates a child definition with fields matching inherited field names | Server returns `400 Bad Request` with `"code": "duplicate_field_name"` — own fields must not collide with inherited field names. |
| User sets `parent_def_id` to a descendant (cycle) | Server walks ancestor chain. Returns `400 Bad Request` with `"code": "cycle_detected"`. |
| User sets `parent_def_id` to itself | Server returns `400 Bad Request` with `"code": "self_parent"`. |
| User overrides a sealed field | Server returns `400 Bad Request` with `"code": "field_sealed"`. |
| User sets an override `default_value` that violates the enum values | Server returns `400 Bad Request` with `"code": "invalid_enum_value"`. |
| User deletes a parent definition that has children with instances | Blocked by `ON DELETE RESTRICT` on `parent_def_id`. Server returns 409 with child count. Facts: the children may or may not have instances — doesn't matter, the restrict blocks it either way. |
| Parent definition's field is deleted or type is changed | Dynamic inheritance resolves the current state. Changing a parent field's type could invalidate existing child overrides. Server validates overrides against current field type on read; invalid overrides are silently ignored (field falls back to parent default). |
| Grandparent chain of 10 levels deep | Service resolves entire chain in one recursive walk. Performance test: < 50ms for 10-level chain. |
| User deletes a field from a definition that has children inheriting from it | Children lose the inherited field on next GET (dynamic resolution). No cascade issues since inherited fields are not stored on children. Override rows for that field cascade-delete via FK. |
| User renames a definition to match an existing one | Returns `409 Conflict` with `"code": "duplicate_name"`. |
| User creates a definition with zero fields | Allowed. The definition acts as a simple "name + unit" template. Instances of this definition have no custom fields. |
| Definition has 100+ instances across 50+ locations | `instances_summary.by_location` groups via a SQL GROUP BY query. Performance is fine with proper indexing. |
| User navigates directly to `/definitions/:id` for a non-existent definition | React Router renders the detail page. TanStack Query fetches, gets 404. Page shows "Definition not found" state. |

---

## 7. Non-Goals & Scope Boundaries

- **Drag-and-drop field reordering:** v1 uses up/down buttons. Deferred to v2.
- **Field-level validation on instances:** Defined in this PRD's schema but enforced in PRD #8 (Item Instances).
- **Complex field types (date pickers, rich text, file uploads):** v1 supports text/number/boolean/date/enum as string-typed values. Date is a string in ISO format.
- **Definition-level images/photos:** Deferred to future photo attachment feature.
- **Versioning / change history for definitions:** Not in v1.
- **Bulk operations:** No multi-select, batch delete, or import/export of definitions.
- **Field value formulas / computed fields:** Not in scope.
- **Definition duplication / clone:** Not in v1.
- **Inline field editing with auto-save:** v1 uses explicit "Save Fields" button to batch changes.

---

## 8. Open Questions & Deferred Items

| # | Question | Status |
|---|----------|--------|
| OQ-1 | Should `field_name` collision check between own and inherited fields be case-insensitive? | Deferred to implementation — case-sensitive by default, revisit if user feedback demands it. |
| OQ-2 | Should the create-definition form be a modal on the list page or a separate route? | Deferred to implementation. Recommend separate `/definitions/new` route for complex field/tag editing during creation. |
| OQ-3 | Should the field table auto-save each change or batch-save? | Resolved — batch save via explicit "Save Fields" button for v1. |
| OQ-4 | Instance summary grouping: should "item inside item" instances (nested) be counted under the location or under the parent instance? | Deferred to PRD #8 (Item Instances). For the definition detail, count direct instances only grouped by `location_id`. |
