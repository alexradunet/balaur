# Plan 147: Remove the never-used Composer "deciding mode"

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat ab2c0a9..HEAD -- internal/ui/composer.go internal/ui/composer_test.go internal/feature/storybook/stories_chat.go internal/web/assets/static/basm.css internal/web/home.go internal/ui/chat/choices.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `ab2c0a9`, 2026-06-22

## Why this matters

`internal/ui/composer.go` carries a whole "deciding mode" — props (`Prompt`,
`Choices`, `Decision`), a `ComposerChoice` type, a `composerChoices` helper, two
render branches, and ~15 lines of CSS — that **no production code path ever
exercises**. The only live composer (`composerNode` in `internal/web/home.go`)
constructs the draft form alone; it never sets `Prompt`, `Choices`, or
`Decision`. The live in-chat decision UI is rendered by a different, separately
maintained component — `chat.Choices` in `internal/ui/chat/choices.go` — which
is what the gateway actually uses. So the composer's deciding mode is duplicate,
divergent dead weight: it exists only in the storybook catalog and a unit test.

Removing it shrinks `composer.go` (the file is one of the design system's atoms,
so keeping it lean matters), deletes a misleading second source of truth for
"how a choice is rendered", and follows the repo's SUCKLESS rule ("delete dead
code rather than commenting it out; one source of truth per concern"). After
this lands, `ui.Composer` does exactly one thing — the owner's draft seat — and
`chat.Choices` is the unambiguous home for embedded dialogue choices.

## Current state

The facts the executor needs, inlined.

### Files involved

- `internal/ui/composer.go` — the Composer atom. Defines `ComposerChoice`,
  `ComposerProps` (with the deciding fields), the `Composer()` function with its
  draft/choices/decision switch, and the `composerChoices()` helper. **Primary
  edit.**
- `internal/ui/composer_test.go` — has two tests for the deciding mode
  (`TestComposerDeciding`, `TestComposerDecision`) plus two for draft mode
  (`TestComposer`, `TestComposerDefaults`). **Edit: remove the two deciding
  tests.**
- `internal/feature/storybook/stories_chat.go` — `composerStory()` declares the
  composer's variants and props; four of its five `Variants` and two `Props`
  rows describe the deciding mode. **Edit: trim those + fix an orphaned import.**
- `internal/web/assets/static/basm.css` — ~15 lines of composer-deciding-only
  CSS. **Edit: delete the dead rules (keep the shared ones).**
- `internal/web/home.go` — `composerNode()` is the ONLY production caller; it
  uses draft fields only. **Read-only reference; do NOT edit.**
- `internal/ui/chat/choices.go` — the live replacement (`chat.Choices`). **Do
  NOT edit; it stays.**

### Excerpt: the dead code in `internal/ui/composer.go`

The `ComposerChoice` type (`internal/ui/composer.go:10-15`):

```go
// ComposerChoice is one embedded dialogue option: a Label (the spoken reply)
// and an optional Hint.
type ComposerChoice struct {
	Label string
	Hint  string
}
```

The deciding doc-comment + deciding props on `ComposerProps`
(`internal/ui/composer.go:22-40` — note the `Prompt`/`Choices`/`Decision`
fields are the deciding props; the `Who`/`AvatarSrc`/`Placeholder`/`Hint`/
`SendLabel`/`Tools` above them, and the `PostURL`/`ID`/`Disabled`/`Palette`
below them, are draft-mode props that STAY):

```go
// Prompt, Choices and Decision switch the composer into "deciding" mode: the
// draft is replaced in place by the decision Balaur surfaced — embedded dialogue
// choices, or a Decision card (a TaskCard's Done/Snooze/Drop, a proposed
// KnowledgeCard's Approve/Dismiss). Every owner decision is taken here, so the
// owner only ever looks in one place.
//
// Decision is a pre-rendered card node, not a card package import: internal/ui
// must not depend on internal/feature/* (they compose ui), so the caller renders
// the card and hands it in — the same dependency-injection the export uses.
type ComposerProps struct {
	Who         string
	AvatarSrc   string
	Placeholder string
	Hint        string
	SendLabel   string
	Tools       []string
	Prompt      string
	Choices     []ComposerChoice
	Decision    g.Node
	// ... PostURL/ID/Disabled/Palette below stay ...
```

