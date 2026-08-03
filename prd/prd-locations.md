# PRD: Locations — InventoryManagement

> **Status:** Draft v2.0
> **Scope:** Locations CRUD (API + UI), unified location+instance browse tree, container instance nesting, breadcrumb navigation, move/reparent with cycle detection, deletion guard.

---

## 0. Cross-PRD Consistency Report

### PRDs Reviewed
- `prd-overarching-architecture.md` — High-level data model, API conventions, frontend views, mobile-first constraints.
- `prd-database-schema.md` — Canonical `locations` table schema, auto-seeding, migration system, connection tuning.
- `prd-backend-architecture.md` — Go layering (handler/service/db), chi router, error mapping, payload validation.
- `prd-frontend-architecture.md` — CSS Modules, TanStack Query, React Router v6, Radix UI, mobile/desktop layouts.
- `prd-project-setup.md` — Makefile targets, test commands, directory structure (not directly impacted).

### Conflicts & Resolutions

| # | Conflict | PRDs Involved | Resolution |
|---|---|---|---|
| 1 | Overarching §4.2 describes a "Delete all?" prompt implying cascade, but `prd-database-schema.md` enforces `ON DELETE RESTRICT`. | prd-overarching-architecture.md, prd-database-schema.md | This PRD adopts **hard block (RESTRICT)**. No cascade delete. UI shows error with counts of blocking children/items. Overarching PRD updated to reflect RESTRICT enforcement. |
| 2 | Root location behavior undefined — database PRD auto-seeds it but doesn't specify mutability rules. | prd-database-schema.md | Root location is **special**: cannot be deleted or reparented, but **can be renamed**. Root is identified via `settings.root_location_id`. |
| - | `root_location_id` was not in the initial database PRD schema. | prd-database-schema.md | **Resolved:** `root_location_id TEXT REFERENCES locations(id)` is included in the initial migration defined in `prd-database-schema.md`. No additional change needed. |

### Confirmed Alignments
- Data model: Uses `locations` table exactly as defined in `prd-database-schema.md` TR-3.
- API patterns: Follows `/api/v1/` prefix, JSON bodies, UUID IDs, `{"error":"...","code":"..."}` error format per overarching §5.
- Error mapping: `ErrNotFound` → 404, `ErrConflict` → 409, `ErrInvalidInput` → 400 per `prd-backend-architecture.md` TR-2.
- UI: Radix UI primitives + CSS Modules + TanStack Query per `prd-frontend-architecture.md`.
- Mobile-first: Bottom nav on mobile, sidebar on desktop, 44x44px tap targets.
- Scope does not contradict any non-goal in overarching PRD.

---

## 1. Overview & Problem Statement

Locations are the hierarchical backbone of the inventory system. They represent physical or logical places where items are stored. A location can contain sub-locations (unlimited depth) and item instances. This PRD defines the full CRUD API, the tree browser UI, and the guard rails (deletion blocking, cycle prevention on move/reparent, breadcrumb navigation).

**v2 (Unified Tree):** The location tree page (`/locations`) is upgraded from a locations-only tree to a **unified browse tree** showing both locations and item instances. Container instances are expandable to show nested child instances. Two action buttons per tree node allow adding sub-locations and instances directly from the tree. This replaces the v1 non-goal that kept instances out of the tree.

### Core Deliverables
1. REST API: Create, read, update, delete, list, tree, children, contents, breadcrumb.
2. **NEW** `GET /api/v1/browse` endpoint returning the full unified tree (locations + instances).
3. Server-side cycle detection on reparent.
4. Hard-block deletion when children or item instances exist.
5. Special root location handling (auto-seeded, non-deletable, non-reparentable, renamable).
6. Frontend: unified browse tree, location detail, create/edit form, delete guard dialog.
7. Instance display in tree with quantity badges, container nesting, and per-node action buttons.

---

## 2. Goals & Measurable Success Metrics

| Goal | Metric |
|---|---|
| Full CRUD | All location operations complete in < 200ms p95 |
| Tree rendering | Full tree (500 nodes) renders in < 1s on the frontend |
| Cycle prevention | 100% of invalid reparent attempts rejected by server |
| Deletion safety | 100% of attempted deletes on non-empty locations return 409 |
| Mobile usability | Tree expand/collapse usable on 375px viewport with thumb |
| Breadcrumb correctness | Ancestor chain from root to any location always correct |

---

## 3. Critical Risks & Technical Assumptions

