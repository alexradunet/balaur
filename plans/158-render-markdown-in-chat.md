# 158 — Render Markdown in chat message bubbles

**Status:** TODO
**Written against commit:** `c57ca72`
**Effort:** S–M
**Risk:** Low–Medium (introduces an HTML trust boundary — sanitization is mandatory, see Step 2)
**Category:** Correctness / UX

---

## Problem (why this matters)

Balaur's assistant replies are written in Markdown (bold, bullet lists, headings,
inline code). The web chat UI renders the reply text as **escaped plain text**, so
the owner literally sees the raw Markdown source:

```
**What I can do for you right now:**

- **Remember & organize:** Save facts...
- **Tasks & reminders:** Add, list, mark done...
```

instead of a bold heading followed by a real bullet list. This is the exact symptom
the owner reported. The fix: convert the assistant's Markdown to sanitized HTML and
render it, for **assistant (balaur) turns only**.

### Root cause (confirmed by reading the code)

`internal/ui/chat/message.go:95` emits the body via `g.Text(content)`, which
HTML-escapes the string — no Markdown is ever interpreted:

```go
// internal/ui/chat/message.go  (messageBody, lines 91–96)
kids := []g.Node{h.Class("cmsg-body")}
if bodyID != "" {
    kids = append(kids, h.ID(bodyID))
}
kids = append(kids, g.Text(content))   // <-- escaped plain text; Markdown shows raw
return h.Div(kids...)
```

Both render paths flow through this one function:

1. **Static / final render** — `Message()` (`message.go:34`) → `messageBody(...)`.
2. **Streaming token morphs** — the SSE gateway `internal/web/chatstream.go`
   accumulates tokens in a buffer and morphs the body by id on every chunk:
   - `emit()` (`chatstream.go:154`): `s.morphNode(chat.MessageBody(s.bodyID, s.buf.String()))`
   - `finalizeBubble()` (`chatstream.go:148`): `s.morphNode(s.balaurBubble(s.buf.String(), false))`
   - `MessageBody()` (`message.go:101`) is the exported morph helper; it calls the
     same `messageBody`.

Because everything funnels through `messageBody`, fixing that one function fixes
both the streamed and finalized renders.

### Why assistant turns ONLY

- The owner's own typed turns (`Role == "user"`) are plain text — rendering them as
  Markdown is needless and would change their bubble. The existing test
  `TestMessageUserDefaultName` and `internal/ui/chat/message_test.go:36` assert the
  user body stays `<div class="cmsg-body">Hi</div>`; **do not break that.**
- Tool-result rows (`internal/ui/chat/toolrow.go`, class `.cmsg-tool`) are a mono
  audit trail, not prose. **Out of scope** — leave them as-is.

---

## Scope

### In scope (the only files you may touch)

- `internal/ui/chat/markdown.go` — **new file**: the Markdown→sanitized-HTML helper.
- `internal/ui/chat/message.go` — thread a `markdown bool` through `messageBody`;
  render Markdown for assistant turns.
- `internal/ui/chat/message_test.go` — update the balaur expectations; add a
  Markdown-rendering test.
- `internal/web/assets/static/basm.css` — prose styles for the rendered-Markdown
  body (`.cmsg-md`).
- `internal/feature/storybook/stories_chat.go` — add a "markdown" variant so the
  storybook documents it.
- `go.mod` / `go.sum` — new dependencies (added automatically by `go get`).

### Explicitly OUT of scope — do NOT touch

- `internal/web/chatstream.go` — needs **no** change; it already calls
  `chat.MessageBody` / `chat.Message`, which you are fixing underneath it. If you
  find yourself editing chatstream.go, STOP — you've misread the design.
- `internal/ui/chat/toolrow.go` and `.cmsg-tool` CSS — tool rows stay plain.
- User-turn rendering — must remain byte-for-byte identical.
- Any other package.

---

## Dependencies to add

Both are pure-Go and build with `CGO_ENABLED=0` (required — see `AGENTS.md`).

- **`github.com/yuin/goldmark`** — CommonMark-compliant Markdown → HTML. Chosen over
  the transitive-but-unused `russross/blackfriday/v2` because goldmark is the
  modern, maintained standard **and its default config escapes raw HTML blocks**
  (it does not pass `<script>` through unless you opt into `html.WithUnsafe()`,
  which you must NOT do). This is the safer default for an LLM→browser boundary.
- **`github.com/microcosm-cc/bluemonday`** — HTML sanitizer. goldmark's default
  blocks raw HTML, but it does **not** sanitize link URLs, so a model could emit
  `[click](javascript:...)`. bluemonday's `UGCPolicy` strips dangerous URL schemes
  and any stray markup. This closes the remaining XSS vector at the trust boundary.

