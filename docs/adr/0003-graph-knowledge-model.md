# ADR-0003: Graph-first knowledge model replacing Johnny Decimal

**Status:** Accepted
**Date:** 2026-07-24
**Deciders:** Repository owner
**Supersedes:** The Johnny Decimal subsystem (dialog, `goToJD`, numeric codes, sidecar `johnnyDecimal` index, `<!-- orbit:jd -->` markers-as-structure)

## Context

Balaur shipped a Johnny Decimal (JD) subsystem: numeric area/category/item codes, a validation dialog, `goToJD` lookup, `<!-- orbit:jd -->` markers treated as structure, and a sidecar `johnnyDecimal` index. JD is heavier than the app needs, has no lifecycle (capture → project → archive), and forces a dialog/validation layer that fights the canvas-as-editable-graph medium.

The settled direction reframes the app as "Obsidian + AI + Canvas as the operating layer": the editable spatial canvas graph *is* the knowledge layer, and the AI reads/traverses it as memory.

## Decision

Replace the JD subsystem with a graph-first knowledge model:

1. **Four hub canvases off Home** — Inbox (capture pending processing), Projects (committed efforts with a finish line), Wiki (durable reference), Archive (dormant and completed). Home is the root canvas and the canonical AI + user entry point.

2. **Node typing via three existing channels** (no new mechanism):
   - Canvas kinds (sidecar `kind`): `hub`, `project`
   - Note kinds (inert body markers): `<!-- orbit:inbox -->` (capture pending processing), `<!-- orbit:reference -->` (durable wiki page)
   - Entity files (frontmatter `orbit-type`): `task`, `journal` (both already exist)
   - `<!-- orbit:ai-card -->` (existing marker)
   - `<!-- orbit:jd -->` markers from pre-graph vaults remain harmless inert text; no migration is performed.

3. **Relation labels as convention, not enforced schema** (an enforced enum would be a proprietary Canvas dialect, forbidden by AGENTS.md §4.1): `part-of` (structural), `relates-to` (associative), `filed-to` (lifecycle: inbox→Project/Wiki, Project→Archive). `AI output` stays reserved. Edge `label` is already accepted by the validator, rendered, and editable.

4. **AI memory layer** = read/traverse from Home via labelled edges + type markers + one-line summaries, bounded depth (2 canvas hops, 60-node cap). Propose changes only through the existing confirmed AI-operation flow (AGENTS.md §10). No autonomous mutation, auto-tagging, auto-linking, or roaming.

5. **Journaling as a Today feature** — daily-note panel in the Today view with date navigation (prev/Today/next), editable body (debounced create-or-update), and an explicit "Place on canvas" button that adds a standard `file` node for the viewed date's journal onto the currently open canvas. Journal files use the existing `FileJournalRepository` and `journal/YYYY/YYYY-MM-DD.md` layout.

6. **Archive convention** — reparent a node/portal under the Archive hub and set the dormant node color `#6c757d` (a named constant `DORMANT_NODE_COLOR`). Never delete. Manual and explicit; no automated archive button for v1.

7. **Sidecar schema** — drop `johnnyDecimal`; add optional canvas `kind` (`hub`/`project`) as app metadata; `parseSidecar` strips legacy `johnnyDecimal` and unknown `kind` values on read so old sidecars load. `SIDECAR_VERSION` stays at 2 (bumping would break every existing vault and backup).

8. **No migration** — existing JD vaults keep their `<!-- orbit:jd -->` markers as inert text and their canvases as valid JSON Canvas. `normalizeWorkspace` and `parseSidecar` strip legacy sidecar JD on load. Users can export a whole-space backup (still valid) or load the new graph starter.

9. **Self-teaching graph starter** replaces the JD "Alex, age 30" starter: Home (root) + four hub portals + example nodes demonstrating the pipeline (an inbox note `filed-to` a Project canvas containing a task; reference pages `relates-to` each other under Wiki; a journal node; an archived item under Archive), with labelled edges and one-line summaries. Bundled widget seeding retained.

10. **One-line summary convention** — derived from existing content, no new field. A node's summary is its heading/title: `nodeTitle(node)` for structure; for a text note, the first `# Heading` if present, else the first non-empty, non-marker body line, truncated to ~120 chars.

## Consequences

- The JD dialog, `goToJD`, `createJDEntry`, `loadJohnnyDecimalStarter`, `createJohnnyDecimalStarterWorkspace`, `seedStarterTasks`, and all `jd*` helper functions are removed from `app.js`.
- The sidecar no longer carries a `johnnyDecimal` index; `parseSidecar` strips legacy JD fields on read.
- The `window.orbitCanvas` surface exposes `loadGraphStarter` instead of `createJDEntry`/`goToJD`/`loadJohnnyDecimalStarter`.
- The Today view gains a daily-note panel with date navigation and place-on-canvas.
- The AI assistant receives a bounded graph memory digest (traversed from Home) as context for both local and remote providers.
- The inspector color swatches include the dormant grey `#6c757d` for the archive convention.
- Existing JD vaults load without error; their JD markers become inert text and their sidecar JD index is stripped on read.

## Verification boundary

- Node suite: 169 tests pass (phase4 updated for the new `kind` schema and legacy-strip behavior; phase9/phase10/phase4-backup pass unchanged).
- Browser smoke: 12–13 checks pass (fresh profile boots into the graph starter; all nodes render; file index reports no SQLite; selection frame works; document stays valid JSON Canvas; reload preserves workspace; offline renders from cache).
- Browser-pending: IndexedDB durability, first-render timing, journal create/edit/place in a real browser, timezone/local-date boundaries, malformed-file repair in the running UI, and the AI memory digest against a real provider.
