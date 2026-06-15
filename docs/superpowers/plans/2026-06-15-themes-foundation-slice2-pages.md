# Themes — Slice 2: Foundation Pages (Colors / Typography / Materials) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the three Foundation guideline pages — **Colors**, **Typography**, **Materials** — as storybook stories under a new "Foundations" group (the first content group). Colors is the live theme showcase: every swatch reads `var(--token)` so it re-tints as the palette cycles.

**Architecture:** Each page is a canvas func in `internal/feature/storybook/storybook.go` + a `Story` registry entry + new tokenized CSS in `basm.css`. Colors composes `ui.SectionLabel` for group headings; Materials composes `ui.FolkBand` + `ui.Stitch`. All faithful ports of the export's `.dc.html` foundation pages.

**Tech Stack:** Go, gomponents, vanilla `basm.css`.

**Spec:** `docs/superpowers/specs/2026-06-15-themes-and-foundations-design.md` (Slice 2). Slice 1 (the theme system) is merged.

**Conventions:** This is COMPONENT/documentation CSS — tokenized (`var(--token)`, no raw hex, single-dash classes), appended at the END of basm.css. The one exception is the swatch chip background, set via an inline `--sw` custom property (`style="--sw:var(--gold)"`) read by `.swatch-chip { background: var(--sw) }` — the same idiom as `ui.SectionLabel`'s `--sl-accent` (still a var reference, no raw hex). Stories register at the TOP of the `stories` slice so Foundations is the first content group. `story_test.go` is `package storybook_test` (EXTERNAL) and already imports `strings` + the package as `storybook`; new tests assert per-canvas golden via `storybook.Lookup(id).Canvas()` (NOT bare `Lookup` — it's the external package). After each task: `go test ./...`, `CGO_ENABLED=0 go build ./...`, `go vet ./...`. If `git status` shows any file other than the task's own, `git checkout --` it.

Verified facts: tokens `--font-display/-pixel/-body/-mono`, `--bevel-up/-in`, `--parch-bevel`, `--gold-ink`, `--grain-ink`, `--grain-warm`, `--wood-planks`, `--outline-2`, `--surface*`, `--parch-edge`, `--ink*`, `--fg-strong`, `--muted`, `--bg` all exist. `ui.SectionLabel(SectionLabelProps{Text})`, `ui.Stitch(...attrs)`→`.stitch`, `ui.FolkBand(...attrs)`→`.folk-band` exist. `storybook.go` imports `g`, `h`, `ui`, `chat`; helper `section(label, ...items)` exists. The registry test in `story_test.go` covers new IDs automatically.

---

## File Structure

- **Modify** `internal/web/assets/static/basm.css` — append Colors, then Typography, then Materials CSS.
- **Modify** `internal/feature/storybook/storybook.go` — three canvas funcs + a package-level `colorGroups` table.
- **Modify** `internal/feature/storybook/story.go` — three `Story` entries at the top of the slice.
- **Modify** `internal/feature/storybook/story_test.go` (or add a focused test) — assert the three Foundation canvases render their key markers.

---

## Task 1: Colors page

**Files:** `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`, `internal/web/assets/static/basm.css`, `internal/feature/storybook/story_test.go`.

- [ ] **Step 1: Write the failing test**

In `internal/feature/storybook/story_test.go`, add (the package is `storybook` — internal test — so it can call `Lookup`):
```go
func TestColorsCanvas(t *testing.T) {
	s, ok := storybook.Lookup("colors")
	if !ok {
		t.Fatal("colors story not registered")
	}
	var b strings.Builder
	if err := s.Canvas().Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()
	for _, want := range []string{
		`<span class="section-label-text">Accents</span>`,
		`<div class="swatch-chip" style="--sw:var(--gold)"></div>`,
		`<div class="swatch-name">--bg</div>`,
		`<div class="swatch-label">gold</div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("colors canvas missing %q", want)
		}
	}
}
```
(Ensure `strings` is imported in the test file.)

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/feature/storybook/ -run TestColorsCanvas -v` — Expected: FAIL (`colors story not registered`).

