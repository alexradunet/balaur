# Plan 114: The chatbar + model/head switcher fragments render via gomponents instead of `chat_bar` / `model_switcher` / `head_switcher` templates

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm its expected result before moving on. If a
> "STOP conditions" item occurs, stop and report — do not improvise. When done,
> update this plan's status row in `plans/readme.md` unless a reviewer told you
> they maintain the index.
>
> **Drift check (run first)**: `git diff --stat ea79dae..HEAD -- internal/web/home.go internal/web/models.go internal/web/heads.go internal/ui/chat/dock.go web/templates/home.html internal/self/knowledge.md`
> If any in-scope file changed since this plan was written, compare the "Current
> state" excerpts against the live code; on a mismatch, treat it as a STOP
> condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED
- **Depends on**: none (independent of 111–113, 115; all of 111–115 must land before 116/117)
- **Category**: migration / tech-debt
- **Planned at**: commit `0dd2457`, 2026-06-19 — **refreshed 2026-06-22 against `ea79dae`; see "## Refresh" below**

## Refresh (2026-06-22, against `ea79dae`)

Still **valid and unstarted**. Live: `chat_bar` → `models.go:113`, `head_switcher`
→ `heads.go:30`. **`model_switcher` is DEAD** (only invoked via `{{template}}`
inside `home.html`, never `ExecuteTemplate`'d) — drop it, do not port. Anchors:
`patchChatbar` → `models.go:111-128` (excerpt byte-accurate); `homeData`/`headChoice`
`models.go:25-46`/:49 intact; `setActiveHead` `heads.go:27-33` unchanged (preserve
the `#ucard-heads` patch right after, at heads.go:34-39). **Resolve the open
`h.FormEl` vs `h.Form` question: use `h.Form(...)`** — `knowledgecards/memory.go:121`
uses `h.Form` in this repo's gomponents (change the Step-1 draft from `h.FormEl(`
to `h.Form(`). Docs anchors for Step 5: knowledge.md `:161-163`, `dock.go:45`,
`stories_chat.go:185`. Note: `models.go:534` `template.HTMLEscapeString` does NOT
match the `ExecuteTemplate` grep, so the done-criterion stays clean. The
homeData-fields / Switchers-slot STOP conditions are verified clear at HEAD
(home.go still passes only Convo+Composer).

## Why this matters

Three fragments in `web/templates/home.html` — `chat_bar`, `model_switcher`
(nested inside `chat_bar`), and `head_switcher` — are still `ExecuteTemplate`'d
at runtime as SSE patch targets, the last templates in `home.html`. Porting them
to gomponents node builders removes two more `ExecuteTemplate` callers and lets
plan 117 delete `home.html`. This is a behavior-preserving port: the handlers
keep patching the same element ids (`#chatbar` outer, `#head-switcher` outer)
with the same content.

**Pre-existing state to preserve (not fix here):** `home.go`'s `homePage` does
**not** pass `chat.Dock`'s `Switchers` slot, so `#chatbar` is not currently
rendered into the home DOM at page load — the switcher integration was deferred
(see the `Switchers` slot comment in `dock.go:31,44-45`). Whether to wire the
chatbar into the dock is a separate product decision **out of scope** here. This
plan only swaps the renderer; the handlers behave exactly as today (blind outer
patches that are no-ops if the target is absent).

## Current state

