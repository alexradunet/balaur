# Gomponents Atomic — Overlay Atoms (Plan 02b-v) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the overlay/feedback atoms `Toast` and `Dialog` to `internal/ui`, with new tokenized CSS, registering both as storybook stories under the existing "Feedback" group.

**Architecture:** `Toast` is an accent-bordered parchment status pill (`role=status`) with a per-tone default icon. `Dialog` is a native `<dialog>` element — a gold-bordered parchment panel with four corner brackets, an optional kicker + display title, a body, and right-aligned small `ui.Button` actions; `Open` adds the `open` attribute so it renders in place (storybook display + non-JS reveal), and a `::backdrop` rule styles it for a future `showModal()` modal use. Both are pure server-rendered; new CSS lifts the export's inline styles into tokenized `basm.css` rules.

**Tech Stack:** Go, gomponents (`h.Dialog`/`h.Open` exist in v1.3.0), vanilla `basm.css`.

**Scope:** Plan 02b-v (Phase 2b catalog). `SectionLabel`/`ScreenTitle` follow in a later sub-plan; then organisms.

**Conventions:** package `ui` uses QUALIFIED `g`/`h` imports (NO dot-import — a package-level `func Button` collides with `html.Button`). New CSS appends at the END of `basm.css`, tokenized (`var(--token)`, no raw hex, single-dash classes — raw `rgba()` is allowed for scrims/shadows, matching existing `--parch-bevel`). Atom tests are `package ui_test` and use the shared `render(t, node)` helper (in `internal/ui/helpers_test.go` — do NOT redefine). Stories: append a canvas func to `internal/feature/storybook/storybook.go` and a `Story` entry to the `stories` slice in `internal/feature/storybook/story.go` (positional `{ID, Group, Title, Canvas}`). After each task: `go test ./...`, `CGO_ENABLED=0 go build ./...`, `go vet ./...`. If `git status` shows any file other than the task's own as modified (e.g. a stray `chatstream.go` from a linter), do NOT stage it — `git checkout --` it.

---

## File Structure

- **Create** `internal/ui/toast.go`+`_test.go`, `internal/ui/dialog.go`+`_test.go`.
- **Modify** `internal/web/assets/static/basm.css` (append Toast CSS, then Dialog CSS).
- **Modify** `internal/feature/storybook/storybook.go` (canvas funcs) + `internal/feature/storybook/story.go` (register Toast + Dialog).

Verified facts (do not re-derive): tokens `--gold-deep`, `--good-ink`, `--ember-deep`, `--gold-ink`, `--ink`, `--surface`, `--grain-ink`, `--parch-bevel`, `--font-mono`, `--font-display` all exist in `basm.css`. Icons `quill.png`, `check.png`, `shield.png` exist in `internal/web/assets/static/icons/`. Classes `.toast*` and `.dlg*` are unused. `ui.Button(ButtonProps{Variant, Size:"sm", Href}, g.Text(label))` exists.

---

## Task 1: `ui.Toast` + story

An accent-bordered parchment status pill: a pixel icon + a message. The tone sets
the border accent (info→gold, success→green, warn→ember) and the default icon
(info→quill, success→check, warn→shield); `Icon` overrides the icon.

**Files:** Create `internal/ui/toast.go`, `internal/ui/toast_test.go`; modify `internal/web/assets/static/basm.css`, `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/toast_test.go`:
```go
package ui_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestToastWarn(t *testing.T) {
	got := render(t, ui.Toast(ui.ToastProps{Tone: "warn"}, g.Text("Heads up")))
	for _, want := range []string{
		`<div class="toast toast-warn" role="status">`,
		`<img class="toast-icon" src="/static/icons/shield.png" alt="" decoding="async">`,
		`<span class="toast-msg">Heads up</span>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("toast missing %q in: %s", want, got)
		}
	}
}

func TestToastDefaultInfo(t *testing.T) {
	got := render(t, ui.Toast(ui.ToastProps{}, g.Text("Saved")))
	if !strings.Contains(got, `<div class="toast toast-info" role="status">`) {
		t.Errorf("default tone should be info: %s", got)
	}
	if !strings.Contains(got, `src="/static/icons/quill.png"`) {
		t.Errorf("info default icon should be quill: %s", got)
	}
}