The `deciding` flag + kicker + the switch's two dead branches + the deciding
root class (`internal/ui/composer.go:62-133`, the load-bearing lines):

```go
func Composer(p ComposerProps, attrs ...g.Node) g.Node {
	deciding := len(p.Choices) > 0 || p.Decision != nil   // line 63
	live := p.PostURL != ""
	// ... hint/send/tools defaults (KEEP) ...
	// ... toolRow build (KEEP) ...

	// Center slot of the top row: empty in draft mode, the prompt kicker when deciding.
	kicker := h.Div()                                      // line 90
	if deciding {
		label := p.Prompt
		if label == "" {
			label = "Your word"
		}
		kicker = h.Div(h.Class("composer-kicker"), g.Text(label))
	}                                                     // line 97

	var main g.Node                                        // line 99
	switch {
	case len(p.Choices) > 0:                               // DEAD branch
		main = composerChoices(p.Choices)
	case p.Decision != nil:                                // DEAD branch
		// A surfaced TaskCard / KnowledgeCard, carried in by the caller.
		main = h.Div(h.Class("composer-decision"), p.Decision)
	default:                                               // the ONLY live path
		ta := []g.Node{ /* ... textarea ... */ }
		// ... draft form build (KEEP) ...
		main = h.Form(formAttrs...)
	}                                                     // line 128

	rootCls := "composer"                                 // line 130
	if deciding {
		rootCls += " composer-deciding"
	}                                                     // line 133
	// ... root assembly uses kicker + main (KEEP, but kicker is always h.Div() now) ...
```

The `composerChoices` helper (`internal/ui/composer.go:161-184`):

```go
// composerChoices renders the embedded dialogue choices — numbered choice
// buttons plus a final manual-input row — so a decision is answered without
// ever leaving the composer.
func composerChoices(choices []ComposerChoice) g.Node {
	panel := []g.Node{h.Class("choices-panel composer-choices")}
	// ... builds numbered .choice buttons + a .choice-type manual-input row ...
	return h.Div(panel...)
}
```

Note `composer.go` imports `"strconv"` (line 4) — `composerChoices` is the ONLY
user of `strconv` in the file (it calls `strconv.Itoa`). Removing the helper
orphans that import; **you must remove the `strconv` import in the same step** or
the build fails (staticcheck/compiler).

### Excerpt: the only production caller is draft-only (`internal/web/home.go:43-53`)

```go
func composerNode(d homeData) g.Node {
	return ui.Composer(ui.ComposerProps{
		AvatarSrc:   d.SoulAvatarURL,
		Placeholder: d.ChatPlaceholder,
		Hint:        "enter speaks · / for pages",
		PostURL:     "/ui/chat",
		ID:          "chat-draft",
		Disabled:    !d.ChatReady,
		Palette:     commandPaletteNode(),
	})
}
```

It sets no `Prompt`, no `Choices`, no `Decision` — confirming the deciding mode
is unreachable in production.

### Excerpt: the live replacement that STAYS (`internal/ui/chat/choices.go`)

`chat.Choices` (used by the web gateway) renders the live dialogue panel with
its own `ChoiceItem` type. It shares the CSS classes `.choices-panel`,
`.choice`, `.choice-key`, `.choice-label`, `.choice-hint` with the dead
composer code, so **those CSS rules must NOT be deleted** — only the
composer-deciding-only rules below are dead. `choices.go:32-43`:

