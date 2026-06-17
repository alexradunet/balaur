# Plan 085: Refresh DESIGN.md's honesty ledger to match the running domain-rail + Kronk architecture

> **Executor instructions**: Follow step by step. Run every Verify and confirm before moving on. On a STOP condition, stop and report — do not improvise. When done, update the 085 row in plans/readme.md (add the row if it is not present yet, matching the existing column format).
>
> **Drift check (run first)**: `git diff --stat 12a2ff5..HEAD -- DESIGN.md internal/self/knowledge.md internal/ui/shell/shell.go internal/web/assets/static/basm.css` — if any in-scope file (DESIGN.md, internal/self/knowledge.md) changed since this plan was written, compare the "Current state" excerpts below to the live code; on mismatch, STOP and report which excerpt drifted.

## Status
- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: soft — this plan documents the state established by plans 075–082 (layout tokens) and 084 (chat.Dock port). It does NOT require them to have landed; it documents the CURRENT state and gates any claim whose truth depends on an unshipped plan. Hard dependency: none.
- **Category**: docs
- **Planned at**: commit `12a2ff5`, 2026-06-17

## Why this matters

`DESIGN.md` section 3 is named the "Honesty ledger" and opens with "All copy must match this. Update it the moment shape changes." (DESIGN.md:85–87). It has drifted from the running binary in two material ways: (1) it describes the information architecture as **card-first, boards-as-home, retired routes 302 → /boards**, but the shipped product navigates from a **domain-rail topbar** (`internal/ui/shell/shell.go` Topbar → `/focus/{quests,memory,lifelog,journal,heads,settings}`); (2) it says inference is **"local inference via Ollama, run as a subprocess … reached over the OpenAI-compatible API · OpenAI-compatible remote providers"**, but inference is now **in-process via the embedded Kronk SDK** with a single local path and no Ollama / no remote provider (AGENTS.md and `internal/self/knowledge.md` already say so — DESIGN.md is the outlier). A document titled the honesty ledger that lies about its own architecture is the worst possible stale doc: it makes Balaur misdescribe itself. This plan reconciles DESIGN.md with the running code, AGENTS.md, and `internal/self/knowledge.md` so all three agree, keeping the warm/plain/no-hype/no-emoji voice mandated by DESIGN.md section 2.

## Current state

This is a docs-only reconciliation. The authoritative facts, confirmed by reading the live code at HEAD `12a2ff5`:

**The live top-level nav is a domain rail, not boards.** `internal/ui/shell/shell.go:74–87`:

```go
// Topbar is the wood-chrome header: ... the product's top-level domain nav (the
// active domain rides gold) ... The domain links are the single top-level
// navigation — there is no side rail.
func Topbar(active string) g.Node {
	return h.Header(h.Class("topbar"),
		h.A(h.Class("brand"), h.Href("/"), ...),  // brand links Home (full-screen chat)
		h.Nav(
			navLink("/focus/quests", "Quests", "quests", active),
			navLink("/focus/memory", "Knowledge", "knowledge", active),
			navLink("/focus/lifelog", "Life", "life", active),
			navLink("/focus/journal", "Journal", "journal", active),
			navLink("/focus/heads", "Heads", "heads", active),
			navLink("/focus/settings", "Settings", "settings", active),
		),
		...theme-cycle + theme-toggle buttons...
	)
}
```

So `/` is Home (the full-screen companion chat), the topbar nav is the six domains above, and `/focus/{type}` are real reachable surfaces — NOT 302-redirected stubs. Boards still exist at `/boards` as an owner-composed surface, but they are no longer the home/IA and the domain routes are not retired.

**`internal/self/knowledge.md` already describes this correctly** (do NOT "fix" what is already right) — `internal/self/knowledge.md:113–120`:

```
The domain pages (Quests, Knowledge, Life, Journal, Heads) + Settings are the
top-level nav in the wood topbar (no side rail); navigating to one drops the
"home" class so the dock returns to its right rail ... Those domains are still
served by the legacy /focus/memory, /focus/skills, /focus/quests, /focus/heads,
/focus/journal, /focus/day?date={date}, /focus/lifelog,
/focus/settings?section=profile|models, and owner-composed boards remain at
/boards until those surfaces are migrated to gomponents and retired ...
```

