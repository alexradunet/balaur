# Plan 105 — Arrow-key navigation + Enter-to-select in the composer `/`-command menu

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in "STOP conditions" occurs, stop and report — do not improvise.
> When done, update this plan's row in `plans/readme.md` (unless a reviewer
> dispatched you and said they maintain the index).
>
> **Drift check (run first)**:
> ```
> git rev-parse --short HEAD
> git diff --stat d0b44a5..HEAD -- internal/web/assets/static/basm.js internal/web/assets/static/basm.css internal/ui/command_palette.go internal/ui/composer.go internal/feature/storybook/stories_chat.go
> ```
> If any of those files changed since `d0b44a5`, compare the "Current state"
> excerpts below against the live code before editing. On a mismatch in any
> block you are told to edit, treat it as a STOP condition.

## Status

- **Priority**: P2 (UX polish on the primary navigation launcher; not a breakage)
- **Effort**: S (one JS file + one CSS rule + one guard test + one story-doc update)
- **Risk**: LOW (additive client behavior; the no-arrow path is preserved byte-for-byte)
- **Depends on**: none (plan 102 shipped the palette; this builds on it)
- **Category**: dx (interaction polish)
- **Planned at**: commit `d0b44a5`, 2026-06-18

## Why this matters

When the owner types `/` in the chat composer, a command menu (`.cmd-palette`)
appears and filters as they type — it is **the** navigation launcher that
replaced the domain rail (plan 102). Today the only way to pick a non-top item
is the mouse: pressing **Enter** always fires the *first visible* row, and the
arrow keys just move the textarea caret. The owner asked to navigate the menu
with **↑/↓** and act on the highlighted item with **Enter** — the universal
command-palette interaction (VS Code, Spotlight, Slack). This makes the
launcher fully keyboard-drivable without lifting hands from the keyboard.

The fix is small and self-contained: the menu already renders every row and
filters them via Datastar `data-show`; we add a client-side "active row"
highlight that ↑/↓ move, and we teach the existing Enter handler to prefer the
highlighted row (falling back to the first visible row, so current behavior is
unchanged when the owner never presses an arrow).

## Key design decision (read before coding)

**The active-row highlight is client-owned DOM state (`.cmd-item.is-active`
class), set imperatively in `basm.js` — NOT a Datastar signal.**

Why, so you don't "improve" this into a signal:

- Which rows are *visible* depends on the typed filter, evaluated in the
  browser by Datastar `data-show` (it toggles inline `display`). "The active
  row" is "the Nth **currently-visible** row" — a value only the browser knows
  after layout. A server-rendered prop or a `$cmdActive` signal cannot compute
  it, and expressing "Nth visible" in a `data-show`/`data-class` expression is
  far more code than a 6-line DOM helper.
- `basm.js` already owns exactly this kind of imperative, delegated keyboard
  behavior — see the digit-shortcut `keydown` listener (lines 106–119) and
  `balaurSubmitOnEnter` (lines 135–151). This plan matches that established
  pattern. **No new Datastar signal, no change to `command_palette.go`.**

## Current state — the exact code you will work with

### (A) The menu markup — `internal/ui/command_palette.go` (NOT edited; for context)

The palette is a `<div class="cmd-palette">` shown only while the draft starts
with `/`; each item is an `<a class="cmd-item">` shown only while its `Key`
prefix-matches what the owner typed. Excerpt (lines 26–53):

```go
func CommandPalette(items []CommandItem) g.Node {
	list := []g.Node{h.Class("cmd-list")}
	for _, it := range items {
		show := "$message.startsWith('/') && '" + it.Key +
			"'.startsWith($message.slice(1).toLowerCase().trim())"
		item := []g.Node{
			h.Class("cmd-item"),
			g.Attr("data-show", show),
			h.Href(it.URL), // no-JS fallback
			g.Attr("data-on:click__prevent", "@get('"+it.URL+"'); $message = ''"),
		}
		// ... icon, label, key span ...
		list = append(list, h.A(item...))
	}
	return h.Div(
		h.Class("cmd-palette"),
		g.Attr("data-show", "$message.startsWith('/')"),
		h.Div(list...),
	)
}
```

