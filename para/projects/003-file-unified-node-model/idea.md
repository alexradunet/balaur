---
phase: idea
status: grilled
project: 003-file-unified-node-model
date: 2026-07-24
blocks: 002-daily-canvas-journal
---

# Idea: File-unified node model (everything is a file; canvas is the connecting layer)

Raised while grilling 002 (daily canvas journal). The user proposed collapsing
the app's two content-node models into one: **every content node is a standard
`file` node referencing a canonical `.md` / `.html` / `.canvas`; the canvas is
purely the spatial + edge connecting layer.** This is the foundation the daily
canvas (002) is built on, so it is sequenced *before* 002.

## Why now
- The current design has two content models: inline `text` nodes (content lives
  in the canvas JSON; no id, no placements) and `file` nodes (canonical files
  with placements/repositories). Notes are the only inline holdouts.
- The daily canvas needs a **draining queue**: capture notes + tasks into today's
  canvas, then physically re-place them out to project/Wiki/Archive canvases.
  Inline text notes cannot drain (no file to re-place). Unifying the model makes
  a note drain exactly like a task.
- One mental model, file-canonical (ADR-0001), still 100% JSON Canvas 1.0
  compliant (`file` is a standard node type).

## Scope
- Retire inline `text` content authoring; notes/inbox/reference/goal/AI cards
  become `.md` files. `group` (layout) and `link` (external URL) stay.
- Notes are path-identified (`notes/<slug>.md`), no mandatory balaur-id.
- Hard-cut migration (early development): regenerate the starter, no migration
  of existing inline text nodes. `text` stays a valid *rendered* type for
  imported/external canvases (JSON Canvas interop).
- Lands as **ADR-0004** (reverses AGENTS.md §4.2 "notes are text nodes").

See `grill-2026-07-24.md` for the settled model and downstream summary.
