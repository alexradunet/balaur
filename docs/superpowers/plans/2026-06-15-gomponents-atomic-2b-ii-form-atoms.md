# Gomponents Atomic — Form Atoms (Plan 02b-ii) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add three form atoms to `internal/ui` — `TextField`, `Select`, `Toggle` — each a typed gomponents atom with new tokenized `basm.css` rules, sharing one form shell, and showcase them at `/storybook`.

**Architecture:** `TextField` and `Select` share ONE `.prim-field` base + `.prim-control`/`.prim-label`/`.prim-msg` shell (defined once in the TextField task; Select adds only its specializers). All three are pure server-rendered controls: the atom owns no behavior — value/state live in the DOM (uncontrolled input, `<option selected>`, `aria-checked`) and reach the server when the caller submits the enclosing `<form>` or wires a Datastar action through the variadic passthrough. `Toggle` is a real `<button role="switch">` (keyboard-operable). Atoms use the qualified `h "maragu.dev/gomponents/html"` import.

**Tech Stack:** Go, gomponents (`maragu.dev/gomponents` + `.../html`), vanilla `basm.css`.

**Scope:** Plan 02b-ii of the migration (Phase 2b). Nav (`Tabs`/`Breadcrumb`/`Pagination`), `List`, and the text helpers follow in later sub-plans.

**Design provenance:** Go atoms + CSS extracted from the export and vetted by an adversarial a11y-weighted review (workflow `wf_08b5a603-c40`). The critic caught and these tasks already incorporate: (a) the **shared `.prim-field` collision** — TextField and Select now share ONE base; (b) the **unified `prim-control`/`prim-label`/`prim-msg` taxonomy**; (c) **`--bevel-in`** as the single inset-shadow token for both fields; (d) **form a11y** — TextField wires `aria-describedby` to its error when an `ID` is given; Toggle is `role="switch"` with `aria-checked`, and gets an accessible name via `aria-labelledby` (when `ID`+`Label`) or `aria-label` (when `Label` only — id-free, no duplicate-id risk); (e) the dead `toggle-on` class removed (state lives in `[aria-checked]`).

**Conventions for every task:**
- Package `ui` files: `g "maragu.dev/gomponents"` + `h "maragu.dev/gomponents/html"` (qualified; no dot-import).
- New CSS goes at the **very end** of `internal/web/assets/static/basm.css`. Tokenized — no raw hex; single-dash class names.
- After each task: `go test ./...` (full suite), `CGO_ENABLED=0 go build ./...`, `go vet ./...`.
- Atom tests are `package ui_test` and use the shared `render(t, node)` helper.

---

## File Structure

**Created:** `internal/ui/textfield.go`+`_test.go`, `internal/ui/select.go`+`_test.go`, `internal/ui/toggle.go`+`_test.go`.
**Modified:** `internal/web/assets/static/basm.css` (append form shell + select + toggle blocks); `internal/feature/storybook/storybook.go`+`_test.go` (showcase).

---

## Task 1: `ui.TextField` atom + the shared form shell

Adds the labelled text input AND the shared `.prim-control`/`.prim-label`/
`.prim-field`/`.prim-msg` base that Select (Task 2) reuses.

**Files:** Create `internal/ui/textfield.go`, `internal/ui/textfield_test.go`; modify `basm.css`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/textfield_test.go`:
```go
package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestTextFieldBasic(t *testing.T) {
	got := render(t, ui.TextField(ui.FieldProps{Label: "Name", Placeholder: "Your name", Name: "name"}))
	want := `<label class="prim-control"><span class="prim-label">Name</span><input class="prim-field prim-field-text" type="text" placeholder="Your name" name="name"></label>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}

func TestTextFieldError(t *testing.T) {
	got := render(t, ui.TextField(ui.FieldProps{Label: "Token", Type: "password", ID: "tok", Name: "token", Error: "Required."}))
	for _, want := range []string{
		`class="prim-field prim-field-text prim-field-error"`,
		`type="password"`,
		`id="tok"`,
		`aria-invalid="true"`,
		`aria-describedby="tok-msg"`,
		`<span class="prim-msg prim-msg-error" id="tok-msg">Required.</span>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("textfield missing %q in: %s", want, got)
		}
	}
}