Two facts this plan relies on:
1. A hidden item gets inline `display: none` (Datastar `data-show`), so
   `el.offsetParent === null` reliably means "filtered out / menu closed".
2. Clicking an item fires its `data-on:click__prevent` (`@get(URL)` +
   clear draft). So **selecting = calling `.click()` on the row** — exactly
   what `balaurSubmitOnEnter` already does for the first visible row.

### (B) The Enter handler — `internal/web/assets/static/basm.js` lines 135–151

This is the function you will EDIT in Step 1:

```js
window.balaurSubmitOnEnter = function (event) {
  if (event.key !== 'Enter' || event.shiftKey || event.altKey ||
      event.ctrlKey || event.metaKey || event.isComposing) return;
  event.preventDefault();
  var ta = event.currentTarget;
  // Slash-command: pick the first visible palette item instead of sending.
  if (ta.value.trimStart().startsWith('/')) {
    var palette = ta.closest('.composer') &&
      ta.closest('.composer').querySelector('.cmd-palette');
    var first = palette && Array.prototype.find.call(
      palette.querySelectorAll('.cmd-item'),
      function (el) { return el.offsetParent !== null; }); // visible
    if (first) first.click();   // triggers the item's data-on:click @get
    return;                     // never post a "/foo" line to chat
  }
  ta.form && ta.form.requestSubmit();
};
```

It is wired via an inline attribute on the live textarea —
`internal/ui/composer.go:109`:
`g.Attr("onkeydown", "balaurSubmitOnEnter(event)")`. **You will NOT change
`composer.go`**: arrow handling is added as a separate document-level listener,
and Enter stays in this same function.

### (C) The existing imperative keydown pattern — `basm.js` lines 106–119 (for style reference)

```js
document.addEventListener('keydown', (e) => {
  if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.tagName === 'SELECT') return;
  if (e.metaKey || e.ctrlKey || e.altKey) return;
  // ... digit 1–9 clicks the matching .choice ...
});
```

(That one *ignores* textareas; your new arrow listener will instead *act* when
focus is in the composer textarea and the palette is open. They do not
conflict — different keys, different conditions.)

### (D) The palette CSS — `internal/web/assets/static/basm.css` lines 3504–3524

```css
.cmd-palette {
  position: absolute;
  left: var(--space-3); right: var(--space-3);
  bottom: calc(100% + var(--space-2));
  z-index: var(--z-overlay);
  max-height: 40vh; overflow-y: auto;            /* scrolls — active row must scrollIntoView */
  background-color: var(--surface-2); border: 2px solid var(--parch-edge);
  box-shadow: var(--drop-hard);
}
.cmd-list { display: flex; flex-direction: column; }
.cmd-item {
  display: flex; align-items: center; gap: var(--space-2);
  padding: var(--space-2) var(--space-3); text-decoration: none;
  color: var(--ink); font-family: var(--font-mono); font-size: 13px;
}
.cmd-item:hover { background: var(--surface-3, var(--surface)); color: var(--gold); }
.cmd-item-icon { width: 20px; height: 20px; image-rendering: pixelated; }
.cmd-item-label { flex: 1; }
.cmd-item-key { font-family: var(--font-pixel); font-size: 12px; color: var(--gold); opacity: .7; }
```

Tokens you may use (already defined in `:root`): `--gold-deep` (muted
theme-aware gold, used by the scrollbar styling), `--gold`, `--surface-3`
(with `--surface` fallback as above), `--space-2` (8px). All are theme-aware,
so one rule tracks every palette.

### (E) The storybook story — `internal/feature/storybook/stories_chat.go` `commandpaletteStory()` lines 277–313

You will edit only its `Blurb` and `Dos` (Step 4) to document the new keyboard
nav. The single existing `Variant` and the `Props` table stay as-is.

## Commands you will need

| Purpose | Command | Expected on success |
|---|---|---|
| Format check | `gofmt -l internal/` | prints nothing |
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Tests | `CGO_ENABLED=0 go test ./... -count=1` | exit 0, no FAIL |
| Whitespace | `git diff --check` | prints nothing |

