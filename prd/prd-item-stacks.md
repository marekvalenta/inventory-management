# PRD: Item Stacks — InventoryManagement

> **Status:** Implemented v1.0
> **Scope:** UI-level grouping of item instances sharing the same definition and parent (location or container) into "Item Stacks" — a query-time aggregation with no new DB tables. Stack detail page (matching instance detail page layout), stack-level move and bulk delete, browse tree integration, search integration, and location detail page integration.

---

## 0. Cross-PRD Consistency Report

### PRDs Reviewed
- `prd-overarching-architecture.md` — Core data model (§4), API conventions (§5), frontend views (§6.1). No Item Stack concept yet — **this PRD requires an update to the overarching PRD §4.2.**
- `prd-database-schema.md` — No schema changes. Stacks are query-time aggregates over existing `item_instances` rows.
- `prd-backend-architecture.md` — Go layering (handler→service→db), error mapping, chi router, payload validation.
- `prd-frontend-architecture.md` — CSS Modules, TanStack Query, React Router v6, Radix UI, mobile/desktop layouts.
- `prd-visual-design.md` — Golden Amber tokens, component patterns, page layouts. Stack detail page is a new layout.
- `prd-locations.md` — Browse tree US-007 currently shows individual `InstanceNode` per instance. **Must be updated to show group `StackNode` rows.** Browse endpoint US-010 currently returns `BrowseInstance[]`, **must be changed to `BrowseStack[]`.**
- `prd-item-instances.md` — Individual instance CRUD unchanged. Stack operations (move, bulk delete) are new complementary operations. Instance detail page at `/instances/:id` remains.
- `prd-search.md` — Search results (US-003, US-004, FR-2) currently return individual instance entries. **Must be updated to return Item Stack entries** instead (one result per definition+parent pair with total quantity). Instance-level search becomes obsolete with stacks — a stack containing "AA Battery" is the natural search result.
- `prd-dashboard.md` — No direct impact (dashboard counts are already aggregate). Location breakdown is location-based, not definition-based, so no change needed.
- `prd-tags.md`, `prd-item-definitions.md`, `prd-settings.md` — No impact.
- `prd-testing.md`, `prd-docker-deployment.md` — No impact.

### Conflicts & Resolutions

| # | Conflict | PRDs Involved | Resolution |
|---|---|---|---|
| 1 | Overarching §4.2 "Smart quantity merging" describes DB-level merge only (same definition + same field values + same parent). No UI-level grouping concept exists. | `prd-overarching-architecture.md` | **Add "Item Stack" concept** to §4.2. Item Stack = UI grouping of instances sharing (definition_id, location_id, parent_instance_id) regardless of field values. DB-level merge remains unchanged. |
| 2 | `prd-locations.md` US-007 browse tree renders `InstanceNode` per individual instance. Clicking navigates to `/instances/:id`. | `prd-locations.md` | **Update browse tree to render `StackNode` per definition+parent pair.** Clicking navigates to `/stacks?definition_id=X&location_id=Y` (or `&parent_instance_id=Y`). Individual instance pages remain accessible as drill-down from stack detail. |
| 3 | `prd-locations.md` US-010 `/browse` endpoint returns `instances: BrowseInstance[]` with per-instance fields (`id`, `definition_id`, `definition_name`, `quantity`, `is_container`, `child_count`). | `prd-locations.md` | **Change to `stacks: BrowseStack[]`** with aggregated fields (`definition_id`, `definition_name`, `unit`, `total_quantity`, `instance_count`, `is_container`, `child_count`). The `id` field is removed (stacks have no UUID). `instance_truncated` and `instance_count` now refer to stack count (not individual instances). Cap per location becomes 50 stacks (not 50 instances). |
| 4 | `prd-search.md` FR-2 search response returns individual `instances[]` with per-instance `id`, `quantity`, `location_name`. | `prd-search.md` | **Change to return `stacks[]`** with aggregated `definition_id`, `definition_name`, `total_quantity`, `location_id`, `location_name`, `parent_instance_id`, `parent_instance_name`. Multiple instances of the same definition at the same location merge to one search result. |
| 5 | `prd-item-instances.md` US-006 list endpoint returns flat `InstanceSummary[]`. | `prd-item-instances.md` | **No change** — `GET /api/v1/instances` retains flat list for API flexibility. Stack aggregation is provided by the new `/stacks` endpoint. The flat list endpoint is useful for debugging and edge cases but the browse tree and search consume stacks. |
| 6 | No `/stacks` route exists in any PRD. New page + API endpoint group required. | All prior PRDs | This PRD defines the `/stacks` route, `StackDetailPage`, and `GET|POST|DELETE /api/v1/stacks` endpoints. |

### Confirmed Alignments
- Data model: Zero schema changes. All stack queries are `GROUP BY` aggregates over `item_instances` + `item_definitions` JOINs.
- API patterns: `/api/v1/stacks` prefix, JSON bodies, UUID IDs for underlying instances, standard `{"error":"...","code":"..."}` format.
- Error mapping: `ErrNotFound` → 404, `ErrConflict` → 409, `ErrInvalidInput` → 400 per `prd-backend-architecture.md` TR-2.
- UI: CSS Modules + CSS variables (Golden Amber tokens), TanStack Query, React Router v6. Mobile-first: 375px–1920px. 44x44px tap targets.
- Go layering: Handler → Service → DB. Services receive `*sql.DB`.
- Existing individual instance CRUD, move/split, and breadcrumb endpoints are **unchanged** and continue to work. Stacks are additive, not replacement.