```go
for i, c := range p.Choices {
	kids := []g.Node{
		h.Class("choice"), h.Type("button"),
		// ...
		h.Span(h.Class("choice-key"), g.Text(strconv.Itoa(i+1))),
		h.Span(h.Class("choice-label"), g.Text(c.Label)),
	}
	if c.Hint != "" {
		kids = append(kids, h.Span(h.Class("choice-hint"), g.Text(c.Hint)))
	}
	buttons = append(buttons, h.Button(kids...))
}
```

### Excerpt: composer-deciding-only CSS (`internal/web/assets/static/basm.css:3075-3094`)

```css
@media (max-width: 540px) { .composer-top { grid-template-columns: 1fr auto; } }
.list-title, .list-sub { overflow-wrap: anywhere; }

/* Composer "deciding" mode — a decision (dialogue choices) is embedded in place
   of the draft: gold ledge, dimmed tools, the prompt as the top-row kicker. */
.composer-deciding { border-color: var(--gold-deep); }
.composer-deciding .composer-tool { opacity: .38; pointer-events: none; }
.composer-kicker {
  min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; text-align: center;
  font-family: var(--font-mono); font-size: 10px; font-weight: 700; letter-spacing: .08em;
  text-transform: uppercase; color: var(--gold);
}
.composer-choices { min-height: 0; }
.composer-decision { min-width: 0; }
.choice-type { align-items: center; cursor: text; }
.choice-type input {
  flex: 1 1 auto; min-width: 0; background: transparent; border: 0; padding: 0; outline: none;
  font: 16px/1.45 var(--font-body); color: var(--ink); caret-color: var(--ember-deep);
}
.choice-type input::placeholder { color: var(--ink-muted); }
```

Delete the block from the comment `/* Composer "deciding" mode … */` through
`.choice-type input::placeholder { … }` inclusive — that is lines 3078-3094.
**Keep** the `@media (max-width: 540px) { .composer-top … }` rule at line 3075
(`.composer-top` is used by draft mode) and the `.list-title, .list-sub` rule at
line 3076.

### Excerpt: the storybook story (`internal/feature/storybook/stories_chat.go`)

`composerStory()` (starts at line 132). Its `Blurb` (line 135), `Variants` (lines
136-172), and `Props` (lines 173-183) describe the deciding mode. The four
deciding variants to remove are at lines 138-171 (`"deciding · choices"`,
`"deciding · task"`, `"deciding · memory"`, `"deciding · guardian"`); the lone
surviving variant is `"draft"` at line 137. Two Props rows to remove:

```go
{"Prompt", "string", `"Your word"`, "Kicker question shown in the top row when deciding."},
{"Choices", "[]ComposerChoice", "nil", "When set, the draft is replaced by these numbered choices + a manual-input row."},
{"Decision", "g.Node", "nil", "A surfaced card (TaskCard / KnowledgeCard / GuardianCard, rendered by the caller) shown in place of the draft — its own actions are the decision."},
```

**Import-orphan trap (critical):** `stories_chat.go` imports
`knowledgecards` and `taskcards`. After removing the deciding variants:

- `knowledgecards` is used ONLY by the `"deciding · memory"` variant (line 157).
  Removing that variant orphans the import → build fails. **You must remove the
  `knowledgecards` import line.**
- `taskcards` is ALSO used by other stories (`chatdockStory` line 199,
  `chatclusterStory` lines 251-260, `chatpanelStory` line 296), so its import
  STAYS.
- `ui` and `chat` are used throughout the file, so their imports STAY.

Current import block (`internal/feature/storybook/stories_chat.go:3-11`):

```go
import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/feature/knowledgecards"
	"github.com/alexradunet/balaur/internal/feature/taskcards"
	"github.com/alexradunet/balaur/internal/ui"
	"github.com/alexradunet/balaur/internal/ui/chat"
)
```

### Repo conventions that apply

- gomponents: `h "maragu.dev/gomponents/html"`, `g "maragu.dev/gomponents"`;
  text renders through escaping `g.Text`. (No change needed — just don't break
  it.)