There is **no JavaScript unit-test harness** in this repo (`basm.js` is plain
platform JS, loaded by the page; its behavior is exercised manually / in a
browser). So the JS in Steps 1–2 is gated by `go build` + `go vet` + the
browser check in Step 6, and the CSS in Step 3 is pinned by a Go string-assert
test (Step 5), matching the repo's convention in
`internal/web/assets/css_tokens_test.go`.

## Scope

**In scope (the only files you may modify):**
- `internal/web/assets/static/basm.js` — Steps 1 & 2 (edit `balaurSubmitOnEnter`; add a helper + two delegated listeners)
- `internal/web/assets/static/basm.css` — Step 3 (the `.cmd-item.is-active` highlight rule)
- `internal/web/assets/css_tokens_test.go` — Step 5 (one guard test)
- `internal/feature/storybook/stories_chat.go` — Step 4 (`commandpaletteStory` Blurb/Dos doc update)

**Out of scope (do NOT touch, even though they look related):**
- `internal/ui/command_palette.go` — no markup change. The highlight is
  client-computed (see "Key design decision"); adding an `ActiveKey`/`is-active`
  prop here would be dead at runtime (the server can't know the post-filter
  first-visible row). Do not add one.
- `internal/ui/composer.go` — keep the existing inline
  `onkeydown="balaurSubmitOnEnter(event)"`. Arrows are handled by a new
  document-level listener; the textarea needs no new attribute.
- The Datastar layer / any `data-signals:*` — do NOT introduce a `$cmdActive`
  (or similar) signal. The highlight is DOM state, by design.
- The digit-shortcut `keydown` listener (`basm.js` 106–119) and the
  choices-panel — unrelated feature.
- Escape-to-close the menu — deferred (clearing the open menu means clearing
  the Datastar-owned `$message`, which JS can't do cleanly here; see
  Maintenance notes). Do not attempt it in this plan.

## Git workflow

- Branch: `improve/105-cmd-palette-keyboard-nav` (match the repo's
  `improve/NNN-slug` convention used throughout `plans/readme.md`).
- One commit is fine. Message style: conventional commits, e.g.
  `feat(web): keyboard-navigate the composer /-command menu (↑/↓ + Enter)`.
- Do NOT push or open a PR unless the operator asks.

## Steps

### Step 1 — Add a visible-items helper and teach Enter to prefer the active row

In `internal/web/assets/static/basm.js`, **replace** the existing
`window.balaurSubmitOnEnter` function (lines 135–151, shown verbatim in
"Current state (B)") with the version below. It adds a small
`balaurCmdVisibleItems` helper and makes the slash branch pick the highlighted
row when one exists, else the first visible row (unchanged fallback):

```js
// Visible (not filtered-out, menu-open) command rows, top-to-bottom.
// A row hidden by Datastar data-show has inline display:none → offsetParent null.
function balaurCmdVisibleItems(palette) {
  if (!palette) return [];
  return Array.prototype.filter.call(
    palette.querySelectorAll('.cmd-item'),
    function (el) { return el.offsetParent !== null; });
}

window.balaurSubmitOnEnter = function (event) {
  if (event.key !== 'Enter' || event.shiftKey || event.altKey ||
      event.ctrlKey || event.metaKey || event.isComposing) return;
  event.preventDefault();
  var ta = event.currentTarget;
  // Slash-command: act on the highlighted palette row (or the first visible
  // one) instead of sending the "/foo" line to chat.
  if (ta.value.trimStart().startsWith('/')) {
    var palette = ta.closest('.composer') &&
      ta.closest('.composer').querySelector('.cmd-palette');
    var items = balaurCmdVisibleItems(palette);
    var target = items.filter(function (el) {
      return el.classList.contains('is-active');
    })[0] || items[0];
    if (target) target.click();   // triggers the row's data-on:click @get
    return;                       // never post a "/foo" line to chat
  }
  ta.form && ta.form.requestSubmit();
};
```

**Verify**: `grep -n "balaurCmdVisibleItems" internal/web/assets/static/basm.js`
→ at least 2 hits (the function definition + its use in `balaurSubmitOnEnter`).

### Step 2 — Add ↑/↓ navigation and a typed-filter highlight reset