func TestTextFieldHintNoError(t *testing.T) {
	got := render(t, ui.TextField(ui.FieldProps{Label: "Email", Hint: "On your box only."}))
	if !strings.Contains(got, `<span class="prim-msg">On your box only.</span>`) {
		t.Errorf("hint missing: %s", got)
	}
	if strings.Contains(got, "prim-msg-error") || strings.Contains(got, "aria-invalid") {
		t.Errorf("hint-only field must not be an error: %s", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run TestTextField -v` — Expected: FAIL (`undefined: ui.TextField`/`ui.FieldProps`).

- [ ] **Step 3: Implement the atom**

Create `internal/ui/textfield.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// FieldProps configures a TextField atom. All fields optional; the zero value
// renders a bare unlabelled text input. Value seeds the uncontrolled input;
// Name is the form key. ID, when set, lets the atom wire aria-describedby to
// the message span for full screen-reader association. Error replaces Hint,
// recolors the border (prim-field-error), and sets aria-invalid.
type FieldProps struct {
	Label       string
	Type        string // default "text"
	Placeholder string
	Value       string
	Name        string
	ID          string // optional; enables aria-describedby wiring
	Hint        string
	Error       string
	Disabled    bool
}

// TextField renders the Hearthwood labelled text input: a <label class=prim-control>
// wrapping a mono label span, an <input class="prim-field prim-field-text">, and an
// optional hint/error message. Pure render — callers wire behavior by submitting the
// enclosing <form> (Datastar action on the form) and may pass extra attributes
// (data-bind, autofocus, …) through attrs.
func TextField(p FieldProps, attrs ...g.Node) g.Node {
	typ := p.Type
	if typ == "" {
		typ = "text"
	}

	cls := "prim-field prim-field-text"
	if p.Error != "" {
		cls += " prim-field-error"
	}

	msgID := ""
	if p.ID != "" && (p.Error != "" || p.Hint != "") {
		msgID = p.ID + "-msg"
	}

	input := []g.Node{
		h.Class(cls),
		h.Type(typ),
		g.If(p.ID != "", h.ID(p.ID)),
		g.If(p.Placeholder != "", h.Placeholder(p.Placeholder)),
		g.If(p.Value != "", h.Value(p.Value)),
		g.If(p.Name != "", h.Name(p.Name)),
		g.If(p.Disabled, h.Disabled()),
		g.If(p.Error != "", g.Attr("aria-invalid", "true")),
		g.If(msgID != "", g.Attr("aria-describedby", msgID)),
	}
	input = append(input, attrs...)

	msg := func(extra, text string) g.Node {
		mcls := "prim-msg"
		if extra != "" {
			mcls += " " + extra
		}
		nodes := []g.Node{h.Class(mcls)}
		if msgID != "" {
			nodes = append(nodes, h.ID(msgID))
		}
		nodes = append(nodes, g.Text(text))
		return h.Span(nodes...)
	}

	return h.Label(
		h.Class("prim-control"),
		g.If(p.Label != "", h.Span(h.Class("prim-label"), g.Text(p.Label))),
		h.Input(input...),
		g.If(p.Error != "", msg("prim-msg-error", p.Error)),
		g.If(p.Error == "" && p.Hint != "", msg("", p.Hint)),
	)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run TestTextField -v` — Expected: PASS (all three).

- [ ] **Step 5: Append the shared form-shell CSS to the end of basm.css**

Append exactly this at the very end of `internal/web/assets/static/basm.css`:
```css

/* ── Primitives: shared form shell (TextField + Select) ────────────────
   Mono uppercase caption stacked over a parchment control, full width.
   The .prim-field base is defined ONCE here; Select adds only specializers. */
.prim-control { display: flex; flex-direction: column; gap: 6px; width: 100%; }
.prim-label {
  font-family: var(--font-mono); font-size: 10px; font-weight: 700;
  letter-spacing: .07em; text-transform: uppercase; color: var(--muted);
}
.prim-field {
  width: 100%; box-sizing: border-box;
  color: var(--ink); background-color: var(--surface);
  background-image: var(--grain-ink); background-size: 4px 4px;
  border: 2px solid var(--parch-edge); border-radius: var(--radius);
  box-shadow: var(--bevel-in); outline: none; caret-color: var(--ember-deep);
}
.prim-field:disabled { opacity: .55; cursor: default; }
.prim-field-text { padding: 10px 12px; font: 15px var(--font-body); }
.prim-field-error { border-color: var(--ember-red); }
.prim-msg {
  font-family: var(--font-mono); font-size: 10.5px; letter-spacing: .02em; color: var(--muted);
}
.prim-msg-error { color: var(--ember-red); }
```

- [ ] **Step 6: Verify** — `CGO_ENABLED=0 go build ./... && go test ./internal/web/assets/ && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"` (build clean, token test passes, suite green).

- [ ] **Step 7: Commit**
```bash
git add internal/ui/textfield.go internal/ui/textfield_test.go internal/web/assets/static/basm.css
git commit -m "$(printf 'feat(ui): add TextField atom + shared form shell\n\nui.TextField(FieldProps, attrs) — labelled parchment input; error sets the\nprim-field-error border + aria-invalid, and aria-describedby to the message\nwhen an ID is given. Adds the shared .prim-control/.prim-label/.prim-field\nbase (reused by Select). Tokenized CSS.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 2: `ui.Select` atom

A native `<select>` styled with the shared `.prim-field` base + select
specializers (suppressed browser chevron + a custom aria-hidden glyph). Relies
on the shared base from Task 1.

**Files:** Create `internal/ui/select.go`, `internal/ui/select_test.go`; modify `basm.css`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/select_test.go`:
```go
package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestSelect(t *testing.T) {
	got := render(t, ui.Select(ui.SelectProps{Label: "Model", Options: []string{"local", "openai"}, Value: "openai", Name: "model"}))
	for _, want := range []string{
		`<label class="prim-control"><span class="prim-label">Model</span>`,
		`<div class="prim-select">`,
		`<select class="prim-field prim-field-select" name="model">`,
		`<option value="local">local</option>`,
		`<option value="openai" selected>openai</option>`,
		`<span class="prim-select-chevron" aria-hidden="true">▾</span>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("select missing %q in: %s", want, got)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run TestSelect -v` — Expected: FAIL (`undefined: ui.Select`/`ui.SelectProps`).

- [ ] **Step 3: Implement the atom**

Create `internal/ui/select.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// SelectProps configures a Select dropdown. Label is the optional uppercase mono
// caption. Options are the literal values, rendered verbatim as both value and
// visible text. Value marks the matching option selected (no match selects the
// browser default). Name is the form field name. Disabled greys and blocks it.
type SelectProps struct {
	Label    string
	Options  []string
	Value    string
	Name     string
	Disabled bool
}

// Select renders the Hearthwood form dropdown: a <label class=prim-control> caption
// over a positioned wrapper holding a native <select class="prim-field prim-field-select">
// (browser chevron suppressed in CSS) and a custom aria-hidden ▾ glyph. Pure render —
// selection lives in <option selected>; callers pass attrs (data-on-change, an id, …)
// to wire change/submit into the form pipeline.
func Select(p SelectProps, attrs ...g.Node) g.Node {
	opts := make([]g.Node, 0, len(p.Options))
	for _, o := range p.Options {
		opt := []g.Node{h.Value(o), g.Text(o)}
		if o == p.Value {
			opt = append(opt, h.Selected())
		}
		opts = append(opts, h.Option(opt...))
	}

	selectAttrs := []g.Node{h.Class("prim-field prim-field-select")}
	if p.Name != "" {
		selectAttrs = append(selectAttrs, h.Name(p.Name))
	}
	if p.Disabled {
		selectAttrs = append(selectAttrs, h.Disabled())
	}
	selectAttrs = append(selectAttrs, attrs...)
	selectAttrs = append(selectAttrs, opts...)

	control := h.Div(
		h.Class("prim-select"),
		h.Select(selectAttrs...),
		h.Span(h.Class("prim-select-chevron"), g.Attr("aria-hidden", "true"), g.Text("▾")),
	)

	rows := []g.Node{h.Class("prim-control")}
	if p.Label != "" {
		rows = append(rows, h.Span(h.Class("prim-label"), g.Text(p.Label)))
	}
	rows = append(rows, control)

	return h.Label(rows...)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run TestSelect -v` — Expected: PASS.

- [ ] **Step 5: Append the Select specializer CSS to the end of basm.css**

Append exactly this at the very end of `internal/web/assets/static/basm.css`
(it relies on the shared `.prim-field` base added in Task 1):
```css

/* ── Primitives: Select (extends the shared .prim-field base) ──────────── */
.prim-select { position: relative; width: 100%; }
.prim-field-select {
  appearance: none; -webkit-appearance: none; -moz-appearance: none; cursor: pointer;
  font: 13px var(--font-mono); text-transform: uppercase; letter-spacing: .03em;
  padding: 11px 34px 11px 12px;
}
.prim-select-chevron {
  position: absolute; right: 12px; top: 50%; transform: translateY(-50%);
  pointer-events: none; font-family: var(--font-mono); font-size: 12px; color: var(--gold-ink);
}
```

- [ ] **Step 6: Verify** — `CGO_ENABLED=0 go build ./... && go test ./internal/web/assets/ && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"`.

- [ ] **Step 7: Commit**
```bash
git add internal/ui/select.go internal/ui/select_test.go internal/web/assets/static/basm.css
git commit -m "$(printf 'feat(ui): add Select atom\n\nui.Select(SelectProps, attrs) — native <select> on the shared .prim-field base\nwith the browser chevron suppressed and a custom aria-hidden glyph. Pure\nrender (option selected); tokenized specializer CSS.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 3: `ui.Toggle` atom

A Hearthwood switch: a real `<button type="button" role="switch">` with a
sliding knob; state lives in `[aria-checked]`/`[disabled]`. Accessible name via
`aria-labelledby` (when ID+Label) or `aria-label` (Label only).

**Files:** Create `internal/ui/toggle.go`, `internal/ui/toggle_test.go`; modify `basm.css`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/toggle_test.go`:
```go
package ui_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestToggleBare(t *testing.T) {
	got := render(t, ui.Toggle(ui.ToggleProps{Checked: true}))
	want := `<button type="button" role="switch" class="toggle" aria-checked="true"><span class="toggle-knob"></span></button>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}

func TestToggleLabelAriaLabel(t *testing.T) {
	got := render(t, ui.Toggle(ui.ToggleProps{Label: "Notifications"}))
	want := `<span class="toggle-row"><button type="button" role="switch" class="toggle" aria-checked="false" aria-label="Notifications"><span class="toggle-knob"></span></button><span class="toggle-label">Notifications</span></span>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}

func TestToggleLabelledby(t *testing.T) {
	got := render(t, ui.Toggle(ui.ToggleProps{Label: "Dark", ID: "theme", Checked: true}))
	want := `<span class="toggle-row"><button type="button" role="switch" class="toggle" aria-checked="true" id="theme" aria-labelledby="theme-label"><span class="toggle-knob"></span></button><span class="toggle-label" id="theme-label">Dark</span></span>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}

func TestToggleDisabled(t *testing.T) {
	got := render(t, ui.Toggle(ui.ToggleProps{Disabled: true}))
	want := `<button type="button" role="switch" class="toggle" aria-checked="false" disabled><span class="toggle-knob"></span></button>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run TestToggle -v` — Expected: FAIL (`undefined: ui.Toggle`/`ui.ToggleProps`).

- [ ] **Step 3: Implement the atom**

Create `internal/ui/toggle.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ToggleProps configures a Toggle switch. Checked is the on/off state. Label is
// the optional caption beside the switch. ID, when set with Label, wires
// aria-labelledby to the caption; without an ID the switch gets aria-label from
// Label (so it is always named, with no duplicate-id risk). Disabled emits the
// native disabled attribute.
type ToggleProps struct {
	Checked  bool
	Disabled bool
	Label    string
	ID       string
}

// Toggle renders the Hearthwood switch: a keyboard-operable
// <button type="button" role="switch"> with a sliding knob. State lives in
// [aria-checked] / [disabled] (no inline style). Callers wire the flip via the
// variadic attrs (e.g. a Datastar data.On("click", …) posting the new state).
func Toggle(props ToggleProps, attrs ...g.Node) g.Node {
	btnAttrs := []g.Node{
		h.Type("button"),
		h.Role("switch"),
		h.Class("toggle"),
		g.Attr("aria-checked", boolStr(props.Checked)),
		g.If(props.ID != "", h.ID(props.ID)),
		g.If(props.Disabled, h.Disabled()),
	}

	labelID := ""
	if props.Label != "" {
		if props.ID != "" {
			labelID = props.ID + "-label"
			btnAttrs = append(btnAttrs, g.Attr("aria-labelledby", labelID))
		} else {
			btnAttrs = append(btnAttrs, h.Aria("label", props.Label))
		}
	}
	btnAttrs = append(btnAttrs, h.Span(h.Class("toggle-knob")))
	btn := h.Button(append(btnAttrs, attrs...)...)

	if props.Label == "" {
		return btn
	}
	labelSpan := []g.Node{h.Class("toggle-label")}
	if labelID != "" {
		labelSpan = append(labelSpan, h.ID(labelID))
	}
	labelSpan = append(labelSpan, g.Text(props.Label))
	return h.Span(h.Class("toggle-row"), btn, h.Span(labelSpan...))
}

// boolStr renders a Go bool as the HTML attribute string "true"/"false".
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run TestToggle -v` — Expected: PASS (all four).

- [ ] **Step 5: Append the Toggle CSS to the end of basm.css**

Append exactly this at the very end of `internal/web/assets/static/basm.css`:
```css

/* ── Toggle — Hearthwood switch (track + sliding knob) ──────────────────
   Pure-render <button role="switch">; state lives in [aria-checked]/[disabled],
   the knob slides via CSS, never inline style. */
.toggle {
  position: relative; display: inline-block; flex-shrink: 0; width: 46px; height: 26px; padding: 0;
  background: var(--chrome-2); background-image: var(--grain-warm); background-size: 4px 4px;
  border: 2px solid var(--outline-2); border-radius: var(--radius); box-shadow: var(--bevel-in); cursor: pointer;
}
.toggle[aria-checked="true"] { background: var(--teal-deep); background-image: none; }
.toggle-knob {
  position: absolute; top: 2px; left: 2px; width: 18px; height: 18px;
  background: var(--surface-2); border: 2px solid var(--outline-2); border-radius: var(--radius);
  box-shadow: var(--bevel-up); transition: left 80ms;
}
.toggle[aria-checked="true"] .toggle-knob { left: 22px; background: var(--gold); }
.toggle[disabled] { opacity: .5; cursor: default; }
.toggle-row { display: inline-flex; align-items: center; gap: 11px; cursor: pointer; }
.toggle-row:has(.toggle[disabled]) { cursor: default; }
.toggle-label {
  font-family: var(--font-mono); font-size: 12px; text-transform: uppercase; letter-spacing: .04em; color: var(--fg);
}
```

- [ ] **Step 6: Verify** — `CGO_ENABLED=0 go build ./... && go test ./internal/web/assets/ && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"`.

- [ ] **Step 7: Commit**
```bash
git add internal/ui/toggle.go internal/ui/toggle_test.go internal/web/assets/static/basm.css
git commit -m "$(printf 'feat(ui): add Toggle atom\n\nui.Toggle(ToggleProps, attrs) — a keyboard-operable <button role=switch> with\na sliding knob; state in [aria-checked]/[disabled]. Accessible name via\naria-labelledby (ID+Label) or aria-label (Label only). Tokenized CSS.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 4: Showcase the form atoms in `/storybook`

**Files:** Modify `internal/feature/storybook/storybook.go`, `storybook_test.go`.

- [ ] **Step 1: Extend the test assertions (failing first)**

In `internal/feature/storybook/storybook_test.go`, add these to the `want` slice
in `TestBodyRendersAtoms` (after the existing `` `class="skeleton skeleton-line"` `` line):
```go
		`class="prim-field prim-field-text"`,
		`class="prim-field prim-field-select"`,
		`class="toggle"`,
		`role="switch"`,
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/feature/storybook/ -v` — Expected: FAIL (new substrings absent).

- [ ] **Step 3: Add the sections to the gallery**

In `internal/feature/storybook/storybook.go`, inside `Body()`, add these sections
after the existing `section("Skeletons", ...)` block and before the closing `)`
of `h.Div(h.Class("sb"), ...)`:
```go
		section("Form fields",
			ui.TextField(ui.FieldProps{Label: "Name", Placeholder: "Your name", Name: "name"}),
			ui.TextField(ui.FieldProps{Label: "Email", Type: "email", Value: "you@yourbox", Name: "email", Hint: "Used only on your box."}),
			ui.TextField(ui.FieldProps{Label: "Token", ID: "tok", Name: "token", Error: "Required."}),
			ui.Select(ui.SelectProps{Label: "Model", Options: []string{"local", "openai", "anthropic"}, Value: "local", Name: "model"}),
		),
		section("Toggles",
			ui.Toggle(ui.ToggleProps{Label: "Notifications", ID: "notif", Checked: true}),
			ui.Toggle(ui.ToggleProps{Label: "OS access", ID: "os"}),
			ui.Toggle(ui.ToggleProps{Label: "Disabled", ID: "dis", Disabled: true}),
		),
```

- [ ] **Step 4: Run to verify it passes + full suite + view**

Run:
```bash
go test ./internal/feature/storybook/ -v 2>&1 | tail -6
go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./... && go vet ./...
```
Expected: storybook test PASS, full suite green, build+vet clean. Optionally view
`/storybook` — fields, the select chevron, and the switches (on/off/disabled)
render; tab to a switch and press Space to confirm keyboard operation.

- [ ] **Step 5: Commit**
```bash
git add internal/feature/storybook/storybook.go internal/feature/storybook/storybook_test.go
git commit -m "$(printf 'feat(storybook): showcase TextField, Select, Toggle\n\nAdd gallery sections for the three form atoms.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Final verification

- [ ] `go vet ./... && go test ./... && CGO_ENABLED=0 go build ./... && git diff --check` — all green.
- [ ] No raw hex in the new CSS declarations:
```bash
awk '/Primitives: shared form shell/{p=1} p' internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX in the new form CSS"
```
Expected: `NO RAW HEX…`.

## What this slice delivers / what's next

**Delivered:** three form atoms (`TextField`, `Select`, `Toggle`) sharing one
`.prim-field` shell, with form-correct a11y (aria-invalid/describedby, role=switch
+ accessible name), tokenized CSS, and a fuller `/storybook`.

**Next (02b sub-plans):** nav (`Tabs` — the load-bearing tab control, plus
`Breadcrumb`, `Pagination`), `List`/`ListItem`, and the `SectionLabel`/
`ScreenTitle` text helpers. Then Phase 2 is complete and Phase 3 (storybook → `/`)
begins.