| Risk | Mitigation |
|---|---|
| Infinite recursion on tree endpoint for cyclic data | Cycle cannot exist at rest (enforced on write). Tree endpoint trusts stored data. Add max depth guard (50) as safety net. |
| N+1 queries on breadcrumb resolution | Use recursive CTE (single query) or batch parent-chain walk. |
| Slow full-tree query on very large hierarchies | Provide lazy-load `/children` endpoint as alternative; full tree endpoint is convenience-only. |
| Race condition: user deletes parent while another is adding a child to it | SQLite serialized writes + FK constraint ensures transactional safety. |
| Root location deleted due to bug | `settings.root_location_id` is checked by delete handler as first validation step. |

---

## 4. User Stories & Acceptance Criteria

### US-001: Create a Location
**Description:** As a user, I want to create a new location (room, shelf, box, etc.) so I can organize my inventory hierarchy.

**Acceptance Criteria:**
- [ ] `POST /api/v1/locations` with `{ "name": "...", "description": "...", "parent_id": "..." }` creates a location.
- [ ] `name` is required (2–200 chars); `description` is optional (max 2000 chars); `parent_id` is optional (NULL = top-level).
- [ ] If `parent_id` is provided, it must reference an existing location.
- [ ] Returns `201 Created` with the new location JSON including generated UUID and timestamps.
- [ ] Invalid input returns `400 Bad Request` with field-level error messages.
- [ ] Typecheck / build / test suite passes.

### US-002: View a Location
**Description:** As a user, I want to view a single location's details including its sub-locations and item instances.

**Acceptance Criteria:**
- [ ] `GET /api/v1/locations/:id` returns location JSON with `id`, `name`, `description`, `parent_id`, `created_at`, `updated_at`.
- [ ] `GET /api/v1/locations/:id/contents` returns `{ "sub_locations": [...], "instances": [...] }` — direct children only, not recursive.
- [ ] Item instances in contents include `id`, `definition_name`, `quantity`, `definition_id`.
- [ ] Non-existent location returns `404 Not Found`.
- [ ] Typecheck / build / test suite passes.

### US-003: List All Locations & Browse Tree
**Description:** As a user, I want to browse the full location hierarchy with item instances visible inline so I understand my inventory's structure at a glance.

**Acceptance Criteria:**
- [ ] `GET /api/v1/locations` returns a flat list of all locations (name, id, parent_id). Accepts optional `?parent_id=` filter.
- [ ] `GET /api/v1/locations/tree` returns the entire nested hierarchy from all root nodes down, with each node containing `id`, `name`, `description`, and `children` array.
- [ ] `GET /api/v1/locations/:id/children` returns flat array of direct children only (no nesting).
- [ ] **NEW** `GET /api/v1/browse` returns the full tree with locations + instances mixed in. See US-010 for details.
- [ ] Tree and list endpoints sorted alphabetically by `name` ascending.
- [ ] Typecheck / build / test suite passes.

### US-004: Update a Location
**Description:** As a user, I want to edit a location's name, description, or parent so I can reorganize my inventory.

**Acceptance Criteria:**
- [ ] `PUT /api/v1/locations/:id` updates `name`, `description`, and/or `parent_id`.
- [ ] Reparenting to a descendant of itself is rejected with `400 Bad Request` ("Cannot move a location into its own subtree").
- [ ] Reparenting the root location is rejected with `400 Bad Request`.
- [ ] Reparenting to a non-existent parent returns `400 Bad Request`.
- [ ] Reparenting to `null` clears the parent (makes it top-level; allowed for non-root locations).
- [ ] Partial updates allowed (only send fields you want to change).
- [ ] `updated_at` timestamp is auto-updated on every change.
- [ ] Typecheck / build / test suite passes.

### US-005: Delete a Location
**Description:** As a user, I want to delete an empty location so I can remove unused entries from my hierarchy.

**Acceptance Criteria:**
- [ ] `DELETE /api/v1/locations/:id` succeeds only if location has zero sub-locations AND zero item instances.
- [ ] If location has children or items, returns `409 Conflict` with `{ "error": "Cannot delete: location has X sub-locations and Y item instances", "code": "location_not_empty" }`.
- [ ] Root location (identified via `settings.root_location_id`) is never deletable — returns `400 Bad Request`.
- [ ] Non-existent location returns `404 Not Found`.
- [ ] Typecheck / build / test suite passes.