- `web/templates/home.html` defines the three fragments (data = `homeData`):
  ```html
  {{define "chat_bar"}}
  <div class="chatbar chatbar-slim" id="chatbar"
       {{if not .ChatReady}}data-on:interval__duration.2s="@get('/ui/chatbar')"{{end}}>
    {{template "head_switcher" .}}
    {{template "model_switcher" .}}
  </div>
  {{end}}

  {{define "model_switcher"}}
  <section class="model-switcher" aria-label="Model">
    <div class="model-switcher-head">
      <span class="model-switcher-kicker">Model</span>
      {{if .ActiveModel}}<span class="model-current">{{.ActiveModel}}</span>{{end}}
      <a class="model-switcher-manage" href="/ui/show/settings?section=models">Manage models →</a>
    </div>
    {{if not .ChatReady}}
      <div class="model-switcher-empty">
        <span>{{if .ModelError}}{{.ModelError}}{{else}}No model is ready yet.{{end}}</span>
        <a href="/ui/show/settings?section=models">Set up a model →</a>
      </div>
    {{end}}
    <div class="chatbar-profile-link">
      <span class="balaur-avatar balaur-avatar-soul" aria-hidden="true">
        <img class="px" src="{{.SoulAvatarURL}}" alt="" decoding="async">
      </span>
      <a href="/ui/show/settings?section=profile" class="chatbar-profile-href">Your avatar &amp; profile →</a>
    </div>
  </section>
  {{end}}

  {{define "head_switcher"}}
  <section class="head-switcher" id="head-switcher" aria-label="Head">
    <span class="model-switcher-kicker">Head</span>
    <span class="head-switcher-current">{{.ActiveHeadName}}</span>
    <ul class="head-switcher-list">
      {{range .HeadChoices}}
      <li>
        <form data-on:submit__prevent="@post('/ui/heads/active', {contentType:'form'})">
          <input type="hidden" name="head" value="{{.ID}}">
          <button type="submit" class="head-switcher-choice{{if .Active}} head-switcher-choice-active{{end}}"
                  data-attr:disabled="$streaming"
                  {{if .Active}}aria-current="true"{{end}}>
            <img class="px" src="{{.AvatarURL}}" alt="" decoding="async">
            <span>{{.Name}}</span>
          </button>
        </form>
      </li>
      {{end}}
    </ul>
  </section>
  {{end}}
  ```

- `homeData` fields used (defined in `internal/web/models.go:25-46`): `ChatReady`
  bool, `ActiveModel` string, `ModelError` string, `SoulAvatarURL` string,
  `ActiveHeadName` string, `HeadChoices []headChoice`. `headChoice`
  (`models.go:49`) = `{ID, Name, AvatarURL string; Active bool}`.

- `internal/web/models.go:116-133` — `patchChatbar` executes `chat_bar`:
  ```go
  func (h *handlers) patchChatbar(sse *datastar.ServerSentEventGenerator, data homeData) error {
  	var b strings.Builder
  	if err := h.tmpl.ExecuteTemplate(&b, "chat_bar", data); err != nil {
  		return err
  	}
  	if err := sse.PatchElements(b.String(),
  		datastar.WithSelectorID("chatbar"), datastar.WithModeOuter()); err != nil {
  		return nil // client gone
  	}
  	if data.ChatReady {
  		var d strings.Builder
  		if err := composerNode(data).Render(&d); err != nil {
  			return err
  		}
  		_ = sse.PatchElements(d.String(), datastar.WithSelectorID("chat-draft"), datastar.WithModeOuter())
  	}
  	return nil
  }
  ```

- `internal/web/heads.go:27-33` — `setActiveHead` executes `head_switcher`:
  ```go
  	sse := datastar.NewSSE(e.Response, e.Request)
  	var sw strings.Builder
  	if err := h.tmpl.ExecuteTemplate(&sw, "head_switcher", data); err != nil {
  		return e.InternalServerError("rendering head switcher", err)
  	}
  	_ = sse.PatchElements(sw.String(), datastar.WithSelectorID("head-switcher"), datastar.WithModeOuter())
  ```

- **Conventions to match**:
  - Web-local chrome node builders: `internal/web/home.go:24` `commandPaletteNode()`
    and `home.go:44` `composerNode(d homeData) g.Node`. Put the new builders here.
  - gomponents: `g "maragu.dev/gomponents"` (already in `home.go`) and
    `h "maragu.dev/gomponents/html"` (add to `home.go`).
  - Emit Datastar attributes verbatim with `g.Attr("data-on:...", expr)` /
    `g.Attr("data-attr:disabled", "$streaming")` to match the templates exactly.
  - Conditional attributes (the `data-on:interval` only when `!ChatReady`, the
    `aria-current` only when active): build the attribute slice conditionally —
    see `dock.go`'s `Dock` which appends nodes into a `[]g.Node` slice.
  - `&amp;` in the template is HTML-escaping of `&` — in gomponents write the
    literal `"Your avatar & profile →"` as `g.Text`; gomponents escapes it.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Tests | `go test ./...` | all pass, exit 0 |