**`internal/self/knowledge.md` already describes Kronk correctly** — `internal/self/knowledge.md:14–18` and `:205–206`:

```
... and local LLM inference run in-process via the embedded Kronk engine
(internal/kronk) — a local GGUF model loaded through yzma/llama.cpp, CGO-free ...
For v1 there is a single LLM path: local; there is no remote provider and no Ollama.
```
```
the model runs in-process via the embedded Kronk engine. There is no remote
provider and no Ollama (both removed in plan 074).
```

So `internal/self/knowledge.md` is in scope ONLY as a cross-check — based on this read it needs NO edits (see Step 3). It is listed in scope solely so the executor can correct it IF a stale architecture line is found; the SPEC's premise that it "may" be stale did not hold on this read.

**AGENTS.md agrees and is the reconciliation target (read-only here).** `AGENTS.md:5–6`, `:74`, `:78`:

```
... and local LLM inference run in-process via the embedded Kronk
engine (llama.cpp through yzma, CGO-free; see `internal/kronk`).
```
```
- Local inference runs in-process via the embedded Kronk SDK (`internal/kronk`): ...
  Ollama (both removed in plan 074).
```

**The stale DESIGN.md text to fix** — `DESIGN.md:89–103` (IA paragraph) and `:105–109` (True today head):

