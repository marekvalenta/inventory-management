# PRD: Search — InventoryManagement

> **Status:** Done v1.0
> **Scope:** Name-based search across locations, definitions, and instances. Persistent header search bar on all pages, quick-results dropdown, full search results page with entity type filtering, and database indexes for LIKE performance.

---

## 0. Cross-PRD Consistency Report

### PRDs Reviewed
- `prd-overarching-architecture.md` — Search endpoint group at `/api/v1/search` (§5.2), dashboard quick search bar (§6.1), non-goal boundary (FTS5 deferred to v2, §11), extensibility hook (§12).
- `prd-database-schema.md` — `locations`, `item_definitions`, `item_instances` tables; OQ-1 deferred index decision to this PRD.
- `prd-backend-architecture.md` — Go layering (handler/service/db), chi router, error mapping, payload validation.
- `prd-frontend-architecture.md` — React Router v6, TanStack Query, CSS Modules, Radix UI, mobile/desktop layouts.
- `prd-visual-design.md` — Golden Amber tokens, cards, inputs, empty/loading/error states, component patterns.
- `prd-dashboard.md` — Search bar placeholder (US-005, FR-17) — this PRD wires it up and updates the dashboard PRD.
- `prd-locations.md` — Location CRUD, tree, contents, breadcrumb.
- `prd-item-definitions.md` — Definition CRUD, field resolution, instances summary.
- `prd-item-instances.md` — Instance CRUD, move/split, breadcrumb, container nesting.
- `prd-tags.md` — Tag CRUD (no search over tags in v1).
- `prd-testing.md` — Integration test patterns, seed data.
- `prd-docker-deployment.md` — No direct impact.

### Conflicts & Resolutions

| # | Conflict | PRDs Involved | Resolution |
|---|---|---|---|
| 1 | Dashboard PRD US-005 describes the search bar as "non-functional placeholder" — "Enter or clicking the icon does nothing." | `prd-dashboard.md` | **Update dashboard PRD.** US-005 and FR-17 changed to describe the wired-up search bar: pressing Enter or clicking the magnifying glass navigates to `/search?q=...` and opens the quick-results dropdown on typing. The `SearchBar` component now imports from this PRD's implementation. |
| 2 | Dashboard PRD conflict #4 anticipated this — "Dashboard renders a non-functional search bar placeholder. PRD #12 wires it up." | `prd-dashboard.md` | Resolved. This PRD implements the wiring. |
| 3 | No `/search` route exists in frontend architecture or visual design PRDs. New page + layout required. | `prd-frontend-architecture.md`, `prd-visual-design.md` | This PRD defines the `/search` route, `SearchResultsPage` component, and page layout using Golden Amber tokens. The visual design PRD's existing component patterns (cards, inputs, empty/loading/error states) are reused — no new design tokens needed. |
| 4 | Overarching §5.2 lists `/api/v1/search` as an endpoint group with zero detail. | `prd-overarching-architecture.md` | This PRD defines the concrete endpoint: `GET /api/v1/search?q=...&type=...&limit=...`. |
| 5 | Database PRD OQ-1 asks whether to index `name` columns. | `prd-database-schema.md` | **Add indexes.** New migration `00002_search_indexes.sql` adds indexes on `locations.name` and `item_definitions.name`. |
| 6 | Item instances have no `name` column — search works via definition name JOIN. | `prd-database-schema.md` | Instance search queries `item_instances` JOIN `item_definitions` on `definition_id`, matching against `item_definitions.name`. |
| 7 | Overarching §11 says FTS5 is a v2 non-goal. §12 says API must be extensible to support it. | `prd-overarching-architecture.md` | v1 uses SQL `LIKE '%term%'`. The API design (`?q=` query param, entity-type grouping, paginated results) is compatible with a future FTS5 backend swap — only the service implementation changes, not the request/response contract. |
| 8 | Persistent header search bar on every page conflicts with the dashboard's prominent centered search bar. Two search bars on one page is redundant. | `prd-dashboard.md` | **Dashboard page suppresses the header search bar.** The dashboard has its own prominent, centered search bar (per `prd-dashboard.md` US-005). On the dashboard route `/`, the header search bar (mobile icon + desktop input) is hidden. On all other routes, the header search bar is visible. The `SearchBar` component is shared — both the dashboard's centered variant and the header's inline variant use the same `SearchBar` component with different `variant` props. |

