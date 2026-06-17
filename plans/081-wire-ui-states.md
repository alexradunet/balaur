# Plan 081: Wire the missing UI states — shared inline empty states, a styled full-page error, and an honest Toast/Skeleton story

> **Executor instructions**: Follow step by step. Run every Verify and confirm before moving on. On a STOP condition, stop and report — do not improvise. When done, update the 081 row in plans/readme.md (add the row if it is not present yet, matching the existing column format).
>
> **Drift check (run first)**: `git diff --stat 12a2ff5..HEAD -- internal/ui internal/feature internal/web` — if any in-scope file changed since this plan was written, compare the "Current state" excerpts to the live code; on mismatch, STOP.

## Status
- **Priority**: P2
- **Effort**: M
- **Risk**: MED
- **Depends on**: soft: plans/080-*.md (atoms gain trailing variadic attrs — not required here; this plan does not add Datastar attrs to the new atom)
- **Category**: consistency/correctness
- **Planned at**: commit `12a2ff5`, 2026-06-17

## Why this matters
Balaur's component system ships three state primitives — `EmptyState`, `Toast`, `Skeleton` — but product code barely uses them, so the storybook (the "source of truth") presents patterns that the running app does not actually run. Concretely: feature cards hand-roll ~18 inline `<p class="k-empty">` placeholders instead of one shared atom; there is no confirmation after a Datastar action even though `Toast` exists; and a server error on a full-page handler (`home.go`, `focus.go`, `boards.go`) drops the user out of the Hearthwood shell into PocketBase's raw JSON error. This plan (1) folds the `k-empty` inline placeholder into the `EmptyState` atom so there is one source of truth, (2) renders full-page handler errors inside the shell, and (3) makes an honest decision on `Toast`/`Skeleton` — wiring one if it is cheap, otherwise marking the storybook story "not yet wired" so the catalog stops lying. DESIGN.md: empty states "wait without nagging"; AGENTS.md: "the storybook is its source of truth … reuse or extend a component instead of hand-rolling markup."

## Current state

### The two empty-state shapes that exist today
- **Full / centered** `ui.EmptyState` (`internal/ui/emptystate.go:17-35`) renders a centered crest + display title + line + wood button. Used in exactly ONE product site: `internal/feature/modelcards/panel.go:33-37`.
  - `internal/ui/emptystate.go:11-13`:
    ```go
    type EmptyProps struct {
    	CrestSrc, Title, Line, ActionLabel, ActionHref string
    }
    ```
  - CSS `internal/web/assets/static/basm.css:2757-2761` (`.empty`, `.empty-crest`, `.empty-title`, `.empty-line`, `.empty-action`).
- **Inline / compact** hand-rolled `P(Class("k-empty"), g.Text(...))`, in ~18 feature-card sites. CSS is a single line `internal/web/assets/static/basm.css:998`:
  ```css
  .k-empty { color: var(--muted); font-size: 15px; }
  ```
  Example sites (feature packages dot-import `html`, so it is `P(...)` not `h.P(...)`):
  - `internal/feature/taskcards/today.go:77-78` — `return P(Class("k-empty"), g.Text("Nothing due today."))`
  - `internal/feature/knowledgecards/memory.go:64-65` — `return P(Class("k-empty"), g.Text("No active memories yet."))`
  - `internal/feature/lifecards/lifelogfocus.go:108-109` — multi-line dynamic text.

