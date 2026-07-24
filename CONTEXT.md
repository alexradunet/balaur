# Balaur domain context

Use these terms consistently. The glossary summarizes names and distinctions; `AGENTS.md`, `docs/architecture.md`, `docs/life-data.md`, and the ADRs remain authoritative for behavior and constraints.

## Core model

### Vault

The source-of-truth collection of logical user files. Browser runtime uses `IndexedDbVault`; Node tooling and tests use `FsVault` or `MemoryVault`. A vault adapter is storage infrastructure, not a different data model.

### Canonical file

A vault file that owns durable user data. This includes JSON Canvas documents, Markdown entities, the workspace sidecar, and reviewed widget files. Do not call disposable index rows or local UI state canonical.

### Canvas document

One independently valid JSON Canvas 1.0 document containing only standard nodes and edges. It owns portable spatial content, not application hierarchy, camera, filters, or entity state. Avoid “workspace” when referring to a single canvas document.

### Workspace

The user-visible collection of canvas documents plus its hierarchy and application metadata. It is persisted as separate `.canvas` files and the metadata-only workspace sidecar, not as one proprietary canvas document.

### Workspace sidecar

`.orbit/workspace.json`. It owns canvas paths and titles, hierarchy and portals, active canvas, cameras, and optional canvas `kind` metadata (`hub`/`project`). It never embeds canvas documents. Legacy `johnnyDecimal` fields are stripped on read (ADR-0003).

### Entity

A life-management record whose identity and fields live in one canonical Markdown file. Tasks, habits, habit check-ins, journals, and calendar events are entities. An entity is not a canvas node.

### Note

A path-identified canonical `notes/*.md` file with no mandatory frontmatter or `orbit-id`; its identity is the path. Kind (inbox, reference, or AI) is an inert body marker rather than a separate type, and the indexer treats the file as valid untyped Markdown. Placed by zero or more standard `file` nodes, the note is the unified content unit that replaces inline text-node authoring (ADR-0004).
_Avoid_: "text note", "inline note", "card", "entity".

### Placement

A standard JSON Canvas `file` node that references a canonical file (ADR-0004): a note (`notes/*.md`), an entity (task/habit/journal/event `.md`), a component card (`cards/*.md`), a widget (`.html`), or a portal sub-canvas (`canvases/*.canvas`). Its node ID identifies that spatial occurrence only; one canonical file may have zero, one, or many placements.

### Projection

Disposable runtime data derived from canonical files for querying or rendering. `MemoryIndex`, `LifeQuery` results, and preloaded catalogs are projections. They may be purged and rebuilt and must never become a second owner.

### Repository

A canonical-file-first service that validates and writes entities, then reconciles projections. Do not use “repository” for the Git clone when discussing runtime architecture; say “Git repository” in that context.

## Life data

### Task

A canonical `tasks/*.md` entity with an immutable Orbit ID and explicit status. A task can exist without a placement.

### Scheduled date

`scheduled-on`, the local date on which work is intended. It is not a deadline.

### Due date

`due-on`, the local-date deadline. Never infer it from the scheduled date.

### Habit definition

A canonical `habits/*.md` entity describing the habit. Completion history does not mutate the definition.

### Habit check-in

An immutable event appended to a daily `habit-logs/YYYY/*.md` file. It is not a recurring task record.

### Component card

A declarative, canonical `cards/*.md` file rendered through an allowlisted recipe. It is data, not executable HTML.

### Widget

A reviewed canonical `widgets/*.html` file run only on explicit request inside an iframe with `sandbox="allow-scripts"`. It is executable content but has no host or vault authority.

## Canvas behavior

### Portal

A standard `file` node that points to another `.canvas` file. The sidecar records the parent relationship; the node itself remains portable JSON Canvas.

### Hub

A light entry canvas (Inbox, Projects, Wiki, or Archive) hanging off Home. Marked with sidecar `kind: "hub"`. Hubs are entry points, not folders.

### Home

The root canvas and the canonical AI + user entry point.

### Inbox note

A note: a path-identified `notes/*.md` file placed by a standard `file` node, whose body carries the inert `<!-- orbit:inbox -->` marker; a capture pending processing.
_Avoid_: "text node", "inbox card".

### Reference page

A note: a path-identified `notes/*.md` file placed by a standard `file` node, whose body carries the inert `<!-- orbit:reference -->` marker; durable wiki content.
_Avoid_: "text node", "reference card", "wiki page".

### Drain

To re-place a path-bound node from one canvas to another: remove the placement here and add a placement for the same path there, leaving the canonical file and its identity unchanged. Because every content node is path-bound, notes, tasks, and journals all drain the same way; this is how the daily canvas (project 002) processes captures.
_Avoid_: "move", "reparent", "copy".

### Relation label

A standard edge `label` used by convention: `part-of` (structural), `relates-to` (associative), `filed-to` (lifecycle). Never enforced as an enum.

### Dormant node

A node with color `#6c757d` indicating archived or completed work.

### Graph memory

A bounded traversal from Home (depth 2, 60-node cap) that digests the workspace graph for the AI assistant. Read-only; writes remain confirmed operations.

### Inert marker

A harmless Markdown or HTML comment that lets Balaur recognize special behavior while remaining readable in other editors. Markers do not create custom Canvas types or a second source of truth. Examples: `<!-- orbit:inbox -->`, `<!-- orbit:reference -->`, `<!-- orbit:ai-card -->`. Legacy `<!-- orbit:jd -->` markers from pre-graph vaults remain harmless inert text.

### AI operator

A standard `file` node referencing a `notes/*.md` file whose body carries the inert `<!-- orbit:ai-card -->` marker; incoming edges define its context, and its output is likewise a file-backed note connected by the reserved `AI output` edge. It proposes allowlisted structured operations; it does not execute generated host-page code or directly mutate the host DOM.
_Avoid_: "text node", "AI note".

### Text node (interop)

A standard JSON Canvas `text` node that Balaur renders read-only for imported or external documents but never authors; its own authoring produces only `file`-backed notes (ADR-0004 guardrail). The validator still accepts `text` for JSON Canvas 1.0 interop, which is a render path, not an authoring license.
_Avoid_: "authored text node", "editable text node", "note".

## Portability and recovery

### Single-canvas export

One valid `.canvas` document. It may reference canonical files not included in that export, so it is portable spatial content but not necessarily a complete backup.

### Whole-space backup

A validated version-2 `.orbit.json` bundle containing the metadata-only sidecar and raw logical vault files. It is the complete life-data backup and never includes a database snapshot.

### Repair placeholder

A read-only in-memory representation used when a referenced canvas file is missing or invalid. It retains raw content and must never be silently saved as an empty document.
