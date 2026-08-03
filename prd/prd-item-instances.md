# PRD: Item Instances — InventoryManagement

> **Status:** Draft v1.0
> **Scope:** Item Instances CRUD (API + UI), smart merge logic, move/split with transaction safety, item-in-item container nesting with cycle detection, mixed instance+location breadcrumb, container contents browsing.

---

## 0. Cross-PRD Consistency Report

### PRDs Reviewed
- `prd-overarching-architecture.md` — Core data model, smart merging rules (§4.2), breadcrumb format (§6.3), API conventions (§5), views (§6.1).
- `prd-database-schema.md` — `item_instances` and `instance_field_values` tables, `chk_single_parent` XOR constraint, UUID generation.
- `prd-backend-architecture.md` — Go layering, error mapping, chi router, payload validation.
- `prd-frontend-architecture.md` — TanStack Query key strategy, Radix UI primitives, mobile constraints, CSS Modules.
- `prd-locations.md` — Breadcrumb CTE (reusable), contents endpoint pattern, deletion guard pattern (RESTRICT).
- `prd-item-definitions.md` — Field resolution, field type validation (enforced here), `instances_summary` structure, definition detail page.
- `prd-tags.md` — Tags apply to definitions only (no conflict).
- `prd-item-stacks.md` — Item Stack concept providing UI-level grouping of instances by definition+parent. Stacks are a query-time aggregation layer above individual instances. Individual instance CRUD, move/split, and breadcrumb endpoints remain unchanged. Stack-level operations (list, detail, move, bulk delete) complement individual instance operations.

### Conflicts & Resolutions

| # | Conflict | PRDs Involved | Resolution |
|---|---|---|---|
| 1 | `parent_instance_id REFERENCES item_instances(id)` has **no explicit ON DELETE clause**. Default SQLite behavior is NO ACTION (effectively RESTRICT), but this must be explicit in the schema. | prd-database-schema.md | **Change FK to `ON DELETE RESTRICT`.** Deleting a parent instance that has child instances is hard-blocked — returns 409 with child count. Matches the deletion guard pattern from locations. |
| 2 | `item_definitions` table lacks `is_container` column. Container semantics must be explicit — the user chose a definition-level boolean flag controlling which definitions' instances can contain other instances. | prd-database-schema.md, prd-item-definitions.md | **Add `is_container BOOLEAN NOT NULL DEFAULT 0`** to `item_definitions` in the initial migration. Instance creation/move validates that `parent_instance_id` targets only instances of container-enabled definitions. The definition detail/list response shapes in `prd-item-definitions.md` must include `is_container`. |
| 3 | `instances_summary` in definition detail currently only has `by_location`. Nested instances (item-in-item) have no location_id and need separate grouping. | prd-item-definitions.md | **Add `by_parent_instance` array** to the `instances_summary` response. Nested instances grouped by parent instance ID with location context (since each parent instance resolves to a location via its own breadcrumb). |

### Confirmed Alignments
- Data model: Uses `item_instances` + `instance_field_values` tables exactly as defined in `prd-database-schema.md`, with two schema additions (is_container, ON DELETE RESTRICT).
- XOR constraint (`chk_single_parent`): Enforced at schema level. An instance is either at a location or inside another instance — never both, never neither.
- API patterns: `/api/v1/instances` prefix, JSON bodies, UUID IDs, `{"error":"...","code":"..."}` format per overarching §5.
- Error mapping: `ErrNotFound` → 404, `ErrConflict` → 409, `ErrInvalidInput` → 400 per `prd-backend-architecture.md` TR-2.
- Breadcrumb: Reuses the locations recursive CTE (prd-locations FR-8) for the location portion; instance-to-location chain resolved via a Go parent-instance walk.
- Smart merging: "Same definition + same field values + same parent" → merge quantities, per overarching §4.2.
- Field value validation: Enforced here per `prd-item-definitions.md` (field type, enum values, required fields).
- Contents: Direct children only, matching `GET /locations/:id/contents` pattern from locations PRD.
- Deletion guard: Hard block (RESTRICT), same semantics as locations.
- UI: Radix UI + CSS Modules + TanStack Query, 44x44px tap targets, mobile/desktop layouts per frontend PRD.
- Scope does not contradict any non-goal in overarching PRD.

---

## 1. Overview & Problem Statement

Item Instances are the physical record layer of the inventory system. While Item Definitions define *what* a thing is, Instances represent *where and how many* of it exist. Instances track quantity, location/container placement, and field values specific to each occurrence. They support smart quantity merging, partial move/split operations, and container nesting (item-in-item, e.g., a Box of Screws). Every instance resolves to a full root-to-instance breadcrumb for at-a-glance location context.

**Complementary layer: Item Stacks** — See `prd-item-stacks.md`. Item Stacks group multiple instances of the same definition at the same parent into a single UI row. The browse tree, search results, and `/stacks` detail page operate at the stack level. Individual instance pages (`/instances/:uuid`) remain accessible as drill-down from stack detail. This PRD defines the individual instance operations that underpins all stack operations.

### Core Deliverables
1. REST API: Create (with auto-merge), read (with field values + breadcrumb), update, delete (guarded), partial move/split (transactional), list with filters, container contents, breadcrumb.
2. Smart merge: Identical instances (same definition, same field values, same parent container) share one record with summed quantity.
3. Move/split: Move N items from source to target, with full transaction safety — source reduction, target merge (or create), and source deletion on exhaustion are atomic.
4. Container nesting: Instances of container-enabled definitions can contain other instances via `parent_instance_id`. Cycle and self-containment detection on write.
5. Breadcrumb resolution: Two-phase chain walk (instance parents → location → location parents) producing a mixed type breadcrumb from root to instance.
6. Frontend: Instance detail page with breadcrumb, field values, container children list; location detail integration; definition detail instance summary.