### Full list of inline `k-empty` PRODUCTION sites to migrate (confirmed by grep at HEAD)
| File:line | Text |
| --- | --- |
| `internal/feature/taskcards/today.go:78` | `Nothing due today.` |
| `internal/feature/taskcards/habits.go:66` | `No habits yet — add a recurring task in chat.` |
| `internal/feature/taskcards/timeline.go:95` | `Nothing upcoming in the window.` |
| `internal/feature/taskcards/timeline.go:106` | `Nothing upcoming in the window.` |
| `internal/feature/taskcards/quests.go:37` | `No quests here yet.` |
| `internal/feature/taskcards/quests.go:78` | `No open quests — add one in chat.` |
| `internal/feature/taskcards/questsfocus.go:143` | `No quests yet. Speak one in the chat.` |
| `internal/feature/taskcards/questsfocus.go:176` | `No quests yet. Speak one in the chat.` |
| `internal/feature/knowledgecards/memory.go:65` | `No active memories yet.` |
| `internal/feature/knowledgecards/memory.go:193` | `Nothing yet — Memory appears as Balaur proposes.` |
| `internal/feature/knowledgecards/skills.go:127` | `No active skills yet.` |
| `internal/feature/knowledgecards/skills.go:234` | `Nothing yet — Skills appears as Balaur proposes.` |
| `internal/feature/knowledgecards/knowledgefocus.go:49` | `Nothing matches %q.` (fmt.Sprintf) |
| `internal/feature/knowledgecards/knowledgefocus.go:51` | `Nothing here yet. Speak with Balaur — …` |
| `internal/feature/lifecards/lifelogfocus.go:108` | `Nothing tracked yet. …` (long, multi-sentence) |
| `internal/feature/lifecards/lines.go:86` | `No "+v.Kind+" entries yet.` (concatenated) |
| `internal/feature/lifecards/measure.go:100` | `No "+v.Kind+" entries yet.` (concatenated) |
| `internal/feature/journalcards/journal.go:89` | `No journal entries yet.` |

(`internal/web/recap.go:153` writes a raw `<p class="k-empty">…</p>` string into a `strings.Builder` — it is NOT a gomponents call site; leave it. See Out of scope.)

### Tests that PIN the `k-empty` output (these MUST stay green — they constrain the atom's markup)
- `internal/feature/lifecards/lifelogfocus_test.go:58` asserts the EXACT substring `<p class="k-empty">Nothing tracked yet.` — so the compact atom MUST render `<p class="k-empty">` with the text as the direct first child (no wrapper element, no extra leading class token).
- `internal/feature/taskcards/questsfocus_test.go:75,122` assert substring `class="k-empty"`.
- `internal/feature/knowledgecards/knowledgefocus_test.go:160,171` assert `class="k-empty"`.
- `internal/feature/journalcards/journal_test.go:61` asserts substring `k-empty`.
- Text-only assertions (do not pin the class): `today_test.go:43` (`Nothing due today.`), `habits_test.go:112` (`No habits yet`), `quests_test.go` (`No quests here yet.` / `No open quests`), `knowledgefocus_test.go` (`Nothing here yet.` / `Nothing matches`).
- **Consequence**: the new compact variant's class string MUST be exactly `k-empty` (the class attribute value `class="k-empty"`, not `class="k-empty empty-compact"`), and the element MUST be a `<p>` with `g.Text(...)` directly inside. The simplest correct shape is byte-identical to today's hand-rolled output, so every pinned test passes by construction.

### EmptyState storybook story (the catalog entry to update)
`internal/feature/storybook/stories_feedback.go:118-146` (`emptyStateStory`) shows only the "with action" full variant. `skeletonStory` is `:91-116`; `toastStory` is `:148-171`.

