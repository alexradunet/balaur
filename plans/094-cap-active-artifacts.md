# Plan 094: At most 3 active artifacts in the chat; older ones collapse to a static "shown earlier" chip

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md`.
>
> **Drift check (run first)**:
> `git diff --stat 766b7aa..HEAD -- internal/web/recap.go internal/web/chatstream.go internal/web/cards.go internal/web/assets/static/basm.js internal/web/assets/static/basm.css internal/web/show_test.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.
>
> **Sandbox note**: in a TLS-intercepting sandbox (Hyperagent), Go commands
> need the GOPROXY shim — see `docs/hyperagent-sandbox.md`. GOSUMDB stays on.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED
- **Depends on**: none (independent of 092 and 093 — it wraps whatever an artifact renders; any order)
- **Category**: direction (UX)
- **Planned at**: commit `766b7aa`, 2026-06-17

## Why this matters

In the single-page chat, every summoned domain card (sidebar click, the agent's
`card_show`, or a `show_cards` cluster) is a **persisted artifact** in the
transcript. As the owner summons more, the chat fills with full-surface cards
and gets cluttered. The decision (confirmed 2026-06-17): **keep at most 3
*active* (fully-rendered) artifacts; collapse older ones to a static one-line
"shown earlier" chip.** The chip is non-interactive — to see an old artifact
again the owner re-summons it from the sidebar (a fresh copy appends at the
bottom, and the cap re-applies).

Two render paths produce artifacts, and both must enforce the cap:
- **Reload / history** (`recap.go` `messageViews` → `renderMessages`) — the
  server knows the full ordered transcript, so it marks all-but-the-newest-3
  collapsed. Fully testable in Go.
- **Live** (`chatstream.go` for the agent path, plus the one-shot SSE in
  `show.go` for sidebar clicks) — a small client helper hooked into the
  **existing `#chat` MutationObserver** (`basm.js:149-154`) keeps the invariant
  as new artifacts arrive this session, across all fragments.

Both use one CSS class (`artifact--collapsed`) and one server-rendered chip
stub, so the two paths never diverge.

## Current state

Files and their roles:

- `internal/web/recap.go` — `messageView` struct (lines 162–181), `messageViews`
  (237–283: builds the views, has the `uicard` and `artifact`/cluster branches),
  `renderMessages` (188–212: the single history/reload render).
- `internal/web/chatstream.go` — `handleToolResult` (176–202) + `endTool`
  (206–213): the LIVE append of an artifact card.
- `internal/web/cards.go` — `uicardBody` (single card) and `artifactBody`
  (cluster). Good home for a shared `artifactWrap` helper (package `web`).
- `internal/web/show.go` — `uiShow` (sidebar door) appends via
  `renderMessages(messageViews([]*core.Record{rec}))`, so it inherits the
  wrapper automatically.
- `internal/web/assets/static/basm.js` — the `#chat` MutationObserver
  (lines 149–154) that already calls `balaurScrollToLatest`.
- `internal/web/assets/static/basm.css` — `.k-inline` (1038–1042): the current
  artifact body wrapper.
- `internal/cards` — `cards.Get(typ) (Spec, bool)`; `Spec` has `Label` and
  `Icon` (icon file stem under `/static/icons`).

### `messageView` + the two artifact branches (`recap.go`)

```go
// recap.go:162
type messageView struct {
	Role            string
	Tool            string
	Content         string
	Origin          string
	CardURL         string
	CardBody        template.HTML // server-rendered inline card, embedded directly
	// ... avatar/name + streaming fields ...
}

// recap.go:262 (inside messageViews, for mv.Role == "tool")
if typ, query, rest, ok := tools.ParseUICard(mv.Content); ok {
	mv.CardBody = h.uicardBody(typ, query)
	mv.Content = rest
} else if _, _, modelText, ok := tools.ParseChoices(mv.Content); ok {
	mv.Content = clipText(modelText, 2000)
} else if kind, id, rest, ok := tools.ParseProposal(mv.Content); ok {
	mv.CardBody, mv.Content = h.proposalBody(kind, id), rest
} else if _, rest, ok := tools.ParseRefresh(mv.Content); ok {
	mv.Content = clipText(rest, 2000)
} else if title, cs, rest, ok := tools.ParseArtifact(mv.Content); ok {
	mv.CardBody, mv.Content = h.artifactBody(title, cs), rest
}
```

### `renderMessages` tool case (`recap.go:196-202`)