- SUCKLESS / dead-code: delete, don't comment out. `staticcheck` U1000 and the
  compiler fail the build on an unused symbol/import — so orphaned imports
  (`strconv` in composer.go, `knowledgecards` in stories_chat.go) MUST be
  removed in the same step as the code that used them.
- The storybook is the component source-of-truth and a build gate: when a
  component's surface changes, its story changes in the same commit. That is why
  the story edit is in scope, not deferred.

## Commands you will need

| Purpose    | Command                                       | Expected on success |
|------------|-----------------------------------------------|---------------------|
| Build      | `CGO_ENABLED=0 go build ./...`                | exit 0              |
| Vet        | `go vet ./...`                                | exit 0              |
| Test (pkg) | `go test ./internal/ui/...`                   | ok / all pass       |
| Test (sb)  | `go test ./internal/feature/storybook/...`    | ok / all pass       |
| Test (all) | `go test ./...`                               | all pass            |
| Format     | `gofmt -l internal/ui/composer.go internal/ui/composer_test.go internal/feature/storybook/stories_chat.go` | prints nothing |
| Lint       | `make lint`                                   | exit 0              |
| Diff check | `git diff --check`                            | no whitespace errors|

(A `PostToolUse` hook runs `gofmt -w` on every edited `.go` file, so formatting
stays clean automatically — but still run the `gofmt -l` gate to be sure.)

## Scope

**In scope** (the only files you should modify):

- `internal/ui/composer.go`
- `internal/ui/composer_test.go`
- `internal/feature/storybook/stories_chat.go`
- `internal/web/assets/static/basm.css`
- `plans/readme.md` (status row only — unless a reviewer says they own it)

**Out of scope** (do NOT touch, even though they look related):

- `internal/web/home.go` — the live caller; it already uses draft fields only.
  No change needed; touching it is out of scope.
- `internal/ui/chat/choices.go` and its story `chatchoicesStory()` — this is the
  live replacement. It stays exactly as-is.
- The SHARED CSS rules `.choices-panel` (basm.css ~748), `.choice` (~807),
  `.choice-key`/`.choice-label`/`.choice-hint` (~827-845) — used by
  `chat.Choices`. Do NOT delete these.
- The `taskcards`, `ui`, `chat` imports in `stories_chat.go` — still used by
  other stories. Only `knowledgecards` becomes orphaned.
- Any change to draft-mode behavior of `Composer` (textarea, send, tools,
  palette, live wiring). The draft path is the product; leave it untouched.

## Git workflow

- Branch: `advisor/147-composer-drop-deciding-mode` (if you create one).
- Conventional-commit subject, e.g.
  `refactor(ui/composer): drop never-used deciding mode (plan 147)`.
- Do NOT commit or push unless the operator explicitly tells you to. Make the
  change, run the gates, and report. If you do commit, end the message with the
  repo's Co-Authored-By trailer.

## Steps

### Step 1: Strip the deciding mode out of `internal/ui/composer.go`

Make these edits to `internal/ui/composer.go`:

1. Delete the `ComposerChoice` type (lines 10-15, the doc-comment + the struct).
2. From `ComposerProps`, delete the three deciding fields and trim the
   deciding-specific paragraphs of the doc-comment. Specifically:
   - Remove the doc-comment paragraphs that begin
     `// Prompt, Choices and Decision switch the composer into "deciding" mode:`
     and `// Decision is a pre-rendered card node, …` (lines 22-30). Keep the
     first paragraph (lines 17-20, describing the draft props).
   - Remove the struct fields `Prompt string`, `Choices []ComposerChoice`, and
     `Decision g.Node` (lines 38-40). Keep every other field.