### US-006: Location Breadcrumb
**Description:** As a user, I want to see the full path from the root location down to the current location so I know exactly where I am in the hierarchy.

**Acceptance Criteria:**
- [ ] `GET /api/v1/locations/:id/breadcrumb` returns an ordered array of ancestor locations from root to the location itself (inclusive).
- [ ] Each breadcrumb entry includes `id` and `name`.
- [ ] The breadcrumb logic (recursive CTE or Go parent-chain walk) is implemented in the service layer so it can be reused for item instance breadcrumbs in PRD #8.
- [ ] A location with no parent returns a single-element breadcrumb (itself).
- [ ] Typecheck / build / test suite passes.

### US-007: Unified Browse Tree (UI) — Locations + Instances
**Description:** As a user, I want a visual tree browser showing both locations and item instances together so I can see my entire inventory structure in one view. This replaces the v1 locations-only tree.

**Acceptance Criteria:**
- [ ] Tree page (`/locations`) loads the full tree upfront via `GET /api/v1/browse`.
- [ ] Tree nodes are mixed: locations (folder icon) and instances (box/tag icon) rendered under each location.
- [ ] **Grouping:** Under each expanded location, sub-locations render first, then a subtle divider, then instances below.
- [ ] Each location node is clickable → navigates to `/locations/:id`. Each instance node is clickable → navigates to `/instances/:id`.
- [ ] Instance nodes show a quantity badge (e.g., "×5") to the right of the name.
- [ ] Container instances (those with `is_container = true` and `child_count > 0`) display a chevron and are expandable. Expanding a container instance lazy-loads child instances via `GET /api/v1/instances/:id/contents`.
- [ ] Non-container instances show no expand toggle.
- [ ] **Instance capping:** Maximum 50 instances shown per location. If truncated, a "(+N more)" link appears below the visible instances, linking to `/locations/:id` (the Location Detail page).
- [ ] **Two action buttons** per location node: "+" folder icon → opens CreateEditModal for a new sub-location (existing behavior); "+" box icon → opens CreateInstanceModal with the location pre-filled as `location_id`.
- [ ] **Two action buttons** per container instance node: "+" box icon → opens CreateInstanceModal with the container instance pre-filled as `parent_instance_id`. No location-add button on instance nodes.
- [ ] Mobile: full-width tree, indentation shows hierarchy, tap to expand/collapse, 44×44px touch targets for expand and action buttons.
- [ ] Desktop: indent markers, hover states, larger spacing between groups.
- [ ] Empty state: if only the root location exists with no instances, show "No locations or items yet — tap + to add" prompt.
- [ ] **[UI]** Verified in browser on both 375px and 1920px viewports.

### US-008: Create/Edit Location Form (UI)
**Description:** As a user, I want to add a sub-location inside the location I'm currently viewing, or edit an existing location. **Creating a location is contextual — you are always placing it inside another location (except the root, see below).**

**Acceptance Criteria:**
- [ ] **"Add Sub-Location" button is ONLY available on the Location Detail page.** It creates a location with the current location pre-selected as parent. There is NO standalone "Create Location" page/route and NO create-location button on any instance page.
- [ ] **Root-level location creation:** creating a location with no parent (top-level) has no obvious UI entry point yet since the user is always "inside" a location context. This is an open question — deferred to implementation. The `parent_id` can be set to `null` via the API, but the UI flow for root-level creation is not designed in v1.
- [ ] Form opens as a modal (mobile: bottom sheet; desktop: centered dialog via Radix Dialog).
- [ ] Fields: Name (required, text input), Description (optional, textarea), Parent Location (pre-filled from context, read-only).
- [ ] When editing: parent dropdown is visible and editable (for reparenting). The dropdown excludes the location itself and all its descendants to prevent cycles (mirrors server validation).
- [ ] Inline validation errors appear under each field on blur/submit.
- [ ] Successful save closes the modal and invalidates the `['locations']` TanStack Query cache.
- [ ] For edit: form is pre-populated with current values.
- [ ] **[UI]** Verified in browser.

### US-009: Delete Location with Guard (UI)
**Description:** As a user, I want clear feedback when I attempt to delete a location that still contains items or sub-locations.

**Acceptance Criteria:**
- [ ] Delete button triggers a confirmation dialog.
- [ ] If deletion succeeds: location removed from tree, query cache invalidated, success toast.
- [ ] If deletion blocked (409): dialog closes, error toast shows "Cannot delete '[name]': it contains X sub-locations and Y items. Move them first."
- [ ] Delete button on the root location is hidden/disabled with a tooltip: "Root location cannot be deleted."
- [ ] **[UI]** Verified in browser.