```go
case "tool":
	nodes = append(nodes, chat.ToolRow(chat.ToolRowProps{
		Tool: mv.Tool, Icon: toolIconFile(mv.Tool), Content: mv.Content,
	}))
	if mv.CardBody != "" {
		nodes = append(nodes, g.El("div", g.Attr("class", "k-inline"), g.Raw(string(mv.CardBody))))
	}
```

### `endTool` + its callers (`chatstream.go:176-213`)

```go
func (s *chatStream) handleToolResult(ev agent.Event) {
	if typ, query, rest, ok := tools.ParseUICard(ev.Text); ok {
		s.endTool(rest, s.h.uicardBody(typ, query)); return
	}
	if prompt, choices, _, ok := tools.ParseChoices(ev.Text); ok {
		s.endTool("choices offered", ""); s.appendChoices(prompt, choices); return
	}
	if kind, id, rest, ok := tools.ParseProposal(ev.Text); ok {
		s.endTool(rest, s.h.proposalBody(kind, id)); return
	}
	if types, rest, ok := tools.ParseRefresh(ev.Text); ok {
		s.endTool(clipText(rest, 2000), ""); for _, typ := range types { s.refreshCard(typ) }; return
	}
	if title, cs, rest, ok := tools.ParseArtifact(ev.Text); ok {
		s.endTool(rest, s.h.artifactBody(title, cs)); return
	}
	s.endTool(clipText(ev.Text, 2000), "")
}

func (s *chatStream) endTool(content string, card template.HTML) {
	s.morphNode(chat.ToolRow(chat.ToolRowProps{
		Tool: s.toolName, Icon: toolIconFile(s.toolName), ID: s.toolID, BodyID: s.toolBody, Content: content,
	}))
	if card != "" {
		s.appendNode(g.El("div", g.Attr("class", "k-inline"), g.Attr("id", s.toolID+"-card"), g.Raw(string(card))))
	}
}
```

### The `#chat` observer (`basm.js:149-154`)

```js
document.addEventListener('DOMContentLoaded', () => {
  const chat = document.getElementById('chat');
  if (!chat) return;
  balaurScrollToLatest();
  new MutationObserver(balaurScrollToLatest).observe(chat, { childList: true, subtree: true });
});
```

### `.k-inline` CSS (`basm.css:1038-1042`)

```css
.k-inline { margin: 10px 0; max-width: 480px; }
.chat .k-inline { margin: 0 0 0 var(--chat-gutter, 124px); max-width: none; }
.chat .k-inline .kcard { padding: 15px 18px; }
```

### Conventions

- Only **uicard** (single card) and **artifact** (cluster) results are
  "artifacts" subject to the cap. **proposal / choices / refresh / plain** tool
  results are NOT capped and keep their current plain markup.
- gomponents in package `web` use `g "maragu.dev/gomponents"`; `template.HTML`
  bodies are embedded with `g.Raw`.
- The chip is **static** (no click handler) — the owner's chosen behavior.

## Commands you will need

| Purpose   | Command                              | Expected on success |
|-----------|--------------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`       | exit 0              |
| Vet       | `go vet ./...`                       | exit 0              |
| Test (pkg)| `go test ./internal/web/...`         | all pass            |
| Test (all)| `go test ./...`                      | all pass            |
| Format    | `gofmt -l internal/`                 | no output           |
| Diff check| `git diff --check`                   | no output           |

## Scope

**In scope** (the only files you should modify):
- `internal/web/cards.go` — add the shared `artifactWrap` + `artifactChip` helpers.
- `internal/web/recap.go` — `messageView` fields, set title/icon in the two artifact branches, the `capArtifacts` post-pass, the `renderMessages` tool case.
- `internal/web/chatstream.go` — thread title/icon through `endTool`; wrap artifacts.
- `internal/web/assets/static/basm.js` — `balaurCapArtifacts` + hook into the `#chat` observer.
- `internal/web/assets/static/basm.css` — chip + collapsed rules.
- `internal/web/show_test.go` (or a new `internal/web/artifact_cap_test.go`) — the cap test.

**Out of scope** (do NOT touch):
- `internal/web/show.go` — it appends through `renderMessages` and inherits the wrapper for free.
- The proposal/choices/refresh handling — not artifacts; leave their markup.
- Any feature/card renderer — the cap wraps whatever they produce.
- Plans 092 / 093 surfaces.

## Git workflow

- Branch: `improve/094-cap-active-artifacts`.
- Commit per logical unit; conventional-commit style, e.g.
  `feat(web): cap active chat artifacts at 3, collapse older to a chip`.
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Add the shared wrapper helpers (`cards.go`)

Add to `internal/web/cards.go` (package `web`):