3. In `Composer()`:
   - Delete the `deciding := len(p.Choices) > 0 || p.Decision != nil` line
     (line 63). Keep `live := p.PostURL != ""`.
   - Replace the kicker block (lines 90-97) with just `kicker := h.Div()` — the
     top-row center slot is always the empty placeholder now. (The root assembly
     at line 150 still references `kicker`, so it must remain a defined node.)
   - Collapse the `switch` (lines 99-128) to the draft path only. Remove the
     `case len(p.Choices) > 0:` and `case p.Decision != nil:` arms; keep the
     body that was under `default:`. The result should build `main` as the draft
     `h.Form(...)` unconditionally. Concretely the shape becomes:

     ```go
     ta := []g.Node{h.Name("message"), h.Placeholder(p.Placeholder), h.Rows("2"), g.Attr("autocomplete", "off")}
     if live {
         ta = append(ta, g.Attr("data-bind:message"), g.Attr("onkeydown", "balaurSubmitOnEnter(event)"), h.Required(), h.AutoFocus())
     }
     if p.Disabled {
         ta = append(ta, h.Disabled())
     }
     formAttrs := []g.Node{h.Class("composer-form")}
     if live {
         formAttrs = append(formAttrs, g.Attr("data-on:submit", "@post('"+p.PostURL+"')"))
     }
     formAttrs = append(formAttrs,
         h.Div(h.Class("composer-draft"),
             h.Textarea(ta...),
             h.Div(h.Class("composer-foot"),
                 h.Span(h.Class("composer-hint"), g.Text(hint)),
                 Button(ButtonProps{Size: "sm"}, h.Type("submit"), g.If(p.Disabled, h.Disabled()), g.Text(send)),
             ),
         ),
     )
     main := h.Form(formAttrs...)
     ```

     (Note: it becomes `main := h.Form(...)` with `:=`, since the `var main
     g.Node` declaration that fed the switch is gone.)
   - Delete the `rootCls := "composer"` + `if deciding { rootCls += " composer-deciding" }`
     block (lines 130-133) and replace the single use of `rootCls` with the
     literal: change `root := []g.Node{h.Class(rootCls)}` (line 135) to
     `root := []g.Node{h.Class("composer")}`.
4. Delete the entire `composerChoices` function (lines 161-184).
5. Remove the now-unused `"strconv"` import (line 4) — `composerChoices` was its
   only user.