- [ ] **Step 3: Implement the canvas + data**

In `internal/feature/storybook/storybook.go`, append:
```go

// colorGroups drives the Colors foundation page — token groups whose swatches
// read var(--token) live, so they re-tint with the active theme.
var colorGroups = []struct {
	Name  string
	Items [][2]string // {label, css-var}
}{
	{"Page & wood", [][2]string{{"bg", "--bg"}, {"chrome", "--chrome"}, {"chrome-2", "--chrome-2"}, {"chrome-fg", "--chrome-fg"}, {"outline-2", "--outline-2"}}},
	{"Parchment & ink", [][2]string{{"surface", "--surface"}, {"surface-2", "--surface-2"}, {"surface-3", "--surface-3"}, {"parch-edge", "--parch-edge"}, {"ink", "--ink"}, {"ink-muted", "--ink-muted"}}},
	{"Accents", [][2]string{{"gold", "--gold"}, {"gold-deep", "--gold-deep"}, {"ember", "--ember"}, {"ember-deep", "--ember-deep"}, {"ember-red", "--ember-red"}, {"teal", "--teal"}, {"teal-deep", "--teal-deep"}, {"folkred", "--folkred"}, {"indigo", "--indigo"}, {"violet", "--violet"}, {"good", "--good"}}},
	{"Text on page", [][2]string{{"fg", "--fg"}, {"fg-strong", "--fg-strong"}, {"muted", "--muted"}, {"hair", "--hair"}}},
}

func colorsCanvas() g.Node {
	groups := make([]g.Node, 0, len(colorGroups))
	for _, grp := range colorGroups {
		swatches := make([]g.Node, 0, len(grp.Items))
		for _, it := range grp.Items {
			swatches = append(swatches, h.Div(h.Class("swatch"),
				h.Div(h.Class("swatch-chip"), h.Style("--sw:var("+it[1]+")")),
				h.Div(h.Class("swatch-label"), g.Text(it[0])),
				h.Div(h.Class("swatch-name"), g.Text(it[1])),
			))
		}
		groups = append(groups, h.Div(
			ui.SectionLabel(ui.SectionLabelProps{Text: grp.Name}),
			h.Div(append([]g.Node{h.Class("swatch-grid")}, swatches...)...),
		))
	}
	return section("Colors", h.Div(append([]g.Node{h.Class("fdn-stack")}, groups...)...))
}
```

- [ ] **Step 4: Register the story (top of slice)**

In `internal/feature/storybook/story.go`, add as the FIRST entry of the `stories` slice, immediately before `{"button", "Atoms", "Button", buttonCanvas},`:
```go
	{"colors", "Foundations", "Colors", colorsCanvas},
```

- [ ] **Step 5: Append the Colors CSS**

At the END of `internal/web/assets/static/basm.css`:
```css

/* ── Foundations · Colors — token swatches (re-tint live per theme) ──────── */
.fdn-stack { display: flex; flex-direction: column; gap: 24px; }
.swatch-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(108px, 1fr)); gap: 13px; }
.swatch { display: flex; flex-direction: column; gap: 7px; }
.swatch-chip { height: 62px; background: var(--sw); border: 2px solid var(--outline-2); box-shadow: var(--bevel-in); }
.swatch-label { font-family: var(--font-mono); font-size: 11px; color: var(--fg-strong); }
.swatch-name { font-family: var(--font-mono); font-size: 10px; color: var(--muted); }
```

- [ ] **Step 6: Run + commit**

