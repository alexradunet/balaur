# Plan 146: Models page UX/UI polish — style the unstyled, sharpen affordances

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat d8a8b66..HEAD -- internal/feature/modelcards/ internal/web/assets/static/basm.css internal/feature/storybook/`
> If any in-scope file changed since this plan was written, compare the "Current
> state" excerpts against the live code before proceeding; on a mismatch, treat
> it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW (CSS + small markup; no behavior change)
- **Depends on**: none (independent of 144/145, but if both touch basm.css,
  land 145 first to avoid a merge conflict in that file — see Maintenance notes)
- **Category**: dx / direction (UX)
- **Planned at**: commit `d8a8b66`, 2026-06-22

## Why this matters

The Models page renders several CSS classes that **have no stylesheet rules at
all**, so parts of it are visually broken today:

- The **download progress meter** (`.model-dl-progress`, `.model-dl-bar`,
  `.model-dl-label`) renders as an unstyled `<div>` with an inline `width:NN%` —
  there is no bar, track, or layout, so a model download shows no real progress
  bar.
- The **runtime variant rows** (`.runtime-row`, `.runtime-row-info`,
  `.runtime-row-action`) are unstyled divs — CPU and Vulkan rows have no
  alignment, rhythm, or separation between the info column and the action.

These are confirmed gaps (verified: `grep -c '\.runtime-row' basm.css` → 0,
same for the three `.model-dl-*` classes). Fixing them is the highest-confidence
UX win on the page. This plan also makes two small, high-value affordance
improvements: a clearer **selected-processor pill** and a clearer **empty
state**. Scope is deliberately tight — style what's unstyled and sharpen what's
weak; no restructuring.

## Current state

### Download meter markup (consumes the missing classes)
`internal/feature/modelcards/modelcard.go:151-161` (inside `modelAction`, the
`StatusDownloading` case):
```go
return g.Group([]g.Node{
    h.Div(h.Class("model-dl-progress"),
        h.ID("model-dl-progress"),
        h.Div(h.Class("model-dl-bar"),
            g.Attr("style", fmt.Sprintf("width:%d%%", v.Progress)),
        ),
        h.P(h.Class("model-dl-label"), g.Text(v.ProgressLabel)),
    ),
    cancelForm(),
})
```
`v.Progress` is `0..100`; `v.ProgressLabel` is a human line like
`"1.2 GB / 5.3 GB · 4.2 MB/s"` (`modelcard.go:111-112`).

### Runtime row markup (consumes the missing classes)
`internal/feature/modelcards/modelcard.go:92-99` (end of `RuntimeCard`):
```go
return h.Div(h.Class("runtime-row"), h.ID("runtime-row-"+v.Processor),
    h.Div(h.Class("runtime-row-info"),
        h.Strong(g.Text(label)),
        h.Span(h.Class("model-detail-line"), g.Text(detail)),
        hostNote,
    ),
    h.Div(h.Class("runtime-row-action"), action),
)
```
The info column holds a bold label, a status `.model-detail-line`, and an
optional host-loader note; the action column holds an Install button / "Not
supported" / an "installed" tag.

### Processor pills (affordance to sharpen)
`internal/feature/modelcards/panel.go:134-156` — `processorPill`. Selected pill
gets class `proc-pill-active` (which exists in basm.css, 1 rule) plus
`aria-current="true"` and `disabled`. There is no visual check/lock glyph; the
selected state is a subtle border/background change only.

### Empty state (weak wayfinding)
`internal/feature/modelcards/panel.go:81-85`:
```go
kids = append(kids, ui.EmptyState(ui.EmptyProps{
    Title: "No local models yet",
    Line:  "Download the official model below to run it in-process.",
}))
```
`ui.EmptyProps` supports more fields — confirm the available fields by reading
`internal/ui/emptystate.go` before using any beyond `Title`/`Line`.

### Stylesheet + tokens
`internal/web/assets/static/basm.css`. Existing related classes to match in
tone/spacing: `.model-card` / `.model-card-active` / `.model-card-cloud`
(~line 2009+), `.model-detail-line` (~2054), `.proc-pill` / `.proc-pill-active`
(~2030/2042), `.k-section` (~999), `.k-grid` (~1030). Design tokens available
(CSS custom properties): spacing `--space-1..--space-7`; colors `--bg`,
`--surface`, `--surface-2/3`, `--ink`, `--ink-muted`, `--gold`, `--gold-deep`,
`--gold-ink`, `--teal`, `--teal-deep`, `--parch-edge`, `--ember`, `--good`.
**Use these tokens — do not introduce raw hex values.**

### Storybook (so changes are visible + regression-checked)
`internal/feature/storybook/stories_settings.go` already has
`runtimesectionStory()` (~line 193) and `modelspanelStory()` (~63) with a
downloading variant (`Detail: "Downloading…"`, around line 109). The render-all
`story_test.go` must stay green.

## Commands you will need

| Purpose   | Command                                              | Expected on success |
|-----------|------------------------------------------------------|---------------------|
| Format    | `gofmt -l internal/feature/`                         | no output           |
| Vet       | `go vet ./internal/feature/...`                      | exit 0              |
| Tests     | `go test ./internal/feature/...`                     | `ok`, all pass      |
| Build     | `CGO_ENABLED=0 go build ./...`                       | exit 0              |
| CSS check | `grep -c '\.runtime-row' internal/web/assets/static/basm.css` | `> 0` after Step 1 |

## Suggested executor toolkit

- Invoke the `ui-development` skill (if available) — it documents the Hearthwood
  tokens, the layout-token scale, and the storybook workflow this plan relies on.

## Scope

**In scope**:
- `internal/web/assets/static/basm.css` (modify — add the missing rules +
  affordance polish; additive, no deletions of existing rules)
- `internal/feature/modelcards/panel.go` (modify — ONLY the small empty-state
  wayfinding tweak in Step 4 and, if needed, a check glyph span in
  `processorPill` in Step 3)
- `internal/feature/modelcards/modelcard.go` (modify — ONLY if Step 2/3 needs a
  tiny markup hook like a `<span class="model-dl-track">` wrapper; prefer
  pure-CSS first)
- `internal/feature/storybook/stories_settings.go` (modify — ensure a
  downloading-state and runtime-section variant exist so the new styles are
  visible; add a variant if missing)

**Out of scope** (do NOT touch):
- Any handler / route / Datastar wiring (`internal/web/*.go`) — this is visual
  only; no endpoint or signal changes.
- `internal/store/*`, `internal/turn/*`, `internal/kronk/*` — no logic changes.
- The cloud preset work (plans 144/145).
- Renaming or removing any existing CSS class — additive changes only.

## Git workflow

- Branch: `advisor/146-models-page-ux-polish`
- Commit per logical unit (download meter, runtime rows, pill, empty state) or
  one squashed commit; conventional subject, e.g.
  `style(ui/models): style download meter + runtime rows, sharpen affordances (plan 146)`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Style the runtime variant rows

Add rules to `basm.css` for `.runtime-row` (a flex row: info column grows,
action column hugs right, vertical centering, `gap:var(--space-3)`, padding via
`var(--space-2/3)`, a subtle `--parch-edge` bottom border between stacked rows),
`.runtime-row-info` (column flex, `gap:var(--space-1)`), and
`.runtime-row-action` (shrink-0, right-aligned). Match the parchment look of the
surrounding `.k-section`. Ensure it reflows to stacked on narrow widths
(`@media` max-width consistent with the repo's existing breakpoint tokens — find
an existing `@media` in basm.css and reuse its breakpoint value).

**Verify**: `grep -c '\.runtime-row' internal/web/assets/static/basm.css` → `>= 3`.

### Step 2: Style the download progress meter

Add rules for `.model-dl-progress` (container: column, `gap:var(--space-1)`),
`.model-dl-bar` (the fill: height ~`var(--space-1)` or 6–8px, `--gold` /
`--gold-deep` fill, rounded, `transition: width .2s ease`; its width comes from
the inline `style="width:NN%"` already emitted — do NOT hardcode width in CSS),
and `.model-dl-label` (small muted text, `--ink-muted`, matching
`.model-detail-line` sizing).

The fill needs a **track** behind it for the bar to read as progress. Prefer
pure CSS: give `.model-dl-progress` a track appearance by adding a
`background: var(--surface-2)` rounded strip on a wrapper. If the current markup
(`.model-dl-progress` directly containing `.model-dl-bar` + `.model-dl-label`)
makes a clean track impossible, add a minimal wrapper
`<div class="model-dl-track">` around just the `.model-dl-bar` in
`modelcard.go:155` and style `.model-dl-track` as the track. Keep the inline
width on `.model-dl-bar` untouched.

**Verify**: `grep -c '\.model-dl-bar' internal/web/assets/static/basm.css` → `>= 1`;
`go build ./internal/feature/modelcards/...` → exit 0.

### Step 3: Sharpen the selected-processor pill

Make the active pill unmistakable. Two options — prefer CSS-only:
- CSS-only: strengthen `.proc-pill-active` (e.g. a `--gold` left border or a
  `::before` "✓ " glyph via `content`) so the selected processor is obvious at a
  glance, distinct from hover.
- If a glyph via `::before content` doesn't render acceptably, add a small
  `<span class="proc-pill-check" aria-hidden="true">✓</span>` inside the selected
  pill button in `processorPill` (`panel.go:151-154`, only when `p.Selected`).

Do not change the pill's form/post behavior or its `aria-current`/`disabled`
attributes.

**Verify**: `go build ./internal/feature/modelcards/...` → exit 0;
`go test ./internal/feature/...` → `ok`.

### Step 4: Sharpen the empty state wayfinding

In `panel.go:81-85`, improve the copy so it points the owner at the "Get a
model" cards below, e.g. `Line: "Download a curated model from the “Get a model”
section below to run one in-process — no account needed."` If `ui.EmptyProps`
exposes a crest/icon or action field (confirm by reading
`internal/ui/emptystate.go`), set the crest for visual anchoring; otherwise
leave structure as-is and change only the copy. Keep it a single
`ui.EmptyState` call — do not hand-roll markup.

**Verify**: `go build ./internal/feature/modelcards/...` → exit 0.

### Step 5: Ensure storybook shows the styled states

Confirm `stories_settings.go` has a variant exercising the **downloading** model
state (so the new meter renders) and the **runtime section**. The downloading
variant exists in `modelspanelStory` (~line 109) — verify it sets a `Progress`
value (e.g. add `Progress: 42, ProgressLabel: "2.2 GB / 5.3 GB · 4.1 MB/s"` to
that fixture `ModelView` if absent, so the bar shows a partial fill).
`runtimesectionStory` already renders runtime rows. Add a `Dos`/`Donts` note if
helpful. No new story registration needed.

**Verify**: `go test ./internal/feature/storybook/...` → `ok`.

### Step 6: Full gates

**Verify**:
- `gofmt -l internal/feature/` → no output
- `go vet ./internal/feature/...` → exit 0
- `go test ./...` → all packages `ok`
- `CGO_ENABLED=0 go build ./...` → exit 0
- `git diff --check` → no whitespace errors

## Test plan

- No new Go logic, so no new unit tests are required. The guardrail is
  `internal/feature/storybook/story_test.go`, which renders every story — it must
  stay green, proving the modified components still produce valid markup.
- Manual visual check (describe in your report, do not block on it): run the app
  / storybook, open the Models story group, confirm (a) the download bar shows a
  partial gold fill on a track, (b) runtime rows align info-left / action-right,
  (c) the selected processor pill is obviously selected.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -c '\.runtime-row' internal/web/assets/static/basm.css` → `>= 3`.
- [ ] `grep -c '\.model-dl-bar' internal/web/assets/static/basm.css` → `>= 1`.
- [ ] `gofmt -l internal/feature/` prints nothing.
- [ ] `go vet ./internal/feature/...` exits 0.
- [ ] `go test ./...` exits 0 (storybook render test green).
- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `git diff --check` reports no errors.
- [ ] No handler/route/store files modified (`git status` shows only the
      in-scope files).
- [ ] `plans/readme.md` status row for plan 146 updated.

## STOP conditions

Stop and report back (do not improvise) if:

- The markup excerpts in "Current state" don't match the live
  `modelcard.go`/`panel.go` (drift — another change already restyled these).
- The classes are in fact already styled (`grep` finds rules) — the gap was
  closed independently; report and skip the redundant steps.
- Closing the progress-track cleanly seems to require restructuring the meter
  markup beyond a single wrapper `<div>` — report your proposed markup change
  and wait rather than reshaping the component.
- Any step pushes you toward editing a handler, route, or store file.

## Maintenance notes

- These styles target the existing class names emitted by `modelcards`. If those
  components are restructured later, keep the class hooks or move the rules with
  them.
- **Merge ordering with plan 145**: both this plan and 145 append to
  `basm.css`. If 145 is also being executed, land 145 first, then rebase this
  plan's CSS additions on top — the additions are independent blocks, so a
  conflict is mechanical. The advisor recommends executing 145 before 146.
- A reviewer should check: only additive CSS (no existing rule deleted/renamed),
  tokens used instead of raw hex, and the inline `width:NN%` on `.model-dl-bar`
  left intact (the bar fill is data-driven, not CSS-driven).
- Deferred out of this plan (do not pull in): toast notifications, per-card
  inline errors, VRAM comparison view, restart-pending banner in the chatbar,
  model search/filter. Those are larger UX bets for a separate roadmap plan.
</content>
