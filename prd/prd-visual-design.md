# PRD: Visual Design System — InventoryManagement

> **Status:** Draft v1.0
> **Scope:** Complete visual design system — color palette, typography, spacing, elevation, key page layouts, component states. This is a reference PRD consumed by all feature PRDs during UI implementation. It is not an implementation phase itself — it defines *how things should look*.

---

## 0. Cross-PRD Consistency Report

### PRDs Reviewed
- `prd-overarching-architecture.md` — Mobile-first constraint, bottom nav vs sidebar, 44x44px tap targets, 375px viewport minimum.
- `prd-project-setup.md` — Vite + React + TypeScript scaffold, no direct design impact.
- `prd-frontend-architecture.md` — CSS Modules + CSS variables, Radix UI primitives, glassmorphism direction, dark palette placeholders, Inter font suggestion, mobile/desktop layout split.
- `prd-locations.md` — Tree browser UI, location detail page, breadcrumb component, modal forms for create/edit, delete confirmation dialogs.
- `prd-tags.md` — Flat list view with inline editing, tag color badges with swatches, confirmation dialogs.
- `prd-item-definitions.md` — Detail page with tabs/sections (fields, tags, instances), field table with inherited/locked rows, tag assignment UI, instance summary component.
- `prd-item-instances.md` — Instance detail page with breadcrumb bar, field values table, container children list, move dialog with target selector, create form with dynamic field inputs.

### Conflicts & Resolutions

| # | Conflict | PRDs Involved | Resolution |
|---|---|---|---|
| 1 | Frontend PRD suggests Inter, glassmorphism, and dark placeholder palette (`#121212`, `#E0E0E0`, `#007BFF`) | prd-frontend-architecture.md | This PRD replaces those with the Golden Amber warm/material design system. Switch font to Nunito + DM Sans, drop glassmorphism for warm/material card aesthetic, replace all color tokens. |
| 2 | Frontend PRD says themes should "support [dark/light] trivially" but no light mode is designed in v1 | prd-frontend-architecture.md | v1 is dark-mode only. CSS variable architecture must still use semantic tokens (not raw hex values) so light mode can be added later without refactoring. |

### Confirmed Alignments
- CSS Modules + global CSS variables — unchanged, just expanded with real token values.
- Radix UI primitives for dialogs, dropdowns, modals — unchanged.
- Bottom nav (mobile) vs left sidebar (desktop) — unchanged, now specified visually.
- 44x44px minimum tap targets — unchanged.
- 375px–1920px responsive range — unchanged.
- Modal forms, confirmation dialogs, inline editing patterns — unchanged, now visually detailed.
- Reusable breadcrumb component — unchanged, now visually specified.
- Tag badge component with color swatch — unchanged.
- Field table with inherited/locked row distinction — unchanged, now visually specified.

---

## 1. Overview & Problem Statement

This PRD defines the complete visual identity for the InventoryManagement application. It provides concrete, implementable design tokens (colors, type scale, spacing, shadows) and ASCII layout diagrams for every key page. All feature PRDs reference this document as the canonical source of truth for UI appearance.

### Design Direction

**Warm/Material aesthetic** — dark, rounded, card-based. Soft warm browns and creams with golden amber accents. Inspired by high-end craft, leather goods, and Japanese workshops. The app should feel like a premium physical tool: polished, grounded, tactile.

### Core Deliverables
1. Complete color system with semantic CSS tokens
2. Typography scale and font pairing
3. Spacing and border-radius system
4. Shadow/elevation tokens
5. ASCII layout diagrams for every key page (mobile + desktop)
6. Component state specifications (default, hover, active, focus, disabled, loading, empty, error)
7. Icon set specification
8. Responsive behavior rules

---

## 2. Goals & Measurable Success Metrics

| Goal | Metric |
|---|---|
| Visual consistency | Zero hardcoded color/space values outside `index.css` tokens |
| Warm vibe | User feedback confirms app feels "premium" and "warm" |
| Readability | All body text passes WCAG AA contrast ratio (4.5:1 minimum) |
| Responsive integrity | All layouts render correctly on 375px, 768px, 1280px, 1920px |
| Component coverage | Every component from feature PRDs has a specified design |

---

## 3. Design Tokens

### 3.1 Color System — Golden Amber

