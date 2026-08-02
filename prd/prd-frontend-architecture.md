# PRD: Frontend Architecture — InventoryManagement

> **Status:** Draft v1.0
> **Scope:** React/Vite/TS scaffold rules, TanStack Query integration, routing strategy, CSS design system, navigation, and UI component architecture.
> **Backlog Position:** #4

---

## 0. Cross-PRD Consistency Report

### PRDs Reviewed
- `prd-overarching-architecture.md` — Top-level tech stack, mobile-first design, frontend framework (React/Vite).
- `prd-project-setup.md` — Repository structure (`frontend/` directory), proxy setup, Vite configuration.
- `prd-database-schema.md` — Data entities (Locations, Items, Tags) which influence the frontend routing and caching.
- `prd-backend-architecture.md` — API response structures (e.g., standard error payloads) and SPA fallback logic.

### Conflicts & Resolutions
No conflicts detected. This PRD is consistent with all prior PRDs.

### Confirmed Alignments
- Uses React, Vite, and TypeScript as defined in overarching PRD.
- Data fetching relies on TanStack Query (React Query) over Redux.
- Follows the mobile-first "bottom nav vs sidebar" directive from the overarching architecture.
- Adheres to the "Vanilla CSS + CSS variables" constraint by utilizing CSS Modules (`.module.css`) + global CSS variables.

---

## 1. Overview & Problem Statement

This PRD establishes the core frontend architecture for the InventoryManagement application. The goal is to build a highly responsive, modern, and premium web application that feels like a native app on mobile devices while leveraging screen space effectively on desktop. It must be extremely lightweight to run efficiently on low-resource devices and NAS systems without compromising on UX.

### Core Deliverables
1. **Routing Strategy:** Configuration of React Router v6 with distinct layouts for Mobile and Desktop.
2. **State Management:** Implementation patterns for TanStack Query for server state and minimal React Context for UI state.
3. **Design System & Styling:** Setup of CSS Modules combined with global CSS variables for a premium, custom aesthetic.
4. **UI Toolkit Integration:** Integration of Radix UI primitives for accessible, complex UI components without heavy styling overhead.
5. **Error & Offline Handling:** Standardized patterns for surfacing API errors and handling network disconnection gracefully.

---

## 2. Goals & Measurable Success Metrics

| Goal | Metric |
|---|---|
| Responsive Design | Flawless usability on 375px wide viewports (mobile) up to 4K displays. |
| Performance | Lighthouse performance score > 90. Initial JS bundle < 300KB gzipped. |
| Maintainability | Zero global CSS class conflicts (via CSS Modules). |
| UX Quality | Seamless transitions, responsive micro-animations, and immediate visual feedback on all interactions. |
| Accessibility | 100% keyboard navigability for interactive components via Radix UI primitives. |

---

## 3. Critical Risks & Technical Assumptions

| Risk | Mitigation |
|---|---|
| CSS Bundle Bloat | Avoid utility-class frameworks or heavy component libraries. Use targeted CSS Modules and Radix primitives. |
| Complex Layout Rendering | Enforce separate layout components at the React Router level (`<MobileLayout>` vs `<DesktopLayout>`) rather than complex CSS media-query gymnastics within a single shell. |
| Stale Data on Move Operations | Precisely invalidate targeted TanStack Query keys (source and destination) upon move success, rather than global invalidations, to prevent unnecessary re-fetches. |
| Native App Feel | Implement touch-friendly tap targets (minimum 44x44px) and prevent text selection on UI controls. |

---

## 4. User Stories & Acceptance Criteria

### US-001: Adaptive Routing Layouts
**Description:** As a user, I want the application to present a layout optimized for my device (bottom navigation for mobile, sidebar for desktop) so that it is easy to navigate.

**Acceptance Criteria:**
- [ ] React Router utilizes a layout wrapper that dynamically renders `<MobileLayout>` or `<DesktopLayout>` based on viewport width breakpoints.
- [ ] Mobile layout includes a fixed bottom navigation bar.
- [ ] Desktop layout includes a fixed left sidebar.
- [ ] Both layouts render the primary application content cleanly in the remaining viewport space.

### US-002: Premium Accessible Components
**Description:** As a user, I want to interact with dropdowns, modals, and tooltips that work perfectly with my keyboard/screen reader and look stunning.

**Acceptance Criteria:**
- [ ] Complex interactive components (e.g., Modals, Dropdowns) are built using Radix UI primitives.
- [ ] Components are styled using CSS Modules and global CSS variables, avoiding default generic styles.
- [ ] Modals correctly trap focus when open.
- [ ] Esc key dismisses modals and popovers.

### US-003: Graceful API Error Handling
**Description:** As a user, I want clear feedback when an action fails or data cannot be loaded, so I understand what went wrong.