| Format check | `gofmt -l internal/` | empty output |
| Whitespace | `git diff --check` | no output |

Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need the
GOPROXY shim — see `docs/hyperagent-sandbox.md`.

## Scope

**In scope**:
- `internal/web/home.go` (add the node builders)
- `internal/web/models.go` (repoint `patchChatbar`)
- `internal/web/heads.go` (repoint `setActiveHead`)
- `internal/web/home_test.go` / `internal/web/heads_test.go` (assertions; see Test plan)
- `internal/ui/chat/dock.go` (comment-only truth fix — see Step 5)
- `internal/feature/storybook/stories_chat.go` (comment-only truth fix — Step 5)
- `internal/self/knowledge.md` (line ~160 truth fix — Step 5)

**Out of scope** (do NOT touch):
- `web/templates/home.html` — plan 117 deletes it.
- Wiring `chat.Dock`'s `Switchers` slot into `homePage` — pre-existing deferred
  decision; do not start rendering `#chatbar` on the home page here.
- `composerNode` markup, the `chat-draft` patch, the model-download handlers in
  `models.go` — unrelated.

## Git workflow

- Branch: `improve/114-chatbar-switchers-gomponents`.
- One commit; conventional message, e.g.
  `refactor(web): render chatbar/head-switcher via gomponents (plan 114)`.
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Add the three node builders to `home.go`

