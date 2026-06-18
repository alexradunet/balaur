# Plan 104 — Stop the chat composer being clipped (reset the leaked `top: 62px` on the app-shell dock) + chat-area spacing polish

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in "STOP conditions" occurs, stop and report — do not improvise.
> When done, update this plan's row in `plans/readme.md` (unless a reviewer
> dispatched you and said they maintain the index).
>
> **Drift check (run first)**:
> ```
> git rev-parse --short HEAD
> git diff --stat 1ba3b62..HEAD -- internal/web/assets/static/basm.css internal/ui/shell/chatshell.go
> ```
> If `basm.css` changed since `1ba3b62`, compare the "Current state" excerpts
> below against the live file before editing. On any mismatch in the two rules
> this plan edits, treat it as a STOP condition.

## Status

- **Priority**: P1 (visible breakage: the primary input is partially off-screen)
- **Effort**: S (CSS-only; one file edited + one test added)
- **Risk**: LOW (the edited rule is scoped to `html.app` only)
- **Depends on**: none (the `html.app` shell already exists on `main`)
- **Category**: bug
- **Planned at**: commit `1ba3b62`, 2026-06-18

## Why this matters

On the single-page chat shell (`html.app`, the live product surface), the
bottom of the composer — its footer row with the hint and the **Send** button —
is cut off below the window edge. The composer is the owner's "single seat of
action" (`internal/ui/composer.go:17`); having Send clipped is a functional
break, not just cosmetic.

**Root cause (confirmed empirically — see "Evidence" below).** The base dock
rule positions the dock as a fixed right-rail that clears a 62px topbar:

```css
/* internal/web/assets/static/basm.css:2309 */
#dock {
  position: fixed;
  top: 62px;                 /* below the sticky 62px topbar */
  right: 0;
  bottom: 0;
  ...
}
```

The single-page chat shell re-purposes that same `#dock` element as a full-height
grid column and switches it to `position: relative` — **but never resets `top`**:

```css
/* internal/web/assets/static/basm.css:3373 */
html.app #dock.app-dock {
  position: relative;
  left: 0;
  width: auto;
  height: 100%;            /* = 100dvh, the grid cell height */
  z-index: var(--z-base);
  border-left: 0;
  box-shadow: none;
  padding-inline: var(--pad);
}
```

Under `position: relative`, the inherited `top: 62px` is no longer a sizing
constraint — it becomes a **downward visual offset**. So the dock box is
`100dvh` tall *and* shoved down 62px, and its parent clips the overflow:

```css
/* internal/web/assets/static/basm.css:3330 */
html.app .app-shell { display: grid; grid-template-columns: 1fr var(--w-panel); height: 100dvh; overflow: hidden; }
```

The dock runs from `y=62` to `y=62+100dvh`, so its bottom 62px — which holds the
pinned composer's footer — is pushed past the viewport and clipped by
`.app-shell { overflow: hidden }`.

`ChatShell` has **no topbar** (`internal/ui/shell/chatshell.go:22` —
*"Unlike Page, there is no topbar"*), so the 62px is pure leaked cruft. The fix
is to reset it to `0` on the app-shell dock. The clipped composer becomes fully
visible with a small inset below it, and the chat above keeps scrolling.

This plan also makes one small, requested spacing tweak so the last chat line
clears the composer ledge comfortably (Step 2).

## Evidence (measured on the running app, commit `1ba3b62`, viewport 1440×681)

Measured with the browser dev console (`getBoundingClientRect` + `getComputedStyle`):

| Element | Before fix | After `top:0` | Note |
|---|---|---|---|
| viewport height | 681 | 681 | `.app-shell` clips at this height |
| `#dock.app-dock` | `position:relative`, `top:62px`, box **62→743** | box **0→681** | leaked `top` shoves it down 62px |
| `.composer` bottom | **733** (52px **below** viewport → clipped) | **671** (10px **above** edge → fully visible) | the Send/footer row was the clipped part |
| `.chat` overflow | `overflow-y:auto`, scrollable | unchanged, still scrollable | the fix does not affect scrolling |