### Confirmed Alignments
- Data model: All queries are read-only against existing tables (`locations`, `item_definitions`, `item_instances`). One schema change: new migration with indexes.
- API patterns: `GET /api/v1/search?q=...` follows the `/api/v1/` prefix, returns JSON, uses standard error format `{"error":"...","code":"..."}`.
- Error mapping: `ErrInvalidInput` → 400 per `prd-backend-architecture.md` TR-2.
- Go layering: handler → service → db, chi router.
- UI: CSS Modules + CSS variables (Golden Amber tokens), Radix UI primitives, TanStack Query, React Router v6. Mobile-first: 375px–1920px range.
- Visual design: Result cards use card pattern (§6.3), search input uses form input pattern (§6.2), dropdown uses shadow/radius tokens, empty/loading/error states follow §6.7–6.9.
- Scope does not contradict any non-goal in overarching PRD: no FTS5, no filters beyond entity type, no ranking algorithm beyond match quality.
- Indexes are additive only — no existing data migration, no breaking schema changes.

---

## 1. Overview & Problem Statement

Users need to quickly find locations, item definitions, and/or item instances by name without browsing through the tree or list views. The current app has a non-functional search bar placeholder on the dashboard — this PRD makes it functional and extends search access to every page via a persistent header search bar.

### Core Deliverables
1. **Persistent header search bar** — visible on every page (mobile: collapsible icon; desktop: inline input). Includes a quick-results dropdown that shows top matches across all three entity types as the user types, with a "Show all results..." link to the full search results page.
2. **Search results page** at route `/search?q=...` — grouped by entity type by default, with entity type tabs to narrow to a single type.
3. **Single REST API endpoint** — `GET /api/v1/search?q=...&type=...&limit=...` — handles both quick-dropdown and full-page queries.
4. **Database indexes** on `locations.name` and `item_definitions.name` for LIKE query performance.
5. **Dashboard PRD update** — search bar placeholder upgraded to functional search bar.

---

## 2. Goals & Measurable Success Metrics

| Goal | Metric |
|---|---|
| Fast search response | `GET /api/v1/search` responds in < 100ms p95 with realistic seed data and cold cache |
| Global search access | Search bar accessible on every page (all 6 routes) |
| Quick dropdown speed | Quick-results dropdown opens within 200ms of typing (debounced 300ms) |
| Grouped clarity | User can see top matches by entity type without scrolling on mobile |
| Type filtering | User can narrow results to a single entity type via tabs on the results page |
| Direct navigation | Every result click navigates to the entity's detail page |
| Zero breaking changes | No existing API or UI regressions |
| Index performance | Indexes improve LIKE query performance without measurable write overhead |

---

## 3. Critical Risks & Technical Assumptions

| Risk | Mitigation |
|---|---|
| LIKE `%term%` scan slow with many entities | Indexes on `name` columns accelerate prefix scanning (LIKE still requires full scan for leading `%`, but indexes help SQLite optimizer). At v1 scale (< 1000 entities), performance is acceptable. FTS5 upgrade path in v2. |
| Search bar in header clutters mobile UI | Mobile: collapsible magnifying glass icon (44x44px tap target). Tap expands an inline input overlay. Quick-results dropdown appears below. Desktop: persistent input field — ample space at 1024px+ viewport. |
| Quick dropdown causes excessive API calls during typing | Frontend debounces input at 300ms. TanStack Query deduplicates in-flight requests. `staleTime: 10_000` keeps recent results cached. |
| Search results page URL with query parameter can be bookmarked/shared | Yes — `/search?q=term&type=locations` is a standard URL. TanStack Query uses `['search', { q, type }]` key. Back/forward navigation works via React Router. |
| Empty search term triggers unnecessary query | Frontend skips the query if `q` is empty or < 2 characters. Dropdown only appears when the input has ≥ 2 chars. |
| Migration adds indexes on existing large DB | Index creation is fast on SQLite (< 100ms for < 1000 rows). Migration runs on startup before server binds. |

---

## 4. User Stories & Acceptance Criteria

### US-001: Persistent Header Search Bar (Mobile)
**Description:** As a mobile user, I want to trigger search from any page via a magnifying glass icon in the header, so I can find items quickly without navigating back to the dashboard.