Add to `internal/web/home.go` (faithful ports; preserve the conditional
`data-on:interval`, the empty-state branch, and the per-choice active state):
```go
// chatBarNode renders the slim chatbar (#chatbar) — the head + model switchers.
// Port of the chat_bar template; patchChatbar outer-patches #chatbar with it.
// While no model is ready it carries the 2s self-refresh poll; the ready chatbar
// drops the interval, so polling stops.
func chatBarNode(d homeData) g.Node {
	attrs := []g.Node{h.Class("chatbar chatbar-slim"), h.ID("chatbar")}
	if !d.ChatReady {
		attrs = append(attrs, g.Attr("data-on:interval__duration.2s", "@get('/ui/chatbar')"))
	}
	attrs = append(attrs, headSwitcherNode(d), modelSwitcherNode(d))
	return h.Div(attrs...)
}

// modelSwitcherNode renders the model panel (nested in the chatbar). Port of
// model_switcher.
func modelSwitcherNode(d homeData) g.Node {
	head := []g.Node{
		h.Span(h.Class("model-switcher-kicker"), g.Text("Model")),
	}
	if d.ActiveModel != "" {
		head = append(head, h.Span(h.Class("model-current"), g.Text(d.ActiveModel)))
	}
	head = append(head, h.A(h.Class("model-switcher-manage"),
		h.Href("/ui/show/settings?section=models"), g.Text("Manage models →")))

	kids := []g.Node{
		g.Attr("aria-label", "Model"),
		h.Div(h.Class("model-switcher-head"), g.Group(head)),
	}
	if !d.ChatReady {
		msg := "No model is ready yet."
		if d.ModelError != "" {
			msg = d.ModelError
		}
		kids = append(kids, h.Div(h.Class("model-switcher-empty"),
			h.Span(g.Text(msg)),
			h.A(h.Href("/ui/show/settings?section=models"), g.Text("Set up a model →")),
		))
	}
	kids = append(kids, h.Div(h.Class("chatbar-profile-link"),
		h.Span(h.Class("balaur-avatar balaur-avatar-soul"), g.Attr("aria-hidden", "true"),
			h.Img(h.Class("px"), h.Src(d.SoulAvatarURL), h.Alt(""), g.Attr("decoding", "async"))),
		h.A(h.Href("/ui/show/settings?section=profile"), h.Class("chatbar-profile-href"),
			g.Text("Your avatar & profile →")),
	))
	return h.Section(append([]g.Node{h.Class("model-switcher")}, kids...)...)
}

// headSwitcherNode renders the dock persona picker (#head-switcher). Port of
// head_switcher; setActiveHead outer-patches #head-switcher with it.
func headSwitcherNode(d homeData) g.Node {
	choices := make([]g.Node, 0, len(d.HeadChoices))
	for _, c := range d.HeadChoices {
		btnClass := "head-switcher-choice"
		if c.Active {
			btnClass += " head-switcher-choice-active"
		}
		btnAttrs := []g.Node{
			h.Type("submit"), h.Class(btnClass),
			g.Attr("data-attr:disabled", "$streaming"),
		}
		if c.Active {
			btnAttrs = append(btnAttrs, g.Attr("aria-current", "true"))
		}
		btnAttrs = append(btnAttrs,
			h.Img(h.Class("px"), h.Src(c.AvatarURL), h.Alt(""), g.Attr("decoding", "async")),
			h.Span(g.Text(c.Name)),
		)
		choices = append(choices, h.Li(
			h.FormEl(g.Attr("data-on:submit__prevent", "@post('/ui/heads/active', {contentType:'form'})"),
				h.Input(h.Type("hidden"), h.Name("head"), h.Value(c.ID)),
				h.Button(btnAttrs...),
			),
		))
	}
	return h.Section(h.Class("head-switcher"), h.ID("head-switcher"), g.Attr("aria-label", "Head"),
		h.Span(h.Class("model-switcher-kicker"), g.Text("Head")),
		h.Span(h.Class("head-switcher-current"), g.Text(d.ActiveHeadName)),
		h.Ul(h.Class("head-switcher-list"), g.Group(choices)),
	)
}
```
Add `h "maragu.dev/gomponents/html"` to `home.go`. **Note**: gomponents'
`<form>` helper is `h.FormEl` (the function `h.Form` is the CSS form attribute in
some versions — confirm by reading how forms are built in
`internal/feature/knowledgecards/memory.go:121` which uses `h.Form(...)`; use the
same identifier that package uses). Pick whichever the codebase already uses for
`<form>` elements and stay consistent.

**Verify**: `go build ./internal/web/` → exit 0.

### Step 2: Repoint `patchChatbar`

In `internal/web/models.go`, replace the `chat_bar` ExecuteTemplate:
```go
func (h *handlers) patchChatbar(sse *datastar.ServerSentEventGenerator, data homeData) error {
	if err := sse.PatchElements(renderNodeHTML(chatBarNode(data)),
		datastar.WithSelectorID("chatbar"), datastar.WithModeOuter()); err != nil {
		return nil // client gone
	}
	if data.ChatReady {
		_ = sse.PatchElements(renderNodeHTML(composerNode(data)),
			datastar.WithSelectorID("chat-draft"), datastar.WithModeOuter())
	}
	return nil
}
```

**Verify**: `go build ./internal/web/` → exit 0.

### Step 3: Repoint `setActiveHead`

In `internal/web/heads.go`, replace the `head_switcher` ExecuteTemplate:
```go
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(renderNodeHTML(headSwitcherNode(data)),
		datastar.WithSelectorID("head-switcher"), datastar.WithModeOuter())
```
(Delete the `var sw strings.Builder` + `ExecuteTemplate` lines. If `strings` is
now unused in `heads.go`, drop the import — `go build` will tell you.)

**Verify**: `go build ./internal/web/` → exit 0.

### Step 4: Build, vet, test

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → all pass, exit 0
- `gofmt -l internal/` → empty

### Step 5: Truth-sync the stale "still a template" comments

These three comments become false the moment this lands; fix them in the same
commit (the repo requires self-knowledge to track architecture changes):