Immediately **after** the `balaurSubmitOnEnter` function from Step 1, add the
two delegated listeners below. Both are document-level (matching the existing
`document.addEventListener(...)` pattern in this file) and cheap — they
early-return unless focus is in a composer textarea with an open palette.

```js
// ── Composer /-command menu: ↑/↓ navigate the active row ───────────────
// The active row is DOM state (.cmd-item.is-active), not a Datastar signal:
// "the active row" = "the Nth currently-visible row", a value only the browser
// knows after data-show filtering. Enter (balaurSubmitOnEnter) reads it.
document.addEventListener('keydown', function (e) {
  if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return;
  var ta = e.target;
  if (!ta || ta.tagName !== 'TEXTAREA' || !ta.closest('.composer')) return;
  var palette = ta.closest('.composer').querySelector('.cmd-palette');
  if (!palette || palette.offsetParent === null) return; // menu not open
  var items = balaurCmdVisibleItems(palette);
  if (!items.length) return;
  e.preventDefault(); // own the arrow: don't move the textarea caret
  var cur = -1;
  for (var i = 0; i < items.length; i++) {
    if (items[i].classList.contains('is-active')) { cur = i; break; }
  }
  var next = e.key === 'ArrowDown'
    ? (cur + 1) % items.length
    : (cur - 1 + items.length) % items.length; // wraps at both ends
  items.forEach(function (el) { el.classList.remove('is-active'); });
  items[next].classList.add('is-active');
  items[next].scrollIntoView({ block: 'nearest' });
});

// As the owner types and the menu re-filters, default the highlight to the top
// match so Enter's target is always visible. Deferred one frame so Datastar's
// data-show has re-evaluated visibility before we read offsetParent.
document.addEventListener('input', function (e) {
  var ta = e.target;
  if (!ta || ta.tagName !== 'TEXTAREA' || !ta.closest('.composer')) return;
  var palette = ta.closest('.composer').querySelector('.cmd-palette');
  if (!palette) return;
  requestAnimationFrame(function () {
    var items = balaurCmdVisibleItems(palette);
    items.forEach(function (el, i) { el.classList.toggle('is-active', i === 0); });
  });
});
```

Notes for you (do not paste into the file):
- Wrapping at both ends (↓ from last → first, ↑ from first → last; and ↑ from
  "none" → last) is intentional and matches common palettes.
- The `input` reset re-defaults to the top match on every keystroke; arrow keys
  don't fire `input`, so an arrow selection persists until the owner types
  again. This keeps Enter's default identical to today (top match) while making
  the selection *visible*.

**Verify**: `grep -c "addEventListener('keydown'\|addEventListener('input'\|addEventListener(\"keydown\"\|addEventListener(\"input\"" internal/web/assets/static/basm.js`
→ count increased by 2 versus before this step (there is now one more keydown
listener and one more input listener). Simpler check:
`grep -n "ArrowDown" internal/web/assets/static/basm.js` → at least one hit in
the block you just added.

### Step 3 — Style the active row (mirror hover + a clearer keyboard marker)

In `internal/web/assets/static/basm.css`, **replace** the single hover rule at
line 3521:

```css
.cmd-item:hover { background: var(--surface-3, var(--surface)); color: var(--gold); }
```

with the following (hover and keyboard-active share the fill; the active row
gets an extra inset keyline so a keyboard selection reads even mid-list, and a
little scroll-margin so `scrollIntoView` leaves breathing room in the 40vh
scroll box):

```css
.cmd-item:hover,
.cmd-item.is-active { background: var(--surface-3, var(--surface)); color: var(--gold); }
.cmd-item.is-active { outline: 1px solid var(--gold-deep); outline-offset: -1px; scroll-margin: var(--space-2); }
```

Do not change any other rule in the palette block.

**Verify**: `grep -n "cmd-item.is-active" internal/web/assets/static/basm.css`
→ two hits (the shared fill rule and the keyline rule).

### Step 4 — Document the keyboard nav in the storybook story

In `internal/feature/storybook/stories_chat.go`, in `commandpaletteStory()`
(lines 277–313), make two small doc edits so the story (the component's source
of truth) describes the new behavior. The static storybook can't *render* the
JS-applied `.is-active` highlight, so this is documentation, not a new variant.