```
89  **Information architecture — card-first, no feature pages.** The UI is three
...
99  ... There are no feature pages: the
100 retired routes (`/tasks`, `/journal`, `/day`,
101 `/memory`, `/skills`, `/life`, `/heads`, `/models`,
102 `/settings`, `/profile`) **302 → `/boards`**, while their write endpoints
103 (`/ui/*`) live on, now driving card focuses.
```
```
105 **True today:** single Go binary embedding PocketBase · Datastar web UI ·
106 PocketBase collections for conversations, messages, memories, skills, heads,
107 audit log · local inference via Ollama, run as a subprocess and reached over
108 the OpenAI-compatible API · OpenAI-compatible
109 remote providers by explicit choice · heads are switchable personas ...
```

**The layout-token claim is NOT yet true (gate it).** The canonical layout tokens that plans 075–082 introduce (`--space-1..7`, `--measure`, `--w-chat-home`, `--w-chat-overlay`, `--pad`, `--z-base`/`--z-overlay`/`--z-drawer`) are **absent from basm.css at this HEAD** — a grep for them returns nothing. Therefore DESIGN.md must NOT claim a layout-token layer exists yet. Confirm before writing:

```
grep -nE "\-\-space-[1-7]|\-\-measure|\-\-w-chat-home|\-\-z-overlay" internal/web/assets/static/basm.css   # expected: NO output at HEAD 12a2ff5
```

**`--shadow-hard` is STILL live (do NOT delete its DESIGN.md mention).** Plan 082 (which would delete it) has not landed. `internal/web/assets/static/basm.css:169` and `DESIGN.md:308` both still define `--shadow-hard: 5px 5px 0;`. The DESIGN.md line is currently ACCURATE. Leave it.

```
grep -n "shadow-hard" internal/web/assets/static/basm.css DESIGN.md
# basm.css:169:  --shadow-hard: 5px 5px 0;
# DESIGN.md:308:--shadow-hard: 5px 5px 0;
```

**`chat.Dock` is NOT yet ported (do NOT claim it as shipped).** `internal/ui/chat/` contains only `message.go` and `toolrow.go` (plus tests) — there is no `Dock`. `internal/self/knowledge.md:111–112` already correctly says the dock chrome "is still the legacy template injected via g.Raw, pending a full chat.Dock port." Any DESIGN.md mention of the dock must describe this CURRENT state.

**The Roadmap fence already lists deferred items** — `DESIGN.md:184–186`:

```
**Roadmap — do not state as shipped:** Johnny Decimal Markdown vault
mirror (one-way export + git) · embedding recall · encrypted export ·
multi-human accounts · channel adapters (Signal/WhatsApp/web).
```

Do not move Kronk or domain-rail facts into Roadmap — they are SHIPPED. Roadmap is for unshipped work only.

**Voice constraint (DESIGN.md section 2, lines 53–83).** Warm, plain, no hype, no emoji. Lead with the companion. The honesty ledger is terse, `·`-separated clauses — match that rhythm; do not editorialize.

**Tours are unaffected.** `tours_test.go` + `.tours/*.tour` do not reference DESIGN.md or knowledge.md (confirmed by grep), so this change cannot break a tour anchor. The suite is run only as a sanity check.

## Commands you will need

| Purpose | Command | Expected |
| --- | --- | --- |
| Confirm stale terms gone | `grep -niE "ollama\|subprocess\|302 .* /boards\|remote provider" DESIGN.md` | no CURRENT-tense matches (history-framed mentions allowed if any) |
| Confirm Kronk present | `grep -niE "kronk\|in-process" DESIGN.md` | at least one match in "True today" |
| Confirm domain rail described | `grep -niE "domain.*rail\|/focus/quests\|topbar.*nav\|six domains" DESIGN.md` | at least one match |
| Confirm shadow-hard untouched | `grep -n "shadow-hard" DESIGN.md internal/web/assets/static/basm.css` | both lines still present, unchanged |
| Confirm layout tokens NOT claimed | `grep -nE "\-\-space-[1-7]\|\-\-w-chat-home\|\-\-measure" DESIGN.md` | no output (not yet shipped) |
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 (docs-only, must stay green) |
| Vet | `go vet ./...` | exit 0 |
| Test | `go test ./...` | all pass |
| Tours still pass | `go test -run Tours ./...` | ok |
| Whitespace | `git diff --check` | no output |
| Only docs changed | `git diff --name-only 12a2ff5` | only DESIGN.md (and knowledge.md if Step 3 edits it) |

## Scope

**In scope** (only files you may modify):
- `DESIGN.md` — the honesty ledger (section 3 IA paragraph + "True today" head). Primary target.
- `internal/self/knowledge.md` — cross-check ONLY; edit only if a concrete architecture line is found stale (Step 3). Based on the planning read it needs no change.

**Out of scope** (do NOT touch, with reason):
- `internal/web/assets/static/basm.css` — code; `--shadow-hard` removal belongs to plan 082, not here.
- `AGENTS.md`, `README.md` — already accurate; touch only if a line is provably FALSE (none found at planning time). If you find one, note it in your report and change only the clearly-false line.
- Any `.go` source, `internal/ui/*`, `internal/feature/*`, route handlers — this plan ships zero behavior change.
- `plans/readme.md` index body beyond the single 085 row update at the end (add the row if it is not present yet, matching the existing column format).
- DESIGN.md Roadmap fence (`:184–186`), Visual system / Datastar sections — do NOT add the layout-token layer there (not shipped); leave them as-is.

## Git workflow

Branch `improve/085-design-honesty-refresh`. Conventional commits, e.g. `docs(design): reconcile honesty ledger with domain-rail + Kronk`. Do NOT push or open a PR unless explicitly told. (Sandbox note: in a TLS-intercepting Hyperagent sandbox, Go commands need the GOPROXY shim — see `docs/hyperagent-sandbox.md`. Docs-only edits here run no Go, but the verify build/test do.)

## Steps

### Step 1: Rewrite the Information-architecture paragraph (DESIGN.md:89–103) to lead with Home + the domain rail

Replace the "card-first, no feature pages … retired routes 302 → /boards" framing with the SHIPPED IA, sourced from `shell.go` Topbar. Keep boards as a real-but-secondary surface (they still live at `/boards`), and keep cards as the composition unit — but stop calling boards the home and stop calling the domain routes retired.

Target shape (match the ledger's terse, plain voice; adjust wording to read naturally but keep every fact accurate to `shell.go:74–87` and `knowledge.md:113–120`):

> **Information architecture — Home is the companion; a domain rail is the nav.** `/` is Home: the full-screen companion chat, the conversation Balaur is built around. The wood topbar carries the single top-level navigation — six domains (Quests, Knowledge, Life, Journal, Heads, Settings), the active one riding gold; there is no side rail. Each domain links its `/focus/{type}` surface (`/focus/quests`, `/focus/memory`, `/focus/lifelog`, `/focus/journal`, `/focus/heads`, `/focus/settings`); leaving Home drops the `home` class so the persistent dock chat returns to its right rail with the domain content in `#main`. A *card* is a typed, parameterized, server-rendered resource (`/ui/cards/{type}`) that renders as a tile or a full-canvas focus; *boards* (`/boards`) are owner-composed dashboards of card tiles, and chat can compose cards/boards (`card_show` / `board_compose` / `board_add_card`). A **head switcher** in the dock changes the active persona without leaving or forking the conversation.

Constraints:
- Do NOT assert "302 → /boards" as current behavior. If you want to preserve the history, frame it explicitly as past ("boards were briefly the home" — only if you can state it truthfully); otherwise simply omit it. Prefer omission over a guessed history claim.
- Do NOT claim the `/focus/*` pages are gomponents-ported (they are still legacy per `knowledge.md:116–120`); say only that they are the domain surfaces.
- Keep the head-switcher sentence (it is still true).

**Verify**:
- `grep -niE "302 .* /boards|no feature pages.*home|boards.*home" DESIGN.md` → no current-tense match
- `grep -niE "/focus/quests|domain|topbar" DESIGN.md` → matches the new paragraph
- `git diff --check` → no output

### Step 2: Fix the "True today" inference clause (DESIGN.md:105–109) — Kronk, not Ollama

In the `· `-separated "True today" run, replace:

> `local inference via Ollama, run as a subprocess and reached over the OpenAI-compatible API · OpenAI-compatible remote providers by explicit choice`

with a Kronk-accurate clause that matches `AGENTS.md:5–6,74,78` and `knowledge.md:14–18,205–206`:

> `local inference run in-process via the embedded Kronk engine (internal/kronk) — a GGUF model loaded through yzma/llama.cpp, CGO-free, the native library dlopen'd at runtime; a single local provider path (no remote provider, no Ollama — both removed in plan 074)`

Constraints:
- Keep it a single `·`-delimited clause in the existing run; do not restructure the surrounding clauses.
- Owner-supplies-the-model detail is optional and already in knowledge.md; keep DESIGN.md's clause terse — "in-process via the embedded Kronk engine, single local path, no Ollama" is the load-bearing correction.
- Do NOT touch the heads/OS-access/memory/recap/tasks clauses that follow — they are accurate and out of scope.

**Verify**:
- `grep -niE "ollama|subprocess|openai-compatible|remote provider" DESIGN.md` → no current-tense match (the "removed in plan 074" parenthetical may name Ollama as history — that is allowed and accurate)
- `grep -niE "kronk|in-process" DESIGN.md` → match in "True today"
- `git diff --check` → no output

### Step 3: Cross-check internal/self/knowledge.md — edit ONLY if a concrete line is stale

Read `internal/self/knowledge.md` lines 14–18, 65–67, 103–128, 199–206. At planning time these already correctly describe Kronk (no Ollama, plan 074) and the domain-rail topbar, so the EXPECTED outcome is **no edit**.

Make an edit here ONLY if you find a specific line that states an architecture fact contradicted by the running code (e.g. a leftover "Ollama" as current, or "boards are the home"). If you do edit, keep it minimal and match the surrounding voice. If you edit nothing, that is correct — do not invent a change to justify the file being in scope.

**STOP** if editing knowledge.md would require describing behavior that is not shipped (e.g. claiming chat.Dock is ported, or the layout tokens exist). In that case, leave the line as-is and report.

**Verify**:
- `git diff --stat 12a2ff5 -- internal/self/knowledge.md` → either empty (no change, expected) or a small, justified diff
- `grep -niE "ollama" internal/self/knowledge.md` → only the "removed in plan 074" history mentions (lines ~16, ~206), never current-tense

### Step 4: Final reconciliation pass + green-build gate

Re-read the full DESIGN.md section 3 (lines 85–186) once more and confirm three documents agree: DESIGN.md "True today" / IA ↔ `AGENTS.md` Product shape + Known limitations ↔ `internal/self/knowledge.md` Overview + Architecture. Confirm you did NOT: add a layout-token claim, delete the `--shadow-hard` mention, or claim chat.Dock is ported.

Run the green-build gate (docs-only, but must not have introduced a stray code edit):

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → all pass
- `go test -run Tours ./...` → ok
- `git diff --name-only 12a2ff5` → only `DESIGN.md` (plus `internal/self/knowledge.md` iff Step 3 edited it)
- `git diff --check` → no output

## Test plan

No code tests (docs-only). Verification is by grep + human review of the diff:
- The grep gates in each Verify block prove the stale terms are gone/reframed and the accurate terms are present.
- `go test ./...` and `go test -run Tours ./...` are run only to prove no source/tour was accidentally touched (tours do not reference these docs, so they must stay green unchanged).
- A human reviewer reads the DESIGN.md diff for VOICE (warm, plain, no hype, no emoji — DESIGN.md section 2) and ACCURACY (every clause traces to `shell.go`, `internal/kronk`, AGENTS.md, or knowledge.md).
- No storybook story changes (no component touched).

## Done criteria

- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go vet ./...` exits 0.
- [ ] `go test ./...` all pass; `go test -run Tours ./...` ok.
- [ ] `grep -niE "ollama|subprocess|openai-compatible|remote provider" DESIGN.md` returns no CURRENT-tense match (only the "removed in plan 074" history parenthetical, if kept).
- [ ] `grep -niE "kronk|in-process" DESIGN.md` matches in "True today".
- [ ] `grep -niE "302 .* /boards" DESIGN.md` returns no current-tense IA claim.
- [ ] `grep -niE "/focus/quests|domain|topbar" DESIGN.md` matches the rewritten IA paragraph.
- [ ] `grep -n "shadow-hard" DESIGN.md internal/web/assets/static/basm.css` shows BOTH lines still present and unchanged.
- [ ] `grep -nE "\-\-space-[1-7]|\-\-w-chat-home|\-\-measure" DESIGN.md` returns no output (no premature layout-token claim).
- [ ] `git diff --name-only 12a2ff5` lists only `DESIGN.md` (and `internal/self/knowledge.md` only if Step 3 found a real stale line).
- [ ] `git diff --check` returns no output.
- [ ] The 085 row in `plans/readme.md` is updated to done (add the row if it is not present yet, matching the existing column format).
- [ ] VISUAL check: N/A — no CSS or markup changed (docs-only). State this explicitly in the report.

## STOP conditions

- **Drift**: the Step-0 drift check shows DESIGN.md or knowledge.md changed since `12a2ff5` and a "Current state" excerpt no longer matches — STOP and report which excerpt drifted.
- **shadow-hard already removed**: if `grep -n "shadow-hard" internal/web/assets/static/basm.css` returns nothing (plan 082 landed first), then DESIGN.md:308's mention IS now stale — STOP and report so the orchestrator can fold the deletion in; do not delete it on a guess.
- **Layout tokens already shipped**: if `grep -nE "\-\-space-[1-7]" internal/web/assets/static/basm.css` returns matches (plans 075–082 landed first), then DESIGN.md MAY accurately gain a layout-token note — STOP and report; do not add the note speculatively if they are absent.
- **chat.Dock shipped**: if `grep -rn "func Dock" internal/ui/chat/` matches (plan 084 landed), the dock-ported claim becomes truthful — STOP and report rather than writing the target state on a guess.
- **Genuine code/doc disagreement you cannot resolve**: if a DESIGN.md claim and the running code disagree in a way the cited sources (shell.go, internal/kronk, AGENTS.md, knowledge.md) do not settle, note the contradiction and ASK — do not guess.
- **Any Verify fails twice**: stop and report the command + output.
- **Need to edit an out-of-scope file** (basm.css, a .go file, AGENTS.md beyond a provably-false line): stop and report — the change has outgrown this docs-only plan.

## Maintenance notes

- DESIGN.md section 3 is the honesty ledger by contract ("Update it the moment shape changes" — DESIGN.md:87). Future plans that change the IA, the inference engine, or the nav MUST update this section in the same commit; this plan only catches up the accumulated drift.
- Three documents must stay in agreement: DESIGN.md "True today"/IA, `AGENTS.md` Product shape + Known limitations, and `internal/self/knowledge.md` Overview + Architecture. A reviewer should diff a claim across all three when any one changes.
- Deferred, intentionally NOT done here (gated on their owning plans): removing the `--shadow-hard` DESIGN.md mention (plan 082), adding the layout-token-layer note to DESIGN.md's Layout/Datastar sections (plans 075–082), and flipping the dock-port wording from "pending" to "ported" (plan 084). When each lands, fold the matching DESIGN.md/knowledge.md sentence in that plan's own commit.
- Watch the Roadmap fence (DESIGN.md:184–186): it is for UNSHIPPED work only. Never let a shipped fact (Kronk, the domain rail) slide into it, and never leave a genuinely-deferred item (vault mirror, embedding recall, multi-human, channel adapters) out of it.