**Acceptance Criteria:**
- [ ] A magnifying glass icon (24x24px, `--color-text-secondary`) appears in the mobile header on every page, positioned left of the `[+]` button.
- [ ] Tapping the icon slides the header down/expands to reveal a search input field (height 44px, full-width, bg `--color-bg-surface`, border `--color-border`, `--radius-md`).
- [ ] The input auto-focuses on expand. Typing ≥ 2 characters triggers the quick-results dropdown (US-003) after a 300ms debounce.
- [ ] Tapping outside the search area or pressing the back button collapses the search input and restores the normal header.
- [ ] The expanded search input includes a magnifying glass icon on the left and a clear (×) button on the right when text is present. Pressing × clears the input.
- [ ] The search bar is accessible via the mobile bottom nav's page transitions — the header state persists per page.
- [ ] Tap target for the magnifying glass icon: ≥ 44x44px.
- [ ] **[UI]** Verified in browser on 375px viewport.

### US-002: Persistent Header Search Bar (Desktop)
**Description:** As a desktop user, I want a persistent search input in the top header bar on every page so I can search without extra clicks.

**Acceptance Criteria:**
- [ ] A search input field is rendered in the desktop header bar on every page, positioned between the page title (left) and the action button (right).
- [ ] Styled as a form input: bg `--color-bg-surface-alt`, border `--color-border`, `--radius-sm`, height 36px, width 240px, padding `--space-md`.
- [ ] Includes a magnifying glass icon (16x16px, `--color-text-secondary`) positioned inside the input on the left.
- [ ] Placeholder text: "Search..." (`--text-small`, `--color-text-secondary`).
- [ ] Typing ≥ 2 characters triggers the quick-results dropdown (US-003) after a 300ms debounce.
- [ ] The dropdown appears below the search input, aligned to the input width (240px min, expands to fit content up to 400px).
- [ ] Clicking outside the dropdown or pressing Escape closes it.
- [ ] Pressing Enter navigates directly to `/search?q=<term>` (the full results page) and closes the dropdown.
- [ ] The input includes a clear (×) button on the right when text is present. Visible only on hover/focus.
- [ ] **[UI]** Verified in browser on 1920px viewport.

### US-003: Quick-Results Dropdown
**Description:** As a user, I want to see instant top results as I type in the search bar, so I can jump directly to what I'm looking for without loading a full results page.

**Acceptance Criteria:**
- [ ] Dropdown appears below the search input when ≥ 2 characters are typed, after a 300ms debounce.
- [ ] Dropdown styling: bg `--color-bg-surface`, border `--color-border`, `--radius-md`, shadow `--shadow-dropdown`, max-height 360px (scrollable), z-index above page content.
- [ ] Top 3 results per entity type, grouped into sections with headers:
  - **Locations (N):** 3 location rows from `GET /api/v1/search?q=term&limit=3`
  - **Definitions (N):** 3 definition rows
  - **Instances (N):** 3 instance rows
  - `N` = total matching count for that type (from `total_counts` in API response).
- [ ] Each section header: `--text-caption`, `--color-text-secondary`, uppercase, padded.
- [ ] Each result row shows:
  - Entity type icon (16x16px): archive for locations, clipboard for definitions, cube for instances.
  - Entity name (`--text-body-strong`).
  - Subtitle line (`--text-caption`, `--color-text-secondary`): location → parent location name (if not root); definition → unit (if set); instance → "xQTY — in LocationName".
- [ ] If a section has 0 results, the section is hidden entirely.
- [ ] A "Show all results..." link is present at the bottom of the dropdown (even when all sections are collapsed, "Show all results for 'term'" is shown), navigating to `/search?q=<term>`.
- [ ] Clicking any result row navigates to the entity's detail page (`/locations/:id`, `/definitions/:id`, `/instances/:id`) and closes the dropdown.
- [ ] Keyboard navigation: Arrow Up/Down moves highlight through result rows. Enter selects the highlighted result. Escape closes the dropdown.
- [ ] TanStack Query key: `['search', { q, limit: 3 }]` with `staleTime: 10_000`, `enabled: q.length >= 2`.
- [ ] **[UI]** Verified in browser.

### US-004: Search Results Page (Full Page)
**Description:** As a user, I want a full-page search results view showing all matches in one place, with the ability to filter by entity type.