### Full-page handlers and the error path
- `internal/web/web.go:160-246` `Register` mounts routes on `se.Router`. The error path is PocketBase's package-level `router.ErrorHandler` (`github.com/pocketbase/pocketbase@v0.39.3/tools/router/router.go:160-184`): when a handler returns an error and nothing has been written, it sets `Content-Type: application/json` and JSON-encodes the `ApiError`. There is **no** documented per-route error-template hook — so the fix is to WRAP the full-page handlers (catch their returned error and render the shell error page ourselves for full-document requests).
- `internal/web/home.go:57-77` `homePage`: returns `e.InternalServerError(...)` on dock/render failure.
- `internal/web/focus.go:111-179` `focusPage`: returns `e.NotFoundError("no such card type", nil)` at `:115`, `e.BadRequestError(...)` at `:121`, and several `e.InternalServerError(...)`. The Datastar branch (`isDatastarRequest`, `internal/web/web.go:292-294`) patches `#main` and must be left alone — only the full-document branch falls out of the shell.
- `internal/web/boards.go` full-page handlers `boardsIndex`/`boardsPage` return `e.InternalServerError`/`e.NotFoundError` (e.g. `boards.go:324` `e.NotFoundError("board not found", nil)`).
- The inline analogue to mirror is `cardErrorStrip` (`internal/web/cards.go:137-141`):
  ```go
  func cardErrorStrip(msg string) template.HTML {
  	return template.HTML(`<div class="card-note card-note-error">` + html.EscapeString(msg) + `</div>`)
  }
  ```
  and its gomponents twin `ui.ErrorStrip` (`internal/ui/components.go:12-14`) — `g.Text` auto-escapes, never `g.Raw`.

### Shell + page-render plumbing to reuse
- `shell.Page(shell.PageProps{...})` (`internal/ui/shell/shell.go:20-51`): `PageProps{Title, Active, HTMLClass, Body, Dock g.Node}`. `Body` fills `#main`, `Dock` fills `#dock`. A page renders with `page.Render(e.Response)` after setting `Content-Type: text/html; charset=utf-8` (see `home.go:72-76`).
- `ui.Button(ui.ButtonProps{Variant:"wood", Href:"/"}, g.Text("…"))` renders a wood link (`internal/ui/button.go:36-41`).

### Toast / Skeleton current usage (the honest-story decision)
- `internal/ui/toast.go` (`Toast`) — NOT referenced by any product code (grep `ui.Toast(` returns only `stories_feedback.go`). There is no `#toast` region in the shell.
- `internal/ui/skeleton.go:22-63` — NOT referenced by any product page. Datastar patches are synchronous (the handler builds HTML then patches), so card focus and grid loads have no async gap to fill. The chat already has a thinking indicator: `.thinking` / `.thinking-dots` CSS at `internal/web/assets/static/basm.css:665-686` (driven by `chatstream.go` + `message.go`) — that is the existing async-gap pattern and is OUT of scope.
- **Decision for THIS plan (Pareto):** do NOT build a new `#toast` SSE region or wire `Skeleton` into a synchronous surface — both are more than an S of net-new plumbing for little user value, and YAGNI/CLAUDE.md forbid speculative wiring. Instead, mark both storybook stories as "not yet wired" so the catalog is honest. (If a future plan adds a genuinely async surface, wire `Skeleton` there then.) This is the explicit choice the SPEC asked the executor to make and state.

### Conventions to honor
- `internal/ui` files: import `g "maragu.dev/gomponents"` + qualified `h "maragu.dev/gomponents/html"` (NEVER dot-import). Atoms are `func Name(p Props, children ...g.Node) g.Node`.
- `internal/feature/*` files dot-import `html` (so `P`, `Class`, `Div` are unqualified) and import `g` + `ui`. Dependency direction is one-way: `internal/feature/*` → `internal/ui`; never the reverse.
- DESIGN.md tokens: colors are `var(--token)` only; `--radius:0`; single-dash class names; CSS rule blocks appended at the END of `basm.css` under a `/* ── Section ── */` banner. The new error-page styles (if any new class is needed) go there; the `.k-empty` rule already exists at `:998` and stays.