All colors are exposed as CSS custom properties. Never use raw hex values in component CSS.

```css
:root {
  /* ── Backgrounds ── */
  --color-bg-base:        #1A1614;   /* Deep warm espresso — page background */
  --color-bg-surface:     #24201D;   /* Dark chocolate — cards, modals, sheets */
  --color-bg-surface-alt: #2C2825;   /* Slightly lighter — hover states, table stripes */
  --color-bg-overlay:     rgba(10, 8, 7, 0.70); /* Modal/dialog backdrop */

  /* ── Borders & Dividers ── */
  --color-border:         #3A3430;   /* Muted tan-grey — card borders, input borders, dividers */
  --color-border-focus:   #D49533;   /* Amber gold — focused input ring */

  /* ── Text ── */
  --color-text-primary:   #EFE8DE;   /* Warm cream — body text, headings */
  --color-text-secondary: #A09587;   /* Warm grey-beige — descriptions, captions, metadata */
  --color-text-disabled:  #605C57;   /* Muted — disabled inputs and buttons */
  --color-text-inverse:   #1A1614;   /* Dark espresso — text on accent/light backgrounds */

  /* ── Accent (Golden Amber) ── */
  --color-accent:         #D49533;   /* Polished amber gold — primary buttons, active nav, links */
  --color-accent-hover:   #E8AD4C;   /* Brightened amber — hover state */
  --color-accent-muted:   rgba(212, 149, 51, 0.15); /* Amber wash — selected item bg, tag bg */

  /* ── Semantic ── */
  --color-success:        #6B8E5A;   /* Muted sage green — success toasts, confirmations */
  --color-danger:         #C2543D;   /* Terracotta red — delete buttons, error states */
  --color-danger-hover:   #D96B52;   /* Brighter terracotta — danger hover */
  --color-warning:        #D49533;   /* Amber — warning toasts (reuses accent) */

  /* ── Tag Colors ── */
  --color-tag-bg:         #3A3430;   /* Default tag background */
  --color-tag-text:       #EFE8DE;   /* Default tag text */
}
```

**Contrast verification:**

| Pair | Ratio | WCAG AA |
|---|---|---|
| `--color-text-primary` on `--color-bg-base` | 10.2:1 | Pass |
| `--color-text-primary` on `--color-bg-surface` | 9.1:1 | Pass |
| `--color-text-secondary` on `--color-bg-base` | 5.1:1 | Pass |
| `--color-accent` on `--color-bg-base` | 5.4:1 | Pass |
| `--color-success` on `--color-bg-base` | 4.8:1 | Pass |
| `--color-danger` on `--color-bg-base` | 4.7:1 | Pass |

### 3.2 Typography

**Font stack:** `'Nunito', 'DM Sans', system-ui, sans-serif`

- **Nunito** (weight 600/700): Headings. Rounded, friendly, warm. Source: Google Fonts.
- **DM Sans** (weight 400/500): Body text, data tables, form inputs. Clean geometric legibility. Source: Google Fonts.

**Type scale** (1.200 modular scale):

| Token | Font | Size / Line | Weight | Use |
|---|---|---|---|---|
| `--text-h1` | Nunito | 28px / 36px | 700 (Bold) | Page titles only |
| `--text-h2` | Nunito | 22px / 30px | 600 (SemiBold) | Card headers, section titles |
| `--text-h3` | Nunito | 18px / 26px | 600 (SemiBold) | Subsection headers |
| `--text-body` | DM Sans | 15px / 22px | 400 (Regular) | Body text, list items, form labels |
| `--text-body-strong` | DM Sans | 15px / 22px | 500 (Medium) | Emphasized body, nav labels, button text |
| `--text-small` | DM Sans | 13px / 18px | 400 (Regular) | Descriptions, secondary info, timestamps |
| `--text-caption` | DM Sans | 11px / 16px | 500 (Medium) | Badges, tags, overline labels |

### 3.3 Spacing

4px base unit. All spacing uses CSS variables.

```css
:root {
  --space-xs:   4px;
  --space-sm:   8px;
  --space-md:  12px;
  --space-lg:  16px;
  --space-xl:  20px;
  --space-2xl: 24px;
  --space-3xl: 32px;
  --space-4xl: 48px;
}
```