**Acceptance Criteria:**
- [ ] Route `/search` renders the `SearchResultsPage` component.
- [ ] Query parameter `?q=` is read from the URL. If `q` is empty or missing, the page shows a prompt: "Enter a search term to find items." with a focused search input.
- [ ] The search input at the top of the page is pre-filled with the query and auto-focused. It mirrors the header search bar — typing here updates the URL and triggers a new search.
- [ ] Below the search input: entity type tabs:
  - **Tab bar:** `[All (24)] [Locations (3)] [Definitions (5)] [Instances (16)]` — counts from `total_counts` in API response.
  - Active tab: bg `--color-accent-muted`, text `--color-accent`, bottom border 2px `--color-accent`.
  - Inactive tab: text `--color-text-secondary`, hover bg `--color-bg-surface-alt`.
  - Default active tab: "All" (shows grouped results).
  - Selecting a specific type tab shows a flat list of only that type's results (sorted by match relevance, descending).
  - Tabs update the URL: `/search?q=term` (All), `/search?q=term&type=locations`, `/search?q=term&type=definitions`, `/search?q=term&type=instances`.
- [ ] **"All" tab — grouped view:**
  - Sections for each entity type with counts, same as dropdown but with ALL matching results (no cap).
  - Section headers: "Locations (3)", "Definitions (5)", "Instances (16)" — `--text-h3`.
  - Results in card rows (bg `--color-bg-surface`, `--radius-sm`, padding `--space-lg`, margin-bottom `--space-sm`).
  - Each card navigates to the entity's detail page on click/tap.
  - If a type has 0 results, its section is hidden.
- [ ] **Specific type tab — flat list:**
  - Single flat list of results for that type only.
  - Sorted by match relevance (same sort as "All" view for that type).
  - Cards are full-width, stacked vertically.
  - Empty result for a specific type: "No [type] match 'term'" with suggestion to try a different term.
- [ ] **Empty state (no results at all):** Centered message: "No results for 'xyz'" with subtitle "Try a different search term or check the spelling." No illustration. A "Browse Locations" button links to `/locations` as a fallback action.
- [ ] **Loading state:** Skeleton cards (3-4 per section) with shimmer animation per visual PRD §6.8.
- [ ] **Error state:** Inline error with retry button per visual PRD §6.9.
- [ ] TanStack Query key: `['search', { q, type }]` with `enabled: q.length >= 2`.
- [ ] Mobile: tabs scroll horizontally, results stacked vertically.
- [ ] Desktop: tabs in a row, max-width 800px centered.
- [ ] **[UI]** Verified in browser on both 375px and 1920px viewports.

### US-005: Search API Endpoint
**Description:** As a frontend, I want a single search endpoint that returns matching locations, definitions, and instances, with optional type filtering and result limiting.

**Acceptance Criteria:**
- [ ] `GET /api/v1/search?q=term` returns JSON with the structure defined in FR-2.
- [ ] Query params:
  - `q` — required, search term (2–200 chars).
  - `type` — optional, filter to entity type: `"all"` (default), `"locations"`, `"definitions"`, `"instances"`.
  - `limit` — optional, integer. Caps results per group. Omit or 0 = unlimited (subject to max 100 per group).
- [ ] Invalid `q` (< 2 chars or empty) returns `400 Bad Request` with `{"error": "Search term must be at least 2 characters", "code": "invalid_query"}`.
- [ ] Invalid `type` returns `400 Bad Request` with `{"error": "Invalid type: 'xyz'. Valid types: all, locations, definitions, instances", "code": "invalid_type"}`.
- [ ] Invalid `limit` (negative, non-integer) returns `400 Bad Request`.
- [ ] When `type=all` (or omitted): response includes all three groups plus `total_counts`.
- [ ] When `type=locations` (or specific type): response includes only that group, no other groups, still includes `total_counts` for all types (so the tabs show accurate counts).
- [ ] `total_counts` always reflects total matches ignoring `limit` (so the dropdown and tabs show accurate N).
- [ ] Results sorted by match quality — exact match first, then starts-with, then contains. Within each tier, alphabetical by name.
- [ ] Results capped at 100 per group when `limit` is not specified.
- [ ] Response time < 100ms p95 with 500+ entities.
- [ ] Typecheck / build / test suite passes.

### US-006: Database Index Migration
**Description:** As a backend system, I need indexes on name columns to support fast LIKE queries for search.