---

## 2. Goals & Measurable Success Metrics

| Goal | Metric |
|---|---|
| Full CRUD | All instance operations < 200ms p95 |
| Move transaction safety | 100% of partial failures leave DB in pre-move state (no split-brain) |
| Merge accuracy | 100% of identical instances merged into single record at create/move time |
| Cycle prevention | 100% of invalid `parent_instance_id` assignments rejected |
| Breadcrumb correctness | Full root→instance chain always correct, computed in < 50ms |
| Container validation | 100% of attempts to nest inside non-container definition instances rejected |
| Mobile usability | Instance detail page with breadcrumb scrollable/readable on 375px viewport |

---

## 3. Critical Risks & Technical Assumptions

| Risk | Mitigation |
|---|---|
| Partial move failure leaving DB in inconsistent state (source reduced but target not created) | Entire move operation wrapped in a single SQLite transaction. Any error triggers rollback. |
| Cyclic item-in-item chains | Server walks `parent_instance_id` chain on every create/move, enforcing max-depth (50) and cycle detection. |
| Deep breadcrumb chains causing slow queries | Two-phase resolution: short instance-chain walk in Go, then a single recursive CTE for locations. |
| Field value validation inconsistency after definition field changes | Validation only enforced on create/update. Existing instances with stale/invalid field values persist until next update (schema evolution is out of scope for v1). |
| Merged instances losing field value history | No audit trail in v1 — quantity merge is a deliberate simplification. |
| SQLite write contention on rapid move operations | Single-user in v1 makes this a non-issue. WAL mode provides sufficient concurrency margin. |
| Container definition flag changed after instances exist | Changing `is_container` from true → false does NOT cascade or affect existing children. It only blocks NEW assignments. Documented non-goal. |

---

## 4. User Stories & Acceptance Criteria

### US-001: Create an Instance (with Auto-Merge)
**Description:** As a user, I want to add item instances to a location or container so I can track my inventory quantities.

**Acceptance Criteria:**
- [ ] `POST /api/v1/instances` creates an instance or merges with an existing identical instance.
- [ ] Request body: `{ "definition_id", "quantity", "location_id?", "parent_instance_id?", "field_values?" }`.
- [ ] `definition_id`: required, must reference an existing definition.
- [ ] `quantity`: required, integer > 0.
- [ ] `location_id` / `parent_instance_id`: exactly one required (XOR). Both null or both set → `400 Bad Request`.
- [ ] If `parent_instance_id` is set, the parent instance's definition must have `is_container = true`. Otherwise `400 Bad Request` with `"code": "not_a_container"`.
- [ ] `field_values`: optional array of `{ "field_id", "value" }`. Each `field_id` must belong to the definition's resolved field schema (own or inherited).
- [ ] If a required field has no `default_value` and no value is provided in `field_values`, return `400 Bad Request` with `"code": "required_field_missing"` listing missing fields.
- [ ] Field values are validated against the field's type: `number` must be numeric, `boolean` must be `"true"` or `"false"`, `enum` must be one of the enum_values. Invalid → `400 Bad Request`.
- [ ] **Auto-merge:** Before creating, the service checks if an instance already exists at this same parent (same `location_id` or same `parent_instance_id`) with the same `definition_id` AND identical field values. If found, increments its quantity. If not, creates a new instance.
- [ ] Returns `201 Created` with the merged-or-created instance JSON (including field values, breadcrumb, child count).
- [ ] Typecheck / build / test suite passes.

