# Plan 025: Adopt the Hearthwood (Basm v2) visual system as the product stylesheet

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 9c77f42..HEAD -- web/static web/templates internal/web DESIGN.md`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: LOW–MED (pure presentation; no behavior change; risk is unstyled corners)
- **Depends on**: none
- **Category**: direction
- **Planned at**: commit `9c77f42`, 2026-06-12

## Why this matters

The owner commissioned a new canon for Balaur's design system: **Hearthwood**
(Basm v2) — tavern-oak chrome, candlelit gold, parchment content panels, 16-bit
bevels. It ships as a complete design-system export in `Balaur_ds/` whose CSS
was authored **against the existing template class vocabulary** (`.kcard`,
`.recap-card`, `.cal-cell`, `.topbar`, `.chatbar`, `.model-switcher`, …), so
the bulk of the migration is a stylesheet replacement plus new fonts and
pixel-art icons. This plan lands the visual foundation; later plans (026+)
restructure chat markup and add new surfaces on top of it.

## Current state

- `web/static/basm.css` (1379 lines) — the current "Forest at Dusk" stylesheet.
  Its header declares it the canonical token source:

  ```css
  /* basm.css:1-6 */
  /* Basm — Balaur design tokens and base styles.
     Canonical token source: if DESIGN.md and this file disagree, this file wins. */
  :root {
    color-scheme: dark light;
    --bg:        light-dark(#f4efe4, #0b1310);   /* Forest at Dusk */
  ```

- The Hearthwood export lives at
  `Balaur_ds/_ds/balaur-basm-design-system-0c1b20fd-0bf4-4b2c-bbd1-bfa417af0a6b/`
  (call it `$DS` below). Its CSS is split into:
  - `$DS/tokens/colors.css` — 54 tokens; e.g. `--bg: light-dark(#efe2bd, #140c06)`,
    parchment `--surface: light-dark(#f4e9c4, #e8d9ae)`, constant `--ink: #2c2012`,
    wood `--chrome`, grain/dither gradients.
  - `$DS/tokens/layout.css` — `--radius: 0px`, bevel shadow system
    (`--bevel-up/in`, `--parch-bevel`, `--drop-hard`), motion + `--focus-ring`.
    Note it keeps a legacy alias `--shadow-hard: 5px 5px 0`.
  - `$DS/tokens/typography.css` — `--font-display: 'Jersey 15', 'Pixelify Sans', …`,
    `--font-body: 'Piazzolla', Georgia, …serif`, plus all `@font-face` rules.
    Font URLs there are `url('../fonts/…')` — they must become
    `/static/fonts/…` in the merged product stylesheet.
  - `$DS/basm/base.css` — reset, `html,body` (17px Piazzolla on oak grain),
    `.px`, materials `.parch`, `.wood`, `.ornate`.
  - `$DS/basm/components.css` — `.topbar`, `.btn*` (incl. new `.btn-wood`),
    `.card`, `.stitch`, `.folk-band`, `.tag`, the full RPG chat block
    (`.msg`, `.portrait`, `.msg-main` with speech tails, `.msg-draft`,
    `.msg-tool`, `.choices*`), `.balaur-avatar`, `basm-glow`, `.chatbar`,
    `dialog`, `.theme-toggle`, reduced-motion block.
  - `$DS/basm/pages.css` — `.h-icon`, knowledge (`.k-*`, `.kcard*`, `.pip`),
    `.card-note(-error)`, recap (`.recap-*`), tasks (`.tcard-*`, `.capture-note`,
    `.task-fresh`), calendar (`.cal-*`), timeline (`.tl-*`), life (`.life-*`,
    `.spark`, `.habit-*`), day (`.day-*`, `.journal-*`), avatar picker
    (`.avatar-choice*`), profile (`.profile-*`), model switcher (`.model-*`).

- New font binaries in `$DS/fonts/`: `jersey-15.ttf`, `piazzolla.ttf`
  (the four existing woff2 files are also there, identical roles).
- 13 pixel-art PNG icons in `Balaur_ds/assets/icons/`:
  `bell check flame gem hourglass key lens orb quill rune_x scroll shield tome`.
- Tool rows currently use Unicode glyphs via the `toolIcon` template func.
  `web/templates/chat-messages.html:30`:

  ```html
  <div class="who"><span class="tool-icon" aria-hidden="true">{{toolIcon .Tool}}</span>tool · {{.Tool}}</div>
  ```

  `toolIcon` is registered in the `funcs` map in `internal/web/web.go` and
  implemented as `toolGlyph()` in `internal/web/chat.go` (a `switch` on tool
  name returning a one-character string). Hearthwood replaces the glyph with a
  bare pixel PNG (`img.tool-icon`, 18px, styled in `$DS/basm/components.css:368-374`).
- `DESIGN.md` section 4 ("Visual system") still documents Forest at Dusk,
  Pixelify Sans/Work Sans, `--radius: 3px`, `--shadow-hard: 5px 5px 0`, and
  Unicode-glyph icons. Per its own contract ("If prose and code disagree,
  `web/static/basm.css` wins"), it must be rewritten to Hearthwood in the same
  change.
- Templates are embedded via `web/embed.go` (`embed.FS` over `templates/` and
  `static/`); new static files under `web/static/` are picked up automatically
  if the embed pattern covers the directory — verify, don't assume (Step 2).

## Commands you will need

| Purpose   | Command                          | Expected on success |
|-----------|----------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`   | exit 0              |
| Tests     | `go test ./...`                  | all packages ok     |
| Vet       | `go vet ./...`                   | exit 0              |
| Fmt check | `gofmt -l .`                     | no output           |
| Run (manual look) | `make run` then open `http://127.0.0.1:8090/` | pages render |

Sandbox note: in a TLS-intercepting sandbox, Go module downloads need the
GOPROXY shim — see `docs/hyperagent-sandbox.md`.

## Scope

**In scope** (the only files you should modify/create):
- `web/static/basm.css` (replace content)
- `web/static/fonts/jersey-15.ttf`, `web/static/fonts/piazzolla.ttf` (copy in)
- `web/static/icons/*.png` (create dir, copy 13 icons)
- `web/templates/chat-messages.html` (tool-icon `<img>` swap only)
- `web/templates/knowledge.html`, `web/templates/tasks.html`,
  `web/templates/life.html`, `web/templates/heads.html` (page-heading icons only)
- `internal/web/chat.go` (`toolGlyph` → icon-file mapping), `internal/web/web.go`
  (funcs map entry) — smallest change that swaps glyph for image
- `DESIGN.md` (section 4 + typography/motif prose)
- Any `*_test.go` under `internal/web` whose assertions mention the old glyphs

**Out of scope** (do NOT touch, even though they look related):
- Chat message *structure* (portraits, nameplates, draft composer) — plan 026.
- `Balaur_ds/` itself — it is the read-only source export. Never edit it.
- `web/static/basm.js`, `web/static/htmx.min.js` — no JS changes in this plan.
- `ds-base.js`, `support.js`, `_ds_bundle.js`, `Balaur App.dc.html` — design-tool
  runtime; nothing from them ships.
- Avatars, crest, logo PNGs — unchanged.

## Git workflow

- Branch: `advisor/025-hearthwood-visual-foundation`
- Commit style: conventional, e.g. `feat(ui): adopt Hearthwood (Basm v2) tokens, fonts, and pixel icons`
  (matches repo history: `feat(llm): …`, `fix(tools): …`)
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Copy fonts and icons into web/static

```sh
DS='Balaur_ds/_ds/balaur-basm-design-system-0c1b20fd-0bf4-4b2c-bbd1-bfa417af0a6b'
cp "$DS/fonts/jersey-15.ttf" "$DS/fonts/piazzolla.ttf" web/static/fonts/
mkdir -p web/static/icons
cp Balaur_ds/assets/icons/*.png web/static/icons/
```

**Verify**: `ls web/static/icons | wc -l` → `13`; `ls web/static/fonts` → 6 files.

### Step 2: Confirm the embed covers the new files

Read `web/embed.go`. If the `//go:embed` directive is a directory pattern like
`templates static` or `all:static`, new files are included automatically. If it
enumerates extensions/paths that exclude `static/icons` or `.ttf`, extend the
directive minimally.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0, then
`go run . serve` briefly and `curl -sI http://127.0.0.1:8090/static/icons/scroll.png | head -1` → `HTTP/1.1 200 OK`.
(If you cannot run a server in your environment, instead write a tiny temporary
Go test that opens `webassets.FS` and stats `static/icons/scroll.png` and
`static/fonts/piazzolla.ttf`; delete it after, or keep it as a permanent
asset-presence test in `web/` — keeping it is preferred.)

### Step 3: Build the new basm.css

Replace the content of `web/static/basm.css` with, in order:

1. A header comment (keep the "canonical token source" contract sentence, note
   "Hearthwood — Basm v2 canon, imported from the Balaur_ds export").
2. `$DS/tokens/colors.css` body (including the `:root.light/:root.dark`
   override block at the bottom).
3. `$DS/tokens/typography.css` body — **rewrite every `url('../fonts/X')` to
   `url('/static/fonts/X')`**.
4. `$DS/tokens/layout.css` body.
5. `$DS/basm/base.css` body.
6. `$DS/basm/components.css` body.
7. `$DS/basm/pages.css` body.
8. A final section `/* ── Carried forward (not in the Hearthwood export) ── */`
   containing every rule block from the OLD basm.css whose selectors are not
   defined anywhere in sections 2–7. Build it like this:
   - Save the old file first: `git show HEAD:web/static/basm.css > /tmp/old-basm.css`.
   - Extract the class inventory actually used by templates:
     `grep -roh 'class="[^"]*"' web/templates | tr ' "' '\n\n' | grep -v '^class=$' | sed 's/class=//' | sort -u > /tmp/classes.txt`
   - For each class in `/tmp/classes.txt` not matched by a selector in the new
     concatenation, copy its rule block(s) from `/tmp/old-basm.css` into the
     carried-forward section and **restyle only its raw color values to
     Hearthwood tokens** (no `#hex` literals — use `var(--…)`). Expected gaps
     (from advisor recon — verify, the list may not be exhaustive): the
     settings shell (`.settings-*`), `.chatbar-download*`, `.gguf-progress-bar`,
     `.model-switcher-manage`, `.model-switcher-empty`, `.chatbar-profile-link`,
     `.chatbar-profile-href`, `.chatbar-setup` (partial), `.model-error`,
     `.modal-card`, head-chat page classes (`.chatbar-head-context`, `.head-*`),
     `.t-views` internals, `.dev-*`, responsive media queries from the old file
     that cover pages the export didn't.

**Verify**:
```sh
for c in $(cat /tmp/classes.txt); do grep -q "\.$c[ ,:{.\[]" web/static/basm.css || echo "UNSTYLED: $c"; done
```
→ no `UNSTYLED:` lines (a handful of intentionally style-free utility/state
classes may appear — e.g. `msg-with-avatar`, `balaur-avatar-live`,
`client_rendered` artifacts; list any you accept as fine in your report).

### Step 4: Swap tool glyphs for pixel icons

In `internal/web/chat.go`, find `toolGlyph` (a switch mapping tool names like
`task_add`, `remember`, `skill`, `journal_write`, `log_entry`, OS tools, etc. to
single Unicode characters). Add alongside it:

```go
// toolIconFile maps a tool name to a pixel icon under /static/icons/.
// Unmapped tools fall back to "orb" — every row gets an icon.
func toolIconFile(name string) string {
    switch {
    case strings.HasPrefix(name, "task_"):    return "scroll"
    case name == "remember", strings.Contains(name, "memor"): return "tome"
    case strings.Contains(name, "skill"):     return "key"
    case name == "journal_write":             return "quill"
    case strings.HasPrefix(name, "log_"), strings.HasPrefix(name, "entry_"): return "orb"
    case strings.Contains(name, "search"), strings.Contains(name, "recall"): return "lens"
    case strings.HasPrefix(name, "os_"), name == "bash": return "shield"
    }
    return "orb"
}
```

Match the exact tool-name set you find in `toolGlyph` — the prefixes above are
the advisor's read of `internal/tools/{tasks,knowledge,life,journal,os}.go`;
align with reality. Register it in the `funcs` template map in
`internal/web/web.go` as `toolIcon` (replacing the glyph func registration —
keep `toolGlyph` itself only if other code references it; otherwise delete it).

Then in `web/templates/chat-messages.html`, change BOTH tool fragments
(`chat-msg-tool` at line 30 and `chat-msg-tool-start` at line 50) from:

```html
<span class="tool-icon" aria-hidden="true">{{toolIcon .Tool}}</span>
```
to:
```html
<img class="tool-icon" src="/static/icons/{{toolIcon .Tool}}.png" alt="" aria-hidden="true">
```

**Verify**: `go test ./internal/web/...` → ok (fix any test asserting the old
glyph characters — change the assertion to the icon `<img`), and
`grep -n 'toolIcon' web/templates/*.html` → only the two `img` usages.

### Step 5: Page-heading icons

Hearthwood puts a bare 26px pixel icon beside page titles (`.h-icon`,
`$DS/basm/pages.css:7-13`). Add to the `<h1>`/section headings of:
- `/tasks` (`web/templates/tasks.html`) → `scroll.png`
- `/memory` (`web/templates/knowledge.html`) → `tome.png`
- `/life` (`web/templates/life.html`) → `orb.png`
- `/heads` (`web/templates/heads.html`) → `tome.png`

Pattern: `<img class="h-icon" src="/static/icons/scroll.png" alt="">` placed
immediately before the title text. Never boxed, never bordered (DESIGN.md rule:
"Art ships borderless").

**Verify**: `go test ./internal/web/...` → ok (templates parse test passes).

### Step 6: Update DESIGN.md to the Hearthwood canon

Rewrite section 4 ("Visual system") of `DESIGN.md`:
- Color tokens reference copy → the Hearthwood set (copy the dark values from
  `$DS/tokens/colors.css`; name the palette "Hearthwood"; note parchment
  surfaces are constant across modes and `--ink` is the on-parchment text).
- Typography → Jersey 15 display / Piazzolla body / Silkscreen nameplate-only /
  JetBrains Mono functional; note Jersey 15 + Piazzolla ship as TTF.
- Layout & motifs → `--radius: 0`, the bevel system (`--bevel-up/in`,
  `--parch-bevel`, `--drop-hard`), materials `.parch`/`.wood`/`.ornate`,
  buttons press-sink 3px.
- Icons → 13 pixel PNGs under `web/static/icons/` (list them and the tool→icon
  mapping from Step 4); Unicode glyph language is retired.
- Keep sections 1–3 and 5 intact except: in the checklist, change
  "3px radius, hard shadows" to the Hearthwood equivalents.
Do NOT touch the honesty ledger (no capability changed).

**Verify**: `grep -n "Forest at Dusk\|Pixelify Sans.*hero\|Work Sans.*body" DESIGN.md`
→ no matches describing them as *current* (a historical aside is fine);
`grep -c "Hearthwood" DESIGN.md` → ≥ 1.

### Step 7: Full check + visual pass

Run the full gate, then `make run` and visually open: `/`, `/tasks` (all three
views), `/memory`, `/life`, `/heads`, `/settings/profile`, `/settings/models`,
a `/day/{today}` page, and toggle light/dark. You are looking for unstyled
white boxes, unreadable text contrast, or broken layout.

**Verify**: `gofmt -l .` → empty; `go vet ./...` → exit 0; `go test ./...` →
ok; `CGO_ENABLED=0 go build ./...` → exit 0; `git diff --check` → empty.

## Test plan

- Extend `internal/web/templates_test.go` (pattern: it parses all templates via
  `ParseFS` with the same funcs map): add a render assertion that a tool
  message renders `<img class="tool-icon" src="/static/icons/` (model after the
  existing fragment-render tests in that file).
- Asset-presence test from Step 2 (embed FS contains
  `static/icons/scroll.png`, `static/fonts/piazzolla.ttf`).
- Verification: `go test ./internal/web/... ./web/...` → all pass.

## Done criteria

- [ ] `go test ./...`, `go vet ./...`, `gofmt -l .`, CGO-free build all clean
- [ ] `grep -n "0b1310" web/static/basm.css` → no matches (Forest at Dusk gone)
- [ ] `grep -n "140c06" web/static/basm.css` → at least one match (Hearthwood in)
- [ ] `grep -rn "url('../fonts" web/static/basm.css` → no matches
- [ ] Step 3 class-coverage script reports no unexplained `UNSTYLED:` lines
- [ ] 13 icons + 2 new fonts exist under `web/static/` and are embedded
- [ ] DESIGN.md describes Hearthwood; no prose claims Forest at Dusk is current
- [ ] No files outside the in-scope list modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- `web/static/basm.css` at HEAD does not start with the "Canonical token
  source" header (drift — someone already touched the stylesheet).
- The `$DS` directory or any of its six CSS files is missing or its content
  diverges materially from the excerpts above.
- The embed directive in `web/embed.go` cannot include `.ttf`/`icons` without
  restructuring the package.
- More than ~25 template classes turn up unstyled in Step 3 — that means the
  export's class vocabulary diverges from the templates more than the advisor
  measured, and the merge strategy needs rethinking.
- Any test failure you cannot attribute to an old-glyph/old-color assertion.

## Maintenance notes

- Plan 026 restructures chat markup against the `.portrait`/`.msg-main` styles
  this plan lands — land 025 first.
- The carried-forward CSS section is intentional debt: as later plans restyle
  the settings shell and head-chat pages natively in Hearthwood, shrink it.
- Reviewer should scrutinize: light-mode contrast on parchment (`--ink` on
  `--surface`), the reduced-motion block survived the merge, and that no
  `#hex` literals leaked into carried-forward rules.
- Deferred: `.msg-draft` composer adoption (plan 026 decides), folk-band use on
  app pages (sparingly, per DESIGN.md).