---

## 1. Overview & Problem Statement

### Problem

The current system displays every `item_instances` row as a separate entry in the browse tree, search results, and lists. When items have distinguishing field values (e.g., serial numbers), each gets a separate row. If a user has 300 serial-numbered AA Batteries at one location, the browse tree shows **300 rows** — making it impossible to see the forest for the trees.

The existing DB-level auto-merge (same definition + same field values + same parent) helps only when field values are identical. It does not solve the "300 serialized items" problem.

### Solution: Item Stacks

An **Item Stack** is a UI-level aggregation of all `item_instances` rows that share the same **definition** and **parent** (either `location_id` or `parent_instance_id`). It is a pure query-time construct — no new table, no migration.

- Browse tree shows **one row per Stack** (e.g., "AA Battery ×300").
- Clicking a Stack navigates to a **Stack Detail page** showing the aggregated summary + a paginated table of the individual instances within it.
- Individual instance detail pages (`/instances/:uuid`) remain for drill-down field-level work.
- Stacks support **move** (backend picks arbitrary instances) and **bulk delete** (all instances in the stack).

### Core Deliverables

1. **Stack API endpoints:** List stacks, get stack detail, move from stack, bulk delete stack.
2. **Modified browse endpoint:** `/api/v1/browse` returns `BrowseStack[]` instead of `BrowseInstance[]`.
3. **Stack Detail page** at `/stacks?definition_id=X&location_id=Y` (or `&parent_instance_id=Y`): summary header + paginated instance list.
4. **Browse tree update:** `BrowseTree` renders `StackNode` rows, navigating to stack detail.
5. **Search update:** Search endpoint and UI return stacks, not individual instances.
6. **Stack-level move:** `POST /api/v1/stacks/move` with arbitrary instance selection.
7. **Stack-level bulk delete:** `DELETE /api/v1/stacks?definition_id=X&location_id=Y` with strong confirmation.

---

## 2. Goals & Measurable Success Metrics

| Goal | Metric |
|---|---|
| Browse readability | Each location shows at most 1 row per definition (stack), regardless of how many individual instances exist |
| Stack detail load time | Stack detail (summary + first page of instances) < 200ms p95 |
| Move transaction safety | Stack move is atomic — partial failure rolls back all changes |
| Bulk delete safety | Bulk delete requires type-to-confirm in UI; backend deletes in a single transaction |
| Individual instance access preserved | Individual instance detail page remains functional via drill-down from stack detail |
| Zero data model impact | No schema migration needed. Existing instance CRUD continues to work. |

---

## 3. Critical Risks & Technical Assumptions

| Risk | Mitigation |
|---|---|
| Stack move picks "wrong" instances | Backend picks arbitrary instances (oldest first). User accepts this tradeoff — field values are not selectable. If the user needs to move a specific serial number, they use the individual move endpoint (`POST /instances/:id/move`). |
| Stack detail with 300 instances is slow | Frontend paginates the instance list (50 per page). Backend returns paginated results. |
| Bulk delete removes instances that are containers with children | Backend checks each instance for children before deleting. If any have children, the entire operation returns 409. No partial deletion. |
| Browse tree performance with many stacks | Stack count is always ≤ instance count. 50-stack cap per location ensures worst-case is no worse than before. |
| Stack detail URL with query params breaks React Router | React Router v6 natively supports `useSearchParams`. The `StackDetailPage` reads query params on mount and on URL change. |
| Definition deleted while viewing its stack | 404 on the stack detail. No cascade — the definition FK prevents deletion while instances exist. |

---

## 4. User Stories & Acceptance Criteria

### US-001: Stack Detail Page
**Description:** As a user, I want to see all instances of the same item type at the same location grouped together, with total quantity prominently displayed and individual instances visible in a list below.

**Acceptance Criteria:**
- [ ] Route `/stacks` with query params `?definition_id=X&location_id=Y` renders `StackDetailPage`.
- [ ] Route `/stacks` with query params `?definition_id=X&parent_instance_id=Y` renders `StackDetailPage`.
- [ ] Exactly one of `location_id` or `parent_instance_id` must be present along with `definition_id`. Missing or both → page shows validation error.
- [ ] **Header section:**
  - Definition name (`--text-h1`), linked to `/definitions/:id`.
  - Unit badge (e.g., "pcs") next to name.
  - Total quantity: large number (`--text-h1`), `--color-accent`.
  - Instance count: "15 individual instances" (`--text-small`, `--color-text-secondary`).
  - Placement: "Located in: [Location Name]" or "Inside: [Parent Instance Name]". Location/container name is a clickable link.