Run (this also updates `go.sum`):

```bash
go get github.com/yuin/goldmark@latest github.com/microcosm-cc/bluemonday@latest
```

---

## Implementation steps

### Step 1 — Create the renderer: `internal/ui/chat/markdown.go`

Create a new file with a package-level goldmark instance and bluemonday policy
(both are safe for concurrent reuse once built — do **not** construct them per call).
The helper returns a `g.Node`. On any conversion error it falls back to escaped
plain text so a bad render can never blank the bubble.

```go
package chat

import (
	"bytes"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	g "maragu.dev/gomponents"
)

// md converts assistant Markdown to HTML. Default goldmark config (no
// html.WithUnsafe) escapes raw HTML blocks, so model-emitted <script> never
// renders; the bluemonday pass below then strips dangerous link schemes
// (javascript:, data:) and any stray markup goldmark passed through. Built once
// and reused — both values are concurrency-safe after construction.
var (
	md     = goldmark.New()
	mdSane = bluemonday.UGCPolicy()
)

// renderMarkdown turns assistant Markdown into a trusted, sanitized HTML node.
// On any error it falls back to escaped plain text — a render failure must never
// blank or unescape the bubble.
//
// ponytail: re-renders the whole accumulated buffer on every streamed token.
// Fine for a local single-owner app with short replies; revisit only if a
// measurement shows it on the hot path.
func renderMarkdown(s string) g.Node {
	var buf bytes.Buffer
	if err := md.Convert([]byte(s), &buf); err != nil {
		return g.Text(s)
	}
	return g.Raw(mdSane.Sanitize(buf.String()))
}
```

**Note on `g.Raw`:** per `AGENTS.md`, `g.Raw` is for "already-trusted, already-rendered
HTML only." That is exactly what this is — HTML produced by goldmark and then run
through bluemonday's sanitizer. This is the one legitimate `g.Raw` use here.

**Verify:**
```bash
CGO_ENABLED=0 go build ./internal/ui/chat/
```
Expected: builds clean, no output. (The function is referenced in Step 2; if you
run this before Step 2 you'll get an "unused" error — that's fine, do Step 2 next.)

---

### Step 2 — Thread Markdown rendering through `message.go`

Edit `internal/ui/chat/message.go`. Make three changes.

**2a.** In `Message()` (around line 63), tell `messageBody` whether this is an
assistant turn. The `user` bool already exists at line 35 (`user := p.Role == "user"`).

Change:
```go
	panel := h.Div(h.Class("cmsg-panel"),
		h.Div(h.Class("cmsg-name"), g.Text(who)),
		messageBody(p.BodyID, p.Content, p.Pending),
	)
```
to:
```go
	panel := h.Div(h.Class("cmsg-panel"),
		h.Div(h.Class("cmsg-name"), g.Text(who)),
		messageBody(p.BodyID, p.Content, p.Pending, !user),
	)
```

**2b.** Update `messageBody`'s signature and body (lines 83–97). Add the
`markdown bool` parameter. When `markdown` is true, add the `cmsg-md` class and
render via `renderMarkdown`; otherwise keep `g.Text` exactly as today (this is the
user-turn path, which must not change). The pending/thinking branch is unchanged.

Replace the whole function:
```go
func messageBody(bodyID, content string, pending bool) g.Node {
	if pending && content == "" {
		thinking := h.Span(h.Class("thinking thinking-dots"), g.Text("thinking"))
		if bodyID == "" {
			return thinking
		}
		return h.Div(h.Class("cmsg-body"), h.ID(bodyID), thinking)
	}
	kids := []g.Node{h.Class("cmsg-body")}
	if bodyID != "" {
		kids = append(kids, h.ID(bodyID))
	}
	kids = append(kids, g.Text(content))
	return h.Div(kids...)
}
```
with:
```go
func messageBody(bodyID, content string, pending, markdown bool) g.Node {
	if pending && content == "" {
		thinking := h.Span(h.Class("thinking thinking-dots"), g.Text("thinking"))
		if bodyID == "" {
			return thinking
		}
		return h.Div(h.Class("cmsg-body"), h.ID(bodyID), thinking)
	}
	cls := "cmsg-body"
	if markdown {
		cls = "cmsg-body cmsg-md"
	}
	kids := []g.Node{h.Class(cls)}
	if bodyID != "" {
		kids = append(kids, h.ID(bodyID))
	}
	if markdown {
		kids = append(kids, renderMarkdown(content))
	} else {
		kids = append(kids, g.Text(content))
	}
	return h.Div(kids...)
}
```

**2c.** Update the exported `MessageBody` (lines 101–103) — the streaming morph
helper. Streaming only ever targets assistant bubbles, so pass `markdown = true`:
```go
func MessageBody(bodyID, content string) g.Node {
	return messageBody(bodyID, content, false, true)
}
```