Setting `top:0` (and nothing else) on `html.app #dock.app-dock` made the entire
composer visible and left the chat scroll behaviour intact. That single property
is the whole fix.

## Current state — the exact lines you will edit

Two rules in `internal/web/assets/static/basm.css`. Open the file and confirm
both match before editing.

**(A) The dock rule with the leak — line 3373:**

```css
html.app #dock.app-dock {
  position: relative;
  left: 0;
  width: auto;
  height: 100%;
  z-index: var(--z-base);
  border-left: 0;
  box-shadow: none;
  padding-inline: var(--pad);
}
```

**(B) The chat scroll area's padding (Step 2 polish) — line 3395:**

```css
html.app #dock.app-dock .chat { --portrait-size: 56px; --chat-gutter: 84px; padding: 20px 0 var(--space-2); }
```

Spacing tokens for reference (`:root`, lines 173–178):
`--space-2: 8px; --space-3: 12px; --space-4: 16px; --space-5: 24px;`.
Note `var(--space-2)` (8px) is the current chat bottom padding.

The repo already string-asserts `basm.css` rules in tests — see
`internal/web/assets/css_tokens_test.go` (read it; Step 3 follows its exact
pattern: `FS.ReadFile("static/basm.css")` + `strings.Contains`).

## Commands you will need

| Purpose | Command | Expected on success |
|---|---|---|
| Format check | `gofmt -l internal/` | prints nothing |
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Tests | `CGO_ENABLED=0 go test ./... -count=1` | exit 0, no FAIL |
| Whitespace | `git diff --check` | prints nothing |
| Fix present | `grep -A1 'html.app #dock.app-dock {' internal/web/assets/static/basm.css` | shows `top: 0;` |

## Scope

**In scope (the only files you may modify):**
- `internal/web/assets/static/basm.css` — the two rules above (Steps 1 & 2)
- `internal/web/assets/css_tokens_test.go` — add one regression test (Step 3)