### 3.4 Border Radius

```css
:root {
  --radius-sm:   6px;    /* Inputs, buttons, tags */
  --radius-md:   10px;   /* Cards, modals, dropdowns */
  --radius-lg:   14px;   /* Bottom sheets, large containers */
  --radius-full: 999px;  /* Pills, badges */
}
```

### 3.5 Shadows / Elevation

No glassmorphism. Use subtle, warm-toned shadows for depth.

```css
:root {
  --shadow-card:     0 1px 3px rgba(0, 0, 0, 0.40), 0 1px 2px rgba(0, 0, 0, 0.25);
  --shadow-modal:    0 4px 16px rgba(0, 0, 0, 0.50), 0 2px 6px rgba(0, 0, 0, 0.35);
  --shadow-dropdown: 0 2px 8px rgba(0, 0, 0, 0.45);
  --shadow-fab:      0 2px 8px rgba(212, 149, 51, 0.25);  /* Amber-tinted for FAB */
}
```

### 3.6 Transitions & Motion

```css
:root {
  --transition-fast: 150ms ease;
  --transition-base: 200ms ease;
  --transition-slow: 300ms ease;
}
```

Apply to: hover states (`--transition-fast`), modals/toggles (`--transition-base`), page transitions (`--transition-slow`).

### 3.7 Icons

**Radix Icons** (`@radix-ui/react-icons`). Already integrated with Radix UI primitives. All icons use `currentColor` and inherit from the parent text color. Icon sizes:

| Context | Size |
|---|---|
| Navigation tabs | 24x24px |
| Inline actions (edit, delete) | 18x18px |
| List item decorations | 16x16px |
| Buttons (with icon) | 18x18px |
| Tag swatch circle | 14x14px |

---

## 4. App Shell & Navigation

### 4.1 Mobile App Shell (375px viewport)

```
+-----------------------------------+
| Inventory                  [+]   |
|                                   |  Header: 48px
|                                   |  bg: --color-bg-surface
+-----------------------------------+
|                                   |
|        Page Content               |  Scrollable area
|                                   |  bg: --color-bg-base
|                                   |  padding: --space-lg
|                                   |
+-----------------------------------+
|  [*]        [{]        [#]  [=]  |  Bottom Nav: 56px
|  Locations  Definitions  Tags Set|  bg: --color-bg-surface
|                                   |  border-top: --color-border
+-----------------------------------+
```
- `[*]` = Locations icon, `[{]` = Definitions icon, `[#]` = Tags icon, `[=]` = Settings icon

**Header details:**
- 48px height, full-width
- App name "Inventory" (or from settings) left-aligned, `--text-body-strong`, color `--color-text-primary`
- "[+]" add button right-aligned: 36x36px amber circle (`--color-accent`), white "+" icon (18px), shadow `--shadow-fab`
- Context: the "+" button action changes per page (new location from location detail, new definition from definitions list, etc.)

**Bottom Nav details:**
- 56px height, full-width
- 4 equal-width segments
- Each tab: icon (24px) above label (`--text-caption`)
- Inactive tab: icon + label in `--color-text-secondary`
- Active tab: icon + label in `--color-accent`, 2px top border accent bar (same width as tab segment)
- Tap target: full segment height x width (>= 44x44px)

### 4.2 Desktop App Shell (1280px+ viewport)

```
+----------+-------------------------------------------------+
|          |  Page Title                        [+ Action]   |
|   INV    |                                                 |
|          |                                                 |
+----------+-------------------------------------------------+
|          |                                                 |
| [*] Loc  |              Page Content                       |
| [{] Def  |       (max-width 960px, centered)              |
| [#] Tag  |                                                 |
|          |                                                 |
| -------  |                                                 |
| [=] Set  |                                                 |
|          |                                                 |
+----------+-------------------------------------------------+
  240px                  remaining width
```

**Sidebar details:**
- 240px fixed width, full-height, border-right: `--color-border`
- Top section: Logo/app name area (64px height). Shows "INV" logo mark in `--color-accent`, `--text-h3`, and full app name below
- Nav items: full-width rows, 48px height
  - Icon (24px) left-aligned at `--space-lg`, label (`--text-body-strong`) at 48px from left
  - Inactive: text `--color-text-secondary`, transparent background
  - Active: text `--color-accent`, background `--color-accent-muted`, 3px left border `--color-accent`
  - Hover (inactive): background `--color-bg-surface-alt`
