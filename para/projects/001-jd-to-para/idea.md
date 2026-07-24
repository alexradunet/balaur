---
phase: grill
status: done
project: 001-jd-to-para
date: 2026-07-24
---

> Grill complete. See `grill-2026-07-24.md` for the settled model and downstream summary. Next: plan.

# Idea: Replace the Johnny Decimal subsystem with PARA

## Scope (confirmed by user)
Option 1 — full subsystem replacement, not just starter content.

Remove the JD numeric model (codes, validation, sorting, `<!-- orbit:jd -->`
markers, the JD dialog, `goToJD`/`createJDEntry`, and the sidecar
`johnnyDecimal.entries` index). Replace with a PARA model (Projects / Areas /
Resources / Archive) built on the existing nested-canvas + file-node engine.

Includes: a new ADR (0003), sidecar schema change, and doc updates
(AGENTS.md §9, docs/architecture.md, docs/life-data.md, README.md, CONTEXT.md).

## Why
PARA is the organizing method the operator already uses (see orchestration
playbook). JD's three-level numeric scheme is heavier than the app needs and
forces a validation/dialog layer that PARA's four flat buckets do not require.

## Constraints carried in
- JSON Canvas must stay standards-compliant (no custom node types/fields).
- Canonical-files-first; sidecar owns only hierarchy/cameras/app metadata.
- Pre-1.0 experimental app ("open-canvas-experiment"): migration burden for
  existing JD vaults is a live question, not a given.