**Verify:**
```bash
CGO_ENABLED=0 go build ./internal/ui/chat/
go vet ./internal/ui/chat/
```
Expected: both clean, no output.

---

### Step 3 — Update and extend `message_test.go`

Editing `message.go` changes the assistant body HTML (goldmark wraps prose in
`<p>…</p>` and adds a trailing newline; the body div gains `cmsg-md`). Update the
two assistant-turn expectations and add a Markdown test. **Leave the user test
(`TestMessageUserDefaultName`) untouched — it must still pass unchanged**, proving
user turns stay plain.

**3a.** In `TestMessageBalaur` (lines 14–20), the body assertion
```go
		`<div class="cmsg-body">Hello there.</div>`,
```
becomes (goldmark wraps the line in a paragraph and emits a trailing `\n`):
```go
		`<div class="cmsg-body cmsg-md"><p>Hello there.</p>`,
```
> If the exact string differs, run the test once (Step 3c) and copy the real
> rendered fragment from the failure output — goldmark's exact whitespace is the
> source of truth, not this plan.

**3b.** Add a new test that proves Markdown is actually rendered (not escaped) for
assistant turns. Append to the file:
```go
func TestMessageBalaurMarkdown(t *testing.T) {
	got := render(t, chat.Message(chat.MessageProps{
		Role: "balaur", Who: "Balaur", AvatarSrc: "/static/crest.png",
		Content: "**bold** and a list:\n\n- one\n- two",
	}))
	for _, want := range []string{
		`<strong>bold</strong>`,
		`<ul>`,
		`<li>one</li>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("balaur markdown not rendered, missing %q in: %s", want, got)
		}
	}
	// Markdown source must NOT survive as literal text.
	if strings.Contains(got, "**bold**") {
		t.Errorf("raw markdown leaked into output: %s", got)
	}
}
```

**3c. Verify:**
```bash
go test ./internal/ui/chat/
```
Expected: `ok  github.com/alexradunet/balaur/internal/ui/chat`. If `TestMessageBalaur`
fails on whitespace, paste the real fragment from the failure into the 3a assertion
and re-run.

---

### Step 4 — Prose CSS for the rendered Markdown body

Edit `internal/web/assets/static/basm.css`. The existing `.cmsg-body` rule
(line 3176) uses `white-space: pre-wrap`, which is correct for plain text but adds
spurious blank lines once the body contains real block HTML (goldmark emits `\n`
between blocks). So scope prose styling to the new `.cmsg-md` modifier and turn
pre-wrap off there.

Add this block immediately **after** the `.cmsg-body` rule at line 3176 (before the
`/* Tool call ... */` comment at line 3178). Reuse the existing design tokens
already used nearby (`--font-mono`, `--surface-2`, `--parch-edge`, `--ink-muted`,
`--space-2`); do not invent new colors.

```css
/* Rendered Markdown body (assistant turns). Plain-text bodies keep .cmsg-body's
   pre-wrap; rendered prose uses normal flow so goldmark's inter-block newlines
   don't open blank gaps. */
.cmsg-md { white-space: normal; }
.cmsg-md > :first-child { margin-top: 0; }
.cmsg-md > :last-child { margin-bottom: 0; }
.cmsg-md p { margin: 0 0 .6em; }
.cmsg-md ul, .cmsg-md ol { margin: 0 0 .6em; padding-left: 1.4em; }
.cmsg-md li { margin: .15em 0; }
.cmsg-md li > p { margin: 0; }
.cmsg-md h1, .cmsg-md h2, .cmsg-md h3 { margin: .6em 0 .3em; font-size: 1.05em; line-height: 1.3; }
.cmsg-md code { font-family: var(--font-mono); font-size: .9em; background: var(--surface-2); border: 1px solid var(--parch-edge); border-radius: 3px; padding: 1px 4px; }
.cmsg-md pre { background: var(--surface-2); border: 1px solid var(--parch-edge); border-radius: 4px; padding: var(--space-2); overflow-x: auto; }
.cmsg-md pre code { background: none; border: 0; padding: 0; }
.cmsg-md a { color: inherit; text-decoration: underline; }
.cmsg-md blockquote { margin: 0 0 .6em; padding-left: .8em; border-left: 3px solid var(--parch-edge); color: var(--ink-muted); }
```

> CSS is not unit-tested. Verify visually in Step 6. If a token name above does not
> exist in basm.css, grep the file for the closest existing one (the `.cmsg-*` rules
> around line 3170–3199 list the real token names) and use that instead.

---

### Step 5 — Add a storybook variant

Edit `internal/feature/storybook/stories_chat.go`. In `chatmessageStory()`
(line 16), add a "markdown" variant to the `Variants` slice (after the "balaur"
variant at line 21) so the storybook renders and documents Markdown support:

```go
				{"markdown", chat.Message(chat.MessageProps{Role: "balaur", Who: "Balaur", AvatarSrc: "/static/crest.png", Content: "Here's what I can do:\n\n- **Remember** facts you approve\n- **Track** tasks and reminders\n- Show `live cards` in the panel"})},