```bash
cd /home/alex/Projects/balaur
go test ./internal/feature/storybook/ -run TestColorsCanvas && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
tail -10 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"
git status --short
```
Expected: PASS, suite green, build clean, NO RAW HEX. Stage only the four files, then:
```bash
git add internal/feature/storybook/storybook.go internal/feature/storybook/story.go internal/web/assets/static/basm.css internal/feature/storybook/story_test.go
git commit -m "$(printf 'feat(ui): Colors foundation page (live theme-swatch showcase)\n\nNew Foundations story group. colorsCanvas — token groups (Page&wood, Parchment&\nink, Accents, Text) as var(--token) swatch chips that re-tint per theme; composes\nui.SectionLabel. New tokenized .swatch CSS.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 2: Typography page

**Files:** `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`, `internal/web/assets/static/basm.css`, `internal/feature/storybook/story_test.go`.

- [ ] **Step 1: Write the failing test**

Add to `internal/feature/storybook/story_test.go`:
```go
func TestTypographyCanvas(t *testing.T) {
	s, ok := storybook.Lookup("typography")
	if !ok {
		t.Fatal("typography story not registered")
	}
	var b strings.Builder
	if err := s.Canvas().Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()
	for _, want := range []string{
		`<div class="type-role">Display</div>`,
		`<div class="type-sample-display">A new head wakes</div>`,
		`<div class="type-sample-pixel">BALAUR</div>`,
		`The hearth is lit`,
		`<span class="type-scale-tag">36</span>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("typography canvas missing %q", want)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/feature/storybook/ -run TestTypographyCanvas -v` — Expected: FAIL (not registered).

- [ ] **Step 3: Implement the canvas**

In `internal/feature/storybook/storybook.go`, append:
```go

func typeRole(role, sampleClass, sample, note string) g.Node {
	return h.Div(h.Class("fdn-card type-row"),
		h.Div(h.Class("type-role"), g.Text(role)),
		h.Div(h.Class(sampleClass), g.Text(sample)),
		h.Div(h.Class("type-note"), g.Text(note)),
	)
}

func typographyCanvas() g.Node {
	scale := []struct{ Tag, Class string }{
		{"36", "type-scale-36"}, {"28", "type-scale-28"}, {"22", "type-scale-22"}, {"17", "type-scale-17"}, {"13", "type-scale-13"},
	}
	rows := make([]g.Node, 0, len(scale))
	for _, s := range scale {
		rows = append(rows, h.Div(h.Class("type-scale-row"),
			h.Span(h.Class("type-scale-tag"), g.Text(s.Tag)),
			h.Span(h.Class(s.Class), g.Text("The hearth is lit")),
		))
	}
	return section("Typography", h.Div(h.Class("fdn-col"),
		typeRole("Display", "type-sample-display", "A new head wakes", "Jersey 15 · headings 20px+"),
		typeRole("Pixel", "type-sample-pixel", "BALAUR", "Silkscreen · nameplate & runes only"),
		typeRole("Body", "type-sample-body", "I shall weigh the matter.", "Piazzolla · 17px / 1.6"),
		typeRole("Mono", "type-sample-mono", "tool · search · used ×3", "JetBrains Mono · meta, nav, code"),
		h.Div(h.Class("fdn-card"),
			h.Div(h.Class("type-scale-head"), g.Text("Scale")),
			h.Div(append([]g.Node{h.Class("type-scale-list")}, rows...)...),
		),
	))
}
```

- [ ] **Step 4: Register the story**

In `internal/feature/storybook/story.go`, add immediately AFTER the `{"colors", "Foundations", "Colors", colorsCanvas},` line:
```go
	{"typography", "Foundations", "Typography", typographyCanvas},
```

- [ ] **Step 5: Append the Typography CSS**

At the END of `internal/web/assets/static/basm.css`:
```css

/* ── Foundations · Typography — the four font roles + a scale ────────────── */
.fdn-col { display: flex; flex-direction: column; gap: 16px; }
.fdn-card { border: 2px solid var(--parch-edge); background-color: var(--surface); background-image: var(--grain-ink); background-size: 4px 4px; box-shadow: var(--parch-bevel); padding: 18px 20px; color: var(--ink); }
.type-row { display: flex; gap: 20px; align-items: baseline; flex-wrap: wrap; }
.type-role { width: 84px; flex-shrink: 0; font-family: var(--font-mono); font-size: 11px; text-transform: uppercase; letter-spacing: .05em; color: var(--gold-ink); }
.type-note { flex: 1; min-width: 160px; text-align: right; font-family: var(--font-mono); font-size: 11px; color: var(--ink-muted); }
.type-sample-display { font-family: var(--font-display); font-size: 30px; color: var(--ink); line-height: 1; }
.type-sample-pixel { font-family: var(--font-pixel); font-size: 17px; letter-spacing: .08em; color: var(--gold-ink); }
.type-sample-body { font-family: var(--font-body); font-size: 22px; color: var(--ink); }
.type-sample-mono { font-family: var(--font-mono); font-size: 15px; color: var(--ink); }
.type-scale-head { font-family: var(--font-mono); font-size: 11px; letter-spacing: .1em; text-transform: uppercase; color: var(--gold-ink); margin-bottom: 14px; }
.type-scale-list { display: flex; flex-direction: column; gap: 6px; }
.type-scale-row { display: flex; align-items: baseline; gap: 16px; border-bottom: 1px solid var(--parch-edge); padding-bottom: 6px; }
.type-scale-tag { width: 46px; flex-shrink: 0; font-family: var(--font-mono); font-size: 11px; color: var(--ink-muted); }
.type-scale-36 { font-family: var(--font-display); font-size: 36px; color: var(--ink); line-height: 1; }
.type-scale-28 { font-family: var(--font-display); font-size: 28px; color: var(--ink); line-height: 1; }
.type-scale-22 { font-family: var(--font-body); font-size: 22px; color: var(--ink); }
.type-scale-17 { font-family: var(--font-body); font-size: 17px; color: var(--ink); }
.type-scale-13 { font-family: var(--font-mono); font-size: 13px; color: var(--ink); }
```

- [ ] **Step 6: Run + commit**

```bash
cd /home/alex/Projects/balaur
go test ./internal/feature/storybook/ -run TestTypographyCanvas && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
tail -24 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"
git status --short
```
Expected: PASS, suite green, build clean, NO RAW HEX. Stage only the four files, then:
```bash
git add internal/feature/storybook/storybook.go internal/feature/storybook/story.go internal/web/assets/static/basm.css internal/feature/storybook/story_test.go
git commit -m "$(printf 'feat(ui): Typography foundation page (4 font roles + scale)\n\ntypographyCanvas — Display/Pixel/Body/Mono role cards + a size scale, each on the\nparchment .fdn-card material. New tokenized .type-* CSS. Registered under\nFoundations.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 3: Materials page

**Files:** `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`, `internal/web/assets/static/basm.css`, `internal/feature/storybook/story_test.go`.

- [ ] **Step 1: Write the failing test**

Add to `internal/feature/storybook/story_test.go`:
```go
func TestMaterialsCanvas(t *testing.T) {
	s, ok := storybook.Lookup("materials")
	if !ok {
		t.Fatal("materials story not registered")
	}
	var b strings.Builder
	if err := s.Canvas().Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()
	for _, want := range []string{
		`<div class="mat-title">Parchment</div>`,
		`<div class="mat-swatch mat-wood"></div>`,
		`<div class="mat-title">Wood chrome</div>`,
		`class="folk-band"`,
		`class="stitch"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("materials canvas missing %q", want)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/feature/storybook/ -run TestMaterialsCanvas -v` — Expected: FAIL (not registered).

- [ ] **Step 3: Implement the canvas**

In `internal/feature/storybook/storybook.go`, append:
```go

func matTile(swatch g.Node, title, desc string) g.Node {
	return h.Div(h.Class("mat-tile"),
		swatch,
		h.Div(h.Class("mat-title"), g.Text(title)),
		h.Div(h.Class("mat-desc"), g.Text(desc)),
	)
}

func materialsCanvas() g.Node {
	return section("Materials", h.Div(h.Class("mat-grid"),
		matTile(h.Div(h.Class("mat-swatch mat-parchment")), "Parchment",
			"surface + ink grain, paper bevel, 3px hard drop. The content material."),
		matTile(h.Div(h.Class("mat-swatch mat-wood")), "Wood chrome",
			"plank lines + raised bevel + 2px near-black outline. Topbar, tags, frames."),
		matTile(h.Div(h.Class("mat-swatch mat-well")), "Carved well",
			"inset bevel — things carved into the wood: tool rows, the chat input."),
		matTile(h.Div(h.Class("mat-swatch mat-ornate")), "Ornate parchment",
			"gold-bordered — reserved for panels that matter: proposals, choices, dialogs."),
		matTile(h.Div(h.Class("mat-swatch mat-frame"), ui.FolkBand()), "Folk band",
			"the woven carpet stripe — the boldest motif, used sparingly."),
		matTile(h.Div(h.Class("mat-swatch mat-frame"), ui.Stitch()), "Stitch · square corners",
			"2px dashed folk separator. Radius is 0 — RPG panels never round. No blur, ever."),
	))
}
```

- [ ] **Step 4: Register the story**

In `internal/feature/storybook/story.go`, add immediately AFTER the `{"typography", "Foundations", "Typography", typographyCanvas},` line:
```go
	{"materials", "Foundations", "Materials", materialsCanvas},
```

- [ ] **Step 5: Append the Materials CSS**

At the END of `internal/web/assets/static/basm.css`:
```css

/* ── Foundations · Materials — the material specimens ───────────────────── */
.mat-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(230px, 100%), 1fr)); gap: 16px; }
.mat-tile { display: flex; flex-direction: column; gap: 10px; }
.mat-swatch { height: 96px; }
.mat-title { font-family: var(--font-mono); font-size: 11px; text-transform: uppercase; color: var(--fg-strong); }
.mat-desc { font-family: var(--font-mono); font-size: 10.5px; color: var(--muted); line-height: 1.5; }
.mat-parchment { background-color: var(--surface); background-image: var(--grain-ink); background-size: 4px 4px; border: 2px solid var(--parch-edge); box-shadow: var(--parch-bevel); }
.mat-wood { background-color: var(--chrome); background-image: var(--wood-planks), var(--grain-warm); background-size: auto, 4px 4px; border: 2px solid var(--outline-2); box-shadow: var(--bevel-up); }
.mat-well { background-color: var(--chrome-2); background-image: var(--grain-warm); background-size: 4px 4px; border: 2px solid var(--outline-2); box-shadow: var(--bevel-in); }
.mat-ornate { background-color: var(--surface); background-image: var(--grain-ink); background-size: 4px 4px; border: 2px solid var(--gold-deep); box-shadow: var(--parch-bevel); }
.mat-frame { background: var(--bg); border: 2px solid var(--outline-2); display: flex; align-items: center; justify-content: center; padding: 0 18px; }
```

- [ ] **Step 6: Run + commit**

```bash
cd /home/alex/Projects/balaur
go test ./internal/feature/storybook/ -run TestMaterialsCanvas && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./... && go vet ./...
tail -12 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"
git status --short
```
Expected: PASS, suite green, build+vet clean, NO RAW HEX. Stage only the four files, then:
```bash
git add internal/feature/storybook/storybook.go internal/feature/storybook/story.go internal/web/assets/static/basm.css internal/feature/storybook/story_test.go
git commit -m "$(printf 'feat(ui): Materials foundation page (6 material specimens)\n\nmaterialsCanvas — Parchment, Wood chrome, Carved well, Ornate parchment, Folk\nband (composes ui.FolkBand), Stitch (composes ui.Stitch). New tokenized .mat-*\nCSS. Registered under Foundations.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Final verification (controller — visual)

- [ ] `go vet ./... && go test ./... && CGO_ENABLED=0 go build ./... && git diff --check` — green.
- [ ] Sidebar shows a **Foundations** group (first content group) with Colors / Typography / Materials; `/storybook/colors|typography|materials` render 200 (content-assert the class, not just status).
- [ ] Screenshot **Colors in Hearth + Forest + Dungeon** — confirm the swatches re-tint (the payoff: Page&wood + Accents groups change, Parchment&ink stays roughly constant).
- [ ] Screenshot Typography + Materials once (Hearth) — fonts/materials render.

## What this delivers / what's next

**Delivered:** the three Foundation pages; Colors is the live theme showcase. The themes + foundations work (the user's request) is complete.

**Next:** back to the organism track (DialogueChoices, Composer, inline cards) → the chat-wiring slice → Phase 3 (storybook → `/`) → cut boards + delete `web/`.