**Acceptance Criteria:**
- [ ] New migration file `migrations/00002_search_indexes.sql`:
  ```sql
  -- +goose Up
  CREATE INDEX idx_locations_name ON locations(name);
  CREATE INDEX idx_item_definitions_name ON item_definitions(name);

  -- +goose Down
  DROP INDEX IF EXISTS idx_locations_name;
  DROP INDEX IF EXISTS idx_item_definitions_name;
  ```
- [ ] Migration runs on startup via goose. Indexes are created if they don't exist (idempotent).
- [ ] No data migration — indexes are additive.
- [ ] Typecheck / build / test suite passes.

---

## 5. Functional & Technical Requirements

### 5.1 Database Changes

**FR-1:** New migration `migrations/00002_search_indexes.sql` adds indexes:
```sql
-- +goose Up
-- +goose StatementBegin
CREATE INDEX idx_locations_name ON locations(name);
CREATE INDEX idx_item_definitions_name ON item_definitions(name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_locations_name;
DROP INDEX IF EXISTS idx_item_definitions_name;
-- +goose StatementEnd
```

No other schema changes. No new tables, no new columns.

### 5.2 REST API Endpoint

| Method | Path | Description | Query Params | Response |
|---|---|---|---|---|
| `GET` | `/search` | Search entities by name | `q` (required), `type` (optional, default `"all"`), `limit` (optional) | `SearchResponse` |

**FR-2:** `SearchResponse` shape:

```json
{
  "locations": [
    {
      "id": "uuid",
      "name": "Living Room",
      "parent_id": "uuid or null",
      "parent_name": "Home or null"
    }
  ],
  "definitions": [
    {
      "id": "uuid",
      "name": "M3 Screw",
      "unit": "pcs",
      "parent_def_name": "Screw or null"
    }
  ],
  "instances": [
    {
      "id": "uuid",
      "definition_id": "uuid",
      "definition_name": "M3 Screw",
      "unit": "pcs or null",
      "location_id": "uuid or null",
      "location_name": "Workshop or null",
      "parent_instance_id": "uuid or null",
      "parent_instance_name": "Toolbox or null",
      "quantity": 50
    }
  ],
  "total_counts": {
    "locations": 3,
    "definitions": 5,
    "instances": 12
  }
}
```

- `locations[].parent_name`: resolved from the parent location. `null` for root location or top-level.
- `definitions[].parent_def_name`: resolved from parent definition. `null` for root definitions.
- `definitions[].unit`: `null` if not set.
- `instances[].definition_name`: always present (via JOIN on `item_definitions`).
- `instances[].unit`: from the definition. `null` if not set.
- `instances[].location_name`: resolved from `location_id`. `null` if instance is placed inside a container.
- `instances[].parent_instance_name`: resolved from `parent_instance_id`. Format: `"definition_name (xQTY)"`. `null` if instance is directly at a location.
- `total_counts` always reflects total matches per type, ignoring `limit`. When `type` is specified, the non-matching groups are omitted from the response body BUT their `total_counts` entries are still present.

**FR-3:** When `type=all` (or omitted), `locations`, `definitions`, and `instances` arrays are all present in the response.

**FR-4:** When `type=locations` (or `definitions` or `instances`), only the matching array is present. The other two arrays are omitted from the response. `total_counts` always includes all three keys regardless.

```json
// type=definitions
{
  "definitions": [ ... ],
  "total_counts": {
    "locations": 3,
    "definitions": 5,
    "instances": 12
  }
}
```

**FR-5:** Search query logic (service layer):

Locations:
```sql
SELECT l.id, l.name, l.parent_id, p.name AS parent_name
FROM locations l
LEFT JOIN locations p ON p.id = l.parent_id
WHERE l.name LIKE '%' || ? || '%'
ORDER BY
  CASE
    WHEN l.name = ? THEN 0          -- exact match
    WHEN l.name LIKE ? || '%' THEN 1 -- starts with
    ELSE 2                           -- contains
  END,
  l.name ASC
LIMIT ?
```

Definitions:
```sql
SELECT d.id, d.name, d.unit, pd.name AS parent_def_name
FROM item_definitions d
LEFT JOIN item_definitions pd ON pd.id = d.parent_def_id
WHERE d.name LIKE '%' || ? || '%'
ORDER BY
  CASE
    WHEN d.name = ? THEN 0
    WHEN d.name LIKE ? || '%' THEN 1
    ELSE 2
  END,
  d.name ASC
LIMIT ?
```