### US-002: View an Instance
**Description:** As a user, I want to see full details of a specific instance including its field values, breadcrumb, and (if it's a container) its direct children count.

**Acceptance Criteria:**
- [ ] `GET /api/v1/instances/:id` returns the full instance detail.
- [ ] Response includes: `id`, `definition_id`, `definition_name`, `unit`, `quantity`, `location_id`, `location_name`, `parent_instance_id`, `parent_instance_name`, `field_values` (array with field metadata), `child_instance_count`, `breadcrumb`, `created_at`, `updated_at`.
- [ ] `field_values`: each entry includes `field_id`, `field_name`, `field_type`, `enum_values` (if enum), `value`. Missing optional fields (no value stored) are included with `value: null`.
- [ ] `child_instance_count`: 0 if the definition's `is_container` is false, otherwise count of direct children.
- [ ] `breadcrumb`: ordered array from root to this instance. Each entry: `{ "id", "name", "kind": "location" | "instance" }`.
- [ ] Non-existent instance returns `404 Not Found`.
- [ ] Typecheck / build / test suite passes.

### US-003: Update an Instance
**Description:** As a user, I want to edit an instance's quantity and/or field values.

**Acceptance Criteria:**
- [ ] `PUT /api/v1/instances/:id` updates `quantity` and/or `field_values`.
- [ ] Cannot change `definition_id`, `location_id`, or `parent_instance_id` via PUT (use the move endpoint for location/container changes).
- [ ] `quantity`: if provided, must be > 0. Setting quantity to 0 is rejected with `400 Bad Request` (use DELETE instead).
- [ ] `field_values`: if provided, replaces all field values for this instance. Missing fields (not in the array) are deleted from `instance_field_values`.
- [ ] Field value validation applies (same rules as US-001).
- [ ] `updated_at` auto-updated.
- [ ] Non-existent instance returns `404 Not Found`.
- [ ] Typecheck / build / test suite passes.

### US-004: Delete an Instance (Guarded)
**Description:** As a user, I want to delete an instance that is no longer needed.

**Acceptance Criteria:**
- [ ] `DELETE /api/v1/instances/:id` succeeds only if the instance has zero child instances.
- [ ] If the instance has children (`parent_instance_id` pointing to it), return `409 Conflict` with `{ "error": "Cannot delete: X items are stored inside this instance. Move them out first.", "code": "instance_has_children" }`.
- [ ] On success: returns `204 No Content`. Instance row deleted. Field values cascade-delete via FK.
- [ ] Non-existent instance returns `404 Not Found`.
- [ ] Typecheck / build / test suite passes.

### US-005: Move/Split an Instance (Transaction-Safe)
**Description:** As a user, I want to move some quantity of an instance to a different location or container. The operation must be atomic — partial failure leaves the database unchanged.

**Acceptance Criteria:**
- [ ] `POST /api/v1/instances/:id/move` with body `{ "quantity", "target_location_id?", "target_parent_instance_id?" }`.
- [ ] `quantity`: required, integer > 0. Must be ≤ the source instance's current quantity. If exceeds → `400 Bad Request`.
- [ ] `target_location_id` / `target_parent_instance_id`: exactly one required (XOR). Both null or both set → `400 Bad Request`.
- [ ] Target must reference an existing entity. Non-existent → `404 Not Found`.
- [ ] If `target_parent_instance_id` is set, the target instance's definition must have `is_container = true`. Otherwise `400 Bad Request` with `"code": "not_a_container"`.
- [ ] **Cycle detection:** If `target_parent_instance_id` is set, walk up the target's parent chain. The source instance (and its descendants, if it's a container) must not appear in that chain. Returns `400 Bad Request` with `"code": "cycle_detected"` if violated.
- [ ] **Self-containment:** Source `id` must not equal `target_parent_instance_id`. Returns `400 Bad Request` with `"code": "self_parent"`.
- [ ] **Transaction (all-or-nothing):**
  a. Decrement source quantity by `quantity`.
  b. If source quantity reaches 0 and source has children → rollback, return `409 Conflict` with `"code": "instance_has_children"` (can't delete a non-empty container).
  c. If source quantity reaches 0 and no children → delete source row.
  d. At target: search for existing matching instance (same `definition_id`, same target parent, same field values as source). If found → increment target's quantity. If not → create new instance with quantity and source's field values copied.
  e. Any error in a–d rolls back entire transaction.
- [ ] Returns `200 OK` with `{ "source": { ... } | null, "target": { ... } }`. `source` is `null` if the source was deleted (quantity exhausted).
- [ ] Non-existent source instance returns `404 Not Found`.
- [ ] Typecheck / build / test suite passes.

### US-006: List Instances with Filters
**Description:** As a user, I want to browse instances, optionally filtered by location, definition, or parent instance.

**Acceptance Criteria:**
- [ ] `GET /api/v1/instances` returns a flat array of instances sorted by `updated_at DESC`.
- [ ] Optional query params: `?location_id=` , `?definition_id=` , `?parent_instance_id=`. Combined filters use AND logic.
- [ ] Each item includes: `id`, `definition_id`, `definition_name`, `quantity`, `location_id`, `location_name`, `parent_instance_id`, `parent_instance_name`, `updated_at`.
- [ ] Response includes `total_count` (un-paginated total matching filters).
- [ ] Capped at 500 results in v1. If total exceeds 500, response includes `"truncated": true`.
- [ ] No filters → returns all instances (subject to 500 cap).
- [ ] Empty result returns `{ "instances": [], "total_count": 0 }` with 200 OK.
- [ ] Typecheck / build / test suite passes.

### US-007: Browse Container Contents
**Description:** As a user, I want to see what items are stored inside a container instance.

**Acceptance Criteria:**
- [ ] `GET /api/v1/instances/:id/contents` returns direct children only (not recursive).
- [ ] Response: `{ "instances": [ { "id", "definition_id", "definition_name", "quantity" } ] }`.
- [ ] Non-existent instance returns `404 Not Found`.
- [ ] Instance of a non-container definition returns an empty array (200 OK).
- [ ] Sorted alphabetically by `definition_name`.
- [ ] Typecheck / build / test suite passes.

### US-008: Instance Breadcrumb
**Description:** As a user, I want to see the full path from the root location down to this instance, regardless of how deeply nested it is.

**Acceptance Criteria:**
- [ ] `GET /api/v1/instances/:id/breadcrumb` returns an ordered array from root to the instance (inclusive).
- [ ] Each entry: `{ "id": "...", "name": "...", "kind": "location" | "instance" }`.
- [ ] **Algorithm (two-phase):**
  1. Walk `parent_instance_id` chain up from the instance until a row with non-null `location_id` is found. Collect instance entries.
  2. Use the locations breadcrumb CTE (from `prd-locations.md` FR-8) starting from that `location_id` up to root.
  3. Merge: location entries (root-first) + instance entries (closest ancestor first) + current instance (last).
- [ ] For an instance directly at a location (no `parent_instance_id`): entries are all `kind: "location"`, with the instance itself as the last entry `kind: "instance"`.
- [ ] Non-existent instance returns `404 Not Found`.
- [ ] Typecheck / build / test suite passes.

### US-009: Instance Detail Page (UI)
**Description:** As a user, I want a comprehensive detail page showing everything about an instance.

**Acceptance Criteria:**
- [ ] Route `/instances/:id` renders the instance detail page.
- [ ] **Breadcrumb bar (top):** Horizontal scrollable breadcrumb. Each segment is clickable — location segments navigate to that location detail, instance segments navigate to that instance detail. Current instance is the last segment and styled distinctly.
- [ ] **Header:** Definition name (linked to definition detail), unit, quantity badge.
- [ ] **Placement section:** Shows "Located in: [Location Name]" or "Inside: [Parent Instance Name] (in [Location Name])" depending on `location_id` / `parent_instance_id`.
- [ ] **Field Values table/section:** All fields from the definition's resolved schema, with current values. Read-only display. Edit button opens an edit modal.
- [ ] **Contents section (if container):** "Items inside (X)" — list of child instances with definition name, quantity, and clickable rows navigating to each child's detail page. "Add Item" button to create a new instance inside this container.
- [ ] **Actions:** Edit button (opens modal for quantity + field values), Move button (opens move/split dialog), Delete button (with confirmation).
- [ ] **Timestamps:** Created/updated at in a subtle footer area.
- [ ] Mobile: sections stacked vertically, breadcrumb scrolls horizontally, tap targets ≥ 44x44px.
- [ ] Desktop: centered max-width layout (~800px), breadcrumb full-width.
- [ ] Loading state: skeleton/spinner while TanStack Query fetches.
- [ ] Error state: "Instance not found" if 404.
- [ ] **[UI]** Verified in browser on both 375px and 1920px viewports.

### US-010: Create Instance Form (UI)
**Description:** As a user, I want to quickly add instances from the location detail page or container detail page, with the location/container pre-filled. **Creating an instance is ALWAYS contextual — you are always placing it inside whatever you're currently viewing.**

**Acceptance Criteria:**
- [ ] **Entry points are restricted to two contexts ONLY:**
  - Location Detail page: "Add Item" button creates an instance inside this location.
  - Container Instance Detail page (for instances whose definition has `is_container = true`): "Add Item" button creates an instance inside this container.
  - There is NO standalone "Create Instance" route, page, or global button. No `/instances/new` route.
- [ ] Form opens as a modal (mobile: bottom sheet; desktop: centered dialog).
- [ ] **Fields:** Definition (searchable dropdown/combobox), Quantity (number input, default 1), Field Values (dynamic form generated from definition's field schema — text/number/boolean/date/enum inputs), Location/Container (auto-filled from context — read-only in form).
- [ ] When opened from a location page, `location_id` is pre-filled and `parent_instance_id` section hidden.
- [ ] When opened from a container instance page, `parent_instance_id` is pre-filled and `location_id` section hidden.
- [ ] Selecting a definition dynamically renders its resolved field schema as form inputs with default values pre-populated.
- [ ] Required fields marked with asterisk. Inline validation on blur/submit.
- [ ] On successful create: modal closes, success toast, relevant TanStack Query caches invalidated (`['instances']` + context-specific keys).
- [ ] Server-side merge happens transparently — the success toast can indicate "Added 5 to existing stack of 10 (total: 15)" if merged.
- [ ] **[UI]** Verified in browser.

### US-011: Move/Split Dialog (UI)
**Description:** As a user, I want to move some or all of an instance's quantity to another location or container.

**Acceptance Criteria:**
- [ ] Accessible from the instance detail page via a "Move" button.
- [ ] Dialog shows: current quantity, "How many to move?" (number input, 1 to current quantity, slider or stepper), "Move to:" (target selector — location tree or "Into another item" toggle).
- [ ] Target selector: if "Location" selected, show a location tree/dropdown. If "Container" selected, show a searchable list of container-capable instances.
- [ ] Validation: quantity must not exceed current quantity. Target must exist. Target cannot be the source instance itself.
- [ ] On confirm: sends `POST /instances/:id/move`. Shows loading state during transaction.
- [ ] On success: success toast ("Moved 3 to [Target Name]"), invalidates source and target queries, refreshes the detail page.
- [ ] On error: error toast with backend message, dialog stays open with data preserved.
- [ ] If all quantity was moved (source deleted): navigate to the target's detail page or the previous page.
- [ ] **[UI]** Verified in browser.

### US-012: Edit Instance Modal (UI)
**Description:** As a user, I want to edit an instance's quantity and field values inline.

**Acceptance Criteria:**
- [ ] Accessible from the instance detail page via an "Edit" button.
- [ ] Modal shows: Quantity (numeric input, min 1), and all definition fields with current values pre-filled.
- [ ] Cannot change definition, location, or parent container in edit mode.
- [ ] Inline validation on blur/submit.
- [ ] On save: sends `PUT /instances/:id`, invalidates instance queries.
- [ ] On cancel: closes modal, no changes applied.
- [ ] **[UI]** Verified in browser.

### US-013: Delete Instance with Guard (UI)
**Description:** As a user, I want clear feedback when I attempt to delete an instance that contains other items.

**Acceptance Criteria:**
- [ ] Delete button triggers a confirmation dialog.
- [ ] If `child_instance_count > 0`: dialog says "This instance contains X items. You must move them out before deleting." with a single "OK" button (blocks the delete — the user must manually move children first). No "Delete Anyway" option.
- [ ] If `child_instance_count === 0`: dialog says "Delete this instance of [definition name]? Quantity: X" with Cancel and Delete buttons.
- [ ] On success: delete, invalidate cache, navigate to previous page or location detail, success toast.
- [ ] On error: error toast with backend message.
- [ ] **[UI]** Verified in browser.

---

## 5. Functional & Technical Requirements

### 5.1 Schema Changes

The following changes must be applied to the initial migration in `prd-database-schema.md`:

**FR-1:** Add `is_container` column to `item_definitions`:
```sql
is_container BOOLEAN NOT NULL DEFAULT 0
```

**FR-2:** Change `parent_instance_id` FK to explicit `ON DELETE RESTRICT`:
```sql
parent_instance_id TEXT REFERENCES item_instances(id) ON DELETE RESTRICT,
```

Both changes are applied to the initial migration — no separate migration needed since no production data exists.

**FR-3:** The `definition_fields` data remains unchanged. The `instance_field_values` table remains unchanged.

### 5.2 API Endpoints

All endpoints under `/api/v1/instances`.

| Method | Path | Description | Request Body | Response |
|---|---|---|---|---|
| `GET` | `/instances` | List instances (filtered) | — | `{ instances: InstanceSummary[], total_count, truncated? }` |
| `GET` | `/instances/:id` | Single instance detail | — | `InstanceDetail` |
| `POST` | `/instances` | Create (with auto-merge) | `{ definition_id, quantity, location_id?, parent_instance_id?, field_values? }` | `InstanceDetail` (201) |
| `PUT` | `/instances/:id` | Update quantity/fields | `{ quantity?, field_values? }` | `InstanceDetail` |
| `DELETE` | `/instances/:id` | Delete (guarded) | — | 204 or 409 |
| `POST` | `/instances/:id/move` | Move/split (transactional) | `{ quantity, target_location_id?, target_parent_instance_id? }` | `{ source: InstanceDetail \| null, target: InstanceDetail }` |
| `GET` | `/instances/:id/contents` | Direct children of container | — | `{ instances: InstanceSummary[] }` |
| `GET` | `/instances/:id/breadcrumb` | Root-to-instance breadcrumb | — | `BreadcrumbEntry[]` |

**FR-4:** `InstanceSummary` (list/contents response):
```json
{
  "id": "uuid",
  "definition_id": "uuid",
  "definition_name": "M3 Screw",
  "quantity": 50,
  "location_id": "uuid or null",
  "location_name": "Workshop or null",
  "parent_instance_id": "uuid or null",
  "parent_instance_name": "Toolbox #3 or null",
  "updated_at": "2026-08-01T12:00:00Z"
}
```

**FR-5:** `InstanceDetail` (single-get + create/update/move response):
```json
{
  "id": "uuid",
  "definition_id": "uuid",
  "definition_name": "M3 Screw",
  "parent_def_id": "uuid or null",
  "parent_def_name": "Screw or null",
  "unit": "pcs",
  "quantity": 50,
  "location_id": "uuid or null",
  "location_name": "Workshop or null",
  "parent_instance_id": "uuid or null",
  "parent_instance_name": "Toolbox #3 or null",
  "field_values": [
    {
      "field_id": "uuid",
      "field_name": "Material",
      "field_type": "enum",
      "enum_values": ["Steel", "Brass", "Aluminum"],
      "value": "Steel"
    }
  ],
  "child_instance_count": 3,
  "breadcrumb": [
    { "id": "uuid", "name": "Home", "kind": "location" },
    { "id": "uuid", "name": "Workshop", "kind": "location" },
    { "id": "uuid", "name": "Toolbox #3 (x1)", "kind": "instance" },
    { "id": "uuid", "name": "M3 Screw (x50)", "kind": "instance" }
  ],
  "created_at": "2026-08-01T12:00:00Z",
  "updated_at": "2026-08-01T12:00:00Z"
}
```

**FR-6:** `BreadcrumbEntry`:
```json
{
  "id": "uuid",
  "name": "Living Room",
  "kind": "location"
}
```

**FR-7:** `POST /instances` validation:
- `definition_id`: required, must exist.
- `quantity`: required, integer > 0.
- `location_id` / `parent_instance_id`: exactly one required (XOR). If both are set or both null → `400 Bad Request` with `"code": "invalid_parent"`.
- If `parent_instance_id` is set:
  - Parent instance must exist (404 if not).
  - Parent instance's definition must have `is_container = true` (400 with `"code": "not_a_container"`).
- `field_values`: optional array. Each entry validates:
  - `field_id` must belong to the definition's resolved field schema (own or inherited).
  - `value` validated against `field_type` (number, boolean, enum).
  - Required fields: if no `default_value` exists on the field and no value provided → `400 Bad Request` with `"code": "required_field_missing"`.

**FR-8:** **Auto-merge logic** on `POST /instances`:
1. Determine the target parent: `location_id` if set, otherwise `parent_instance_id`.
2. Query for an existing instance with: same `definition_id` AND same parent (matching `location_id` or `parent_instance_id`) AND all field values identical to the submitted ones.
3. Field value comparison: join `instance_field_values` on `field_id` for the candidate instance and compare each `value` against submitted values. Null values match if both are null, or both are absent from the stored values.
4. If match found: `UPDATE item_instances SET quantity = quantity + ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`.
5. If no match: `INSERT` new instance + `INSERT` field value rows.
6. Return the merged-or-created instance detail.

**FR-9:** `POST /instances/:id/move` validation and transaction:
1. Validate source exists (404 if not).
2. Validate `quantity` > 0 and ≤ source.quantity (400 if not).
3. Validate target parent (XOR of `target_location_id` / `target_parent_instance_id`, exactly one).
4. Validate target existence (404 if not).
5. If `target_parent_instance_id` set: validate parent's definition has `is_container = true` (400 if not).
6. Cycle check: if `target_parent_instance_id` set, walk `parent_instance_id` chain from target up. Abort if source.id appears (400 `"code": "cycle_detected"`).
7. Self-containment: source.id ≠ target_parent_instance_id (400 `"code": "self_parent"`).
8. **BEGIN TRANSACTION:**
   a. Lock source row (`SELECT ... FOR UPDATE` concept — SQLite serializes writes anyway).
   b. `UPDATE item_instances SET quantity = quantity - :qty, updated_at = CURRENT_TIMESTAMP WHERE id = :source_id`.
   c. If new quantity = 0:
      - Check child count: `SELECT COUNT(*) FROM item_instances WHERE parent_instance_id = :source_id`.
      - If > 0 → ROLLBACK, return `409 Conflict` `"code": "instance_has_children"`.
      - If 0 → `DELETE FROM item_instances WHERE id = :source_id` (field values cascade).
   d. Auto-merge at target: same logic as FR-8, using source's `definition_id` and field values.
   e. `COMMIT`.
9. Return `{ "source": InstanceDetail | null, "target": InstanceDetail }`.

**FR-10:** `GET /instances` query params:
- `location_id` (optional): filter by exact location.
- `definition_id` (optional): filter by exact definition.
- `parent_instance_id` (optional): filter by exact parent instance.
- All filters use AND logic when combined.
- Sorted by `updated_at DESC`.
- Hard cap: 500 rows. If `COUNT(*)` > 500, set `truncated: true`.
- Response includes `total_count` matching filters (before cap).

**FR-11:** `GET /instances/:id/contents`:
- Returns direct children: `SELECT * FROM item_instances WHERE parent_instance_id = :id`.
- Sorted by `definition_name ASC` (joined from `item_definitions`).
- No recursion, no pagination in v1.

**FR-12:** `GET /instances/:id/breadcrumb` — two-phase algorithm:
1. Walk `parent_instance_id` chain:
   ```sql
   WITH RECURSIVE instance_chain AS (
       SELECT id, definition_id, parent_instance_id, location_id, 0 AS depth
       FROM item_instances WHERE id = :start_id
       UNION ALL
       SELECT i.id, i.definition_id, i.parent_instance_id, i.location_id, c.depth + 1
       FROM item_instances i JOIN instance_chain c ON i.id = c.parent_instance_id
   )
   SELECT * FROM instance_chain ORDER BY depth ASC;
   ```
   This produces: requested instance (depth 0), parent instance (depth 1), ..., until a row with non-null `location_id`.
2. Take the first row with non-null `location_id` from the chain. Use that `location_id` as input to the locations breadcrumb CTE from `prd-locations.md` FR-8:
   ```sql
   WITH RECURSIVE ancestors AS (
       SELECT id, name, parent_id, 0 AS depth FROM locations WHERE id = ?
       UNION ALL
       SELECT l.id, l.name, l.parent_id, a.depth + 1
       FROM locations l JOIN ancestors a ON l.id = a.parent_id
   )
   SELECT id, name FROM ancestors ORDER BY depth DESC;
   ```
3. Merge results:
   - Location entries (from step 2, ordered root-first): `{ id, name, kind: "location" }`.
   - Instance entries (from step 1, ordered closest-ancestor-first, i.e., reverse of depth): `{ id, name, kind: "instance" }`.
   - The requested instance (depth 0 from step 1) is the last entry.

**FR-13:** `PUT /instances/:id` validation:
- Only `quantity` and/or `field_values` may be provided. Any other field in body → ignored (not an error).
- `quantity` if provided: must be > 0 (400 if ≤ 0). Setting to 0 is not allowed — use DELETE.
- `field_values` if provided: replaces ALL existing field values. Existing rows deleted, new ones inserted.
- Field value validation: same rules as FR-7.
- Cannot change `definition_id`, `location_id`, or `parent_instance_id`.

**FR-14:** `DELETE /instances/:id` check:
1. Count child instances: `SELECT COUNT(*) FROM item_instances WHERE parent_instance_id = :id`.
2. If > 0 → `409 Conflict` with `"code": "instance_has_children"` and count in error message.
3. If 0 → `DELETE FROM item_instances WHERE id = :id` (instance_field_values cascade).

### 5.3 Definition Detail Instance Summary Update

The `instances_summary` in `prd-item-definitions.md` FR-5 is extended:

**FR-15:** Add `by_parent_instance` grouping:
```json
"instances_summary": {
  "total_instances": 42,
  "total_quantity": 150,
  "by_location": [
    {
      "location_id": "uuid",
      "location_name": "Workshop",
      "instance_count": 5,
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
}
```

- `total_instances` and `total_quantity` include ALL instances of the definition regardless of placement (location-bound + nested). Recursive counting (per Q11: A).
- `by_location`: instances with non-null `location_id`, grouped by location.
- `by_parent_instance`: instances with non-null `parent_instance_id`, grouped by parent instance. Each group includes the parent's location context (resolved via parent's own location chain).
- Deeper nesting (nested inside nested) is not expanded — only direct grouping by immediate parent. Recursive totals still count them.
- Empty arrays when no instances of that type exist.

### 5.4 Service Layer

**FR-16:** `InstanceService` (in `internal/service/instance.go`) implements:
- `Create(ctx, input) (*InstanceDetail, error)` — validate, auto-merge, insert.
- `GetByID(ctx, id) (*InstanceDetail, error)` — full detail with field values, child count, breadcrumb.
- `List(ctx, filters) (*InstanceListResult, error)` — filtered list with cap.
- `Update(ctx, id, input) (*InstanceDetail, error)` — partial update, field value replacement, validation.
- `Delete(ctx, id) error` — guarded delete.
- `Move(ctx, id, input) (*MoveResult, error)` — transactional move/split with merge at target.
- `GetContents(ctx, id) ([]InstanceSummary, error)` — direct children of a container.
- `GetBreadcrumb(ctx, id) ([]BreadcrumbEntry, error)` — two-phase breadcrumb resolution.

**FR-17:** `ResolveBreadcrumb(ctx, id)` — internal helper implementing the two-phase algorithm in FR-12.

**FR-18:** `FindMatchingInstance(ctx, definitionID, parentLocationID, parentInstanceID, fieldValues)` — internal helper for auto-merge:
- Builds query matching on `definition_id` + `location_id`/`parent_instance_id`.
- Joins `instance_field_values` to compare each submitted field value.
- Returns the matching instance ID or nil.

**FR-19:** `ValidateFieldValues(ctx, definitionID, fieldValues)` — shared validation helper:
- Resolves the definition's field schema (own + inherited, via DefinitionService).
- Validates each submitted value against its field's type, enum values, and required/default rules.
- Returns structured validation errors.

### 5.5 Handler Layer

**FR-20:** Handlers in `internal/handler/instance.go` follow the standard backend architecture pattern:
- Decode JSON → validate struct tags → call service → format response.
- Use `RespondWithError(w, err)` for consistent error JSON.
- Use `google/uuid` for ID generation.

**FR-21:** Request structs use `go-playground/validator/v10` with custom validators:
- `validate_instance_parent`: ensures XOR of `LocationID` and `ParentInstanceID`.
- `validate_positive_quantity`: ensures quantity > 0.

### 5.6 Router Registration

**FR-22:** Instance routes registered under `r.Route("/api/v1/instances", ...)`:
```go
r.Get("/", instanceHandler.List)
r.Get("/{id}", instanceHandler.Get)
r.Post("/", instanceHandler.Create)
r.Put("/{id}", instanceHandler.Update)
r.Delete("/{id}", instanceHandler.Delete)
r.Post("/{id}/move", instanceHandler.Move)
r.Get("/{id}/contents", instanceHandler.GetContents)
r.Get("/{id}/breadcrumb", instanceHandler.GetBreadcrumb)
```

### 5.7 Frontend

**FR-23:** Routes:
- `/instances/:id` — `InstanceDetailPage` (US-009)
- No standalone list page — instances are browsed through location detail, definition detail, and search (PRD #10).

**FR-24:** TanStack Query keys (hierarchical for targeted invalidation):
- `['instances']` — base key for list.
- `['instances', { locationId }]` — instances at a specific location.
- `['instances', { definitionId }]` — instances of a specific definition.
- `['instances', { parentInstanceId }]` — instances inside a specific container.
- `['instances', id]` — single instance detail.
- `['instances', id, 'contents']` — container contents.
- `['instances', id, 'breadcrumb']` — instance breadcrumb.

**FR-25:** On move success, invalidate only affected keys:
- Source instance: `['instances', sourceId]` + its contents and breadcrumb.
- Target location/container: destination-specific instance queries (by location_id or parent_instance_id).
- Do NOT globally invalidate `['instances']` — use targeted invalidation.

**FR-26:** On create success, invalidate:
- The specific location or container query where the instance was added.
- Definition detail's instance summary (for the definition used).

**FR-27:** The create instance form dynamically generates field inputs from the selected definition's resolved field schema. Cached via TanStack Query `['definitions', definitionId]`. Field types map to inputs:
- `text`: `<input type="text">`
- `number`: `<input type="number">`
- `boolean`: `<input type="checkbox">` or toggle
- `date`: `<input type="date">`
- `enum`: `<select>` or combobox with enum_values as options

**FR-28:** The move dialog's target selector provides two modes:
- "To Location": tree/dropdown of locations (excludes source's current location for clarity).
- "Into Container": searchable list of instances whose definitions have `is_container = true`. Filtered to exclude the source instance itself and its ancestors.

**FR-29:** Breadcrumb component is a reusable `<BreadcrumbBar>` that accepts an array of `BreadcrumbEntry`. Each entry renders as a clickable link (location entries → `/locations/:id`, instance entries → `/instances/:id`). The last entry (current) is not clickable and styled distinctively. Mobile: scrolls horizontally; Desktop: wraps or truncates with ellipsis.

**FR-30:** Instance edit modal pre-populates all field values and quantity. Submits `PUT /instances/:id`. Field inputs are re-validated on change.

---

## 6. Edge Cases & Failure Modes

| Scenario | System Behavior |
|---|---|
| Move quantity > source quantity | `400 Bad Request` with `"code": "invalid_quantity"` and `"error": "Cannot move X items: only Y available."` |
| Move quantity = source quantity, source has children | Transaction rolls back. `409 Conflict` with `"code": "instance_has_children"` — user must move children out first. |
| Move quantity = source quantity, no children | Source deleted. Target gets new or merged instance. Transaction commits. |
| Move to a container that was deleted between page load and submit | `404 Not Found` for `target_parent_instance_id`. Transaction rolls back. |
| Move target location deleted between page load and submit | `404 Not Found` for `target_location_id`. Transaction rolls back. |
| Create instance with `parent_instance_id` pointing to non-container definition instance | `400 Bad Request` with `"code": "not_a_container"`. |
| Create instance merges into existing | Merged instance quantity updated. Response shows the merged instance, not a new one. Status still `201 Created` (something was "added"). |
| Create instance with `field_values` for a field_id not in the definition's resolved schema | `400 Bad Request` with `"code": "unknown_field"`. |
| Create instance where required field has no default and no user value | `400 Bad Request` with `"code": "required_field_missing"` listing field names. |
| Create instance with `field_type: number` but value is `"abc"` | `400 Bad Request` with `"code": "invalid_field_value"`. |
| Create instance with `field_type: enum` but value not in enum_values | `400 Bad Request` with `"code": "invalid_enum_value"`. |
| Update instance quantity to 0 | `400 Bad Request` — use DELETE instead. |
| Update instance field values with invalid types | `400 Bad Request` with field-level errors. |
| Delete instance that has children (FK RESTRICT) | `409 Conflict` with child count. |
| Delete non-existent instance | `404 Not Found`. |
| Request instance breadcrumb for deeply nested instance (10 levels of containers) | Two-phase algorithm handles arbitrary depth. Max depth guard (50) applies to the instance-chain walk. |
| Definition's `is_container` changed from true to false after instances exist | Existing container instances retain their children. Only NEW assignments to that definition's instances are blocked. Validation on create/move checks the flag at write time. |
| Definition's field schema changed after instances exist | Existing instances' field values are NOT re-validated. Validation only on next update. Stale field_value rows (for fields that were removed) are returned in GET responses but have no corresponding field metadata — they are excluded from the resolved field_values list (orphaned values silently dropped). |
| Parent instance breadcrumb chain is broken (corrupt data) | Max depth guard (50) prevents infinite loop. If location is never found (all ancestors have null location_id), breadcrumb returns only the instance entries — no location prefix. |
| Simultaneous moves of the same source instance | SQLite serialized writes prevent true concurrency. Second request sees updated quantity and proceeds correctly. |
| Request container contents for an instance of a non-container definition | Returns empty array `{ "instances": [] }` with 200 OK. Not an error. |
| User adds instance at a location and the location is then deleted | Blocked by FK `ON DELETE RESTRICT` on `item_instances.location_id`. Location can't be deleted while instances exist. |
| Move target is the same as the source's current parent | Allowed and treated as a no-op for the parent change, but quantity and field values are not re-checked for merge (would merge into itself — conceptually a no-op). API returns `400 Bad Request` with `"code": "same_parent"` to avoid confusion. |
| `GET /instances` with no filters returns > 500 results | Response returns first 500 rows (by `updated_at DESC`) with `"truncated": true` and accurate `total_count`. |
| Breadcrumb requested for instance at root location (no parent location chain) | First entry is the root location, last entry is the instance. Single location entry if root is the only location in chain. |
| Instance quantity updated to negative value | Rejected by `CHECK (quantity > 0)` at DB level AND by service validation. |
| Field value submitted as empty string for a non-required field | Stored as empty string `""`. Not treated as null. |
| Field value submitted as null for a field with default_value | Stored as null (override explicitly). The default_value is only used when no value is stored at all. |

---

## 7. Non-Goals & Scope Boundaries

- **Audit trail / move history:** No activity log or change tracking in v1. Only `updated_at` timestamp.
- **Bulk operations:** No multi-select, batch move, batch create, or batch delete.
- **Drag-and-drop move/relocate:** v1 uses form-based move dialog only.
- **Undo/redo for moves:** No undo window. Move is final once the transaction commits.
- **Item-in-item depth beyond the max guard:** Max depth 50 enforced. Deeper nesting is a user error.
- **Definition change cascading to instances:** Changing a definition's fields does NOT migrate or delete existing instance field values. Orphaned values silently dropped on read.
- **Instance-level tags:** Tags apply to definitions only (per overarching PRD §4.2 and OQ-4).
- **Photos/image attachments on instances:** Deferred to future photo feature.
- **Instance duplicate detection beyond field values:** Only definition_id + parent + field values are considered. Name/description are not instance-level properties.
- **Manually merging two different instances:** No explicit merge endpoint. Merge only happens automatically on create or move to same parent with identical properties.
- **Un-merging / splitting by property:** No way to split a merged instance by field value divergence after creation. User must create a separate instance manually.
- **Standalone "Create Instance" page or global button:** Instance creation is always contextual — initiated from a Location or Container Instance detail page with the parent pre-filled. No `/instances/new` route exists.
- **Export/import of instances:** Deferred.
- **Barcode/QR scanning for instance creation:** Deferred.

---

## 8. Open Questions & Deferred Items

| # | Question | Status |
|---|---|---|
| OQ-1 | Should the move endpoint support moving to the same parent but changing field values ("split by property")? | Deferred — v1 move preserves field values. Changing field values during move is a separate concern (edit then move). |
| OQ-2 | Should instances support a "notes" or "comment" free-text field independent of definition fields? | Deferred — could be a future `notes TEXT` column on `item_instances`. |
| OQ-3 | Should the `instances_summary.by_parent_instance` resolve multi-level nesting (recursive grouping) or only direct parent? | Resolved in v1 — `by_parent_instance` shows direct parent only. Deeper nesting totals are included in `total_instances`/`total_quantity` but not expanded in the grouping. |
| OQ-4 | Should moving a container instance also move all its children? | Deferred — v1 move moves quantity only. Moving a container with children is not supported as a unit (children stay at their current location/container). This is consistent with the RESTRICT guard preventing source deletion when children exist. |
| OQ-5 | Should there be a "Create & Add Another" quick-flow button on the create form? | Deferred to implementation. Can be added as a convenience if needed. |
| OQ-6 | Should the list endpoint support cursor-based pagination? | Deferred — v1 uses hard 500-row cap with `truncated` flag. Cursor pagination added in v2 if the cap is reached in real usage. |