### US-010: Unified Browse Endpoint (API)
**Description:** As the frontend, I need a single endpoint that returns the full location tree with instances attached to each node so the unified browse tree can render on initial load.

**Acceptance Criteria:**
- [ ] `GET /api/v1/browse` returns the full nested location tree with `instances` array on each location node.
- [ ] Each location node has `kind: "location"`, recursive `children` (sub-locations), and `instances` array.
- [ ] Each instance entry includes: `id`, `definition_id`, `definition_name`, `quantity`, `is_container`, `child_count`.
- [ ] Instances are capped at **50 per location** node. When capped, `instance_truncated: true` and `instance_count` reflects the total (untruncated) count.
- [ ] Instance `child_count` reflects the number of child instances inside a container (0 for non-containers). The container's children are NOT recursively expanded in the browse response — they are loaded on demand via `/instances/:id/contents`.
- [ ] Locations and instances are sorted alphabetically by name (locations first, then instances grouped after).
- [ ] Container instances at root-level (i.e., instances with `location_id` = root and `is_container = true`) are included in the root location's `instances` array.
- [ ] Typecheck / build / test suite passes.

### US-011: Instance Management from Tree (UI)
**Description:** As a user, I want to create new instances directly from the browse tree without navigating to a detail page.

**Acceptance Criteria:**
- [ ] "+" box icon on a location node opens `CreateInstanceModal` with `location_id` pre-filled.
- [ ] "+" box icon on a container instance node opens `CreateInstanceModal` with `parent_instance_id` pre-filled.
- [ ] After successful creation, the tree node's instance list is invalidated and re-fetched (via TanStack Query invalidation of `['locations', 'tree-instances', locationId]`).
- [ ] Successful creation shows a success toast. Failed creation shows an error toast.
- [ ] **[UI]** Verified in browser.

---

## 5. Functional & Technical Requirements

### 5.1 Database Dependencies

The `locations` table and `settings.root_location_id` column are defined in the initial migration of `prd-database-schema.md`. This PRD introduces no additional schema changes.

**FR-1:** The root location is identified by `settings.root_location_id`. It cannot be deleted or reparented. Renaming is allowed.

**FR-2:** Auto-seeding logic (defined in `prd-database-schema.md` US-003) must insert a root location named "Home" and store its UUID in `settings.root_location_id`.

### 5.2 REST API Endpoints

All endpoints under `/api/v1/locations` except `/api/v1/browse` (which is a top-level route).

| Method | Path | Description | Request Body | Response |
|---|---|---|---|---|
| `GET` | `/locations` | Flat list, optional `?parent_id=` filter | — | `Location[]` |
| `GET` | `/locations/tree` | Full recursive tree (locations only) | — | `TreeNode[]` |
| `GET` | `/locations/:id` | Single location | — | `Location` |
| `GET` | `/locations/:id/children` | Direct children only | — | `Location[]` |
| `GET` | `/locations/:id/contents` | Sub-locations + item instances | — | `{ sub_locations: Location[], instances: InstanceSummary[] }` |
| `GET` | `/locations/:id/breadcrumb` | Ancestor chain root→self | — | `BreadcrumbNode[]` |
| `POST` | `/locations` | Create | `{ name, description?, parent_id? }` | `Location` (201) |
| `PUT` | `/locations/:id` | Update (partial) | `{ name?, description?, parent_id? }` | `Location` |
| `DELETE` | `/locations/:id` | Delete (guarded) | — | 204 or 409 |
| **`GET`** | **`/browse`** | **NEW** Full unified tree (locations + instances) | — | `BrowseNode[]` |

**FR-2:** `POST /locations` validates:
- `name`: required, 2–200 characters.
- `description`: optional, max 2000 characters.
- `parent_id`: optional. If provided, must reference an existing location. If omitted, location is top-level.