Instances:
```sql
SELECT
  i.id, i.definition_id, d.name AS definition_name, d.unit,
  i.location_id, l.name AS location_name,
  i.parent_instance_id,
  pi_def.name AS parent_instance_def_name,
  pi.quantity AS parent_instance_qty,
  i.quantity
FROM item_instances i
JOIN item_definitions d ON d.id = i.definition_id
LEFT JOIN locations l ON l.id = i.location_id
LEFT JOIN item_instances pi ON pi.id = i.parent_instance_id
LEFT JOIN item_definitions pi_def ON pi_def.id = pi.definition_id
WHERE d.name LIKE '%' || ? || '%'
ORDER BY
  CASE
    WHEN d.name = ? THEN 0
    WHEN d.name LIKE ? || '%' THEN 1
    ELSE 2
  END,
  d.name ASC
LIMIT ?
```

**FR-6:** Each query uses parameterized inputs to prevent SQL injection. Three parameter bindings per query: the `%term%` wildcard, the exact term (for exact match comparison), and the starts-with pattern.

**FR-7:** When `limit` is 0 or omitted from the request, a default cap of 100 per group is applied. When `limit` is explicitly set (e.g., `limit=3` for dropdown), that cap is used.

**FR-8:** The `total_counts` are computed via three lightweight `COUNT(*)` queries that run regardless of `type`. These execute with the same `LIKE` filter but without `LIMIT`.

### 5.3 Service Layer

**FR-9:** `SearchService` (in `internal/service/search.go`):

```go
type SearchService struct {
    db *sql.DB
}

type SearchParams struct {
    Query string
    Type  string // "all", "locations", "definitions", "instances"
    Limit int    // 0 = default cap (100)
}

type SearchResponse struct {
    Locations    []LocationResult    `json:"locations,omitempty"`
    Definitions  []DefinitionResult  `json:"definitions,omitempty"`
    Instances    []InstanceResult    `json:"instances,omitempty"`
    TotalCounts  TotalCounts         `json:"total_counts"`
}

type TotalCounts struct {
    Locations   int `json:"locations"`
    Definitions int `json:"definitions"`
    Instances   int `json:"instances"`
}
```

**FR-10:** `SearchService.Search(ctx, params) (*SearchResponse, error)` — main entry point:
1. Validate `params.Query` (≥ 2 chars).
2. Validate `params.Type` (must be valid or "all").
3. Compute `total_counts` first (3 lightweight COUNT queries, always run).
4. If `type=all` or `type=locations`: execute locations query.
5. If `type=all` or `type=definitions`: execute definitions query.
6. If `type=all` or `type=instances`: execute instances query.
7. Assemble and return response.

**FR-11:** Validation:
- `q` < 2 characters → return `ErrInvalidInput` (handler maps to 400).
- Invalid `type` → return `ErrInvalidInput`.

### 5.4 Handler Layer

**FR-12:** `SearchHandler` in `internal/handler/search.go`:
- `GET /api/v1/search` → reads `q`, `type`, `limit` from query params.
- Validates params, calls `SearchService.Search(ctx, params)`.
- Returns `200 OK` with `SearchResponse` JSON.
- Invalid params → `400 Bad Request` with standard error format.
- Follows the standard handler pattern: decode params → call service → format response.

### 5.5 Router Registration

**FR-13:** Search route registered in `internal/router/`:

```go
// Under r.Route("/api/v1/search", ...):
r.Get("/", searchHandler.Search)
```

### 5.6 Frontend

**FR-14:** Route: `/search` → `SearchResultsPage` component (reads `?q=` and `?type=` from URL).

**FR-15:** `SearchBar` component at `frontend/src/components/SearchBar.tsx` (shared, imported by both the header shell and `SearchResultsPage`):

```typescript
interface SearchBarProps {
  variant: 'header-mobile' | 'header-desktop' | 'page';
  initialValue?: string;
  onSearch: (query: string) => void;
}
```

- **`header-mobile`**: collapses to a magnifying glass icon. Expands to a full-width input on tap. Triggers quick dropdown on typing. Enter navigates to `/search?q=...`.
- **`header-desktop`**: persistent 240px input in the header bar. Triggers quick dropdown on typing. Enter navigates to `/search?q=...`.
- **`page`**: full-width input at the top of the search results page. Updates the URL on submit (`replace: false` for back-navigation support). No dropdown — the page itself is the full results view.