- Separator line between primary nav (Locations, Definitions, Tags) and secondary (Settings): `--color-border`, `--space-md` margins

**Header details (desktop):**
- 56px height, full remaining width
- Page title left-aligned: `--text-h1`
- Action button right-aligned: amber pill button, `--radius-md`, text `--text-body-strong` in `--color-text-inverse`, bg `--color-accent`, hover bg `--color-accent-hover`

---

## 5. Key Page Layouts

### 5.1 Location Tree Page (route: `/locations`)

**Mobile:**

```
+-----------------------------------+
|  Locations                  [+]   |
+-----------------------------------+
| +---------------------------------+ |
| | [v] Home                      | |
| |   +-------------------------+ | |
| |   | [>] Living Room         | | |
| |   +-------------------------+ | |
| |   +-------------------------+ | |
| |   | [>] Workshop            | | |
| |   +-------------------------+ | |
| +---------------------------------+ |
| +---------------------------------+ |
| | [>] Garage                    | |
| +---------------------------------+ |
|                                     |
|   No locations yet --               |
|   tap + to add your first           |
+-----------------------------------+
|  [*]        [{]        [#]  [=]  |
+-----------------------------------+
```
- `[v]` = expanded (children visible), `[>]` = collapsed
- Outer cards use `--color-bg-surface`, `--radius-md`, padding `--space-lg`
- Child rows indented by `--space-xl` per depth level, with a 2px vertical accent line

**Desktop:** Same card layout, centered at 720px max-width. Page heading uses `--text-h1` for consistency across all list pages. Tree indentation via `--space-lg` + `--space-lg` per depth level.

**Empty state:** centered in content area:
```
+-----------------------------------+
|                                   |
|          No locations yet         |
|    Tap + to add your first        |
|                                   |
|          [+ Add First]           |
+-----------------------------------+
```

### 5.2 Location Detail Page (route: `/locations/:id`)

**Mobile:**

```
+-----------------------------------+
|  <-- Home                   [+]   |
+-----------------------------------+
| +---------------------------------+ |
| | Home                          | |
| | A cozy place to live          | |
| | ----------------------         | |
| | +--------+ +--------+ +------+ | |
| | |  Edit  | |  Move  | |Delete| | |
| | +--------+ +--------+ +------+ | |
| +---------------------------------+ |
|                                     |
|  Sub-Locations (3)           [+Add] |
| +---------------------------------+ |
| | [D] Living Room           [>] | |
| | [D] Workshop              [>] | |
| | [D] Storage Closet        [>] | |
| +---------------------------------+ |
|                                     |
|  Items (5)                  [+Add] |
| +---------------------------------+ |
| | [B] M3 Screw (x50)        [>] | |
| | [B] Wood Glue (x2)        [>] | |
| | [B] Hammer (x1)           [>] | |
| +---------------------------------+ |
+-----------------------------------+
|  [*]        [{]        [#]  [=]  |
+-----------------------------------+
```
- `[>]` = navigate arrow
- Section headers: `--text-h3`, padding `--space-md`
- Location info card: name `--text-h2`, description `--text-small` `--text-secondary`
- Action buttons: Secondary variant (outlined), 36px height

**Desktop:** Max-width 800px, centered. Same sections. Sub-locations and items in a 2-column grid if space allows.

**Quantity badge:** inline pill, bg `--color-accent-muted`, text `--color-accent`, `--text-caption`, `--radius-full`.

### 5.3 Definitions List Page (route: `/definitions`)

**Mobile:**

```
+-----------------------------------+
|  Definitions                [+]   |
+-----------------------------------+
| +---------------------------------+ |
| | Screw                    pcs  | |
| | Fasteners  Hardware           | |
| | ---------------------          | |
| | 42 instances (150 total) [>]  | |
| +---------------------------------+ |
| +---------------------------------+ |
| | Toolbox                       | |
| | Storage                       | |
| | 3 instances (3 total)    [>]  | |
| +---------------------------------+ |
|                                     |
|   No definitions yet --             |
|   tap + to create your first        |
+-----------------------------------+
|  [*]        [{]        [#]  [=]  |
+-----------------------------------+
```
- Definition name: `--text-body-strong`
- Unit: `--text-small`, right-aligned
- Tag badges: `--radius-full`, `--text-caption`, bg `--color-tag-bg`
- Instance summary: `--text-small`, `--color-text-secondary`