(a) Append one sentence to the end of the `Blurb` string (before its closing
quote):

```
" Navigate the menu with ↑/↓ and select the highlighted row with Enter; the active row carries .cmd-item.is-active (set by basm.js)."
```

(b) Add one entry to the `Dos` slice:

```go
"Drive it from the keyboard: ↑/↓ move the active row, Enter selects it, mouse hover and click still work.",
```

Leave the existing `Variants`, `Props`, and `Donts` unchanged.

**Verify**: `grep -n "Navigate the menu with" internal/feature/storybook/stories_chat.go`
→ one hit. And `CGO_ENABLED=0 go build ./internal/feature/storybook/` → exit 0.

### Step 5 — Add a CSS guard test for the highlight rule

Add a test to `internal/web/assets/css_tokens_test.go` (package `assets`,
`strings` already imported) that fails if the `.cmd-item.is-active` highlight
is ever dropped — the keyboard nav is invisible without it. Follow the existing
`FS.ReadFile("static/basm.css")` + `strings.Contains` pattern in that file:

```go
// TestCmdPaletteActiveStyle guards plan 105: the composer /-command menu is
// keyboard-navigable (↑/↓ move .cmd-item.is-active; Enter selects it via
// balaurSubmitOnEnter). The highlight is invisible without this CSS rule.
func TestCmdPaletteActiveStyle(t *testing.T) {
	b, err := FS.ReadFile("static/basm.css")
	if err != nil {
		t.Fatalf("read basm.css: %v", err)
	}
	if !strings.Contains(string(b), ".cmd-item.is-active") {
		t.Error(".cmd-item.is-active highlight is missing — keyboard nav in the /-command menu would be invisible (plan 105)")
	}
}
```

**Verify**: `CGO_ENABLED=0 go test ./internal/web/assets/ -run TestCmdPaletteActiveStyle -count=1`
→ `ok`. Sanity-check it guards the fix: it must PASS now (Step 3 done) and would
FAIL if `.cmd-item.is-active` were removed.

### Step 6 — Full gates + browser verification

Run all gates:

```
gofmt -l internal/                          # prints nothing
CGO_ENABLED=0 go build ./...                # exit 0
go vet ./...                                # exit 0
CGO_ENABLED=0 go test ./... -count=1        # exit 0, no FAIL
git diff --check                            # prints nothing
```

**Browser check (the feature is interactive — confirm it in a real browser).**
Build/run the app (`make run`, or it may already be on `:8090`) and open
`http://localhost:8090/`. Then:

1. Click into the composer textarea and type `/`. The `.cmd-palette` appears
   and the **first row is highlighted** (gold fill + keyline).
2. Press **↓** a few times: the highlight moves down row to row and **wraps**
   to the top after the last. Press **↑**: it moves up and wraps to the bottom.
   The textarea caret must NOT move while the menu is open.
3. With a row highlighted, press **Enter**: that row's artifact opens in the
   panel and the draft clears (the menu closes). It must NOT post a `/…` line
   into the chat.
4. Type to filter (e.g. `/se`): the list narrows and the highlight re-defaults
   to the **top match**; Enter selects it.
5. Mouse hover and click still work unchanged.

Optional scripted check (Playwright MCP or dev console) to assert the core
invariant without manual keypress-watching — paste in the dev console after
typing `/` in the composer:

```js
(() => {
  const ta = document.querySelector('.composer textarea');
  const palette = ta.closest('.composer').querySelector('.cmd-palette');
  const vis = () => [...palette.querySelectorAll('.cmd-item')].filter(el => el.offsetParent !== null);
  ta.focus();
  ta.dispatchEvent(new KeyboardEvent('keydown', {key:'ArrowDown', bubbles:true}));
  const active = palette.querySelector('.cmd-item.is-active');
  return { open: palette.offsetParent !== null, visibleCount: vis().length, hasActive: !!active };
})()
// expect: { open: true, visibleCount: >0, hasActive: true }
```

If no browser is available in your environment, say so and rely on the build +
`go vet` + the Step 5 unit test; **do not claim the interaction was verified.**