- `internal/self/knowledge.md` (~line 160): the sentence "The head/model switcher
  fragments (chat_bar/model_switcher/head_switcher) remain as legacy template SSE
  patch targets for patchChatbar and setActiveHead — deferred from plan 084."
  Reword to: the switcher fragments now render via gomponents node builders
  (`chatBarNode`/`headSwitcherNode` in `internal/web/home.go`); they are still SSE
  patch targets for `patchChatbar`/`setActiveHead`. Keep it one sentence.
- `internal/ui/chat/dock.go:31` and `:44-45`: drop "still executed as templates"
  / "still a template fragment" wording from the `Switchers` slot comments — the
  caller-injected node is now a gomponents node like the others. Keep the
  "deferred / may be nil" note (the slot is still unwired — that part is true).
- `internal/feature/storybook/stories_chat.go:185`: the `Switchers` prop doc
  `"still a template fragment — deferred from this plan"` → drop "a template
  fragment"; keep the "deferred" note.

**Verify**:
- `grep -rn 'chat_bar\|model_switcher\|head_switcher' internal/` returns matches
  **only** in test files / `internal/self/knowledge.md` prose (no `.go` runtime
  `ExecuteTemplate` of those names).
- `go test ./...` → all pass.

## Test plan

- Add render assertions (model after `internal/web/home_test.go`'s existing
  `*handlers` markup tests):
  - `chatBarNode(homeData{ChatReady:false})` → contains
    `data-on:interval__duration.2s="@get('/ui/chatbar')"` and `id="chatbar"`.
  - `chatBarNode(homeData{ChatReady:true})` → does **not** contain
    `data-on:interval`.
  - `modelSwitcherNode` with `ActiveModel:""` and `ChatReady:false` →
    `model-switcher-empty` present; with `ActiveModel:"x"`/`ChatReady:true` →
    `model-current` present, no empty block.
  - `headSwitcherNode` with two `HeadChoices`, one `Active` → exactly one
    `head-switcher-choice-active` and one `aria-current="true"`; both `<form>`s
    `@post('/ui/heads/active', ...)`.
- `templates_test.go` still parses `home.html`; tests there that execute
  `chat_bar`/`head_switcher` stay passing (the template file still exists) —
  leave them; plan 117 removes them with the file.
- Verification: `go test ./internal/web/...` → all pass.

## Done criteria

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0; new switcher node tests exist and pass
- [ ] `gofmt -l internal/` prints nothing
- [ ] `git diff --check` prints nothing
- [ ] `grep -rn 'ExecuteTemplate' internal/web/models.go internal/web/heads.go` returns **no** matches
- [ ] `grep -rn '"chat_bar"\|"model_switcher"\|"head_switcher"' internal/web/*.go` (excluding `_test.go`) returns **no** matches
- [ ] `web/templates/home.html` still exists (untouched)
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report (do not improvise) if:

- The "Current state" excerpts don't match the live code (drift since `0dd2457`).
- `homeData` / `headChoice` no longer have the fields above.
- The drift check shows `homePage` now **does** pass `chat.Dock`'s `Switchers`
  slot — then `#chatbar` IS rendered at page load and you should wire
  `chatBarNode(dock)` into that `Switchers` value too (report this; it changes
  the scope).
- You cannot determine the correct gomponents `<form>` element helper — report
  what `knowledgecards` uses and match it.

## Maintenance notes

- After this lands, `home.html` is fully dead and removed in plan 117.
- The `chat.Dock` `Switchers` slot remains **unwired** in `homePage` (pre-existing
  deferral). Wiring `chatBarNode(dock)` into it — so the chatbar actually renders
  on the home page — is a separate product change, not part of the migration.
- Reviewer: confirm the patched ids (`#chatbar`, `#head-switcher`) and the
  `data-attr:disabled="$streaming"` binding are byte-identical to the template,
  and that the `chat-draft` composer patch in `patchChatbar` is unchanged.