**Out of scope (do NOT touch, even though they look related):**
- The base `#dock { ... top: 62px ... }` rule (basm.css:2309). It is **shared**
  with the right-rail and `.dock-full` layouts that genuinely sit below a 62px
  topbar. Removing `top: 62px` there would break those. Reset it per-context
  (this plan's Step 1), never at the base.
- `html.home #dock` (basm.css:3123) — that dock is `position: fixed` with
  `top:62px`/`bottom:0`, where `top:62px` correctly *sizes* a fixed box. It is
  not broken; do not change it.
- `internal/ui/shell/chatshell.go`, `internal/ui/chat/dock.go`,
  `internal/ui/composer.go` — no Go/markup change is needed; the bug is purely
  the CSS cascade.
- The composer's internal density/height (it is ~208px tall) — a separate design
  concern, explicitly deferred (see Maintenance notes).
- `#dock .composer { margin: ... 10px; }` (basm.css:3158) — shared across rail /
  home / app docks; the measured 10px bottom inset is already adequate. Leave it.

## Git workflow

- Branch: `improve/104-app-dock-composer-clip` (match the repo's `improve/NNN-slug`
  convention seen throughout `plans/readme.md`).
- One commit is fine (small CSS + test). Message style: conventional commits,
  e.g. `fix(web): app-shell dock — reset leaked top:62px so composer isn't clipped`.
- Do NOT push or open a PR unless the operator asks.

## Steps

### Step 1 — Reset the leaked `top` on the app-shell dock (the fix)

In `internal/web/assets/static/basm.css`, in the `html.app #dock.app-dock { ... }`
rule (line 3373), add `top: 0;` immediately after `position: relative;`. Result:

```css
html.app #dock.app-dock {
  position: relative;
  top: 0;                  /* reset the base #dock top:62px (no topbar here) — else the composer is clipped off the bottom */
  left: 0;
  width: auto;
  height: 100%;
  z-index: var(--z-base);
  border-left: 0;
  box-shadow: none;
  padding-inline: var(--pad);
}
```

Do not change any other property in this block.

**Verify**: `grep -A1 'html.app #dock.app-dock {' internal/web/assets/static/basm.css`
→ the line after the selector is `  top: 0;`.

### Step 2 — Give the chat a little more clearance above the composer ledge (requested polish)

In the same file, change the chat scroll area's **bottom** padding (line 3395)
from `var(--space-2)` (8px) to `var(--space-4)` (16px). Only the third padding
value changes:

```css
html.app #dock.app-dock .chat { --portrait-size: 56px; --chat-gutter: 84px; padding: 20px 0 var(--space-4); }
```

Rationale: combined with the composer's 8px top margin this gives ~24px between
the last message and the ledge — matching the 24px (`--space-5`) inter-message
gap, so the conversation no longer feels jammed under the input when scrolled to
the bottom. This rule is `html.app`-scoped, so it does not affect the home/rail
docks. Leave the `20px 0` top/side values unchanged.

**Verify**: `grep 'html.app #dock.app-dock .chat {' internal/web/assets/static/basm.css`
→ the line ends `padding: 20px 0 var(--space-4); }`.

### Step 3 — Add a regression test for the leak

Add a test to `internal/web/assets/css_tokens_test.go` that fails if the
`html.app #dock.app-dock` rule ever loses its `top` reset again. Follow the
existing `FS.ReadFile` + `strings` pattern in that file:

```go
// TestAppDockResetsTop guards plan 104: the single-page chat shell re-uses the
// base #dock element (which is position:fixed; top:62px to clear a topbar) as a
// position:relative grid column. Under relative positioning that inherited
// top:62px becomes a downward offset that shoves the composer's footer past the
// clipped viewport. The html.app dock MUST reset it, or the Send button is cut off.
func TestAppDockResetsTop(t *testing.T) {
	b, err := FS.ReadFile("static/basm.css")
	if err != nil {
		t.Fatalf("read basm.css: %v", err)
	}
	css := string(b)

	const sel = "html.app #dock.app-dock {"
	i := strings.Index(css, sel)
	if i < 0 {
		t.Fatalf("rule %q not found — the app-shell dock was renamed; re-check plan 104", sel)
	}
	end := strings.Index(css[i:], "}")
	if end < 0 {
		t.Fatalf("unterminated rule for %q", sel)
	}
	block := css[i : i+end]
	if !strings.Contains(block, "top: 0") {
		t.Errorf("html.app #dock.app-dock must reset top (e.g. `top: 0`) so the leaked base #dock top:62px does not clip the composer; block was:\n%s", block)
	}
}
```

`strings` is already imported in this file (the existing tests use it), so no
import change is needed.

**Verify**: `CGO_ENABLED=0 go test ./internal/web/assets/ -run TestAppDockResetsTop -count=1`
→ `ok`, the test passes. Then sanity-check it actually guards the fix: it must
PASS now (Step 1 done) and would FAIL without `top: 0`.

### Step 4 — Full gates + manual browser check

Run all gates:

```
gofmt -l internal/                          # prints nothing
CGO_ENABLED=0 go build ./...                # exit 0
go vet ./...                                # exit 0
CGO_ENABLED=0 go test ./... -count=1        # exit 0, no FAIL
git diff --check                            # prints nothing
```

**Manual browser check (the bug is visual; confirm it in a browser):**
Build/run the app (`make run`, or it may already be running on `:8090`), open
`http://localhost:8090/`, and in the dev console run:

```js
const c = document.querySelector('.composer');
const d = document.querySelector('#dock.app-dock');
({
  dockTop: getComputedStyle(d).top,                                   // expect "0px"
  composerFullyVisible: c.getBoundingClientRect().bottom <= window.innerHeight, // expect true
  chatScrolls: getComputedStyle(document.querySelector('.chat')).overflowY,     // expect "auto"
})
```

Expected: `dockTop: "0px"`, `composerFullyVisible: true`, `chatScrolls: "auto"`.
Also eyeball it: the composer's footer row (hint + **Send** button) is visible
with a small gap below it, and the message history scrolls above the composer.

If a browser is not available in your environment, say so and rely on the
`grep` + unit-test gates; do not claim the visual was verified.

## Test plan

- **New unit test** `TestAppDockResetsTop` in
  `internal/web/assets/css_tokens_test.go` — pins the `top` reset on the
  app-shell dock so the leak cannot silently return. Modeled on the existing
  `TestThemePaletteBlocks` / `TestNoUndefinedHearthwoodTokens` in the same file.
- **No other Go tests change.** This is a CSS-asset edit; the gomponents/template
  tests (`internal/web/home_test.go`, `internal/ui/shell/shell_test.go`) assert
  rendered HTML structure, which is untouched — confirm they still pass via the
  full `go test ./...` run.
- **Verification**: `CGO_ENABLED=0 go test ./... -count=1` → all pass, including
  the one new test.

## Done criteria (ALL must hold)

- [ ] `grep -A1 'html.app #dock.app-dock {' internal/web/assets/static/basm.css` shows `  top: 0;`
- [ ] `grep 'html.app #dock.app-dock .chat {' internal/web/assets/static/basm.css` ends with `padding: 20px 0 var(--space-4); }`
- [ ] `gofmt -l internal/` prints nothing
- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=0 go test ./... -count=1` exits 0; `TestAppDockResetsTop` exists and passes
- [ ] `git diff --check` prints nothing
- [ ] Only the two in-scope files are modified (`git status`)
- [ ] (If a browser is available) console check returns `dockTop:"0px"`, `composerFullyVisible:true`
- [ ] This plan's row in `plans/readme.md` is updated

## STOP conditions

Stop and report (do not improvise) if:

- The `html.app #dock.app-dock {` rule is missing, or no longer sets
  `position: relative` — the layout has been refactored and this plan's premise
  no longer holds.
- The base `#dock` rule (basm.css:2309) no longer contains `top: 62px` — someone
  may have already addressed the leak differently; re-evaluate whether the bug
  still reproduces before editing.
- After Step 1, the browser check shows the composer is *still* clipped
  (`composerFullyVisible: false`) — the 62px offset is not the (only) cause;
  report the new measurements rather than piling on more CSS.
- The fix appears to require editing any out-of-scope file (e.g. the base `#dock`
  rule, `chatshell.go`, or the shared `#dock .composer` margin).

## Maintenance notes

- **The leak pattern, for reviewers:** the base `#dock` is a `position: fixed`
  element designed to clear a 62px topbar; every layout that re-purposes `#dock`
  with a *different* `position` (the app shell uses `relative`) must explicitly
  reset `top`/`inset`, or the 62px re-appears as a stray offset. If a future
  layout re-uses `#dock` again, reset its `top` in that context too.
- **The composer pins via a flex chain** — `#dock-convo { flex:1 1 auto; min-height:0 }`
  (basm.css:3157) + `#dock .chat { overflow-y:auto; flex:1 1 auto; min-height:0 }`
  (basm.css:2332) + `#dock .composer { flex:0 0 auto }` (basm.css:3158). If the
  composer ever stops pinning to the bottom or the chat stops scrolling, check
  those three rules first.
- **Deferred (not this plan):** the composer is ~208px tall (~30% of an 681px
  viewport) because it stacks the tool-well row, soul portrait, a 2-row textarea,
  and the footer. Slimming it is a composer-density redesign, orthogonal to this
  clip fix; raise it separately if the owner wants a leaner input.
- This change also fixes the same clip on narrow viewports (≤720px), where the
  dock uses the same `html.app #dock.app-dock` rule — worth a quick eyeball at a
  phone width if you have a browser.