Update the `Composer` doc-comment (lines 56-61) if it still claims a choices
branch ("either the parchment draft … or — when Choices are given — the embedded
dialogue choices") — reword to describe the draft only, e.g. "renders the wood
input ledge: corner brackets, a top row of tool wells + a sound toggle + the
soul portrait, and the parchment draft (textarea + send)." Keep it short; this
is a comment, not behavior.

**Verify**:
- `gofmt -l internal/ui/composer.go` → prints nothing.
- `CGO_ENABLED=0 go build ./internal/ui/...` → exit 0 (will FAIL here if the
  deciding tests still reference deleted symbols; that is expected — they get
  fixed in Step 2, but the non-test build of the package should pass). If `go
  build` of the package fails on an unused `strconv` import, you missed step 1.5.

### Step 2: Remove the two deciding tests from `internal/ui/composer_test.go`

Delete `TestComposerDeciding` (lines 45-66) and `TestComposerDecision` (lines
68-86) in their entirety. Keep `TestComposer` (lines 12-33) and
`TestComposerDefaults` (lines 35-43).

After deletion, check the imports at the top of the file (lines 3-10). The
`g "maragu.dev/gomponents"` import is used only by `TestComposerDecision`
(`g.El(...)`, `g.Attr(...)`, `g.Text(...)` at line 69). If no remaining test in
the file uses `g.`, remove that import line too. Verify with:
`grep -n 'g\.' internal/ui/composer_test.go` — if it returns nothing, delete the
`g "maragu.dev/gomponents"` import. The `strings`, `testing`, and `ui` imports
stay (used by the surviving tests).

**Verify**:
- `gofmt -l internal/ui/composer_test.go` → prints nothing.
- `go test ./internal/ui/...` → ok, all remaining tests pass.

### Step 3: Trim the deciding mode from the storybook story

In `internal/feature/storybook/stories_chat.go`, `composerStory()`:

1. In `Variants` (lines 136-172), delete the four deciding variants — the
   `"deciding · choices"` (138-146), `"deciding · task"` (147-153), `"deciding ·
   memory"` (154-161), and `"deciding · guardian"` (162-171) entries. Keep the
   `"draft"` variant (line 137). The `Variants` slice should then contain only
   the draft entry.
2. In `Props` (lines 173-183), delete the `"Prompt"`, `"Choices"`, and
   `"Decision"` rows (lines 180-182). Keep the `Who`/`AvatarSrc`/`Placeholder`/
   `Hint`/`SendLabel`/`Tools` rows.
3. Update the `Blurb` (line 135) so it no longer promises deciding behavior —
   it currently reads "Draft mode is the textarea; when Balaur surfaces a
   decision it embeds in place of the draft: dialogue choices …, a TaskCard …,
   a proposed KnowledgeCard …, or a GuardianCard …". Reduce it to describe the
   draft seat only, e.g.: "The owner's single seat of action — every input is
   given here. The parchment draft is a textarea with a send button, the tool
   wells, the sound toggle, and the owner's soul portrait." Keep it one or two
   sentences.
4. Remove the now-orphaned `knowledgecards` import (line 7 of the import block):
   `"github.com/alexradunet/balaur/internal/feature/knowledgecards"`. Do NOT
   remove `taskcards`, `ui`, `chat`, `g`, or `h` — they are still used by other
   stories in this file.

**Verify**:
- `grep -n 'knowledgecards' internal/feature/storybook/stories_chat.go` → prints
  nothing (import and only use are both gone).
- `grep -n 'taskcards' internal/feature/storybook/stories_chat.go` → still prints
  lines (its other uses remain — confirms you did NOT over-delete).
- `gofmt -l internal/feature/storybook/stories_chat.go` → prints nothing.
- `go test ./internal/feature/storybook/...` → ok.

### Step 4: Delete the composer-deciding-only CSS

In `internal/web/assets/static/basm.css`, delete the block from the comment
`/* Composer "deciding" mode … */` through `.choice-type input::placeholder { … }`
inclusive — lines 3078-3094 in the "Current state" excerpt above (eight rules:
`.composer-deciding`, `.composer-deciding .composer-tool`, `.composer-kicker`,
`.composer-choices`, `.composer-decision`, `.choice-type`, `.choice-type input`,
`.choice-type input::placeholder`, plus the leading comment).

Do NOT delete:
- The `@media (max-width: 540px) { .composer-top … }` rule (line 3075 above) —
  `.composer-top` is draft chrome and stays.
- The shared `.choices-panel` / `.choice` / `.choice-key` / `.choice-label` /
  `.choice-hint` rules (around basm.css lines 748-845) — used by `chat.Choices`.

**Verify**:
- `grep -n 'composer-deciding\|composer-kicker\|composer-choices\|composer-decision\|choice-type' internal/web/assets/static/basm.css`
  → prints nothing.
- `grep -n 'composer-top\|choices-panel\|choice-key\|choice-label\|choice-hint' internal/web/assets/static/basm.css`
  → still prints lines (the kept rules survive).
- `git diff --check` → no whitespace errors.

### Step 5: Confirm no surviving references and run the full gates

**Verify**:
- `grep -rn 'ComposerChoice\|composerChoices\|composer-deciding\|composer-kicker\|composer-decision\|composer-choices' internal/ main.go`
  → prints nothing. (This is the definitive "no dead reference left" check —
  including in production code, tests, and stories.)
- `CGO_ENABLED=0 go build ./...` → exit 0.
- `go vet ./...` → exit 0.
- `go test ./...` → all pass.
- `make lint` → exit 0 (staticcheck must report no U1000 unused-symbol/import
  errors from this change).
- `git diff --check` → no whitespace errors.
- `git status --porcelain` → shows ONLY the four in-scope source files (and, if
  you updated it, `plans/readme.md`). No file outside the in-scope list.

## Test plan

This is a deletion, so the "test" is: the surviving draft tests still pass and
no new symbol is dangling. No new tests are needed.

- Existing tests that MUST still pass after the change:
  `TestComposer` and `TestComposerDefaults` in `internal/ui/composer_test.go`
  (they exercise the draft path, which is unchanged).
- Tests removed in this plan: `TestComposerDeciding`, `TestComposerDecision`
  (they only covered the deleted deciding mode).
- The storybook package test (`internal/feature/storybook`) must still build and
  pass — it renders every story, so a broken `composerStory()` would fail it.
- Verification: `go test ./...` → all pass, with the two deciding tests gone and
  the storybook still rendering.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0; `TestComposer` and `TestComposerDefaults` still pass
- [ ] `make lint` exits 0 (no U1000 unused symbol/import)
- [ ] `grep -rn 'ComposerChoice\|composerChoices' internal/ main.go` returns no matches
- [ ] `grep -n 'composer-deciding\|composer-kicker\|composer-decision\|composer-choices\|choice-type' internal/web/assets/static/basm.css` returns no matches
- [ ] `grep -n 'knowledgecards' internal/feature/storybook/stories_chat.go` returns no matches
- [ ] `grep -n 'choices-panel' internal/web/assets/static/basm.css` STILL returns a match (shared CSS preserved)
- [ ] `git diff --check` shows no whitespace errors
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/readme.md` status row updated (unless a reviewer maintains it)

## STOP conditions

Stop and report back (do not improvise) if:

- The code at the locations in "Current state" doesn't match the excerpts — e.g.
  `composer.go` no longer has the `deciding` flag at line 63 or the
  `composerChoices` helper, or the line numbers are off by more than a few. The
  tree has drifted; re-verify before deleting anything.
- `grep -rn 'ComposerChoice\|composerChoices' internal/ main.go` finds a
  reference in a file OTHER than `internal/ui/composer.go`,
  `internal/ui/composer_test.go`, and `internal/feature/storybook/stories_chat.go`
  — especially any file under `internal/web/`, `internal/turn/`, or
  `internal/agent/`. That would mean a PRODUCTION caller exists and the finding
  is wrong. **Do not delete; report the caller's file:line.**
- `internal/web/home.go`'s `composerNode` (or any other production composer
  construction) is found to set `Prompt`, `Choices`, or `Decision`. The deciding
  mode would then be live — STOP.
- After Step 4, `go build ./...` fails with an unused-import error you cannot
  trace to `strconv` (composer.go) or `knowledgecards` (stories_chat.go) — an
  unexpected orphan means the dependency graph differs from this plan; report it.
- Any verification command fails twice after a reasonable fix attempt.
- The fix appears to require editing a file not in the in-scope list.

## Maintenance notes

For the human/agent who owns this code after the change lands:

- `ui.Composer` is now draft-only: one input surface, one render path. If a
  future feature needs an embedded decision UI in the composer ledge, reuse or
  extend `chat.Choices` (`internal/ui/chat/choices.go`) — the live, maintained
  component — rather than reviving a parallel path inside the composer.
- A reviewer should scrutinize: (1) that the SHARED `.choice*`/`.choices-panel`
  CSS was preserved (a `chat.Choices` visual regression would mean those rules
  were over-deleted); (2) that `taskcards`/`ui`/`chat` imports in
  `stories_chat.go` survived (only `knowledgecards` should be gone); (3) that the
  `strconv` import in `composer.go` is gone (its only user was deleted).
- Deferred out of this plan (deliberately): nothing. This is a complete deletion
  of the deciding mode — props, type, helper, render branches, tests, story, and
  CSS — leaving `chat.Choices` as the single source of truth for embedded
  dialogue choices.