## Commands you will need
| Purpose | Command | Expected |
| --- | --- | --- |
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Full test | `go test ./...` | all pass |
| Atom tests | `go test ./internal/ui/...` | ok |
| Feature card tests | `go test ./internal/feature/...` | ok |
| Web handler tests | `go test ./internal/web/...` | ok |
| Storybook render | `go test ./internal/feature/storybook/...` | ok |
| Format | `gofmt -l <changed.go>` | empty output |
| Whitespace | `git diff --check` | no output |
| Confirm no inline k-empty left in features | `grep -rn 'Class("k-empty")' internal/feature` | no output (after Step 2) |
| Confirm Toast/Skeleton still unused in product | `grep -rn 'ui.Toast(\|ui.Skeleton(\|ui.SkeletonLine(' internal/web internal/feature/*cards` | no output |

## Scope
**In scope** (only files you may modify):
- `internal/ui/emptystate.go` — add the compact inline variant.
- `internal/ui/emptystate_test.go` — cover the compact variant.
- The 12 feature-card files in the migration table above (under `internal/feature/taskcards`, `internal/feature/knowledgecards`, `internal/feature/lifecards`, `internal/feature/journalcards`).
- `internal/web/home.go`, `internal/web/focus.go`, `internal/web/boards.go` — wrap full-page errors in the shell.
- `internal/web/web.go` — only if a small shared `renderPageError` helper or a wrapper is best placed here (keep it minimal; it MAY instead live in one of the handler files — choose the smallest diff).
- A new web handler test file (e.g. `internal/web/page_error_test.go`) — asserts a forced full-page error renders HTML in the shell, not JSON.
- `internal/web/assets/static/basm.css` — ONLY if a genuinely new class is needed for the error page; prefer reusing `.empty` + existing tokens so no CSS change is required.
- `internal/feature/storybook/stories_feedback.go` — add the compact `EmptyState` variant; mark `Skeleton` + `Toast` stories "not yet wired".

**Out of scope** (do NOT touch):
- The chat thinking indicator — `chatstream.go`, `message.go`, `basm.css:665-686`. It already covers the only real async gap; changing it is a different concern.
- `chat.Dock` port (plan 084) — `home.go`/`focus.go` still inject the legacy `chat_dock` template via `g.Raw`; do not port it here.
- `internal/web/recap.go:153` — that `<p class="k-empty">` is a raw string written into a `strings.Builder`, not a gomponents call; migrating it is a string-building refactor outside this plan's atom-reuse goal. Leave it (the `.k-empty` CSS stays for it).
- The existing `.k-empty` CSS rule (`basm.css:998`) — keep it; the compact atom reuses the same class, and recap.go still depends on it.
- Building a new `#toast` SSE region or wiring `Skeleton` — explicitly deferred (see Decision above); do not add speculative plumbing.

## Git workflow
Branch `improve/081-wire-ui-states`. Conventional commits, e.g. `refactor(ui): fold inline k-empty into EmptyState compact variant`, then `feat(web): render full-page handler errors inside the shell`, then `docs(storybook): mark Toast/Skeleton not-yet-wired`. Do NOT push or open a PR unless told.

## Steps

### Step 1: Add a compact inline variant to the EmptyState atom
In `internal/ui/emptystate.go`, add a `Compact bool` to `EmptyProps` and short-circuit at the top of `EmptyState`. The compact branch MUST render byte-identically to the hand-rolled site so pinned tests pass:

```go
type EmptyProps struct {
	CrestSrc, Title, Line, ActionLabel, ActionHref string
	Compact bool // inline placeholder for small card tiles (renders <p class="k-empty">…</p>)
}

func EmptyState(p EmptyProps) g.Node {
	if p.Compact {
		// Inline tile placeholder. Text is Line if set, else Title — one short
		// sentence. Markup is byte-identical to the legacy hand-rolled
		// P(Class("k-empty"), g.Text(...)) so pinned card tests stay green.
		msg := p.Line
		if msg == "" {
			msg = p.Title
		}
		return h.P(h.Class("k-empty"), g.Text(msg))
	}
	// …existing full/centered body unchanged…
}
```