```go
// activeArtifactCap bounds how many artifacts stay fully rendered in the chat;
// older ones collapse to a static "shown earlier" chip (plan 094). The live
// path enforces the same cap client-side (balaurCapArtifacts in basm.js).
const activeArtifactCap = 3

// artifactChip is the static, non-interactive summary shown when an artifact
// is collapsed. icon is a /static/icons stem ("" → no icon).
func artifactChip(title, icon string) g.Node {
	if title == "" {
		title = "Artifact"
	}
	kids := []g.Node{g.Attr("class", "artifact-chip"), g.Attr("aria-hidden", "true")}
	if icon != "" {
		kids = append(kids, g.El("img", g.Attr("class", "artifact-chip-icon"),
			g.Attr("src", "/static/icons/"+icon+".png"), g.Attr("alt", ""), g.Attr("decoding", "async")))
	}
	kids = append(kids, g.El("span", g.Text(title+" (shown earlier)")))
	return g.El("div", kids...)
}

// artifactWrap wraps a rendered artifact body in the .artifact container with
// its (hidden) chip. collapsed adds .artifact--collapsed (CSS then hides the
// body and reveals the chip). innerID, when set, is placed on the .k-inline
// body (preserves the live path's tool-card id).
func artifactWrap(title, icon string, collapsed bool, innerID string, body template.HTML) g.Node {
	cls := "artifact"
	if collapsed {
		cls += " artifact--collapsed"
	}
	inner := []g.Node{g.Attr("class", "k-inline")}
	if innerID != "" {
		inner = append(inner, g.Attr("id", innerID))
	}
	inner = append(inner, g.Raw(string(body)))
	return g.El("div", g.Attr("class", cls), artifactChip(title, icon), g.El("div", inner...))
}
```

(`cards.go` already imports `g "maragu.dev/gomponents"` and `html/template`.)

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 2: messageView fields + set title/icon in the artifact branches

In `recap.go`:

1. Add to `messageView` (after `CardBody`):
   ```go
   ArtifactTitle     string // non-empty only for uicard/cluster artifacts (drives the chip + the cap)
   ArtifactIcon      string // /static/icons stem for the chip ("" = none)
   ArtifactCollapsed bool   // true → render collapsed (older than the newest activeArtifactCap)
   ```

2. In `messageViews`, set the title/icon in the two artifact branches (and ONLY
   those — not proposal):
   - uicard branch: after `mv.CardBody = h.uicardBody(typ, query)`, add
     `if spec, ok := cards.Get(typ); ok { mv.ArtifactTitle, mv.ArtifactIcon = spec.Label, spec.Icon }`.
   - artifact (cluster) branch: after assigning `mv.CardBody`, add
     `mv.ArtifactTitle = title` (icon stays "").
   - Import `"github.com/alexradunet/balaur/internal/cards"` in `recap.go`
     (it is not yet imported).

3. Before `return out`, run the cap post-pass:
   ```go
   capArtifacts(out)
   return out
   ```
   And define it in `recap.go`:
   ```go
   // capArtifacts marks all but the newest activeArtifactCap artifacts collapsed,
   // in transcript order. Only uicard/cluster artifacts (ArtifactTitle != "")
   // count; proposals/notes are never collapsed. This is the server-side half of
   // the cap (the full reload); the live/cross-fragment half is balaurCapArtifacts
   // in basm.js, which re-applies it over the whole #chat DOM.
   func capArtifacts(views []messageView) {
   	var idx []int
   	for i := range views {
   		if views[i].ArtifactTitle != "" {
   			idx = append(idx, i)
   		}
   	}
   	if len(idx) <= activeArtifactCap {
   		return
   	}
   	for _, i := range idx[:len(idx)-activeArtifactCap] {
   		views[i].ArtifactCollapsed = true
   	}
   }
   ```

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 3: Wrap artifacts in `renderMessages`

In `recap.go` `renderMessages`, replace the tool-case card append so real
artifacts use `artifactWrap` and everything else (proposals) keeps the plain
`.k-inline`:

```go
	if mv.CardBody != "" {
		if mv.ArtifactTitle != "" {
			nodes = append(nodes, artifactWrap(mv.ArtifactTitle, mv.ArtifactIcon, mv.ArtifactCollapsed, "", mv.CardBody))
		} else {
			nodes = append(nodes, g.El("div", g.Attr("class", "k-inline"), g.Raw(string(mv.CardBody))))
		}
	}
```

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 4: Wrap artifacts in the live path (`chatstream.go`)

Change `endTool` to accept the artifact title/icon, and wrap only when a title
is present:

