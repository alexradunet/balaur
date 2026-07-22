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

`.orbit/workspace.json`. It owns canvas paths and titles, hierarchy and portals, active canvas, cameras, and Johnny Decimal metadata. It never embeds canvas documents.

### Entity

A life-management record whose identity and fields live in one canonical Markdown file. Tasks, habits, habit check-ins, journals, and calendar events are entities. An entity is not a canvas node.

### Placement

A standard JSON Canvas `file` node that references a canonical entity or component-card file. Its node ID identifies that spatial occurrence only. One entity may have zero, one, or many placements.

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

### Johnny Decimal projection

The validated area/category/item view over the nested-canvas hierarchy. It does not replace the hierarchy or define a proprietary node type.

### Inert marker

A harmless Markdown or HTML comment that lets Balaur recognize special behavior while remaining readable in other editors. Markers do not create custom Canvas types or a second source of truth.

### AI operator

A standard text node marked as an AI card whose incoming edges define context. It proposes allowlisted structured operations; it does not execute generated host-page code or directly mutate the host DOM.

## Portability and recovery

### Single-canvas export

One valid `.canvas` document. It may reference canonical files not included in that export, so it is portable spatial content but not necessarily a complete backup.

### Whole-space backup

A validated version-2 `.orbit.json` bundle containing the metadata-only sidecar and raw logical vault files. It is the complete life-data backup and never includes a database snapshot.

### Repair placeholder

A read-only in-memory representation used when a referenced canvas file is missing or invalid. It retains raw content and must never be silently saved as an empty document.