**FR-16:** TanStack Query keys:
- `['search', { q, type?, limit? }]` — single key for all search queries.
- `staleTime: 10_000` (10 seconds) for dropdown queries, `staleTime: 30_000` for page queries.
- `enabled: q.length >= 2` — skip queries for short terms.
- Debounce input at 300ms before firing query.

**FR-17:** `SearchResultsPage` component structure:

```
<SearchResultsPage>
  <SearchBar variant="page" />        ← pre-filled with q, updates URL on submit
  <TypeTabs />                         ← [All (24)] [Locations (3)] [Definitions (5)] [Instances (16)]
  {type === 'all' ? (
    <GroupedResults />                 ← sections for each type with full result lists
  ) : (
    <FlatResults type={type} />        ← single flat list for specific type
  )}
</SearchResultsPage>
```

**FR-18:** `QuickResultsDropdown` component:
- Rendered below the header search bar.
- Uses the same `['search', { q, limit: 3 }]` query key.
- Results grouped by type, 3 per group max.
- "Show all results..." link at bottom navigates to `/search?q=...`.
- Keyboard navigation: Arrow Up/Down + Enter + Escape.
- Click outside → close. Tapping magnifying glass → toggle.

**FR-19:** Result cards in both dropdown and page views are clickable `<Link>` components:
- Location → `/locations/:id`
- Definition → `/definitions/:id`
- Instance → `/instances/:id`

**FR-20:** Entity type badges (used in result cards):

```
[L] Location    [D] Definition    [I] Instance
```

- 16x16px icon + 2-char abbreviation
- `--text-caption`, `--color-text-secondary`
- bg `--color-bg-surface-alt`, `--radius-sm`, padding `--space-xs`
- Rendered to the left of the entity name in each result card.

**FR-21:** Visual design alignment:
- All colors, spacing, border-radius, shadows use CSS variables from `prd-visual-design.md` §3. No raw hex or pixel values.
- Search input: `--color-bg-surface`, border `--color-border`, `--radius-md`, height 44px (mobile) or 36px (desktop).
- Quick dropdown: bg `--color-bg-surface`, border `--color-border`, `--radius-md`, shadow `--shadow-dropdown`.
- Result cards: bg `--color-bg-surface`, `--radius-sm`, padding `--space-lg`, margin-bottom `--space-sm`.
- Type tabs: underline indicator pattern, `--color-accent` active, `--color-text-secondary` inactive.
- Fonts: Nunito 600 for section headers (`--text-h3`), DM Sans 400/500 for body/labels.
- Empty state: follows visual PRD §6.7 pattern.
- Loading state: skeleton cards per visual PRD §6.8.
- Error state: inline error per visual PRD §6.9.

---

## 6. Edge Cases & Failure Modes