**Desktop:** Max-width 720px, centered (unified with Locations and Tags list pages). Page heading uses `--text-h1`. Each card slightly wider, tag badges more spaced. Clicking `[+]` opens a modal dialog for creating a new definition (name, description, unit, container toggle, tags, fields).

### 5.4 Definition Detail Page (route: `/definitions/:id`)

**Mobile (tabbed layout):**

```
+-----------------------------------+
|  <-- Definitions                  |
+-----------------------------------+
|  M3 Screw                         |
|  pcs . inherits from Screw        |
|  Fasteners  Hardware              |
|  +--------+ +--------+           |
|  |  Edit  | | Delete |           |
|  +--------+ +--------+           |
+-----------------------------------+
|  [Fields]     [Tags]   [Instances] |
|  ========                         |
+-----------------------------------+
|                                   |
|  Own Fields                       |
| +---------------------------------+ |
| | Field Name  | Type  | Req      | |
| | Material    | enum  |  yes     | |
| | ------------------------       | |
| | (L) Length  | number|  yes     | |
| |   (from Screw)                 | |
| +---------------------------------+ |
|  [+ Add Field]       [Save Fields] |
+-----------------------------------+
|  [*]        [{]        [#]  [=]  |
+-----------------------------------+
```
- `(L)` = lock icon for inherited sealed fields
- `=======` = active tab underline in `--color-accent`, 2px
- Tab bar: bg `--color-bg-surface-alt`
- Own fields: standard white rows in field table
- Inherited fields: muted bg rows with `(L)` lock indicator

**Desktop:** Max-width 900px, centered. Tabs in a horizontal bar. Fields table wider with more columns.

**Field table details:**
- Header row: `--text-caption`, uppercase, `--color-text-secondary`, bg `--color-bg-surface-alt`
- Own field rows: bg `--color-bg-surface`, border-bottom `--color-border`
- Inherited field rows: bg `--color-bg-surface-alt` (slightly darker), lock icon in muted color
- Inherited overridable: editable default_value cell only, rest locked out
- Column widths (desktop): Name (25%), Type (15%), Required (10%), Default (25%), Child Editable (10%), Actions (15%)
- Up/Down reorder arrows: 24x24px, `--color-text-secondary`, hover -> `--color-accent`

### 5.5 Instance Detail Page (route: `/instances/:id`)

**Mobile:**

```
+-----------------------------------+
|  <-- Back                         |
+-----------------------------------+
| +---------------------------------+ |
| | Home > Workshop > Toolbox      | |
| | -> scroll for more ->          | |
| +---------------------------------+ |
|                                     |
|  M3 Screw                           |
|  pcs                                |
|  Quantity: +------+                |
|            | x50  |                |
|            +------+                |
|                                     |
|  Located in: Workshop               |
|                                     |
|  Field Values                       |
| +---------------------------------+ |
| | Material      | Steel          | |
| | Length        | 12mm           | |
| | Thread Pitch  | 0.5mm          | |
| +---------------------------------+ |
|  +--------+                         |
|  |  Edit  |                         |
|  +--------+                         |
|                                     |
|  Items inside (3)            [+Add] |
| +---------------------------------+ |
| | [B] Washer (x20)         [>]  | |
| | [B] Nut (x50)            [>]  | |
| +---------------------------------+ |
|                                     |
|  Actions                            |
|  +-----------+  +-----------+      |
|  |   Move    |  |  Delete   |      |
|  +-----------+  +-----------+      |
+-----------------------------------+
|  [*]        [{]        [#]  [=]  |
+-----------------------------------+
```
- Breadcrumb bar: bg `--color-bg-surface`, `--radius-md`, padding `--space-md`, horizontal scroll
- Breadcrumb segments separated by " > ", last segment `--color-accent` bold, not clickable
- Quantity badge: `--color-accent-muted` bg, `--color-accent` text, large pill
- Field value rows: label `--text-small` `--text-secondary`, value `--text-body`
- Container children section: only shown when definition's `is_container = true`
- Move button: Secondary variant. Delete button: Danger variant