```

Also update the `Content` prop doc (line 30) from `"The spoken line."` to note
Markdown rendering on assistant turns, e.g.:
```go
				{"Content", "string", "—", "The spoken line. Rendered as Markdown for balaur turns (escaped plain text for the owner)."},
```

**Verify:**
```bash
CGO_ENABLED=0 go build ./internal/feature/storybook/
```
Expected: clean.

---

### Step 6 — Full verification gate

Run the full suite and build (per `AGENTS.md` — never push red, CGO must be off):

```bash
go vet ./...
go test ./...
CGO_ENABLED=0 go build ./...
git diff --check
```
All must pass. Then visually confirm (optional but recommended): `make run`, open
`/storybook`, find Chat → Message → the "markdown" variant, and confirm bold text,
a real bullet list, and inline code render (not raw `**`/`-`/backticks). Also send a
real chat message that includes a list and confirm it renders live and streams
without raw markup surviving.

---

## Done criteria (machine-checkable)

- [ ] `go test ./internal/ui/chat/` passes, including the new `TestMessageBalaurMarkdown`.
- [ ] `TestMessageUserDefaultName` still passes **unchanged** (user turns stay plain).
- [ ] `grep -n 'cmsg-md' internal/web/assets/static/basm.css` returns the new rules.
- [ ] `go vet ./... && go test ./... && CGO_ENABLED=0 go build ./...` all succeed.
- [ ] `git diff --check` is clean.
- [ ] `internal/web/chatstream.go` is **unchanged** (`git diff --stat` does not list it).
- [ ] `go.mod` now requires `github.com/yuin/goldmark` and
      `github.com/microcosm-cc/bluemonday`.

---

## Test plan

- **New unit test:** `TestMessageBalaurMarkdown` in `internal/ui/chat/message_test.go`
  (pattern: the existing `TestMessageBalaur`) — asserts `<strong>`, `<ul>`, `<li>`
  appear and that `**bold**` does NOT survive as literal text.
- **Regression:** existing `TestMessageBalaur` updated for the `<p>`/`cmsg-md`
  wrapping; `TestMessageUserDefaultName` and `TestMessagePending` must pass with no
  changes to their assertions.
- **Security smoke (optional, recommended):** add a quick assertion (can live in the
  same new test or a sibling) that
  `chat.Message({Role:"balaur", Content:"[x](javascript:alert(1))"})` renders a link
  WITHOUT a `javascript:` href — confirms bluemonday is in the path. If you add it,
  derive the exact expected string from a first run rather than guessing.

---

## Maintenance note (for the reviewer)

- The trust boundary is `renderMarkdown` in `internal/ui/chat/markdown.go`. **Never**
  add `goldmark.WithRendererOptions(html.WithUnsafe())` and never drop the bluemonday
  pass — either change would let model output inject live HTML into the owner's
  session. This is the one place `g.Raw` is justified in the chat package; any other
  `g.Raw` on model/user text is a bug.
- Re-rendering Markdown on every streamed token is intentional and acceptable for a
  local single-owner app (see the `ponytail:` comment). If profiling ever shows it on
  the hot path, the cheaper fix is to stream plain text and render Markdown only in
  `finalizeBubble` — but that reintroduces a brief raw-markup flash mid-stream, so
  only do it with a measurement in hand.
- Tool-result rows (`.cmsg-tool`) deliberately stay plain mono text. If a future
  change wants Markdown there too, it's a separate decision — tool output is an audit
  trail, not prose.

---

## Escape hatches (STOP and report instead of improvising)

- If `go get` fails with "certificate signed by unknown authority" inside a sandbox,
  that's the TLS-intercepting proxy — apply the GOPROXY shim per
  `docs/hyperagent-sandbox.md` (do NOT weaken `GOSUMDB`), then retry. If it still
  fails, STOP and report.
- If you discover `internal/web/chatstream.go` does NOT route through
  `chat.Message` / `chat.MessageBody` (i.e. it renders bodies some other way), STOP
  — the plan's central assumption is wrong; report what you found.
- If updating `TestMessageBalaur`'s expected fragment requires more than swapping the
  one body string (e.g. the surrounding structure changed unexpectedly), STOP and
  report rather than rewriting the test wholesale.