| Scenario | System Behavior |
|---|---|
| Search term is empty or whitespace-only | Frontend: query disabled (`enabled: false`). If user navigates to `/search?q=`, page shows "Enter a search term" prompt. Backend: if `q` < 2 chars, returns `400 Bad Request`. |
| Search term is 1 character | Frontend: query disabled. Input does not trigger search. Backend: if called directly, returns `400 Bad Request`. |
| Search term contains SQL special characters (`%`, `_`, `'`) | Parameterized queries prevent injection. Wildcard characters in `q` are treated as literal characters by the application — they're embedded inside the `LIKE '%term%'` pattern, so `%` and `_` have their LIKE wildcard meaning (this is intentional — searching for "%" would match everything, but that's an edge case the user is unlikely to hit). |
| Search term is very long (200+ characters) | Frontend limits input to 200 chars. Backend validates: > 200 chars → `400 Bad Request`. |
| Zero matches across all entity types | API returns `locations: [], definitions: [], instances: [], total_counts: { locations: 0, definitions: 0, instances: 0 }`. Frontend shows "No results for 'term'" empty state. |
| `type` parameter is invalid | Backend returns `400 Bad Request` with valid types listed. Frontend: tabs control the `type` param so invalid values shouldn't occur from UI. URL tampering → error state. |
| `limit` parameter is negative or non-integer | Backend returns `400 Bad Request`. Frontend controls `limit` (always 3 for dropdown, unset for page). |
| Rapid typing triggers many API calls | Frontend debounces at 300ms. TanStack Query cancels previous in-flight queries for the same key. Only the latest query completes. |
| User types, selects a result from dropdown, then hits back | Browser back navigation returns to the page they were on before the dropdown interaction (no URL change for dropdown navigation, so back returns to the previous page). |
| User navigates to `/search` with no `q` param | Page renders with an empty, focused search input and the prompt "Enter a search term to find items." No API call. |
| User navigates to `/search?q=term` directly (bookmark) | Page loads, fetches search results for `term`, displays them. TanStack Query fetches on mount. |
| User clears the search input on the results page | URL updates to `/search`, query is disabled, page shows prompt. |
| Root location matches a search term | Root location ("Home") is included in location search results. Clicking navigates to `/locations/:root_id`. This is fine — the user might want to navigate to root. |
| Instance with `parent_instance_id` set (nested in a container) appears in search results | Instance result shows: `"location_name": null, "parent_instance_name": "Toolbox (x2)"`. The user clicks → navigates to `/instances/:id` which has its own breadcrumb. |
| Multiple instances of the same definition in different locations match the search term | Each instance appears as a separate result row. Sorted by match quality then alphabetically by `definition_name`. Instances with the same name are distinguished by their location/container subtitle. |
| Definition has many matching instances (e.g., 200 "M3 Screw" instances distributed across 50 locations) | All matching instances are returned (subject to the 100-per-group cap). Instance results include distinct `id`, `location_name`, and `quantity` so the user can pick the right one. |
| Search results page is opened on mobile with keyboard visible | Page layout accommodates the keyboard. Search bar stays visible at top. Results scroll beneath. |
| Database indexes fail to create (migration error) | Migration fails on startup, container exits with error. Same behavior as any failed migration per database PRD. |

---

## 7. Non-Goals & Scope Boundaries

- **Full-text search (FTS5):** v1 uses `LIKE '%term%'` only. FTS5 is a v2 upgrade with no breaking API changes.
- **Advanced filters:** No filtering by tags, location, quantity range, date range, container status, or field values. Entity type is the only filter.
- **Relevance ranking beyond simple tiering:** v1 uses exact → starts-with → contains sorting. No TF-IDF, no Levenshtein/fuzzy matching, no result scoring.
- **Search within a specific location or container:** Search is global across all entities. No contextual "search inside this location."
- **Search by instance field values:** Instance search matches against `item_definitions.name` only, not field values (e.g., searching for "Steel" to find instances with Material=Steel is not supported).
- **Search history or recent searches:** No client-side or server-side search history.
- **Autocomplete suggestions:** No "did you mean?" or term suggestions from the database. The quick dropdown shows *results*, not query suggestions.
- **Highlighted matching text in results:** Result names are not highlighted with bold or colored matched substrings in v1.
- **Keyboard shortcut (Cmd+K / Ctrl+K):** No global keyboard shortcut to focus the search bar.
- **Search by tag name:** Tag names are not searchable. Search is over locations, definitions, and instances by name only.

---

## 8. Open Questions & Deferred Items

| # | Question | Status |
|---|---|---|
| OQ-1 | Should search results include a "breadcrumb preview" for instances (e.g., "Home > Workshop > M3 Screw") to show context without clicking through? | Deferred — v1 shows location/container name in the subtitle line. Full breadcrumb would require additional queries per result. |
| OQ-2 | Should the search bar support a global keyboard shortcut (Ctrl+K / Cmd+K)? | Deferred — keyboard shortcut can be added in a later polish pass if users request it. |
| OQ-3 | Should the search results page support URL-based sharing of the entity type filter (`?type=locations`)? | Resolved — yes, this is part of the core design. Tabs update the URL with `?type=`. |
| OQ-4 | Should instance results show the definition's field values in the card? | Deferred — v1 shows definition name, quantity, and location/container only. Field values would clutter the card. |
| OQ-5 | Should search be integrated with the location tree and definition list (e.g., filtered list narrowing as you type)? | Deferred — v1 search is a dedicated page + dropdown. In-page filtering (search-as-you-type narrowing the list) is a different UX pattern deferred to v2. |
| OQ-6 | Should there be a shortcut link from empty search results to create a new entity? | Deferred — "No results" has a "Browse Locations" link as fallback. "Create" from search is out of scope. |