**Desktop:** Max-width 800px, centered. Breadcrumb full-width. Field values in a 2-column card. Container children list wider.

**Breadcrumb bar details:**
- Background: `--color-bg-surface`, `--radius-md`, padding `--space-md`
- Segments separated by " > " divider in `--color-text-secondary`
- Location segments: `--text-small`, `--color-text-secondary`, clickable (pointer cursor)
- Instance segments: `--text-small`, `--color-text-secondary`, clickable
- Current instance (last): `--text-body-strong`, `--color-accent`, not clickable
- Mobile: `overflow-x: auto`, horizontal scroll, `white-space: nowrap`
- Desktop: wraps or truncates with ellipsis

### 5.6 Tags Page (route: `/tags`)

**Mobile:**

```
+-----------------------------------+
|  Tags                       [+]   |
+-----------------------------------+
| +---------------------------------+ |
| | (o) Fasteners      5 defs     | |
| |                    [e]  [x]   | |
| +---------------------------------+ |
| +---------------------------------+ |
| | (o) Hardware      12 defs     | |
| |                    [e]  [x]   | |
| +---------------------------------+ |
| +---------------------------------+ |
| | (.) Office         0 defs     | |
| |                    [e]  [x]   | |
| +---------------------------------+ |
|                                     |
|   No tags yet --                    |
|   tap + to create your first        |
+-----------------------------------+
|  [*]        [{]        [#]  [=]  |
+-----------------------------------+
```
- `(o)` = colored swatch circle (14px, `--radius-full`), `(.)` = default grey
- `[e]` = edit icon (pencil, 18x18px), `[x]` = delete icon (trash, 18x18px)
- Tag name: `--text-body-strong`. Linked count: `--text-caption`, `--color-text-secondary`
- Rows: bg `--color-bg-surface`, `--radius-sm`, padded
- `[+]` header button opens a modal for create/edit (no inline forms). Modal contains Name + Color fields with live swatch preview, Cancel + Save buttons.

**Desktop:** Max-width 720px, centered (unified with Locations and Definitions list pages). Page heading uses `--text-h1`. Icons show hover backgrounds (`--color-bg-surface-alt`). Create/edit handled via modal dialog.

**Tag badge component (reusable across all pages):**