Rationale for one-arg text: every inline site today is a single short string; mapping it to `Line` (the "supporting sentence" field) keeps the full-variant fields meaningful. Document in the doc comment that `Compact` ignores `CrestSrc`/`ActionLabel`/`ActionHref` (tiles have no room).

Do NOT add a new CSS class — the compact variant reuses the existing `.k-empty` rule at `basm.css:998`.

**Verify**: `go test ./internal/ui/...` → ok (after Step 1b adds the test, or run `CGO_ENABLED=0 go build ./internal/ui/...` → exit 0 now).

### Step 1b: Cover the compact variant in emptystate_test.go
In `internal/ui/emptystate_test.go` (pattern: `TestEmptyStateFull` at `:10`, uses `render(t, …)` helper), add:

```go
func TestEmptyStateCompact(t *testing.T) {
	got := render(t, ui.EmptyState(ui.EmptyProps{Compact: true, Line: "Nothing due today."}))
	if got != `<p class="k-empty">Nothing due today.</p>` {
		t.Errorf("compact empty state not byte-stable; got: %s", got)
	}
}
```
If the `render` helper trims or wraps output, assert with `strings.Contains` on `<p class="k-empty">Nothing due today.</p>` instead (check the existing helper's behavior first — `TestEmptyStateFull` uses `strings.Contains`, so match that style). Also add a case proving `Compact` with only `Title` set falls back to the title text.

**Verify**: `go test ./internal/ui/...` → ok.

### Step 2: Migrate the 12 feature-card inline sites to ui.EmptyState{Compact:true}
For each row in the migration table, replace `P(Class("k-empty"), g.Text(<text>))` with `ui.EmptyState(ui.EmptyProps{Compact: true, Line: <text>})`. Keep the surrounding control flow identical. Concatenated/`fmt.Sprintf` texts pass through unchanged as the `Line` value, e.g.:
- `lines.go:86`: `ui.EmptyState(ui.EmptyProps{Compact: true, Line: "No " + v.Kind + " entries yet."})`
- `knowledgefocus.go:49`: `ui.EmptyState(ui.EmptyProps{Compact: true, Line: fmt.Sprintf("Nothing matches %q.", query)})`

Each feature package already imports `ui` (e.g. `today.go` calls `ui.CardHead`); if any of these 12 files does NOT yet import `github.com/alexradunet/balaur/internal/ui`, add it. If a file no longer references `g` after the edit, the build will flag the unused import — only remove an import your edit actually orphaned (CLAUDE.md surgical-changes rule); most sites still use `g.Text` elsewhere so `g` stays.

Do these one file at a time and run that package's test after each, so a regression is localized.

**Verify (run after each file, then once at the end)**:
- `go test ./internal/feature/...` → ok (the pinned `class="k-empty"` and text assertions all pass because the markup is byte-identical).
- `grep -rn 'Class("k-empty")' internal/feature` → no output.

### Step 3: Render full-page handler errors inside the Hearthwood shell
Add a small helper that renders an error page in the shell, then route the full-document error returns of the three full-page handlers through it. Keep the Datastar (`isDatastarRequest`) branches UNCHANGED.

Add the helper (place it in `internal/web/web.go` near `render`, or in `home.go` — smallest diff wins):

```go
// renderPageError renders a sanitized error inside the Hearthwood shell so a
// full-page handler failure keeps the user in-app instead of falling out to
// PocketBase's raw JSON error. status sets the HTTP code; msg is a short,
// owner-safe sentence (NEVER a raw error string — those may leak paths/tokens).
func (h *handlers) renderPageError(e *core.RequestEvent, status int, title, msg string) error {
	page := shell.Page(shell.PageProps{
		Title:  title,
		Active: "",
		Body: h.Div(h.Class("empty"),
			ui.EmptyState(ui.EmptyProps{
				Title:       title,
				Line:        msg,
				ActionLabel: "Back home",
				ActionHref:  "/",
			}),
		),
		Dock: g.Text(""), // no dock on the error page — keep it deterministic
	})
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	e.Response.WriteHeader(status)
	return page.Render(e.Response)
}
```
(Import `h "maragu.dev/gomponents/html"` is already aliased differently in web — web uses `template`, not gomponents html. Check the file's existing imports: `home.go` imports `g "maragu.dev/gomponents"`, `internal/ui`, `internal/ui/shell`. Do NOT introduce a dot/qualified html import if `EmptyState` alone suffices — simplest form is `Body: ui.EmptyState(ui.EmptyProps{Title: title, Line: msg, ActionLabel: "Back home", ActionHref: "/"})` with no extra wrapper Div, since `.empty` is already the atom's own root class.)

Simplify to:
```go
Body: ui.EmptyState(ui.EmptyProps{Title: title, Line: msg, ActionLabel: "Back home", ActionHref: "/"}),
```

Then in each full-page handler, for the FULL-DOCUMENT path only, replace the bare `return e.InternalServerError(...)` / `e.NotFoundError(...)` with `return h.renderPageError(e, http.StatusInternalServerError, "Something went wrong", "Balaur could not open this page. Try again, or head back home.")` (and `http.StatusNotFound` + "Not found" / "There is nothing at this address." for the NotFound cases). Sanitize: pass a fixed sentence, NOT `err.Error()`.

**Imports**: `focus.go` does NOT yet import `net/http` at HEAD (its imports are `fmt`, `html/template`, `net/url`, `regexp`, `strings`, `core`, `datastar`, `gomponents`, `internal/cards`, `internal/ui/shell`) — add `"net/http"` to its import block when you introduce the `http.StatusNotFound`/`http.StatusInternalServerError` calls. `boards.go` (line 9) and `home.go` already import `net/http`. The build Verify catches a missing import, but add it deliberately.

Targets (full-document branches only):
- `home.go:60,64,74` (`homePage`).
- `focus.go:115` (NotFound — note `focusPage` reaches `:115` BEFORE the `isDatastarRequest` branch split at `:131`, so it runs for both branches. Wrapping it is correct despite the maintenance-note SSE caution because unknown card types are never reached via a Datastar `@get` navigation (links only target known types), so no real SSE patch stream is ever corrupted by the full-document error here), `focus.go:158,162,166,176` (the full-document branch after `if isDatastarRequest`).
- `boards.go` `boardsIndex`/`boardsPage` full-document returns: `boardsIndex:275` (seeding) and `:279` (loading boards); `boardsPage:298` (seeding), `:303` (loading boards), `:324` NotFound, and `:354` (loading companion dock). **WARNING**: `boards.go:336` (`e.InternalServerError("rendering board", err)`) is INSIDE the `if ds {` Datastar/SSE branch — leave it as-is. Confirm by reading the function bodies which returns are in a full-document path vs an SSE/fragment path; only convert the full-document ones.

Leave `focus.go:121` `BadRequestError` and `:135` (the SSE-branch `InternalServerError`) as-is — those are fragment/SSE responses, not full-document. (`focus.go:137` is the `sse.PatchElements(...)` call, not an error return; it returns `nil` on client-gone — also untouched.)

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0; `go vet ./...` → exit 0.

### Step 4: Add a web handler test asserting the error page is HTML in the shell
New file `internal/web/page_error_test.go`. Pattern: `internal/web/home_test.go` uses `tests.ApiScenario` with `TestAppFactory: newWebApp`. The cleanest deterministic trigger is `focusPage`'s unknown-card-type path: `GET /focus/__nope__` returns `NotFoundError` → now the shell error page.

```go
func TestFocusUnknownTypeRendersShellError(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET /focus/<unknown> renders the shell error page, not JSON",
		Method:         "GET",
		URL:            "/focus/__nope__",
		TestAppFactory: newWebApp,
		ExpectedStatus: 404,
		ExpectedContent: []string{
			"<!doctype html>",
			`class="empty"`,        // the EmptyState body
			`href="/"`,             // Back home link
		},
		NotExpectedContent: []string{
			`"status":404`, // not the raw JSON ApiError
		},
	}
	s.Test(t)
}
```
Confirm `newWebApp` is the factory used by sibling tests (grep `newWebApp` in `internal/web`). If `/focus/__nope__` is intercepted before `focusPage` (e.g. by the `/` catch-all), pick a different deterministic trigger and document it; the unknown-type path is the expected one because `/focus/{type}` is a registered route (`web.go:235`).

**Verify**: `go test ./internal/web/...` → ok (new test passes; existing pass).

### Step 5: Update the storybook stories (EmptyState compact + honest Toast/Skeleton)
In `internal/feature/storybook/stories_feedback.go`:
- `emptyStateStory` (`:118`): add a second variant showing the compact form, e.g. inside the existing `Variants` slice:
  ```go
  {"compact (in a card)", ui.EmptyState(ui.EmptyProps{Compact: true, Line: "Nothing due today."})},
  ```
  and add a `Prop` row `{"Compact", "bool", "false", "Inline tile placeholder — renders the small k-empty line instead of the centered crest body."}`. Note in the blurb that the compact variant is what the domain cards use.
- `skeletonStory` (`:91`): add to the `Blurb` (or a `Donts`/note) an honest line that Skeleton is **not yet wired into any product surface** — Datastar patches are synchronous, so there is no async gap today; the chat's thinking indicator covers the one real gap. Do not present it as a shipped loading pattern. (If the `Story` struct has a dedicated status/note field, use it; otherwise fold the sentence into the Blurb.)
- `toastStory` (`:148`): similarly note Toast is **not yet wired into any owner action** — there is no `#toast` region yet; it is a designed-but-unused primitive pending a surface that needs post-action confirmation.

Check the `Story` struct fields first (`internal/feature/storybook/story.go`) so you use a real field for any note; if there is no note field, the Blurb is the place.

**Verify**: `go test ./internal/feature/storybook/...` → ok (`TestAllStoriesRender` still green).

### Step 6: Final full verification
**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → all pass
- `gofmt -l internal/ui/emptystate.go internal/ui/emptystate_test.go internal/web/home.go internal/web/focus.go internal/web/boards.go internal/web/web.go internal/web/page_error_test.go internal/feature/storybook/stories_feedback.go` (+ the 12 migrated card files) → empty
- `git diff --check` → no output

### Step 7: Visual check in BOTH modes
Run the app (it may already be serving on `127.0.0.1:8090`; else `go run . serve --http=127.0.0.1:8090`). In a browser/CDP:
1. Open `/focus/__nope__` — confirm the error renders inside the Hearthwood shell (topbar present, parchment EmptyState, a "Back home" wood button linking `/`), NOT raw JSON. Force `document.documentElement.className='theme-hearthwood dark'` then `'theme-hearthwood light'` and confirm the title/line use legible tokens in both (the `.empty-title` uses `--fg-strong`, `.empty-line` uses `--muted` — these read on the page bg; confirm no flip-to-illegible).
2. Open `/storybook/emptystate` — confirm both the full and the new compact variant render.
3. Open `/focus/quests` with no quests (or any empty card) — confirm the inline placeholder still reads `Nothing …` exactly as before (no layout shift).
4. Check `<=920px`: the error page EmptyState stays centered and the button is reachable.

## Test plan
- `internal/ui/emptystate_test.go`: new `TestEmptyStateCompact` (byte-stable `<p class="k-empty">…</p>`; Title fallback). Pattern: existing `TestEmptyStateFull`.
- `internal/web/page_error_test.go`: new `TestFocusUnknownTypeRendersShellError` (HTML-in-shell, status 404, not JSON). Pattern: `internal/web/home_test.go` `tests.ApiScenario`.
- Existing pinned card tests (`lifelogfocus_test.go`, `questsfocus_test.go`, `knowledgefocus_test.go`, `journal_test.go`, `today_test.go`, `habits_test.go`, `quests_test.go`) must stay green WITHOUT edits — the migration is byte-stable by design. If any fails, the atom's compact markup drifted; fix the atom, not the test.
- Storybook: `TestAllStoriesRender` (`internal/feature/storybook`) covers the updated `emptyStateStory`/`skeletonStory`/`toastStory`.

## Done criteria
- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go vet ./...` exits 0.
- [ ] `go test ./...` all pass (including the unchanged pinned card tests).
- [ ] `go test ./internal/feature/storybook/...` ok.
- [ ] `grep -rn 'Class("k-empty")' internal/feature` → no output.
- [ ] `grep -rn 'ui.Toast(\|ui.Skeleton(\|ui.SkeletonLine(' internal/web internal/feature/*cards` → no output (Toast/Skeleton intentionally still unused in product).
- [ ] New `TestEmptyStateCompact` and `TestFocusUnknownTypeRendersShellError` pass.
- [ ] `gofmt -l` on all changed Go files → empty; `git diff --check` → no output.
- [ ] Only in-scope files changed (`git status` shows nothing outside the Scope list).
- [ ] VISUAL: `/focus/__nope__` renders the in-shell error page in BOTH dark and light; `/storybook/emptystate` shows the compact variant; an empty card still shows the inline placeholder unchanged.
- [ ] update the 081 row in plans/readme.md (add the row if it is not present yet, matching the existing column format).

## STOP conditions
- **Drift**: the Step-0 `git diff --stat` shows an in-scope file changed since `12a2ff5` and its excerpt no longer matches — STOP and report which file.
- **Pinned test breaks**: migrating a `k-empty` site turns `lifelogfocus_test.go:58`/`questsfocus_test.go`/`knowledgefocus_test.go`/`journal_test.go` red. First confirm the compact atom renders EXACTLY `<p class="k-empty">{text}</p>` (no extra class token, no wrapper). If it still diverges after that, STOP and report the exact got/want diff — do NOT edit the pinned test to match a worse output.
- **Page-error needs PocketBase internals**: if wrapping the full-page handlers turns out to require overriding `router.ErrorHandler` or any PocketBase-internal hook beyond catching the handler's own return value and rendering the shell, STOP — scope shrinks to only the handlers you can wrap with a returned value, and report what could not be reached.
- **Error trigger intercepted**: if `GET /focus/__nope__` does NOT reach `focusPage` (e.g. routed to the `/` catch-all and redirected), the test trigger is wrong — pick another deterministic full-page error and document it; if none is reachable in a test app, STOP and report.
- **Any Verify fails twice** after a fix attempt — STOP and report the command + output.

## Maintenance notes
- The compact `EmptyState` reuses the legacy `.k-empty` class so `internal/web/recap.go:153` (a raw string, deliberately left un-migrated) keeps the same styling from one rule. If a future plan migrates recap.go's empty line to gomponents, route it through `ui.EmptyState{Compact:true}` too, then the `.k-empty` selector has a single conceptual owner.
- A reviewer should scrutinize: (1) that NO pinned test was edited to accommodate a markup change — the migration must be byte-stable; (2) that `renderPageError` never interpolates `err.Error()` (path/token leak — Safety section); (3) that the Datastar/SSE branches of `focusPage`/`boards.go` were left untouched (converting an SSE-branch error to a full HTML page would corrupt the patch stream).
- Deferred (explicitly): wiring `Toast` into a real owner action (needs a `#toast` SSE region + a chosen action like task-transition/memory-approve) and wiring `Skeleton` into a genuinely async surface (e.g. a pre-first-token assistant bubble) — both wait for a concrete need per YAGNI. The storybook notes added here are the honest interim state.