```go
func (s *chatStream) endTool(content string, card template.HTML, artTitle, artIcon string) {
	s.morphNode(chat.ToolRow(chat.ToolRowProps{
		Tool: s.toolName, Icon: toolIconFile(s.toolName), ID: s.toolID, BodyID: s.toolBody, Content: content,
	}))
	if card == "" {
		return
	}
	if artTitle == "" { // proposal etc. — not an artifact; keep the plain inline card
		s.appendNode(g.El("div", g.Attr("class", "k-inline"), g.Attr("id", s.toolID+"-card"), g.Raw(string(card))))
		return
	}
	// New artifacts append expanded; balaurCapArtifacts collapses any displaced
	// older one client-side (the server enforces the cap on reload).
	s.appendNode(artifactWrap(artTitle, artIcon, false, s.toolID+"-card", card))
}
```

Update the callers in `handleToolResult`:
- uicard: `if spec, ok := cards.Get(typ); ok { s.endTool(rest, s.h.uicardBody(typ, query), spec.Label, spec.Icon) } else { s.endTool(rest, s.h.uicardBody(typ, query), typ, "") }; return`
- choices: `s.endTool("choices offered", "", "", "")`
- proposal: `s.endTool(rest, s.h.proposalBody(kind, id), "", "")`
- refresh: `s.endTool(clipText(rest, 2000), "", "", "")`
- cluster: `s.endTool(rest, s.h.artifactBody(title, cs), title, "")`
- plain tail: `s.endTool(clipText(ev.Text, 2000), "", "", "")`

Add `"github.com/alexradunet/balaur/internal/cards"` to `chatstream.go` imports.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0; `go vet ./...` → exit 0.

### Step 5: Client-side cap (`basm.js`)

Add the helper and hook it into the existing `#chat` observer:

```js
// ── Cap active artifacts (plan 094) ────────────────────────────────
// Keep at most ACTIVE_ARTIFACT_CAP artifacts expanded; older ones collapse to
// their static "shown earlier" chip. Runs on load and on every #chat mutation
// (covers sidebar injects, agent card_show, and clusters across all fragments).
var ACTIVE_ARTIFACT_CAP = 3;
function balaurCapArtifacts() {
  var chat = document.getElementById('chat');
  if (!chat) return;
  var arts = chat.querySelectorAll('.artifact');
  var cutoff = arts.length - ACTIVE_ARTIFACT_CAP;
  arts.forEach(function (el, i) {
    el.classList.toggle('artifact--collapsed', i < cutoff);
  });
}
```

Then update the observer block (lines ~149-154) so the callback runs BOTH
helpers, and call `balaurCapArtifacts()` once on load:

```js
document.addEventListener('DOMContentLoaded', () => {
  const chat = document.getElementById('chat');
  if (!chat) return;
  balaurScrollToLatest();
  balaurCapArtifacts();
  new MutationObserver(() => { balaurCapArtifacts(); balaurScrollToLatest(); })
    .observe(chat, { childList: true, subtree: true });
});
```

**Verify**: `grep -n "balaurCapArtifacts" internal/web/assets/static/basm.js`
→ defined and called in the observer.

### Step 6: CSS for the chip + collapsed state

In `basm.css`, near `.k-inline` (after line 1042), add:

```css
/* Artifact cap (plan 094): collapsed artifacts hide the body, show the chip. */
.artifact-chip { display: none; }
.artifact--collapsed > .k-inline { display: none; }
.artifact--collapsed > .artifact-chip {
  display: flex; align-items: center; gap: var(--space-2);
  margin: 10px 0; padding: var(--space-2) var(--space-3);
  font-family: var(--font-mono); font-size: 11px; text-transform: uppercase;
  letter-spacing: .06em; color: var(--ink-muted);
}
.chat .artifact--collapsed > .artifact-chip { margin-left: var(--chat-gutter, 124px); }
.artifact-chip-icon { width: 16px; height: 16px; }
```

(The gutter rule mirrors `.chat .k-inline` at line 1041 so the chip aligns with
where the card sat.)

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 7: Test the server-side cap

Add `internal/web/artifact_cap_test.go`:

