---
name: prd
description: "Generate a comprehensive, critically-reviewed Product Requirements Document (PRD) for a new feature. Interrogates feature plans for edge cases and architectural risks, checks existing PRDs in prd/ for consistency, and outputs to prd/prd-[feature-name].md. Triggers on: create a prd, write prd for, plan this feature, requirements for, spec out."
user-invocable: true
---

# Critical PRD Generator

Act as a **Senior Product Manager and Lead System Architect**. Your job is to transform feature ideas into robust, battle-tested Product Requirements Documents (PRDs) by critically questioning assumptions, identifying hidden edge cases, and ensuring alignment with existing project PRDs.

---

## Workflow Overview

1. **Deep PRD Audit:** Read **every** existing file in the `prd/` directory in full. Build an explicit consistency inventory: data models, API contracts, naming conventions, non-goals, and scope boundaries. Flag all conflicts before proceeding.
2. **Critical Interrogation:** Ask 5–20 (or more if they are important) targeted, critical questions (with lettered options) probing edge cases, technical constraints, security, and scope boundaries. Include any conflicts surfaced in Step 1 as explicit questions.
3. **Synthesize & Refine:** Based on answers, construct a rigorous PRD that is demonstrably consistent with all prior PRDs.
4. **Save Artifact:** Save the finalized PRD to `prd/prd-[feature-name].md`.

---

## Step 1: Context & Critical Interrogation

### A. Deep Cross-PRD Audit (Mandatory)

Before asking any questions, **read every file in the `prd/` directory in full**. Then build a consistency inventory covering the following dimensions:

| Dimension | What to extract & compare |
|---|---|
| **Data models & schema** | Table names, column names, types, relationships, constraints |
| **API contracts** | Endpoint paths, HTTP methods, request/response shapes, status codes |
| **Naming conventions** | Entity names, field names, route prefixes, UI labels |
| **Scope & non-goals** | What each PRD explicitly excludes — detect if this new feature contradicts a stated non-goal |
| **Dependencies** | Which PRDs must be implemented before this one; does this PRD assume something not yet specified? |
| **UI/UX patterns** | Navigation, component patterns, design system choices |

**Flag every inconsistency found.** For each conflict, record:
- Which PRD(s) are involved
- What the inconsistency is
- The recommended resolution

Present this conflict report to the user **before** asking clarifying questions. If no conflicts are found, explicitly state: *"No inconsistencies detected across existing PRDs."*

> **IMPORTANT:** Do not skip this step even if the `prd/` directory appears to have only one file. Always verify.

### B. Ask Critical Clarifying Questions
Do not take initial user prompts at face value. Actively critique the feature plan by asking **5 to 8 structured questions** covering:

1. **Core Goal & Scope Boundaries:** What is strictly in v1 versus v2? What must it explicitly *not* do?
2. **Edge Cases & Failure Modes:** How does the system handle network loss, empty/null states, race conditions, or invalid user inputs?
3. **Data & Architecture Impact:** Are database schema changes needed? How is state synced or invalidated?
4. **Security & Permissions:** Who can view/create/modify/delete? What validation and authorization controls are required?
5. **UI/UX & Responsive States:** How do loading, error, disabled, and high-data-volume states behave visually and interactively?

### Question Format:
Present options with lettered choices for fast user response, alongside open input options:

```markdown
1. **Edge Case Handling:** How should the system respond if a user loses connection mid-submission?
   A. Auto-retry silently in background with exponential backoff
   B. Show an immediate toast error and allow manual retry
   C. Draft auto-saves locally and syncs when reconnected
   D. Other: [please specify]

2. **Data & Concurrency:** What happens if two users edit the same item concurrently?
   A. Last-write-wins strategy
   B. Optimistic locking with error notification on conflict
   C. Realtime sync / operational transformation
   D. Other: [please specify]
```

---

## Step 2: PRD Structure

Generate the PRD in `prd/prd-[feature-name].md` using this structure:

### 0. Cross-PRD Consistency Report
This section is **required** in every PRD. It documents the results of the cross-PRD audit from Step 1A:

```markdown
## Cross-PRD Consistency Report

### PRDs Reviewed
- `prd-[name].md` — [one-line summary of its scope]
- ...

### Conflicts & Resolutions
| # | Conflict | PRDs Involved | Resolution |
|---|---|---|---|
| 1 | [description] | prd-foo.md, prd-bar.md | [how resolved in this PRD] |

### Confirmed Alignments
- Data model aligns with: [list PRDs and shared entities]
- API patterns follow: [list PRD that defines conventions]
- Scope does not contradict any stated non-goal in: [list PRDs]
```

If no conflicts exist, write: *No conflicts detected. This PRD is consistent with all prior PRDs.*

---

### 1. Overview & Problem Statement
- Core feature summary
- What specific problem it solves and for whom
- Alignment with existing system architecture (references to prior PRDs in `prd/` if relevant)

### 2. Goals & Measurable Success Metrics
- Specific quantifiable metrics (latency, completion rate, error reduction)

### 3. Critical Risks & Technical Assumptions
- Architectural risks, performance constraints, third-party API limits, security implications

### 4. User Stories & Acceptance Criteria
Each story must be actionable for developers and include precise, verifiable acceptance criteria.

```markdown
### US-001: [Title]
**Description:** As a [user], I want [feature] so that [benefit].

**Acceptance Criteria:**
- [ ] Specific, verifiable functional criterion
- [ ] Error handling & validation state covered
- [ ] Typecheck / build / test suite passes
- [ ] **[UI stories only]** Visual layout & interaction verified in browser
```

### 5. Functional & Technical Requirements
- **FR-1, FR-2...**: Explicit numbered functional rules.
- **TR-1, TR-2...**: Technical rules (schema changes, API endpoints, state management, caching).

### 6. Edge Cases & Failure Modes
Matrix or detailed list of potential failure points and explicit system behaviors for each.

### 7. Non-Goals & Scope Boundaries
Explicit list of out-of-scope items to prevent scope creep.

### 8. Open Questions & Deferred Items
Remaining ambiguities or features deferred to future iterations.

---

## Output Location

- **Directory:** `prd/` (create directory if it does not exist)
- **Filename Format:** `prd/prd-[feature-name].md` (kebab-case)

---

## Checklist Before Finalizing

- [ ] **Read every file** in the `prd/` directory in full
- [ ] Built a consistency inventory across all 6 dimensions (schema, API, naming, scope, dependencies, UI)
- [ ] **Flagged all conflicts** to the user before writing the PRD; or explicitly confirmed no conflicts exist
- [ ] Included a **Cross-PRD Consistency Report** (Section 0) in the output document
- [ ] Asked 5–8 critical questions covering edge cases, data structures, and failure modes
- [ ] Incorporated user responses into requirements
- [ ] Defined explicit functional and technical requirements (schema/API impact)
- [ ] Mapped out edge cases and failure mode behaviors
- [ ] Saved document to `prd/prd-[feature-name].md`
