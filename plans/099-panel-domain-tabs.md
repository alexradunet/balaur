# Plan 099 — Domain tabs inside the panel; simplify the left rail to top-level domains

- **Status:** PLANNED (expand to step detail once plan **098** lands — this repo
  authors program plans against the real tree their foundation produces)
- **Priority:** P1
- **Effort:** M
- **Risk:** MED
- **Planned against commit:** `a1955f8` (intent), to be re-anchored post-098
- **Depends on:** **098** (the right panel + `chat.Panel` + the chip/restore
  contract must exist first)

## Intent

This is the "**apply the same treatment to the other sidebar entries**" half of
the canvas pivot. Today the left rail over-lists: **Knowledge** explodes into six
sub-entries (Awaiting / Facts / Preferences / People / Projects / Context) and
**Settings** into four (Profile / Appearance / Models / Heads). Each is its own
`@get('/ui/show/…')`. With a single-active panel, those sub-views belong as
**tabs inside the panel**, not as eleven separate rail rows.

Target after 099:

- The rail collapses to **top-level domains**: Quests, Life, **Knowledge**,
  Skills, **Settings** (Awaiting may stay as a top-level Knowledge entry if the
  owner wants the approval queue one click away — decide at expand time).
- Opening **Knowledge** or **Settings** renders the panel with an in-panel
  **tab strip** (`ui.Tabs`, which already exists — `internal/ui/tabs.go`) for its
  sub-views. Switching a tab `@get`s the panel body for that sub-view
  (`/ui/show/memory?category=…`, `/ui/show/settings?section=…`) and patches
  `#panel-body` (inner) — the panel head + tab strip stay; only the body swaps.
  This is the established "switch a sub-view inside one container" pattern
  (`web/templates/journal-focus.html` does class-toggle + `@get` of an inner
  region; here it is `ui.Tabs` chrome + an inner-mode patch).

## Why it waits for 098

098 introduces `chat.Panel`, the `#panel-body` swap target, and the
chip/restore contract. In-panel tabs are a refinement of that panel: the tab
strip lives in the panel head, and tab clicks patch `#panel-body`. None of that
is expressible until the panel exists.

## Known design decisions to settle at expand time

- **Reverses plans 092 & 095 partially.** 092 made Settings "per-section,
  nav-free artifacts (no in-artifact tabs)"; 095 made Knowledge "nav-free
  per-slice; categories become a sidebar group." 099 brings *in-panel* tabs
  back. That is intended (the panel is a different surface than the old inline
  artifact), but the plan must say so explicitly and update the same storybook
  Don'ts (`stories_navigation.go`) that 092/095 wrote — currently those Don'ts
  forbid in-artifact category tabs.
- **Reload/restore with tabs:** `panel_active` (098) is a `/ui/show` URL that
  already carries `?category=` / `?section=`, so restore lands on the right tab
  for free — confirm the tab strip marks the active tab from the URL.
- Whether the tab strip is part of `chat.Panel` (a `Tabs g.Node` slot) or a
  per-card concern (the memory/settings Focus body renders its own strip). Lean
  toward a `chat.Panel` slot so the head owns the tabs and only the body swaps.

## Out of scope (still)

- The agent live path (done in 098), mobile a11y parity (plan 100), switcher
  placement (plan 100).