- [ ] **Individual instances table:**
  - Columns: Quantity, Key Field Values (first 2 non-empty field values shown inline), Updated date, Actions (View → `/instances/:id`).
  - Paginated: 50 instances per page. Controls: Previous/Next with "Page X of Y".
  - Sorted by `updated_at DESC`.
  - Empty: stack detail still shows header with `instance_count: 0` (edge case, shouldn't normally happen).
- [ ] **Action buttons:**
  - "Add Item" — opens `CreateInstanceModal` with definition + location/container pre-filled.
  - "Move Items" — opens `MoveStackDialog`.
  - "Delete All" — opens `BulkDeleteStackDialog`.
- [ ] **Breadcrumb bar at top:** shows the location chain to the stack's placement. Reuses the breadcrumb endpoint logic.
- [ ] Loading state: skeleton for header + table rows.
- [ ] Error state: "Stack not found" if definition + parent combination has no instances.
- [ ] Mobile: sections stacked vertically. Table columns reduced (quantity + first field value + view link). Breadcrumb scrolls horizontally.
- [ ] Desktop: max-width 800px centered. Full table columns.
- [ ] **[UI]** Verified in browser on both 375px and 1920px viewports.
- [ ] Typecheck / build / test suite passes.

### US-002: Stack List API Endpoint
**Description:** As a frontend, I need an endpoint that returns all stacks (grouped instances) for a given location or filter.

**Acceptance Criteria:**
- [ ] `GET /api/v1/stacks` returns grouped stacks.
- [ ] Optional query params: `?location_id=` filters stacks at that location (instances where `location_id = ?`). `?parent_instance_id=` filters stacks inside that container.
- [ ] Each stack entry:
  ```json
  {
    "definition_id": "uuid",
    "definition_name": "AA Battery",
    "unit": "pcs",
    "location_id": "uuid or null",
    "location_name": "Workshop or null",
    "parent_instance_id": "uuid or null",
    "parent_instance_name": "Toolbox #3 or null",
    "total_quantity": 300,
    "instance_count": 15,
    "is_container": false,
    "child_count": 0
  }
  ```
- [ ] `total_quantity`: SUM of all `item_instances.quantity` matching the group key.
- [ ] `instance_count`: COUNT of individual `item_instances` rows in this stack.
- [ ] `is_container`: from `item_definitions.is_container` for the definition.
- [ ] `child_count`: total number of child instances inside any container instances within this stack (SUM of `SELECT COUNT(*) FROM item_instances WHERE parent_instance_id IN (stack's instance IDs)`).
- [ ] Sorted by `definition_name ASC`.
- [ ] **Capped at 500 stacks.** If truncated, response includes `"truncated": true` and `total_count`.
- [ ] Empty result returns `{ "stacks": [], "total_count": 0 }` with 200.
- [ ] Implementation: `GROUP BY definition_id, location_id, parent_instance_id` on `item_instances` JOIN `item_definitions`.
- [ ] Typecheck / build / test suite passes.

### US-003: Stack Detail API Endpoint
**Description:** As a frontend, I need an endpoint that returns full detail for a single stack including paginated individual instances.

**Acceptance Criteria:**
- [ ] `GET /api/v1/stacks/detail?definition_id=X&location_id=Y` returns stack detail for location-based stacks.
- [ ] `GET /api/v1/stacks/detail?definition_id=X&parent_instance_id=Y` returns stack detail for container-based stacks.
- [ ] Exactly one of `location_id` or `parent_instance_id` is required. Both or neither → 400.
- [ ] Optional `?page=1` and `?per_page=50` for pagination. Defaults: page=1, per_page=50. Max per_page=100.
- [ ] **Response shape:**
  ```json
  {
    "definition_id": "uuid",
    "definition_name": "AA Battery",
    "unit": "pcs",
    "is_container": false,
    "parent_def_id": "uuid or null",
    "parent_def_name": "Battery or null",
    "location_id": "uuid or null",
    "location_name": "Workshop or null",
    "parent_instance_id": "uuid or null",
    "parent_instance_name": "Toolbox #3 or null",
    "total_quantity": 300,
    "instance_count": 15,
    "child_count": 3,
    "breadcrumb": [
      { "id": "uuid", "name": "Home", "kind": "location" },
      { "id": "uuid", "name": "Workshop", "kind": "location" }
    ],
    "instances": [
      {
        "id": "uuid",
        "definition_id": "uuid",
        "definition_name": "AA Battery",
        "quantity": 20,
        "field_values": [
          { "field_id": "uuid", "field_name": "Serial", "field_type": "text", "value": "SN-001" }
        ],
        "location_id": "uuid",
        "location_name": "Workshop",
        "parent_instance_id": null,
        "parent_instance_name": null,
        "created_at": "...",
        "updated_at": "..."
      }
    ],
    "pagination": {
      "page": 1,
      "per_page": 50,
      "total_pages": 1,
      "total_instances": 15
    }
  }
  ```
- [ ] `breadcrumb`: location chain from root to the stack's placement location. For container-based stacks, the breadcrumb goes from root → location chain → container instance chain.
- [ ] `instances[]`: paginated subset of individual instances in this stack. Each entry includes `field_values` with field metadata (field_name, field_type, value). Limited to first 5 field values per instance for performance.
- [ ] `child_count`: summed count of child instances inside container instances within this stack.
- [ ] If no instances match the definition+parent combination → 404.
- [ ] Invalid pagination (page < 1, per_page > 100) → 400.
- [ ] Typecheck / build / test suite passes.

### US-004: Stack-Level Move
**Description:** As a user, I want to move N items from a stack to another location or container without picking which specific instances.

**Acceptance Criteria:**
- [ ] `POST /api/v1/stacks/move` with body:
  ```json
  {
    "definition_id": "uuid",
    "source_location_id": "uuid",
    "quantity": 5,
    "target_location_id": "uuid"
  }
  ```
- [ ] OR with container targeting:
  ```json
  {
    "definition_id": "uuid",
    "source_parent_instance_id": "uuid",
    "quantity": 5,
    "target_parent_instance_id": "uuid"
  }
  ```
- [ ] `definition_id`: required.
- [ ] `source_location_id` / `source_parent_instance_id`: exactly one required (XOR).
- [ ] `quantity`: required, integer > 0.
- [ ] `target_location_id` / `target_parent_instance_id`: exactly one required (XOR).
- [ ] Validation:
  - Total quantity in the source stack must be ≥ `quantity`. If less → 400 with `"code": "insufficient_quantity"`.
  - Target must exist (404 if not).
  - If `target_parent_instance_id`: target must be a container definition (400 `"code": "not_a_container"`).
- [ ] **Backend instance selection:** The service selects arbitrary instances from the source stack (oldest first by `created_at`), decrementing or deleting them as needed, until the requested `quantity` is fulfilled. The service copies field values from each moved instance to the target.
- [ ] **Transaction-safe:** Entire operation in one SQLite transaction. Any failure → rollback.
- [ ] **Auto-merge at target:** If an existing instance at the target shares the same definition + same field values as a moved instance, quantities merge (reusing existing DB-level merge logic).
- [ ] **Response:**
  ```json
  {
    "moved_quantity": 5,
    "source": { ...stack detail after move... },
    "target": { ...stack detail after move... }
  }
  ```
- [ ] If after move the source stack is empty (all instances deleted), `source` shows `total_quantity: 0, instance_count: 0`.
- [ ] If the source stack has children (some instances are containers with children), the service skips those instances when selecting which to move. If no eligible instance without children has enough quantity → 409 with `"code": "instance_has_children"`.
- [ ] Typecheck / build / test suite passes.

### US-005: Stack-Level Bulk Delete
**Description:** As a user, I want to delete all instances in a stack in one action.

**Acceptance Criteria:**
- [ ] `DELETE /api/v1/stacks?definition_id=X&location_id=Y` deletes all instances in the stack (location-based).
- [ ] `DELETE /api/v1/stacks?definition_id=X&parent_instance_id=Y` deletes all instances in the stack (container-based).
- [ ] Exactly one of `location_id` or `parent_instance_id` is required.
- [ ] **Guard:** If any instance in the stack has children (`parent_instance_id` pointing to it), return `409 Conflict` with `{ "error": "Cannot delete stack: X instances have items stored inside them.", "code": "stack_has_children" }`.
- [ ] On success: returns `204 No Content`. All instance rows + field values deleted in a single transaction.
- [ ] Non-existent stack (no instances matching) → 404.
- [ ] Typecheck / build / test suite passes.

### US-006: Move Stack Dialog (UI)
**Description:** As a user, I want a dialog to move items from the stack detail page.

**Acceptance Criteria:**
- [ ] "Move Items" button on the stack detail page opens `MoveStackDialog`.
- [ ] Dialog shows: total available quantity, "How many to move?" (number input, 1 to total), "Move to:" (location tree or container selector).
- [ ] Target selector uses the same location-tree / container-instance-search pattern as the individual `MoveInstanceDialog` (US-011 in `prd-item-instances.md`).
- [ ] Quantity stepper/slider between 1 and `total_quantity`.
- [ ] Validation: quantity must be ≤ `total_quantity`. Target must exist and be different from source.
- [ ] On confirm: sends `POST /api/v1/stacks/move`. Shows loading state.
- [ ] On success: toast ("Moved 5 AA Batteries to Workshop"), invalidates stack detail and target queries, refreshes stack detail.
- [ ] On error: error toast with backend message.
- [ ] **[UI]** Verified in browser.

### US-007: Bulk Delete Stack Dialog (UI)
**Description:** As a user, I want clear, strong confirmation before deleting all instances in a stack.

**Acceptance Criteria:**
- [ ] "Delete All" button on stack detail page opens `BulkDeleteStackDialog`.
- [ ] Dialog shows: "Delete all X AA Batteries? This will permanently remove X individual instances totalling Y items from [location/container name]."
- [ ] **Type-to-confirm:** A text input where the user must type "DELETE" (uppercase) before the confirm button activates.
- [ ] If instances have children (detected via `child_count > 0`): dialog blocks with "X instances contain other items. Move them out first." — only a "Close" button is available. No delete possible.
- [ ] On confirm: sends `DELETE /api/v1/stacks?...`. Shows loading state.
- [ ] On success: toast, navigates back to previous page (browse tree or definition detail).
- [ ] On error: error toast with backend message.
- [ ] **[UI]** Verified in browser.

### US-008: Browse Tree Stack Integration
**Description:** As a user, I want the unified browse tree to show grouped stacks instead of individual instances.

**Acceptance Criteria:**
- [ ] Browse tree renders `BrowseStack[]` from the modified `/api/v1/browse` endpoint (location PRD update).
- [ ] Each stack row shows: definition name, total quantity badge, instance count ("15 instances").
- [ ] Clicking a stack row navigates to `/stacks?definition_id=X&location_id=Y` (or `&parent_instance_id=Y` for container stacks).
- [ ] Container stacks (is_container=true) show a chevron and are expandable. Expanding lazy-loads child stacks via `GET /api/v1/stacks?parent_instance_id=Y`.
- [ ] Non-container stacks show no expand toggle.
- [ ] Stack capping: maximum 50 stacks per location. Truncated: "(+N more stacks)" link navigates to `/locations/:id`.
- [ ] "+" cube icon on location node opens `CreateInstanceModal` with location pre-filled (unchanged).
- [ ] "+" cube icon on container stack node opens `CreateInstanceModal` with parent_instance pre-filled (unchanged).
- [ ] After creating an instance that extends an existing stack, the browse tree refresh shows the updated total quantity.
- [ ] **[UI]** Verified in browser.

### US-009: Search Stack Integration
**Description:** As a user, I want search results to show stacks, not individual instances.

**Acceptance Criteria:**
- [ ] `GET /api/v1/search?q=term` returns `stacks[]` instead of `instances[]`.
- [ ] Each stack search result:
  ```json
  {
    "definition_id": "uuid",
    "definition_name": "AA Battery",
    "unit": "pcs or null",
    "location_id": "uuid or null",
    "location_name": "Workshop or null",
    "parent_instance_id": "uuid or null",
    "parent_instance_name": "Toolbox or null",
    "total_quantity": 300,
    "instance_count": 15
  }
  ```
- [ ] Stack results are grouped by `definition_id + COALESCE(location_id, '') + COALESCE(parent_instance_id, '')`.
- [ ] `total_counts.instances` is renamed to `total_counts.stacks` (or kept for backward compatibility — TBD).
- [ ] Quick-results dropdown and search results page render stack rows instead of instance rows.
- [ ] Clicking a stack search result navigates to `/stacks?...`.
- [ ] Typecheck / build / test suite passes.

---

## 5. Functional & Technical Requirements

### 5.1 Database Changes

**No schema changes.** All stack operations are query-time aggregations over the existing `item_instances` and `item_definitions` tables.

### 5.2 REST API Endpoints

All endpoints under `/api/v1/stacks`.

| Method | Path | Description | Request / Params | Response |
|---|---|---|---|---|
| `GET` | `/stacks` | List stacks | `?location_id=` or `?parent_instance_id=` | `{ stacks: BrowseStack[], total_count, truncated? }` |
| `GET` | `/stacks/detail` | Stack detail | `?definition_id=X&location_id=Y` or `&parent_instance_id=Y` + `?page=&per_page=` | `StackDetail` |
| `POST` | `/stacks/move` | Move N items from stack | `{ definition_id, source_location_id?, source_parent_instance_id?, quantity, target_location_id?, target_parent_instance_id? }` | `{ moved_quantity, source: StackDetail, target: StackDetail }` |
| `DELETE` | `/stacks` | Bulk delete stack | `?definition_id=X&location_id=Y` or `&parent_instance_id=Y` | 204 or 409 |

**FR-1:** `GET /api/v1/stacks` implementation:

```sql
SELECT
    d.id AS definition_id,
    d.name AS definition_name,
    d.unit,
    d.is_container,
    i.location_id,
    l.name AS location_name,
    i.parent_instance_id,
    pi_def.name AS parent_instance_name,
    COALESCE(SUM(i.quantity), 0) AS total_quantity,
    COUNT(i.id) AS instance_count
FROM item_instances i
JOIN item_definitions d ON d.id = i.definition_id
LEFT JOIN locations l ON l.id = i.location_id
LEFT JOIN item_instances pi ON pi.id = i.parent_instance_id
LEFT JOIN item_definitions pi_def ON pi_def.id = pi.definition_id
WHERE (? IS NULL OR i.location_id = ?)
  AND (? IS NULL OR i.parent_instance_id = ?)
GROUP BY d.id, i.location_id, i.parent_instance_id
ORDER BY d.name ASC
LIMIT 501
```

- Child count computed in a separate aggregation step in Go.
- `total_count` = number of stack groups (before LIMIT 500). `truncated: true` if > 500.
- No stacks endpoint returns instances directly; individual instances are accessed via `/stacks/detail`.

**FR-2:** `GET /api/v1/stacks/detail` implementation:

1. Validate exactly one of `location_id` / `parent_instance_id` is present.
2. Compute stack aggregate (total_quantity, instance_count, definition info, placement info).
3. Compute breadcrumb:
   - If `location_id`: use the location breadcrumb CTE from `prd-locations.md` FR-8.
   - If `parent_instance_id`: use the two-phase instance breadcrumb from `prd-item-instances.md` FR-12, up to but excluding the individual instance entries.
4. Fetch paginated instances:

```sql
SELECT
    i.id, i.definition_id, d.name AS definition_name,
    i.quantity, i.location_id, l.name AS location_name,
    i.parent_instance_id, pi_def.name AS parent_instance_name,
    i.created_at, i.updated_at
FROM item_instances i
JOIN item_definitions d ON d.id = i.definition_id
LEFT JOIN locations l ON l.id = i.location_id
LEFT JOIN item_instances pi ON pi.id = i.parent_instance_id
LEFT JOIN item_definitions pi_def ON pi_def.id = pi.definition_id
WHERE i.definition_id = ?
  AND ((? IS NOT NULL AND i.location_id = ?) OR (? IS NOT NULL AND i.parent_instance_id = ?))
ORDER BY i.updated_at DESC
LIMIT ? OFFSET ?
```

5. For each instance, fetch field values (limited to first 5):

```sql
SELECT ifv.field_id, df.field_name, df.field_type, ifv.value
FROM instance_field_values ifv
JOIN definition_fields df ON df.id = ifv.field_id
WHERE ifv.instance_id = ?
ORDER BY df.display_order ASC
LIMIT 5
```

6. Compute `child_count`: SUM of children inside container instances in this stack.

**FR-3:** `POST /api/v1/stacks/move` implementation:

1. Validate all required fields, XOR constraints, quantities.
2. Compute total quantity available in source stack → must be ≥ requested quantity.
3. **BEGIN TRANSACTION:**
   a. Fetch instances in source stack ordered by `created_at ASC` (oldest first).
   b. For each instance:
      - Determine how many to take from this instance: `MIN(instance.quantity, remaining_quantity)`.
      - Decrement or delete (same as individual move logic in `prd-item-instances.md` FR-9, steps b–c).
      - At target: find matching instance (same definition + same field values + same target parent). Merge or create.
      - `remaining_quantity -= taken`. Break when `remaining_quantity == 0`.
   c. If any instance can't be deleted because it has children → skip it. If no eligible instance can fulfill the remaining quantity → ROLLBACK, return `409 instance_has_children`.
   d. **COMMIT.**
4. Return `{ moved_quantity, source: stack_detail, target: stack_detail }`.

**FR-4:** `DELETE /api/v1/stacks` implementation:

1. Validate query params (XOR of `location_id` / `parent_instance_id`).
2. Fetch all instance IDs in the stack.
3. Check for children: `SELECT COUNT(*) FROM item_instances WHERE parent_instance_id IN (stack instance IDs)`.
4. If children exist → `409 Conflict` `"code": "stack_has_children"`.
5. **DELETE FROM item_instances WHERE id IN (...)** in a single transaction. Field values cascade.
6. Return `204 No Content`.

### 5.3 Service Layer

**FR-5:** `StackService` (in `internal/service/stack.go`):

```go
type StackService struct {
    db *sql.DB
}

type BrowseStack struct {
    DefinitionID       string  `json:"definition_id"`
    DefinitionName     string  `json:"definition_name"`
    Unit               *string `json:"unit"`
    LocationID         *string `json:"location_id"`
    LocationName       *string `json:"location_name"`
    ParentInstanceID   *string `json:"parent_instance_id"`
    ParentInstanceName *string `json:"parent_instance_name"`
    TotalQuantity      int     `json:"total_quantity"`
    InstanceCount      int     `json:"instance_count"`
    IsContainer        bool    `json:"is_container"`
    ChildCount         int     `json:"child_count"`
}

type StackDetail struct {
    DefinitionInfo // definition_id, definition_name, unit, is_container, parent_def_id, parent_def_name
    PlacementInfo  // location_id, location_name, parent_instance_id, parent_instance_name
    TotalQuantity  int                `json:"total_quantity"`
    InstanceCount  int                `json:"instance_count"`
    ChildCount     int                `json:"child_count"`
    Breadcrumb     []BreadcrumbEntry  `json:"breadcrumb"`
    Instances      []InstanceInStack  `json:"instances"`
    Pagination     PaginationInfo     `json:"pagination"`
}

type InstanceInStack struct {
    ID                 string             `json:"id"`
    DefinitionID       string             `json:"definition_id"`
    DefinitionName     string             `json:"definition_name"`
    Quantity           int                `json:"quantity"`
    FieldValues        []FieldValueBrief  `json:"field_values"`
    LocationID         *string            `json:"location_id"`
    LocationName       *string            `json:"location_name"`
    ParentInstanceID   *string            `json:"parent_instance_id"`
    ParentInstanceName *string            `json:"parent_instance_name"`
    CreatedAt          string             `json:"created_at"`
    UpdatedAt          string             `json:"updated_at"`
}

type MoveStackRequest struct {
    DefinitionID              string  `json:"definition_id" validate:"required"`
    SourceLocationID          *string `json:"source_location_id"`
    SourceParentInstanceID    *string `json:"source_parent_instance_id"`
    Quantity                  int     `json:"quantity" validate:"required,gt=0"`
    TargetLocationID          *string `json:"target_location_id"`
    TargetParentInstanceID    *string `json:"target_parent_instance_id"`
}

type MoveStackResponse struct {
    MovedQuantity int         `json:"moved_quantity"`
    Source        *StackDetail `json:"source"`
    Target        *StackDetail `json:"target"`
}
```

Methods:
- `List(ctx, locationID, parentInstanceID *string) (*StackListResult, error)`
- `GetDetail(ctx, definitionID string, locationID, parentInstanceID *string, page, perPage int) (*StackDetail, error)`
- `Move(ctx, req MoveStackRequest) (*MoveStackResponse, error)`
- `Delete(ctx, definitionID string, locationID, parentInstanceID *string) error`

**FR-6:** Stack service reuses helper functions from `InstanceService`:
- `validateFieldValues()` — field type validation
- `findMatchingInstanceTx()` — merge-on-move logic
- `ResolveBreadcrumb()` — for stack detail breadcrumb

**FR-7:** Stack detail breadcrumb resolution:
- For location-based stacks: use the location breadcrumb CTE to the placement location. The stack itself is not an instance, so the breadcrumb ends with the location (no instance entry for "the stack").
- For container-based stacks: use the two-phase instance breadcrumb to the parent container instance. Breadcrumb entries include location chain + instance chain up to the parent container.

### 5.4 Handler Layer

**FR-8:** `StackHandler` (in `internal/handler/stack.go`):
- `GET /api/v1/stacks` → `List`
- `GET /api/v1/stacks/detail` → `Detail`
- `POST /api/v1/stacks/move` → `Move`
- `DELETE /api/v1/stacks` → `Delete`
- Standard handler pattern: decode params/body → validate → call service → format response.
- Uses `RespondWithError(w, err)` for consistent error JSON.

### 5.5 Router Registration

**FR-9:** Stack routes registered in `internal/router/`:

```go
r.Route("/api/v1/stacks", func(r chi.Router) {
    r.Get("/", stackHandler.List)
    r.Get("/detail", stackHandler.Detail)
    r.Post("/move", stackHandler.Move)
    r.Delete("/", stackHandler.Delete)
})
```

**FR-10:** Chi router must support `GET /detail` as a sub-path without conflicting with `{id}` patterns from other route groups. Since stacks have no `{id}` param route, both `/stacks` and `/stacks/detail` use static paths. The order matters: detail must be registered before any wildcard route (there are none, so no conflict).

### 5.6 Frontend

**FR-11:** Routes:
- `/stacks` → `StackDetailPage` (reads `?definition_id=X&location_id=Y` or `?definition_id=X&parent_instance_id=Y` from search params).

**FR-12:** TanStack Query keys:
- `['stacks', { locationId }]` — stacks at a location (browse tree lazy-load).
- `['stacks', { parentInstanceId }]` — stacks inside a container (browse tree lazy-load).
- `['stacks', 'detail', { definitionId, locationId }]` — stack detail (location-based).
- `['stacks', 'detail', { definitionId, parentInstanceId }]` — stack detail (container-based).

**FR-13:** `StackDetailPage` component (`frontend/src/pages/StackDetailPage.tsx`):
- Reads `definition_id`, `location_id`, `parent_instance_id` from `useSearchParams()`.
- Validates the XOR constraint client-side.
- Fetches stack detail via `useQuery`.
- Renders: breadcrumb bar, header (definition name + unit + total quantity + instance count), placement info, action buttons, paginated instance table.
- Action buttons: "Add Item" (opens `CreateInstanceModal`), "Move Items" (opens `MoveStackDialog`), "Delete All" (opens `BulkDeleteStackDialog`).

**FR-14:** `MoveStackDialog` component (`frontend/src/components/stacks/MoveStackDialog.tsx`):
- Reuses the target selector pattern from `MoveInstanceDialog`.
- Quantity stepper between 1 and `total_quantity`.
- Submits `POST /api/v1/stacks/move`.

**FR-15:** `BulkDeleteStackDialog` component (`frontend/src/components/stacks/BulkDeleteStackDialog.tsx`):
- Shows total quantity and instance count.
- Type-to-confirm: text input that must match "DELETE" exactly.
- If `child_count > 0`: shows blocking message, only Close button.
- Submits `DELETE /api/v1/stacks?...`.

**FR-16:** Browse tree `StackNode` replaces `InstanceNode` in `BrowseTree.tsx`:
- Renders one row per `BrowseStack`.
- Shows: cube icon, definition name, total quantity badge, instance count label.
- Click → navigate to `/stacks?definition_id=X&location_id=Y`.
- Container stacks show expandable chevron + children list.
- "+" add button on container stacks opens `CreateInstanceModal`.

**FR-17:** Search integration:
- `QuickResultsDropdown` and `SearchResultsPage` render stacks instead of individual instances.
- Instance-related props and API types updated to stack types.
- Clicking a stack result navigates to `/stacks?...`.

**FR-18:** API client (`frontend/src/api/stacks.ts`):
```typescript
interface BrowseStack {
  definition_id: string;
  definition_name: string;
  unit: string | null;
  location_id: string | null;
  location_name: string | null;
  parent_instance_id: string | null;
  parent_instance_name: string | null;
  total_quantity: number;
  instance_count: number;
  is_container: boolean;
  child_count: number;
}

interface StackDetail {
  definition_id: string;
  definition_name: string;
  unit: string | null;
  is_container: boolean;
  parent_def_id: string | null;
  parent_def_name: string | null;
  location_id: string | null;
  location_name: string | null;
  parent_instance_id: string | null;
  parent_instance_name: string | null;
  total_quantity: number;
  instance_count: number;
  child_count: number;
  breadcrumb: BreadcrumbEntry[];
  instances: InstanceInStack[];
  pagination: PaginationInfo;
}
```

**FR-19:** Query invalidation after stack-level mutations:
- After stack move: invalidate `['stacks']` keys for source and target contexts, plus individual instance queries for affected instances.
- After stack bulk delete: invalidate `['browse']`, `['stacks']`, and navigate away from the deleted stack page.
- After creating an instance from the stack page: invalidate the current stack detail query so the page refreshes with the new total.

---

## 6. Edge Cases & Failure Modes

| Scenario | System Behavior |
|---|---|
| Stack detail with 0 instances | Page shows header with `total_quantity: 0, instance_count: 0`. Empty table message. Action buttons hidden except "Add Item". |
| Move more than available from a stack | `400 Bad Request` with `"code": "insufficient_quantity"` and `"error": "Cannot move X items: only Y available."` |
| Stack move where all instances are containers with children | No eligible instance can be moved. `409 Conflict` with `"code": "instance_has_children"`. |
| Stack move partially fulfilled (e.g., 150 available but 50 are in containers with children) | Service skips child-having instances. Only 100 are eligible. If the user requested ≤ 100, the move proceeds with eligible instances. If they requested > 100 → 409. |
| Bulk delete stack where some instances have children | `409 Conflict`. Entire operation blocked. No partial deletion. |
| Stack detail URL with missing params | Page shows validation prompt: "Select an item type and location to view its stack." |
| Stack detail URL with non-existent definition or location | `404 Not Found` — the stack has no instances (all 0). Page shows "No items of this type found at this location." |
| User creates an instance with field values identical to an existing instance at the same location | DB-level auto-merge kicks in → instance_count stays the same, total_quantity increases. Stack shows one fewer instance row, one higher quantity on the merged row. |
| User creates an instance with different field values | New instance row created. Stack shows instance_count +1, updated total_quantity. |
| Browse tree with 0 stacks at a location | Location node shows no instance rows. Empty state under the location: "No items here yet." |
| Stack at location with `parent_instance_id` set (container-nested) and location-based stacks mixed | Separate stacks: one for `location_id=X, parent_instance_id=NULL`, another for `location_id=NULL, parent_instance_id=Y`. They don't merge. |
| Definition's `is_container` changed after stacks are displayed | No impact — the flag is read from `item_definitions` at query time. Container stacks show expand toggle dynamically. |
| Very large stack (1000+ instances) | Stack detail endpoint paginates at 50 per page. List endpoint capped at 500. Browse tree capped at 50 stacks per location. |

---

## 7. Non-Goals & Scope Boundaries

- **Stack-level edit (field values):** Not supported. Editing field values requires the individual instance detail page or edit modal.
- **Stack-level split by field value:** No way to split a stack into sub-stacks by field value divergence from the stack view. Use individual instance operations.
- **Stack as a DB entity:** Stacks are query-time aggregates only. No `item_stacks` table, no UUID for stacks, no stack-level timestamps.
- **Stack sorting customization:** Stack list is sorted alphabetically by definition name. No user-configurable sorting.
- **Stack-level field value display in browse tree:** Browse tree shows total quantity only. Field values are visible only in stack detail and individual instance detail.
- **Recursive stack expansion in browse tree:** Container stack children are shown as lazy-loaded stacks, not recursively expanded. One level deep per expand.
- **Stack merge across locations:** Stacks are strictly per-parent. Items at different locations or different containers do not merge. Same definition at Workshop + Garage = two separate stacks.
- **Stack breadcrumb including the stack itself:** The breadcrumb ends at the location or parent container. There is no "Stack" breadcrumb entry since stacks are not entities.
- **Creating an instance from global context:** Instance creation is contextual only (from location, container, or stack detail), per existing `prd-item-instances.md` non-goal. No standalone `/instances/new`.

---

## 8. Open Questions & Deferred Items

| # | Question | Status |
|---|---|---|
| OQ-1 | Should the stack detail page show an "Edit All" action to bulk-change field values? | Deferred — v1 has no bulk field value editing. Individual edit only via `/instances/:id`. |
| OQ-2 | Should the `child_count` in stack detail distinguish between children-at-this-level and recursive children? | Deferred — v1 uses direct children only. Recursive expansion deferred. |
| OQ-3 | Should the stack `is_container` flag consider whether any instance in the stack is a container, or only the definition's flag? | Resolved — `is_container` comes from the definition, not individual instances. All instances in a stack share the same definition, so the flag is uniform. |
| OQ-4 | Should `GET /instances/:id/contents` continue to return individual instances or stacks? | Deferred — v1 keeps individual instances for container contents to maintain consistency with the direct-instance model. Stack grouping at the container level is a browse tree concern. |
| OQ-5 | Should the definition detail page's `instances_summary` be changed to show stacks? | Deferred — definition detail remains instance-level. Stack aggregation is the browse/search concern. |