```go
package web

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	_ "github.com/alexradunet/balaur/migrations"
)

// TestArtifactCapCollapsesOldest summons 4 artifacts, then loads the home
// transcript and asserts exactly one (the oldest) renders collapsed — the
// newest activeArtifactCap (3) stay expanded.
func TestArtifactCapCollapsesOldest(t *testing.T) {
	app := newWebApp(t)
	for i := 0; i < 4; i++ {
		s := tests.ApiScenario{
			Name: "summon artifact", Method: "GET", URL: "/ui/show/quests",
			TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus: 200,
		}
		s.Test(t)
	}
	s := tests.ApiScenario{
		Name: "home collapses the oldest artifact", Method: "GET", URL: "/",
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus: 200,
		ExpectedContent: []string{"artifact--collapsed", "(shown earlier)"},
		AfterTestFunc: func(tb testing.TB, _ *tests.TestApp, res *http.Response) {
			b, _ := io.ReadAll(res.Body)
			if n := strings.Count(string(b), "artifact--collapsed"); n != 1 {
				tb.Fatalf("want exactly 1 collapsed artifact (4 summoned, cap 3), got %d", n)
			}
		},
	}
	s.Test(t)
}

// TestArtifactCapKeepsFewExpanded: 2 artifacts → none collapsed.
func TestArtifactCapKeepsFewExpanded(t *testing.T) {
	app := newWebApp(t)
	for i := 0; i < 2; i++ {
		s := tests.ApiScenario{
			Name: "summon artifact", Method: "GET", URL: "/ui/show/quests",
			TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus: 200,
		}
		s.Test(t)
	}
	s := tests.ApiScenario{
		Name: "home keeps both expanded", Method: "GET", URL: "/",
		TestAppFactory:     func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus:     200,
		NotExpectedContent: []string{"artifact--collapsed"},
	}
	s.Test(t)
}
```

If `newWebApp` / the `/ui/show` + `/` routes are not both reachable in one
reused app instance (the existing `show_test.go` reuses one app via the same
closure pattern, so this should work), STOP and report rather than reshaping
the harness.

**Verify**: `go test ./internal/web/ -run TestArtifactCap` → all pass.

### Step 8: Full gates

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → all pass
- `gofmt -l internal/` → no output
- `git diff --check` → no output

## Test plan

- `internal/web/artifact_cap_test.go` (new): 4 artifacts → exactly 1 collapsed
  with the "(shown earlier)" chip; 2 artifacts → none collapsed.
- Pattern to follow: `internal/web/show_test.go` (the `/ui/show/quests`
  `ApiScenario` + `AfterTestFunc` reading `res.Body`).
- The live (JS) collapse is not Go-testable; it is exercised by the shared
  `#chat` MutationObserver — note for the reviewer to spot-check in a browser
  (summon 4 artifacts in one session → the oldest collapses without reload).
- Verification: `go test ./...` → all pass.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0 (incl. the two new cap tests)
- [ ] `gofmt -l internal/` prints nothing
- [ ] `grep -n "balaurCapArtifacts" internal/web/assets/static/basm.js` shows it defined AND called in the observer
- [ ] `grep -n "artifact--collapsed" internal/web/assets/static/basm.css` shows the collapse rules
- [ ] `git status` shows only in-scope files modified
- [ ] `plans/readme.md` status row for 094 updated

## STOP conditions

Stop and report back (do not improvise) if:

- The `messageView` / `messageViews` / `renderMessages` / `endTool` code does
  not match the excerpts (drift since `766b7aa`).
- The home transcript (`GET /`) does not include recently-summoned artifacts in
  the test (e.g. history is paginated below 4) — the server-side cap test then
  can't observe them; report so the test approach can be adjusted (e.g. assert
  via `messageViews` directly).
- Adding the `cards` import to `recap.go`/`chatstream.go` creates an import
  cycle (it should not — `internal/cards` is a leaf used widely) — STOP.
- A verification fails twice after a reasonable fix attempt.

## Maintenance notes

- `activeArtifactCap` (Go const in `cards.go`) and `ACTIVE_ARTIFACT_CAP` (JS in
  `basm.js`) must stay in sync — they are the same number expressed in two
  runtimes (Go can't inject into the static JS without a build step, which this
  project avoids). If the cap changes, change both.
- The cap counts **uicard + cluster** artifacts only (`ArtifactTitle != ""`).
  Proposals/choices/notes are deliberately never collapsed.
- The chip is **static** by decision; if a future change makes it clickable to
  re-expand in place, reconcile that with the cap (re-expanding would exceed 3)
  — see the rejected "clickable chip" option in the 2026-06-17 design Q&A.
- The server-side `capArtifacts` runs per `messageViews` batch; the JS
  `balaurCapArtifacts` is the authoritative cross-fragment enforcer over the
  whole `#chat` DOM (covers nudges, day-expand, and live appends together).
- Reviewer should scrutinize: that proposals still render as plain `.k-inline`
  (not wrapped/collapsed), and that the `#chat` observer still scrolls to latest
  (both helpers run in the callback).
- Independent of plans 092 (settings) and 093 (day/quests): this wraps whatever
  an artifact renders, so it composes with their nav-free bodies unchanged.