**Acceptance Criteria:**
- [ ] Form submission errors are displayed inline next to the relevant field.
- [ ] Global errors (e.g., 500 server errors, unexpected conflicts) trigger a global toast notification.
- [ ] Failed data fetches show a localized error state within the component area, rather than breaking the entire page layout.

### US-004: Offline Resilience (Read-Only Mode)
**Description:** As a user, if my device loses connection to the NAS backend, I want to still see the data I have loaded rather than a broken page.

**Acceptance Criteria:**
- [ ] Application detects network loss or API unreachability.
- [ ] A global "Offline" banner appears.
- [ ] The app allows read-only browsing of data already cached by TanStack Query.
- [ ] Mutation actions (buttons to add, edit, delete) are visually disabled while offline.

---

## 5. Functional & Technical Requirements

### TR-1: Styling & Design System
- **CSS Strategy:** Use standard CSS Modules (`[name].module.css`).
- **Design Tokens:** The canonical design tokens (colors, typography, spacing, shadows, border-radius, transitions) are defined in `prd-visual-design.md` §3. All component CSS must use semantic CSS variables — never raw hex/spacing values. The palette is Golden Amber (warm dark mode), font stack is Nunito (headings) + DM Sans (body).
- **Aesthetics:** The design uses a warm/material card aesthetic — dark, rounded, card-based. Soft warm browns and creams with golden amber accents. Subtle warm-toned shadows for depth. Micro-animations for hover/active states.

### TR-2: State Management (TanStack Query)
- Use React Query for all API interactions.
- **Query Key Strategy:** Keys must be structured hierarchically for easy invalidation (e.g., `['locations', 'list']`, `['instances', { locationId: '123' }]`).
- **Targeted Invalidation:** When an item is moved, only the `['instances', { locationId: sourceId }]` and `['instances', { locationId: destinationId }]` queries should be invalidated.

### TR-3: Component Library
- Use **Radix UI Primitives** (`@radix-ui/react-dialog`, `@radix-ui/react-dropdown-menu`, etc.) for complex components.
- Style them via CSS Modules to maintain a lightweight, custom look.
- Build basic primitives (Buttons, Inputs, Cards) from scratch using native HTML elements and CSS Modules.

### TR-4: Layout & Routing
- Define routing in `frontend/src/App.tsx` (or a dedicated router file).
- Use a root wrapper component to listen to `window.innerWidth` (or `window.matchMedia`) to decide whether to render the `DesktopLayout` or `MobileLayout`.
- Both layouts will render the `<Outlet />` for the page content.

### TR-5: API Client
- Centralize API calls using the native `fetch` API.
- Implement an interceptor pattern or a wrapper function to globally handle 401/403 responses or network failures, triggering the offline/error states.

---

## 6. Edge Cases & Failure Modes

| Scenario | System Behavior |
|---|---|
| User navigates to a non-existent route | React Router catches the 404 and displays a "Page Not Found" component. |
| User moves an item, but the API returns a 409 Conflict | The optimistic UI (if any) rolls back. A toast notification displays the backend error message cleanly. |
| NAS server reboots while user is browsing | Global fetch interceptor detects network failure. Displays the global "Offline" banner. Cached data remains viewable. |
| Form submission with missing required fields | Native HTML5 validation acts as the first line of defense. The React component handles secondary validation before sending the request. Inline errors display below fields. |
| Slow network (high latency) | TanStack Query automatically handles loading states (`isLoading`, `isFetching`). Skeleton loaders or spinners display during data retrieval. |

---

## 7. Non-Goals & Scope Boundaries

- **Full Offline PWA:** The app will not feature full Service Worker offline caching or offline mutations (background sync).
- **Complex Animations:** While micro-animations for interactions are required, heavy page transition animations (e.g., Framer Motion) are excluded to keep the bundle small.
- **Complex UI Frameworks:** Tailwind CSS, Bootstrap, Material UI, and Ant Design are explicitly out of scope.
- **Global State Management:** Redux or Zustand are not needed. TanStack Query handles server state, and React Context handles the minimal global UI state (like the offline flag or theme preference).

---

## 8. Open Questions & Deferred Items

| # | Question | Status |
|---|---|---|
| OQ-1 | Should we implement a Dark Mode / Light Mode toggle in v1? | Deferred to Settings PRD, but the CSS variables should be structured to support it trivially. |
| OQ-2 | What specific font family should be standard? | Resolved — Nunito (600/700 for headings) + DM Sans (400/500 for body) per `prd-visual-design.md` §3.2. |
| OQ-3 | Do we need a unified form validation library (e.g., React Hook Form + Zod)? | Deferred. For simple v1 forms, controlled components might suffice. If forms grow complex, React Hook Form is the preferred choice. |
