# PRD: Dashboard — InventoryManagement

> **Status:** Done v1.0 (updated by PRD #12 — search bar now functional)
> **Scope:** Landing page (route `/`) — top-level stat cards, location breakdown ranked by item count, functional search bar (wired up per PRD #12), and guided 3-step onboarding for new users.

---

## 0. Cross-PRD Consistency Report

### PRDs Reviewed
- `prd-overarching-architecture.md` — Views list (§6.1: "Dashboard — Total item counts, recent activity, quick search bar"), non-goals (§11: no dashboards "beyond basic totals"), API conventions (§5), data model (§4), mobile-first constraints (§6.2).
- `prd-database-schema.md` — All tables (`locations`, `item_definitions`, `item_instances`, `tags`, `settings`), FK constraints, auto-seed root location.
- `prd-backend-architecture.md` — Go layering, chi router, error mapping, `/api/v1/health` endpoint.
- `prd-frontend-architecture.md` — React Router v6, TanStack Query, CSS Modules, Radix UI, mobile/desktop layout split, bottom nav vs sidebar.
- `prd-visual-design.md` — Golden Amber design tokens, component patterns (cards, buttons, empty states, loading states, error states), responsive breakpoints, mobile app shell / desktop sidebar, unified list page layout.
- `prd-locations.md` — Location CRUD, tree, contents, breadcrumb, root location special handling.
- `prd-item-definitions.md` — Definition CRUD, field schema, inheritance, instance summary, tag assignment.
- `prd-item-instances.md` — Instance CRUD, move/split, container nesting, breadcrumb, list filters, auto-merge.
- `prd-tags.md` — Tag CRUD, cascade delete with confirmation.
- `prd-testing.md` — Integration test patterns, seed data, AI agent investigation workflow.
- `prd-docker-deployment.md` — Docker compose, health check, volume mounts (no direct dashboard impact).

### Conflicts & Resolutions

| # | Conflict | PRDs Involved | Resolution |
|---|---|---|---|
| 1 | Overarching PRD §11 lists "No reporting, analytics, or dashboards beyond basic totals" as a non-goal. This dashboard PRD covers basic totals + location breakdown — **consistent** with the stated boundary. | `prd-overarching-architecture.md` | Confirmed alignment. This PRD respects the "basic totals only" constraint: no charts, no time-series, no aggregation by date ranges, no export. Stat cards + ranked location list only. |
| 2 | No dashboard route exists in the frontend architecture or visual design PRDs. The visual design PRD shows 4 bottom nav tabs: Locations, Definitions, Tags, Settings. No dashboard tab. | `prd-frontend-architecture.md`, `prd-visual-design.md` | **Dashboard is the home page at route `/`, not a nav tab.** It is reached via the app name in the header (mobile) or logo in sidebar (desktop). Clicking the app name navigates to `/` from anywhere. Bottom nav stays at 4 items. No new tab added. |
| 3 | No dedicated `/api/v1/dashboard` endpoint group exists in overarching §5.2. | `prd-overarching-architecture.md` | **Add `GET /api/v1/dashboard`** to the endpoint groups. Read-only, no mutations. Returns aggregated stats + location breakdown. |
| 4 | Quick search bar is mentioned in dashboard scope but the search feature is separate. | `prd-search.md` | **Resolved by PRD #12.** The dashboard search bar is now a functional search bar — pressing Enter navigates to `/search?q=...` and typing triggers the quick-results dropdown. See `prd-search.md` for full search implementation. The `SearchBar` component is shared, imported from `frontend/src/components/SearchBar.tsx`. |
| 5 | "Recent activity" was listed in the overarching PRD dashboard description but the database has no activity/audit log table. | `prd-database-schema.md` | **Recent activity is deferred to v2.** No `updated_at`-sorted feed. The dashboard focuses on stat cards + location breakdown only. |

### Confirmed Alignments
- Data model: All queries are read-only against existing tables (`locations`, `item_definitions`, `item_instances`, `definition_tags`, `tags`). No schema changes.
- API patterns: `GET /api/v1/dashboard` follows the `/api/v1/` prefix, returns JSON, uses the standard error format `{"error":"...","code":"..."}`.
- Error mapping: Service returns domain errors → handler maps to HTTP status codes per `prd-backend-architecture.md` TR-2.
- UI: CSS Modules + CSS variables (Golden Amber tokens), Radix UI primitives, TanStack Query, React Router v6. Mobile-first: 375px–1920px responsive range, bottom nav on mobile, sidebar on desktop.
- Visual design: Stat cards use the card component pattern (§6.3), location list rows follow list item patterns, empty state onboarding uses the empty state pattern (§6.7), loading uses skeleton cards (§6.8). All colors/spacing use CSS variables defined in §3.
- Scope does not contradict any non-goal in overarching PRD: no charts, no analytics, no time-series, no export.

---

## 1. Overview & Problem Statement

The dashboard is the first screen a user sees when opening InventoryManagement. Its job is to provide an immediate, at-a-glance understanding of the inventory's state and direct the user to the most relevant next action. For new users with no data, it guides them through the initial setup. For active users, it answers "what do I have and where is it?" in seconds.

### Core Deliverables
1. **Stat cards:** 4 top-level aggregated counts — total locations, definitions, instances, and aggregate quantity.
2. **Location breakdown:** All locations ranked by recursive instance count, with expandable sub-location counts.
3. **Search bar:** Prominent, styled input at the top of the dashboard. Typing triggers a quick-results dropdown; submitting navigates to the full search results page per `prd-search.md`.
4. **3-step onboarding guide:** Shown when the user has not yet completed the core setup loop (no locations beyond root, no definitions, no instances). Clickable CTAs link to the relevant pages.
5. **Single API endpoint:** `GET /api/v1/dashboard` returns all dashboard data in one request.

---

## 2. Goals & Measurable Success Metrics

| Goal | Metric |
|---|---|
| Fast page load | Dashboard API responds in < 100ms p95 with realistic seed data |
| Single request | Entire dashboard data returned in one `GET /api/v1/dashboard` call |
| Immediate insight | User can see total item count and top 3 locations without scrolling (mobile) |
| Clear onboarding | New users presented with the 3-step guide; each step links to the correct page |
| Search bar | Visible search bar at top of dashboard on both mobile and desktop, functional per `prd-search.md` |
| Responsive | Layout correct on 375px, 768px, 1280px, 1920px |
| Zero schema impact | No new tables or columns. All queries are read-only against existing schema. |

---

## 3. Critical Risks & Technical Assumptions

| Risk | Mitigation |
|---|---|
| Slow dashboard query with many instances | Recursive instance count per location uses a single recursive CTE per request. Benchmark: < 100ms for 5000 instances across 200 locations. |
| Onboarding card flickers on page load (shows temporarily even when data exists) | API response includes `is_onboarding` boolean. Card only renders when `is_onboarding: true`. No client-side flicker. |
| Search bar in header on other pages | PRD #12 implements a persistent header search bar on all pages (mobile: collapsible icon; desktop: inline input). The dashboard's centered search bar is the most prominent entry point, but search is accessible everywhere. |
| Large location hierarchy produces cluttered ranked list | If > 20 locations exist, show only top-level locations by default with a "Show all X sub-locations" expand toggle. Sub-location counts are included in the parent's total. |
| Dashboard page navigation from anywhere | App name in header (mobile) and logo area in sidebar (desktop) navigate to `/`. React Router handles this. No new nav tab needed. |

---

## 4. User Stories & Acceptance Criteria

### US-001: Dashboard Landing Page
**Description:** As a user, I want to see my inventory at a glance when I open the app, so I immediately know what I have and where it is.

**Acceptance Criteria:**
- [ ] Route `/` renders the dashboard page. It is the default/index route.
- [ ] The app name "Inventory" in the mobile header navigates to `/` when tapped.
- [ ] The logo/name area in the desktop sidebar navigates to `/` when clicked.
- [ ] The dashboard is NOT a bottom nav tab or sidebar nav item. The 4 existing nav items (Locations, Definitions, Tags, Settings) remain unchanged.
- [ ] Typecheck / build / test suite passes.

### US-002: Dashboard API Endpoint
**Description:** As a frontend, I want a single endpoint that returns all the data needed to render the dashboard.

**Acceptance Criteria:**
- [ ] `GET /api/v1/dashboard` returns JSON with the structure defined in FR-2.
- [ ] Response includes: `stats` (4 numbers), `locations` (ranked array with recursive instance counts), `is_onboarding` (boolean).
- [ ] `stats.locations_count` — total count of ALL locations (including root).
- [ ] `stats.definitions_count` — total count of all item definitions.
- [ ] `stats.instances_count` — total count of all item instance rows.
- [ ] `stats.total_quantity` — SUM of all `item_instances.quantity`.
- [ ] `locations` array sorted by `instance_count DESC`. Each entry includes: `id`, `name`, `instance_count` (recursive), `direct_instance_count`, `sub_location_count`, `children` (array of direct child locations with their own recursive counts, ordered by `instance_count DESC`, max 3 per parent).
- [ ] `is_onboarding` is `true` when all three conditions are met: (1) only the root location exists (no sub-locations), (2) zero item definitions, (3) zero item instances. Otherwise `false`.
- [ ] Non-existent DB or empty tables return `stats` with all zeros, `locations: []`, `is_onboarding: true`. No error.
- [ ] Endpoint responds in < 150ms with 500+ instances across 50+ locations.
- [ ] Typecheck / build / test suite passes.

### US-003: Stat Cards
**Description:** As a user, I want to see key inventory counts at the top of the dashboard so I can quickly grasp the state of my inventory.

**Acceptance Criteria:**
- [ ] Four stat cards displayed in a row (mobile: 2x2 grid; desktop: single row).
- [ ] Cards show: Location count, Definition count, Instance count, Total items (aggregate quantity).
- [ ] Each card has a large number (`--text-h1`), a label below it (`--text-small`, `--color-text-secondary`), and a subtle icon.
- [ ] Cards are tappable: tap "Locations" → navigates to `/locations`, tap "Definitions" → `/definitions`, tap "Instances" or "Total items" → both navigate to `/locations` (the primary browse path).
- [ ] Cards use the visual design card pattern: bg `--color-bg-surface`, border `--color-border`, `--radius-md`, padding `--space-lg`, shadow `--shadow-card`.
- [ ] Cards are non-interactive when the count is 0 (no navigation, muted styling).
- [ ] Loading state: skeleton cards (same dimensions, animated shimmer per visual PRD §6.8).
- [ ] Error state: card area shows error state component with retry button per visual PRD §6.9.
- [ ] **[UI]** Verified in browser on both 375px and 1920px viewports.

### US-004: Location Breakdown (Ranked List)
**Description:** As a user, I want to see where my items are concentrated across locations, ranked by item count, so I know which areas hold the most inventory.

**Acceptance Criteria:**
- [ ] Section heading: "Locations" (`--text-h2`) with total location count in parentheses.
- [ ] Below it: a ranked list of top-level locations (parent_id = NULL), sorted by recursive `instance_count DESC`.
- [ ] Each top-level location row shows:
  - Location name (clickable → `/locations/:id`).
  - Recursive instance count as a number badge (pill, bg `--color-accent-muted`, text `--color-accent`, `--text-caption`).
  - A compact horizontal bar visualizing the proportion relative to the max count (bar height: 4px, bg `--color-bg-surface-alt`, fill `--color-accent`, `--radius-full`). Optional — can be deferred if it complicates the layout.
- [ ] **Expand toggle:** If a location has sub-locations (sub_location_count > 0), a small `[+]` / `[-]` control reveals up to 3 of its direct children, each with their own name + recursive instance count, indented. Children are also clickable.
- [ ] If no locations exist beyond root: show an empty state message "No locations added yet" with a link to `/locations`.
- [ ] Loading state: 4-5 skeleton rows per visual PRD §6.8.
- [ ] Error state: inline error with retry button per visual PRD §6.9.
- [ ] **[UI]** Verified in browser.

### US-005: Search Bar
**Description:** As a user, I want a functional search bar on the dashboard so I can quickly find locations, definitions, and instances by name.

**Acceptance Criteria:**
- [ ] A search input field is rendered at the top of the dashboard page, above the stat cards.
- [ ] Styled as a form input (bg `--color-bg-surface`, border `--color-border`, `--radius-md`, height 44px, padding `--space-md`, full width on mobile, max-width 480px centered on desktop).
- [ ] Includes a magnifying glass icon (Radix Icons, 18x18px, `--color-text-secondary`) positioned inside the input on the left.
- [ ] Placeholder text: "Search inventory..." (`--text-body`, `--color-text-secondary`).
- [ ] The input accepts focus and text entry. Typing ≥ 2 characters triggers a quick-results dropdown showing top 3 matches per entity type (locations, definitions, instances) with a "Show all results..." link. Pressing Enter or clicking the magnifying glass navigates to `/search?q=<term>` — the full search results page.
- [ ] The quick-results dropdown and search wiring are implemented per `prd-search.md` US-003 and US-004. The `SearchBar` component is shared at `frontend/src/components/SearchBar.tsx`.
- [ ] Desktop: the search bar is centered below the page header, max-width 480px, with generous vertical padding (`--space-3xl`) to make it a visual anchor.
- [ ] Mobile: full-width, directly below the header, padded horizontally by `--space-lg`.
- [ ] The search bar is always visible, regardless of whether onboarding is shown.
- [ ] **[UI]** Verified in browser on both 375px and 1920px viewports.

### US-006: 3-Step Onboarding Guide
**Description:** As a new user, I want guided steps that tell me what to do first, so I don't face an empty dashboard with no direction.

**Acceptance Criteria:**
- [ ] Shown when `is_onboarding === true` (API response) — no sub-locations beyond root, no definitions, no instances.
- [ ] The onboarding card appears **between the search bar and the stat cards** (stats still show zeros, location list is empty).
- [ ] Three steps, each with a number, short title, one-line explanation, and a CTA link:

  ```
  +-----------------------------------+
  |       Get Started                 |
  |                                   |
  |  [1] Add locations                |
  |      Rooms, shelves, boxes —      |
  |      set up your storage spaces   |
  |      [+ Add first location]       |
  |                                   |
  |  [2] Define your items            |
  |      WHAT you track — like        |
  |      "M3 Screw" or "Toolbox"      |
  |      [+ Create first definition]  |
  |                                   |
  |  [3] Stock your items             |
  |      Put quantities into          |
  |      locations — the actual       |
  |      physical inventory           |
  |      [+ Add first item]           |
  +-----------------------------------+
  ```

- [ ] Step 1 CTA navigates to `/locations` (user creates sub-locations from there).
- [ ] Step 2 CTA navigates to `/definitions` (user creates definitions).
- [ ] Step 3 CTA navigates to `/locations` (user adds instances from a location detail page).
- [ ] The card uses a subtle warm highlight: bg `--color-bg-surface`, a thin left border `--color-accent` (3px), `--radius-md`, padding `--space-2xl`.
- [ ] Once **any** of the three conditions is satisfied (user adds a sub-location, definition, or instance), `is_onboarding` becomes `false` and the card disappears permanently. The regular dashboard content (stat cards + location breakdown) shows instead. There is no progress-tracking — it's all-or-nothing.
- [ ] The card is not dismissible by the user — it only disappears when real data exists.
- [ ] **[UI]** Verified in browser on both 375px and 1920px viewports.

---

## 5. Functional & Technical Requirements

### 5.1 Database Dependencies

**FR-1:** No schema changes. All dashboard queries are read-only against the existing tables: `locations`, `item_definitions`, `item_instances`. The root location auto-seeded by `prd-database-schema.md` US-003 provides the baseline for the `is_onboarding` check.

### 5.2 REST API Endpoint

| Method | Path | Description | Request Body | Response |
|---|---|---|---|---|
| `GET` | `/dashboard` | Dashboard data | — | `DashboardResponse` |

**FR-2:** `DashboardResponse` shape:

```json
{
  "stats": {
    "locations_count": 12,
    "definitions_count": 32,
    "instances_count": 87,
    "total_quantity": 450
  },
  "locations": [
    {
      "id": "uuid",
      "name": "Workshop",
      "instance_count": 85,
      "direct_instance_count": 40,
      "sub_location_count": 3,
      "children": [
        {
          "id": "uuid",
          "name": "Tool Cabinet",
          "instance_count": 25,
          "direct_instance_count": 20,
          "sub_location_count": 1,
          "children": [
            {
              "id": "uuid",
              "name": "Drawer 1",
              "instance_count": 5,
              "direct_instance_count": 5,
              "sub_location_count": 0,
              "children": []
            }
          ]
        }
      ]
    }
  ],
  "is_onboarding": false
}
```

- `instance_count`: recursive — includes instances directly at this location PLUS all instances in all descendant sub-locations recursively.
- `direct_instance_count`: instances where `item_instances.location_id = this location's id` (not inclusive of sub-locations).
- `sub_location_count`: number of direct child locations.
- `children`: up to 3 direct child locations (the top 3 by `instance_count DESC`), each with the same recursive structure. If a location has > 3 children, only the top 3 are included; the `sub_location_count` field reflects the true total. Users expand in the UI to see more (if needed in v2).
- `locations` array: top-level locations only (parent_id IS NULL), sorted by `instance_count DESC`.
- `locations` array excludes the root location from the dashboard-ranked list. The root is not shown in this list since it aggregates everything and skews the ranking.
- `is_onboarding`: `true` when number of sub-locations (locations where parent_id IS NOT NULL) = 0 AND definitions_count = 0 AND instances_count = 0.

**FR-3:** Stats queries:

```sql
-- locations_count: total count including root
SELECT COUNT(*) FROM locations;

-- definitions_count
SELECT COUNT(*) FROM item_definitions;

-- instances_count
SELECT COUNT(*) FROM item_instances;

-- total_quantity
SELECT COALESCE(SUM(quantity), 0) FROM item_instances;
```

**FR-4:** Location recursive instance count — use a recursive CTE that aggregates instance quantities down the location tree:

```sql
WITH RECURSIVE location_tree AS (
    SELECT id, name, parent_id, 0 AS depth
    FROM locations
    WHERE parent_id IS NULL  -- top-level locations only (excludes root by filtering on the result, not here)

    UNION ALL

    SELECT l.id, l.name, l.parent_id, lt.depth + 1
    FROM locations l
    JOIN location_tree lt ON l.parent_id = lt.id
),
location_instance_counts AS (
    SELECT
        lt.id,
        lt.name,
        lt.parent_id,
        lt.depth,
        COALESCE(SUM(ii.quantity), 0) AS direct_quantity,
        COUNT(ii.id) AS direct_instance_count
    FROM location_tree lt
    LEFT JOIN item_instances ii ON ii.location_id = lt.id
    GROUP BY lt.id
),
location_children AS (
    SELECT parent_id, COUNT(*) AS sub_location_count
    FROM locations
    WHERE parent_id IS NOT NULL
    GROUP BY parent_id
)
-- Combine and compute recursive sums in Go (TEXT sorting by parent chains)
```

**Simpler approach (recommended):** Since this is a read-only endpoint on a single-user app with < 500 locations, fetch all locations + their instance counts in flat queries, then build the tree and compute recursive sums in Go. This is simpler, more testable, and performant at this scale.

```go
// 1. SELECT id, name, parent_id FROM locations ORDER BY name
// 2. SELECT location_id, SUM(quantity) AS qty, COUNT(*) AS cnt
//    FROM item_instances WHERE location_id IS NOT NULL GROUP BY location_id
// 3. Build tree in Go, compute recursive sums post-order
```

**FR-5:** The `/api/v1/dashboard` endpoint is registered in `internal/router/` via a `DashboardHandler`:

```go
r.Get("/", dashboardHandler.Get)
```

under `r.Route("/api/v1/dashboard", ...)`.

### 5.3 Service Layer

**FR-6:** `DashboardService` (in `internal/service/dashboard.go`) implements:

```go
type DashboardService struct {
    db *sql.DB
}

type DashboardData struct {
    Stats        DashboardStats  `json:"stats"`
    Locations    []LocationNode  `json:"locations"`
    IsOnboarding bool            `json:"is_onboarding"`
}

type DashboardStats struct {
    LocationsCount  int `json:"locations_count"`
    DefinitionsCount int `json:"definitions_count"`
    InstancesCount  int `json:"instances_count"`
    TotalQuantity   int `json:"total_quantity"`
}

type LocationNode struct {
    ID                 string          `json:"id"`
    Name               string          `json:"name"`
    InstanceCount      int             `json:"instance_count"`
    DirectInstanceCount int            `json:"direct_instance_count"`
    SubLocationCount   int             `json:"sub_location_count"`
    Children           []LocationNode  `json:"children"`
}

func (s *DashboardService) GetDashboard(ctx context.Context) (*DashboardData, error) {
    // 1. Fetch stats (4 COUNT queries or one combined query)
    // 2. Fetch all locations (id, name, parent_id)
    // 3. Fetch instance counts grouped by location_id
    // 4. Build location tree, compute recursive sums
    // 5. Filter: exclude root location from the ranked list
    // 6. Determine is_onboarding
}
```

**FR-7:** `is_onboarding` detection:
- Fetch `SELECT COUNT(*) FROM locations WHERE parent_id IS NOT NULL` (sub-locations).
- Fetch `SELECT COUNT(*) FROM item_definitions`.
- Fetch `SELECT COUNT(*) FROM item_instances`.
- If all three = 0 → `is_onboarding: true`.
- If any > 0 → `is_onboarding: false`.

**FR-8:** Root location exclusion: the root location (`id = settings.root_location_id`) is excluded from the `locations` array. It aggregates everything and would dominate the ranking, providing no useful signal.

**FR-9:** Location ranking: top-level locations (`parent_id IS NULL` and NOT the root) sorted by `instance_count DESC`. Children within each location are ranked by `instance_count DESC`, capped at 3.

### 5.4 Handler Layer

**FR-10:** `DashboardHandler` in `internal/handler/dashboard.go`:
- `GET /api/v1/dashboard` → calls `DashboardService.GetDashboard(ctx)`.
- Returns `200 OK` with `DashboardData` JSON.
- If service returns an error, handler maps it via the standard error mapping pattern (`RespondWithError`).

### 5.5 Frontend

**FR-11:** Route configuration:
- `/` → `DashboardPage` component (home/landing page).
- The mobile header app name "Inventory" links to `/` via React Router `<Link to="/">`.
- The desktop sidebar logo area links to `/`.

**FR-12:** TanStack Query key:
- `['dashboard']` — single key for the entire dashboard data.

**FR-13:** `DashboardPage` component structure:
```
<DashboardPage>
  <SearchBar />           ← functional search bar (wired per prd-search.md)
  <OnboardingGuide />     ← conditional (is_onboarding === true)
  <StatCards />           ← always shown (zero counts OK)
  <LocationBreakdown />   ← conditional (is_onboarding === false)
</DashboardPage>
```

**FR-14:** `StatCards` component:
- 4 cards in a responsive grid:
  - Mobile: 2 columns × 2 rows, full width.
  - Desktop: 4 columns in a single row, centered, max-width 800px.
- Each card: large number, label, subtle icon.
- Cards are `<Link>` components wrapping the card content when the count > 0, plain `<div>` when count is 0.
- Navigation targets:
  - Locations → `/locations`
  - Definitions → `/definitions`
  - Instances → `/locations` (items are browsed through locations)
  - Total items → `/locations` (same rationale)

**FR-15:** `LocationBreakdown` component:
- Section header: "Locations (X)" with X = locations_count from stats.
- Ranked list of `LocationNode` items from the API response.
- Each top-level row: name (link to `/locations/:id`), recursive instance count badge.
- Expandable children: `[+]` toggle reveals up to 3 child rows (already in the API response). Children are indented by `--space-xl`.
- Empty state: "No locations added yet" → links to `/locations`.
- If a location has `instance_count === 0`, it's still shown (with 0) since the user may have set up the structure but not stocked it yet.

**FR-16:** `OnboardingGuide` component:
- Renders the 3-step card only when `is_onboarding === true`.
- Not dismissible. Disappears automatically when any data is created (next dashboard fetch after a mutation elsewhere invalidates the query).
- Steps are not clickable individually — only the CTA buttons link to the respective pages.
- CTA buttons: Primary variant (`--color-accent` bg), small (`--text-small`, height 32px), inline within each step row.

**FR-17:** `SearchBar` shared component:
- Renders a styled `<input type="search">` with a magnifying glass icon.
- On change (≥ 2 chars, 300ms debounce): fires TanStack Query `['search', { q, limit: 3 }]` and renders the quick-results dropdown below the input.
- On Enter or magnifying glass click: navigates to `/search?q=<term>` via React Router.
- The component is shared at `frontend/src/components/SearchBar.tsx` and also used in the persistent header search bar (mobile + desktop) per `prd-search.md` US-001/US-002.
- The dashboard uses the `SearchBar` component — no separate dashboard-specific search bar implementation.

**FR-18:** Query invalidation: the `['dashboard']` query is invalidated on any successful mutation across the app (create/update/delete location, definition, instance, tag). Use TanStack Query's `queryClient.invalidateQueries({ queryKey: ['dashboard'] })` in mutation `onSuccess` callbacks. Since the dashboard is a landing page and not frequently visited during active use, `staleTime: 30_000` (30s) is appropriate — the dashboard won't refetch unnecessarily during navigation but will be fresh when the user returns to it.

**FR-19:** Visual design alignment:
- All colors, spacing, border-radius, shadows use CSS variables from `prd-visual-design.md` §3. No raw hex or pixel values.
- Cards: `--color-bg-surface`, `--radius-md`, `--shadow-card`, padding `--space-lg`.
- Location list rows: bg `--color-bg-surface`, padding `--space-md`, `--radius-sm`, border-bottom `--color-border`.
- Onboarding card: `--color-bg-surface`, left border 3px `--color-accent`, `--radius-md`, padding `--space-2xl`.
- Search bar: `--color-bg-surface`, border `--color-border`, `--radius-md`, height 44px.
- Fonts: Nunito 600/700 for headings (`--text-h1`, `--text-h2`), DM Sans 400/500 for body/labels (`--text-body`, `--text-body-strong`, `--text-small`, `--text-caption`).

---

## 6. Edge Cases & Failure Modes

| Scenario | System Behavior |
|---|---|
| DB is empty (fresh install, no auto-seed ran) | `GET /api/v1/dashboard` returns all stats = 0, `locations: []`, `is_onboarding: true`. Dashboard shows onboarding card with stat cards showing zeros. No error. |
| Auto-seed ran but user hasn't added anything (only root "Home" location exists) | `stats.locations_count` = 1 (the root). `is_onboarding: true` because no sub-locations, definitions, or instances exist. Root location is excluded from the `locations` array. Onboarding card shown. |
| Many locations (200+) but few instances | Location ranked list shows all locations. Scrolling is acceptable. No pagination. Instances-only: only locations with instances > 0 get a bar/visual weight; others show 0. |
| Many instances (5000+) concentrated in a few locations | Recursive count computed in Go — O(n) over all locations + instances. Benchmarked to < 100ms. Location list correctly ranks the few populated locations at top. |
| User visits dashboard, then adds a location on another page, then navigates back | TanStack Query's `staleTime: 30_000` means the dashboard refetches if the data is stale. If the user navigates within 30s, cached data shows (slightly stale counts). On the next navigation after 30s, the fresh data is fetched. |
| `is_onboarding` transitions from true to false | The next dashboard fetch after the mutation sees `is_onboarding: false`. The onboarding card disappears, stat cards and location breakdown render normally. No animation needed for the transition. |
| API request fails (DB connection lost, query error) | Handler returns `500 Internal Server Error` with standard error JSON. Frontend shows the error state component (visual PRD §6.9) with retry button. Search bar remains visible. |
| User bookmarks `/` and opens it directly | React Router renders the dashboard. TanStack Query fetches `['dashboard']`. If network is slow, loading skeletons show (per visual PRD §6.8). |
| User clicks app name while already on `/` | React Router detects same-route navigation. No refetch. No visual change. Standard SPA behavior. |

---

## 7. Non-Goals & Scope Boundaries

- **Charts / graphs / time-series:** No bar charts, line graphs, or trend visualizations. Stats are displayed as numbers only.
- **"Recent activity" feed:** Deferred to v2. Requires an audit log table (`instance_audit_log` or similar). No activity rows on the dashboard.
- **Date-range filters:** No "last 7 days" or "this month" aggregation. All stats are current totals.
- **Search logic, search results page, and search wiring:** Handled by `prd-search.md`. The dashboard renders the shared `SearchBar` component and does not implement search logic itself.
- **Dashboard customization:** No user-configurable widget layout, no show/hide toggles, no card reordering.
- **Export / print:** No CSV/PDF export of dashboard data.
- **Notification badges:** No "3 new items" or "5 untagged definitions" badges on stat cards.
- **Instance count by tag:** No tag-based aggregation on the dashboard.
- **Definition-level breakdown:** The ranked list is location-based only. Definitions have their own detail page with instance summary per `prd-item-definitions.md`.
- **Recursive instance count for nested item-in-item instances:** Instances inside containers (parent_instance_id IS NOT NULL) are counted in `total_quantity` and `instances_count` but NOT attributed to any location in the location breakdown. They appear under their parent container in the instance detail page, not the dashboard.

---

## 8. Open Questions & Deferred Items

| # | Question | Status |
|---|---|---|
| OQ-1 | Should the stat cards include a "Tags" count? | Deferred — tags are metadata for definitions, not a primary inventory dimension. Stat cards stay at 4. |
| OQ-2 | Should the location breakdown show a horizontal bar proportional to max count? | Deferred to implementation — simple ranked list is the baseline. Bar can be added if it improves scanability without layout complexity. |
| OQ-3 | Should expanding a location's children in the dashboard fetch more data or use the initial response? | The initial API response includes up to 3 children per location. Expanding shows these pre-fetched children. If a location has > 3 children and the user wants to see all, they navigate to the location detail page. No lazy loading on the dashboard. |
| OQ-4 | Should the onboarding card be dismissible? | Resolved — no. It disappears only when real data exists. Dismissing would leave the user on an empty dashboard with no guidance. |
| OQ-5 | Should the dashboard auto-refresh (polling)? | Resolved — no. TanStack Query's `staleTime: 30_000` handles refetch on navigation. No WebSocket or polling. |
| OQ-6 | Should there be a "quick add" FAB on the dashboard? | Deferred — the `[+]` button in the header changes context per page. A dashboard-level FAB would need to decide what to create (location, definition, or instance) — that's ambiguous. The onboarding card provides CTAs for the empty state. |