**FR-3:** `PUT /locations/:id` supports partial updates. Only provided fields are changed. Setting `parent_id` to `null` makes the location top-level (unless it's the root).

**FR-4:** Reparenting (via PUT or POST) must verify the new parent is NOT a descendant of the target location. Use a recursive CTE or Go service-loop to traverse the subtree.

**FR-5:** `DELETE /locations/:id` must check:
1. Location is not the root (return 400).
2. Location has no sub-locations (return 409 with count).
3. Location has no item instances directly at this location (return 409 with count).
All three checks happen in a single transaction.

**FR-6:** `GET /locations/tree` returns recursively nested structure:

```json
[
  {
    "id": "uuid",
    "name": "Home",
    "description": "...",
    "children": [
      {
        "id": "uuid",
        "name": "Living Room",
        "description": "...",
        "children": []
      }
    ]
  }
]
```

Built from the flat `locations` table using a Go service that groups children by `parent_id`.

**FR-7:** `GET /locations/:id/contents` returns:

```json
{
  "sub_locations": [
    { "id": "...", "name": "...", "description": "..." }
  ],
  "instances": [
    { "id": "...", "definition_id": "...", "definition_name": "...", "quantity": 5 }
  ]
}
```

Item instances are NOT fetched recursively (direct children only). Recursive item contents is a concern for PRD #8.

**FR-8:** `GET /locations/:id/breadcrumb` returns ordered array:

```json
[
  { "id": "root-uuid", "name": "Home" },
  { "id": "child-uuid", "name": "Living Room" },
  { "id": "requested-uuid", "name": "Bookshelf" }
]
```

Last element is the requested location. First element is the root ancestor (location with NULL parent_id). Implemented as a recursive CTE:

```sql
WITH RECURSIVE ancestors AS (
    SELECT id, name, parent_id, 0 AS depth FROM locations WHERE id = ?
    UNION ALL
    SELECT l.id, l.name, l.parent_id, a.depth + 1
    FROM locations l JOIN ancestors a ON l.id = a.parent_id
)
SELECT id, name FROM ancestors ORDER BY depth DESC;
```

**FR-8b:** `GET /api/v1/browse` returns the full unified tree:

```json
[
  {
    "id": "uuid",
    "name": "Home",
    "description": "...",
    "kind": "location",
    "children": [
      {
        "id": "child-uuid",
        "name": "Living Room",
        "description": "...",
        "kind": "location",
        "children": [],
        "instances": [
          {
            "id": "inst-uuid",
            "definition_id": "...",
            "definition_name": "Box of Screws",
            "quantity": 1,
            "is_container": true,
            "child_count": 3
          }
        ],
        "instance_count": 12,
        "instance_truncated": false
      }
    ],
    "instances": [],
    "instance_count": 0,
    "instance_truncated": false
  }
]
```

**Implementation details:**
- Built from flat `locations` table (same two-pass algorithm as `GetTree()`) but augmented with instance data per node.
- Instances per location are fetched with `LIMIT 51` (50 + 1 to detect truncation). Sorted alphabetically by `definition_name`.
- `instance_truncated` = `true` when more than 50 instances exist at a location. `instance_count` = total unfiltered count.
- `is_container` and `child_count` come from the item definition (`is_container` field) and a subquery counting child instances.
- Container instance children are NOT recursively expanded — the frontend lazy-loads them via `/instances/:id/contents`.
- Only instances with a direct `location_id` (not `parent_instance_id`) are included at each location node.

### 5.3 Frontend

**FR-9:** Use TanStack Query with hierarchical keys:
- `['browse']` — unified full tree (locations + instances).
- `['locations']` — flat list.
- `['locations', 'tree']` — full tree (locations only).
- `['locations', id]` — single location.
- `['locations', id, 'children']` — direct children.
- `['locations', id, 'contents']` — contents.
- `['locations', id, 'breadcrumb']` — breadcrumb.
- `['locations', id, 'instances']` — instances at a location (lazy-loaded on expand).

**FR-10:** On any successful mutation (create/update/delete), invalidate `['browse']`, `['locations']` and related keys. Use targeted invalidation — don't invalidate the entire tree if only one node changed.

**FR-11:** The parent dropdown in create/edit forms must exclude:
- The location being edited (can't be its own parent)
- All descendants of the location being edited (prevents cycles)
- (Optional optimization: fetch the full tree and filter client-side)

**FR-12:** Unified browse tree (`LocationTree` / `BrowseTree` component):
- Initial load via `GET /api/v1/browse` — full tree with locations + capped instances.
- On expand of a location node: lazy-load instances (if truncated) via `GET /api/v1/instances?location_id=:id&limit=50`.
- On expand of a container instance node: lazy-load child instances via `GET /api/v1/instances/:id/contents`.
- Node types: location (folder icon), container instance (box icon, expandable), non-container instance (tag icon, leaf).
- Grouping: under each location, sub-locations render first (from `children`), then instances (from `instances` array).
- Two action buttons per location node: +folder-icon → create sub-location modal, +box-icon → create instance modal.
- One action button per container instance node: +box-icon → create instance modal (inside container).
- Expanded state is tracked locally in component state. Lazy-loaded instance data is cached by TanStack Query.

**FR-13:** All forms use HTML5 validation as first line + controlled component validation before submit.

---

## 6. Edge Cases & Failure Modes

| Scenario | System Behavior |
|---|---|
| User creates a location with `parent_id` pointing to a non-existent location | Server returns `400 Bad Request` with `"code": "parent_not_found"`. |
| User tries to move a location under one of its own descendants | Server detects cycle via recursive CTE and returns `400 Bad Request` with `"code": "cycle_detected"`. |
| User tries to move the root location | Server returns `400 Bad Request` with `"code": "root_immutable"`. |
| User deletes a location while another request is creating a sub-location under it | SQLite serialized writes prevent race. Either delete or create succeeds first; the other gets a FK violation → 409 or 404. |
| User requests contents of a location with 1000+ instances | `contents` endpoint returns all instances with pagination as a future enhancement. For v1, return up to 500 and add a `total_count` field. |
| Location name is empty or whitespace-only | Server rejects with `400 Bad Request` (`"code": "validation_failed"`, field: `name`). |
| User tries to create a location with a name that's a duplicate of a sibling | Allowed in v1 (no uniqueness constraint on name within parent). Users can have multiple "Shelf" locations under different or same parents. |
| Frontend tree has 50+ expanded nodes | Each expand triggers a fetch via TanStack Query with deduplication. No performance issue. |
| User navigates directly to `/locations/:id` by URL | React Router renders the Location Detail view. TanStack Query fetches the location by ID. If 404, show "Location not found" page. |
| Breadcrumb requested for a non-existent location | Returns `404 Not Found`. |

---

## 7. Non-Goals & Scope Boundaries

- **Manual sort ordering:** Alphabetical only in v1. `display_order` column deferred to v2.
- **Bulk operations:** No multi-select, batch delete, or batch move.
- **Drag-and-drop reordering/reparenting:** v1 uses form-based parent selection only.
- **Location-level tags:** Tags apply to item definitions only (per overarching PRD).
- **Standalone "Create Location" page or global button:** Location creation is always contextual — initiated from a Location Detail page with the parent pre-filled. No `/locations/new` route exists.
- **Creating locations from instance pages:** Locations are created within locations only, never within items.
- **Search within location tree:** Deferred to PRD #10 (Search).
- **Location images/photos:** Deferred to future photo attachment feature.
- **Recursive item instance contents in browse endpoint:** The `/browse` endpoint does NOT recursively expand container instance children. Container children are lazy-loaded on expand via `/instances/:id/contents`.
- **Recursive item instance contents in `/contents`:** `contents` returns direct instances only. Recursive item-in-item resolution is PRD #8.

---

## 8. Open Questions & Deferred Items

| # | Question | Status |
|---|---|---|
| OQ-1 | Should the tree include instances and instance counts? | **Resolved — v2** implements a unified browse tree (`/browse` endpoint, `BrowseTree` component) with instances visible inline under each location. Instance count (`instance_count`) and truncation flag (`instance_truncated`) are part of the browse response. |
| OQ-2 | Should moving a location also move its item instances (current behavior) or leave them orphaned? | Resolved — moving a location does NOT affect item instances (they stay at this location); changing `parent_id` only affects the location's position in the tree. Item instances are bound to the location by `location_id`, not by tree position. |
| OQ-3 | Should location names be unique within a parent? | Deferred — v1 allows duplicates. Revisit if user feedback demands it. |
| OQ-4 | Pagination for contents endpoint when > 500 instances? | Deferred — v1 returns all with a hard cap. Add cursor pagination in v2. |
| OQ-5 | How should the user create a root-level location (no parent)? The UI is always contextual — the user is inside a location, so creating a sub-location is natural, but creating a sibling root is not. | Open — the API supports `parent_id = null` but there is no v1 UI entry point for it. Possible solutions: a "Create Top-Level Location" option in the location tree empty area, or a button on the root location's detail page. Deferred to implementation. |
| OQ-6 | **v2** — Should the browse endpoint support a `?root_id=` filter to show only a subtree? | Deferred — v2 returns the full tree. Subtree filtering can be implemented client-side or added as a query param in v3. |