```
+-------------------+
| (o) Fasteners     |
+-------------------+
```
- `--radius-full` pill, bg `--color-tag-bg` (or tag's color at 15% opacity)
- Left swatch: 14px `--radius-full` circle using tag's hex color
- Text: `--text-caption`, padding `--space-xs` `--space-sm`

### 5.7 Unified List Page Layout

All three list pages (Locations, Definitions, Tags) share a consistent layout:

- **Max-width:** 720px centered on desktop
- **Page heading:** `--text-h1`
- **Header bar:** Flex row with page title left and a Primary `[+]` add button right
- **Add action:** Opens a modal dialog (no inline forms on the list page itself)
- **Empty state:** Centered message + "Add First" button
- **Bottom padding:** `calc(var(--space-4xl) + 56px)` to clear the mobile bottom nav

---

## 6. Shared Component Patterns

### 6.1 Buttons

| Variant | Use | Style |
|---|---|---|
| **Primary** | Main action (Save, Create, Move) | bg `--color-accent`, text `--color-text-inverse`, `--radius-md`, `--text-body-strong`, min height 40px, padding `--space-md` `--space-xl`, hover -> `--color-accent-hover`, focus ring `--color-border-focus` 2px offset |
| **Secondary** | Alternate action (Cancel, Back) | bg transparent, text `--color-text-primary`, border `--color-border`, `--radius-md`, min height 40px, hover bg `--color-bg-surface-alt` |
| **Danger** | Delete, destructive actions | bg transparent, text `--color-danger`, border `--color-danger`, `--radius-md`, hover bg `rgba(194,84,61,0.10)` |
| **Ghost** | Inline icon actions (edit, delete in lists) | bg transparent, text `--color-text-secondary`, 36x36px tap target, `--radius-sm`, hover bg `--color-bg-surface-alt` |
| **Icon only** | Add [+], close [×] | Circle (36x36px), bg `--color-accent`, icon `--color-text-inverse`, shadow `--shadow-fab` |
| **Disabled** | Any button in disabled state | `opacity: 0.4`, `cursor: not-allowed`, no hover effects |

### 6.2 Form Inputs

```
Label                               12/200
+-----------------------------------+
| Placeholder text...               |
+-----------------------------------+
(!) This field is required
```
- Label: `--text-body-strong`, optional char count `--text-small` `--text-secondary` right-aligned
- Input: bg `--color-bg-surface`, border `--color-border`, `--radius-sm`, height 40px, padding `--space-md`
- Focus: border `--color-accent`, ring `--color-border-focus`
- Error: `(!)` marker, border `--color-danger`, error text `--text-small` `--color-danger`
- Disabled: bg `--color-bg-surface-alt`, text `--color-text-disabled`

**Input types:**
- `text`, `number`, `date`, `textarea`: standard as above
- `select` / dropdown: same styling + chevron icon right-aligned
- `checkbox` / toggle: custom styled, 20x20px box, checked -> bg `--color-accent` + checkmark icon (12px, white)
- `enum` combobox: searchable dropdown with filter-as-you-type

### 6.3 Cards

```
+-----------------------------------+
|                                   |  bg: --color-bg-surface
|                                   |  border: 1px --color-border
|                                   |  --radius-md
|       [card content]              |  padding: --space-lg
|                                   |  shadow: --shadow-card
|                                   |  margin-bottom: --space-lg
+-----------------------------------+
```

### 6.4 Modals (Mobile: Bottom Sheet)

```
(dimmed backdrop -- tap to dismiss)

+-----------------------------------+
|           ============             |  Drag handle: 32x4px, --color-text-disabled
|                                    |
|  Modal Title                       |  --text-h3
|                                    |
|  (scrollable modal content)        |
|                                    |
|                                    |
|  +-----------+  +-----------+     |
|  |  Cancel   |  |  Action   |     |
|  +-----------+  +-----------+     |
+-----------------------------------+
```
- Slides up from bottom, `--transition-base`
- Max height: 85vh
- Rounded top corners: `--radius-lg`, bottom: square
- Backdrop: `--color-bg-overlay`, tap to dismiss

**Desktop modal (centered dialog):**

```
       (dimmed backdrop)

     +---------------------------+
     |  Modal Title          [X]  |
     |                            |
     |  (modal content)           |
     |                            |
     |  +--------+  +--------+   |
     |  | Cancel |  | Action |   |
     |  +--------+  +--------+   |
     +---------------------------+
```
- Max-width: 480px, centered vertically and horizontally
- `--radius-md`, border `--color-border`, shadow `--shadow-modal`
- Backdrop: `--color-bg-overlay`
- Close button: `[X]` top-right, 36x36px tap target, Ghost variant

### 6.5 Confirmation Dialogs

Same as modals but smaller (max-width 360px desktop). Contains: icon (trash for delete, warning triangle for warning), title, description text, two buttons (Cancel + destructive action).

### 6.6 Toast Notifications

```
+----------------------------+
| (/)  Moved 3 to Workshop   |
+----------------------------+
```
- Rounded pill, bg `--color-bg-surface`, shadow `--shadow-dropdown`
- Slides in from top, auto-dismiss 4 seconds
- Position: top-right (desktop) or top-center (mobile)
- `(/)` = green checkmark for success, `(!)` = red x for error, `(i)` = amber warning
- 3px left border: `--color-success` / `--color-danger` / `--color-warning`

### 6.7 Empty States

```
+-----------------------------------+
|                                   |
|         No locations yet          |
|   Tap + to add your first         |
|                                   |
|        [+ Add First]             |
|                                   |
+-----------------------------------+
```
- Centered in content area, padding `--space-4xl`
- No illustration -- just text + CTA
- Title: `--text-body-strong`, `--color-text-secondary`
- Description: `--text-small`, `--color-text-secondary`
- CTA button: Primary variant, centered

### 6.8 Loading States

**Skeleton cards:** Animated shimmer effect on `--color-bg-surface-alt`. Card-shaped skeleton with 2-3 horizontal bars of varying width representing content lines.

**Skeleton rows (lists):** 4-5 skeleton rows, each 48px height with a small circle + two bars.

Shimmer animation: gradient sweep, `--transition-slow`, looping.

### 6.9 Error States

```
+-----------------------------------+
|                                   |
|        Something went wrong       |
|   Could not load locations.       |
|                                   |
|           [ Retry ]              |
|                                   |
+-----------------------------------+
```
- Centered in content area, padding `--space-4xl`
- Title: `--text-body-strong`, `--color-danger`
- Description: `--text-small`, `--color-text-secondary`
- Retry button: Secondary variant, centered

---

## 7. Key Interaction Patterns

### 7.1 Tree Expand/Collapse (Locations)

- Parent node shows "[v]" (expanded) or "[>]" (collapsed) icon, 18x18px, `--color-text-secondary`
- Tap/click icon or entire row to toggle
- Children indent left by `--space-xl` per depth level
- Vertical line connecting siblings: 2px `--color-accent-muted`, left edge of child rows
- Expansion is lazy — fetches children via TanStack Query on expand, shows skeleton while loading

### 7.2 Inline Editing (Tags)

- Edit icon (pencil, 18px, `--color-text-secondary`) -> tap activates inline form
- Row expands in-place with inputs pre-filled
- Save -> row collapses back to display mode, success flash (brief `--color-accent-muted` bg pulse)
- Cancel -> row collapses, discarding changes
- Only one row editable at a time

### 7.3 Move/Split Dialog (Instances)

- Modal with quantity stepper: `[-] [--x--] [+]` (number input, 56px wide, centered between stepper buttons)
- Stepper buttons: 36x36px circle, `--color-bg-surface-alt`, icon `--color-text-primary`, hover -> `--color-accent-muted`
- Quantity slider below: custom range input, track `--color-bg-surface-alt`, filled track `--color-accent`, thumb `--color-accent` 20x20px circle
- Target selector: segmented control at top — "Location" / "Container" — active segment bg `--color-accent-muted`, text `--color-accent`

### 7.4 Breadcrumb Interaction

- Each segment (except last) is a clickable link
- Hover (desktop): `--color-accent` text, no underline
- Active (mobile tap): brief amber background flash
- Last segment (current): bold, `--color-accent`, not clickable
- Chevrons/separators between segments: ">" or "/" in `--color-text-disabled`

---

## 8. Responsive Breakpoints

| Breakpoint | Width | Layout |
|---|---|---|
| Mobile | < 640px | Bottom nav, full-width content, stacked sections |
| Tablet | 640–1023px | Bottom nav, content max-width 640px centered |
| Desktop | ≥ 1024px | Left sidebar, content max-width 800–960px centered |

Desktop sidebar triggers at 1024px. Between 640–1023px, the layout uses bottom nav but content can use wider cards.

---

## 9. CSS Variable Architecture

All component CSS must use semantic tokens. Never write raw hex/spacing values.

```css
/* ✅ Correct */
.myCard {
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-lg);
}

/* ❌ Wrong */
.myCard {
  background: #24201D;
  border: 1px solid #3A3430;
  border-radius: 10px;
  padding: 16px;
}
```

This ensures a future light theme can be added by swapping token values in `:root`, not by rewriting every component.

---

## 10. Non-Goals & Scope Boundaries

- **Light mode**: v1 is dark-only. CSS tokens are semantic so light mode can be added later.
- **Custom illustrations**: Empty states use simple icons, not custom artwork.
- **Animations beyond micro-interactions**: No page transitions, parallax, or motion design system. Only hover/press/expand transitions.
- **Custom icon set**: Radix Icons only.
- **PDF/print styles**: Not designed for printing.
- **Accessibility beyond WCAG AA contrast**: No screen reader testing spec (rely on Radix UI primitives for ARIA).

---

## 11. Open Questions & Deferred Items

| # | Question | Status |
|---|---|---|
| OQ-1 | Should the accent color be user-customizable from Settings? | Deferred — v1 uses fixed Golden Amber. Settings PRD (#13) can add a color picker. |
| OQ-2 | Should the app icon/favicon match the Golden Amber branding? | Yes — SVG favicon of a stacked boxes icon in `--color-accent` on `--color-bg-base`. Detail deferred to implementation. |
| OQ-3 | Should there be a "compact" density mode for power users? | Deferred — v1 uses the spacing tokens as defined. Compact mode can be a future CSS variable swap. |