func TestToastIconOverride(t *testing.T) {
	got := render(t, ui.Toast(ui.ToastProps{Tone: "success", Icon: "flame"}, g.Text("x")))
	if !strings.Contains(got, `<div class="toast toast-success" role="status">`) {
		t.Errorf("success tone class missing: %s", got)
	}
	if !strings.Contains(got, `src="/static/icons/flame.png"`) {
		t.Errorf("Icon override should win over the success default: %s", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run TestToast -v` — Expected: FAIL (`undefined: ui.Toast`/`ui.ToastProps`).

- [ ] **Step 3: Implement the atom**

Create `internal/ui/toast.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ToastProps configures a Toast. Tone is "info" (default), "success", or "warn"
// and sets the border accent. Icon overrides the per-tone default icon name (a
// /static/icons name).
type ToastProps struct {
	Tone string
	Icon string
}

// toastIcon resolves the icon name: an explicit override wins, else the per-tone
// default (success→check, warn→shield, info→quill).
func toastIcon(tone, override string) string {
	if override != "" {
		return override
	}
	switch tone {
	case "success":
		return "check"
	case "warn":
		return "shield"
	default:
		return "quill"
	}
}

// Toast renders a status pill: an accent-bordered parchment chip with a pixel icon
// and a message (the variadic children). role=status so assistive tech announces
// it politely.
func Toast(p ToastProps, children ...g.Node) g.Node {
	tone := p.Tone
	if tone == "" {
		tone = "info"
	}
	return h.Div(
		h.Class("toast toast-"+tone), h.Role("status"),
		h.Img(h.Class("toast-icon"), h.Src("/static/icons/"+toastIcon(tone, p.Icon)+".png"), h.Alt(""), g.Attr("decoding", "async")),
		h.Span(append([]g.Node{h.Class("toast-msg")}, children...)...),
	)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run TestToast -v` — Expected: PASS (all three).

- [ ] **Step 5: Append the Toast CSS to the end of basm.css**

Append exactly this at the very end of `internal/web/assets/static/basm.css`:
```css

/* ── Toast — accent-bordered parchment status pill ──────────────────────── */
.toast {
  display: inline-flex; align-items: center; gap: 11px; max-width: 100%;
  background: var(--surface); background-image: var(--grain-ink); background-size: 4px 4px;
  color: var(--ink); border: 2px solid var(--gold-deep); box-shadow: var(--parch-bevel); padding: 11px 15px;
}
.toast-success { border-color: var(--good-ink); }
.toast-warn { border-color: var(--ember-deep); }
.toast-icon { width: 20px; height: 20px; image-rendering: pixelated; flex-shrink: 0; }
.toast-msg { font-size: 14px; line-height: 1.45; }
```
(`.toast-info` needs no rule — the base `.toast` border is already gold.)

- [ ] **Step 6: Add the canvas + register the story**

In `internal/feature/storybook/storybook.go`, append:
```go

func toastCanvas() g.Node {
	return section("Toast",
		ui.Toast(ui.ToastProps{}, g.Text("Saved to the book.")),
		ui.Toast(ui.ToastProps{Tone: "success"}, g.Text("Task marked done.")),
		ui.Toast(ui.ToastProps{Tone: "warn"}, g.Text("Heads up — that's overdue.")),
	)
}
```
In `internal/feature/storybook/story.go`, add to the `stories` slice immediately AFTER the `{"emptystate", "Feedback", "EmptyState", emptyStateCanvas},` line:
```go
	{"toast", "Feedback", "Toast", toastCanvas},
```

- [ ] **Step 7: Verify + commit**

Run:
```bash
cd /home/alex/Projects/balaur
go test ./internal/ui/ -run TestToast && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
tail -12 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"
git status --short
```
Expected: PASS, full suite green, build clean, NO RAW HEX. If `git status --short` shows any file other than `internal/ui/toast.go`, `internal/ui/toast_test.go`, `internal/web/assets/static/basm.css`, `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`, do NOT stage it — `git checkout -- <file>`. Then:
```bash
git add internal/ui/toast.go internal/ui/toast_test.go internal/web/assets/static/basm.css internal/feature/storybook/storybook.go internal/feature/storybook/story.go
git commit -m "$(printf 'feat(ui): add Toast atom + storybook story\n\nui.Toast(ToastProps) — accent-bordered parchment status pill (role=status) with\na per-tone default icon (info/success/warn). New tokenized .toast CSS. Registered\nunder Feedback.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 2: `ui.Dialog` + story

A native `<dialog>` parchment panel: four gold corner brackets, an optional mono
kicker + display title, the body, and right-aligned small `ui.Button` actions.
`Open` adds the `open` attribute so it renders in place (storybook + non-JS
reveal). A `::backdrop` rule styles the scrim for future `showModal()` modal use.

**Files:** Create `internal/ui/dialog.go`, `internal/ui/dialog_test.go`; modify `internal/web/assets/static/basm.css`, `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/dialog_test.go`:
```go
package ui_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestDialogFull(t *testing.T) {
	got := render(t, ui.Dialog(ui.DialogProps{
		Open:   true,
		Kicker: "Confirm",
		Title:  "Forget this thread?",
		Actions: []ui.DialogAction{
			{Label: "Cancel", Variant: "ghost", Href: "#"},
			{Label: "Forget", Variant: "wood"},
		},
	}, g.Text("This cannot be undone.")))
	for _, want := range []string{
		`<dialog class="dlg" open>`,
		`<span class="dlg-corner dlg-corner-tl"></span>`,
		`<span class="dlg-corner dlg-corner-br"></span>`,
		`<div class="dlg-kicker">Confirm</div>`,
		`<h2 class="dlg-title">Forget this thread?</h2>`,
		`<div class="dlg-body">This cannot be undone.</div>`,
		`<div class="dlg-actions">`,
		`<a class="btn btn-ghost btn-sm" href="#">Cancel</a>`,
		`<button class="btn btn-wood btn-sm">Forget</button>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("dialog missing %q in: %s", want, got)
		}
	}
}

func TestDialogBare(t *testing.T) {
	got := render(t, ui.Dialog(ui.DialogProps{}, g.Text("hi")))
	if !strings.Contains(got, `<dialog class="dlg">`) {
		t.Errorf("closed dialog should have no open attr: %s", got)
	}
	if strings.Contains(got, "dlg-kicker") || strings.Contains(got, "dlg-title") || strings.Contains(got, "dlg-actions") {
		t.Errorf("bare dialog should omit kicker/title/actions: %s", got)
	}
	if !strings.Contains(got, `<div class="dlg-body">hi</div>`) {
		t.Errorf("body should always render: %s", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run TestDialog -v` — Expected: FAIL (`undefined: ui.Dialog`/`ui.DialogProps`/`ui.DialogAction`).

- [ ] **Step 3: Implement the atom**

Create `internal/ui/dialog.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// DialogAction is one footer button: a label, an optional Button variant, and an
// optional Href (a link button when set, a plain button otherwise). Rendered at
// "sm" size.
type DialogAction struct {
	Label   string
	Variant string
	Href    string
}

// DialogProps configures a Dialog. Open adds the native `open` attribute so the
// <dialog> renders in place. Kicker and Title are optional headers; Actions are
// the right-aligned footer buttons.
type DialogProps struct {
	Open    bool
	Kicker  string
	Title   string
	Actions []DialogAction
}

// Dialog renders the Hearthwood dialog as a native <dialog>: a gold-bordered
// parchment panel with corner brackets, an optional kicker + display title, the
// body (variadic children), and small Button actions. Open renders it in place;
// otherwise the element is present but hidden until shown (e.g. showModal()).
func Dialog(p DialogProps, body ...g.Node) g.Node {
	kids := []g.Node{h.Class("dlg")}
	if p.Open {
		kids = append(kids, h.Open())
	}
	kids = append(kids,
		h.Span(h.Class("dlg-corner dlg-corner-tl")),
		h.Span(h.Class("dlg-corner dlg-corner-tr")),
		h.Span(h.Class("dlg-corner dlg-corner-bl")),
		h.Span(h.Class("dlg-corner dlg-corner-br")),
	)
	if p.Kicker != "" {
		kids = append(kids, h.Div(h.Class("dlg-kicker"), g.Text(p.Kicker)))
	}
	if p.Title != "" {
		kids = append(kids, h.H2(h.Class("dlg-title"), g.Text(p.Title)))
	}
	kids = append(kids, h.Div(append([]g.Node{h.Class("dlg-body")}, body...)...))
	if len(p.Actions) > 0 {
		acts := []g.Node{h.Class("dlg-actions")}
		for _, a := range p.Actions {
			acts = append(acts, Button(ButtonProps{Variant: a.Variant, Size: "sm", Href: a.Href}, g.Text(a.Label)))
		}
		kids = append(kids, h.Div(acts...))
	}
	return h.Dialog(kids...)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run TestDialog -v` — Expected: PASS (both).

- [ ] **Step 5: Append the Dialog CSS to the end of basm.css**

Append exactly this at the very end of `internal/web/assets/static/basm.css`:
```css

/* ── Dialog — native <dialog> parchment panel w/ corner brackets ────────── */
.dlg {
  position: relative; inset: auto; margin: 0; width: min(520px, calc(100% - 40px)); max-width: 520px;
  background: var(--surface); background-image: var(--grain-ink); background-size: 4px 4px; color: var(--ink);
  border: 2px solid var(--gold-deep); box-shadow: var(--parch-bevel); padding: 22px 22px 20px;
}
.dlg::backdrop { background: rgba(10, 5, 1, .72); }
.dlg-corner { position: absolute; width: 10px; height: 10px; border: 0 solid var(--gold-deep); pointer-events: none; }
.dlg-corner-tl { top: 5px; left: 5px; border-top-width: 3px; border-left-width: 3px; }
.dlg-corner-tr { top: 5px; right: 5px; border-top-width: 3px; border-right-width: 3px; }
.dlg-corner-bl { bottom: 5px; left: 5px; border-bottom-width: 3px; border-left-width: 3px; }
.dlg-corner-br { bottom: 5px; right: 5px; border-bottom-width: 3px; border-right-width: 3px; }
.dlg-kicker { font-family: var(--font-mono); font-size: 10px; font-weight: 700; letter-spacing: .1em; text-transform: uppercase; color: var(--gold-ink); margin-bottom: 8px; }
.dlg-title { margin: 0 0 10px; font-family: var(--font-display); font-size: 24px; color: var(--ink); line-height: 1.1; }
.dlg-body { font-size: 15px; line-height: 1.55; }
.dlg-actions { display: flex; flex-wrap: wrap; gap: 10px; margin-top: 18px; justify-content: flex-end; }
```
(The `position:relative; inset:auto; margin:0` triplet cancels the UA `<dialog>` defaults — `position:absolute; left:0; right:0; margin:auto` — so an `open` dialog flows in place in the storybook canvas. The raw `rgba()` backdrop matches the export and the existing `--parch-bevel` rgba convention.)

- [ ] **Step 6: Add the canvas + register the story**

In `internal/feature/storybook/storybook.go`, append:
```go

func dialogCanvas() g.Node {
	return section("Dialog", ui.Dialog(ui.DialogProps{
		Open:   true,
		Kicker: "Confirm",
		Title:  "Forget this thread?",
		Actions: []ui.DialogAction{
			{Label: "Cancel", Variant: "ghost", Href: "#"},
			{Label: "Forget", Variant: "wood"},
		},
	}, g.Text("This removes the thread and everything Balaur learned in it. This cannot be undone.")))
}
```
In `internal/feature/storybook/story.go`, add to the `stories` slice immediately AFTER the `{"toast", "Feedback", "Toast", toastCanvas},` line:
```go
	{"dialog", "Feedback", "Dialog", dialogCanvas},
```

- [ ] **Step 7: Verify + commit**

Run:
```bash
cd /home/alex/Projects/balaur
go test ./internal/ui/ -run TestDialog && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./... && go vet ./...
tail -16 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"
git status --short
```
Expected: PASS, full suite green, build+vet clean, NO RAW HEX. If `git status --short` shows any file other than the five task files, do NOT stage it — `git checkout -- <file>`. Then:
```bash
git add internal/ui/dialog.go internal/ui/dialog_test.go internal/web/assets/static/basm.css internal/feature/storybook/storybook.go internal/feature/storybook/story.go
git commit -m "$(printf 'feat(ui): add Dialog atom + storybook story\n\nui.Dialog(DialogProps) — native <dialog> parchment panel with corner brackets,\noptional kicker + display title, body, and small Button actions. Open renders it\nin place; ::backdrop styles a future showModal() scrim. New tokenized .dlg CSS.\nRegistered under Feedback.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Final verification

- [ ] `go vet ./... && go test ./... && CGO_ENABLED=0 go build ./... && git diff --check` — all green.
- [ ] No raw hex in the new CSS: `tail -28 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"`.
- [ ] Sidebar **Feedback** group now lists Toast and Dialog; `/storybook/toast` and `/storybook/dialog` render 200.

## What this delivers / what's next

**Delivered:** two overlay/feedback atoms (`Toast`, `Dialog`); both registered as stories. `internal/ui` reaches **20 atoms**.

**Next:** `SectionLabel`/`ScreenTitle` text helpers — then the organisms (chat Message/ToolRow/Composer, knowledge/task cards) that begin replacing real `html/template` surfaces, → Phase 3 (storybook → `/`), → cut boards + delete `web/` (Phase 6).