## Test plan

- **New unit test** `TestCmdPaletteActiveStyle` in
  `internal/web/assets/css_tokens_test.go` — pins the `.cmd-item.is-active`
  highlight rule. Modeled on the existing `TestThemePaletteBlocks` /
  `TestNoUndefinedHearthwoodTokens` in the same file (same `FS.ReadFile` +
  `strings.Contains` shape).
- **No other Go test changes.** The JS (Steps 1–2) has no unit harness in this
  repo — it is gated by `go build`/`go vet` and the Step 6 browser check. The
  existing storybook render tests cover that `commandpaletteStory` still builds
  after the Step 4 doc edit; confirm via the full `go test ./...` run.
- **Verification**: `CGO_ENABLED=0 go test ./... -count=1` → all pass,
  including the one new test.

## Done criteria (ALL must hold)

- [ ] `grep -n "balaurCmdVisibleItems" internal/web/assets/static/basm.js` → ≥2 hits
- [ ] `grep -n "ArrowDown" internal/web/assets/static/basm.js` → ≥1 hit
- [ ] `grep -n "cmd-item.is-active" internal/web/assets/static/basm.css` → 2 hits
- [ ] `grep -n "Navigate the menu with" internal/feature/storybook/stories_chat.go` → 1 hit
- [ ] `gofmt -l internal/` prints nothing
- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=0 go test ./... -count=1` exits 0; `TestCmdPaletteActiveStyle` exists and passes
- [ ] `git diff --check` prints nothing
- [ ] Only the four in-scope files are modified (`git status`)
- [ ] `internal/ui/command_palette.go` and `internal/ui/composer.go` are **unchanged** (`git status`)
- [ ] (If a browser is available) the Step 6 scripted check returns `{ open:true, visibleCount:>0, hasActive:true }`
- [ ] This plan's row in `plans/readme.md` is updated

## STOP conditions

Stop and report back (do not improvise) if:

- `balaurSubmitOnEnter` in `basm.js` no longer matches the "Current state (B)"
  excerpt (e.g. it was refactored, or slash handling moved) — re-confirm where
  Enter-on-slash is handled before editing.
- `command_palette.go` no longer renders `.cmd-palette` / `.cmd-item`, or the
  per-item `data-on:click__prevent` is gone — the "selecting = `.click()` the
  row" assumption no longer holds; report what replaced it.
- A `$cmdActive` (or similar) Datastar signal already exists for this — someone
  implemented active state differently; reconcile rather than adding a second
  mechanism.
- During the browser check, ↑/↓ move the textarea caret instead of the
  highlight (the `e.preventDefault()` / open-menu guard isn't firing), OR Enter
  posts a `/…` line into chat — report the behavior; do not pile on more
  handlers.
- Any step's verification fails twice after a reasonable fix attempt.

## Maintenance notes

For the human/agent who owns this after it lands:

- **The highlight is intentionally DOM-owned, not a Datastar signal** (see "Key
  design decision"). If a future change moves the palette's filtering to the
  server, or introduces a real ARIA combobox, revisit this — but don't convert
  it to a signal just for tidiness; "Nth visible row" is a browser-only value.
- **Three coupled pieces** must move together: the `.is-active` class name is
  shared by the ↑/↓ listener, the `input` reset, `balaurSubmitOnEnter`, the CSS
  rule, and the guard test. Rename it in one place → rename in all five.
- **The `input` reset defers via `requestAnimationFrame`** so it reads
  visibility *after* Datastar's `data-show` re-evaluates. If Datastar's update
  timing changes (a major version bump) and the top-match highlight starts
  lagging a keystroke, that rAF is the thing to revisit.
- **Deferred, not in this plan** (raise separately if wanted):
  - **Escape-to-close** the menu — needs clearing the Datastar-owned `$message`
    from JS, which has no clean hook here.
  - **Full combobox ARIA** (`role="listbox"`/`aria-activedescendant`) — the
    rows are `<a>` links today; proper screen-reader semantics is a larger,
    separate a11y task.
  - **Tab / Shift-Tab** as nav aliases — intentionally left to normal focus
    traversal.
