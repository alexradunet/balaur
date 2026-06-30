# Plan 214: Docs truth-sync — fix DESIGN.md + README claims that contradict shipped code

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. Touch
> ONLY `DESIGN.md` and `README.md`. If a STOP condition occurs, stop and report.
> When done, update this plan's row in `plans/README.md` — unless a reviewer
> dispatched you and maintains the index.
>
> **Drift check (run first)**: `git diff --stat ef9f2df..HEAD -- DESIGN.md README.md internal/llm/openai.go internal/export internal/cli/export.go internal/cards/cards.go`
> If `DESIGN.md`/`README.md` changed since this plan was written, compare the
> "Current state" excerpts against the live files before editing; on a mismatch,
> treat it as a STOP condition for that specific edit (the doc may already be
> fixed — see STOP conditions).

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: docs
- **Planned at**: commit `ef9f2df`, 2026-06-30

## Why this matters

`DESIGN.md` is the project's copy-contract — it literally says "All copy must
match this. Update it the moment shape changes." It currently **denies a shipped
product pillar** (it claims there is "no remote provider", but the opt-in
consent-gated cloud model shipped in plan 123) and describes **two retired data
shapes** (the `tasks` collection, and memories/skills as collections — all folded
into the `nodes` spine). `README.md` — the first audience's entry point — sends a
new self-hoster to a model-install form that does not exist (an "absolute `.gguf`
path"), describes the same retired `tasks` collection, mislabels the **shipped**
encrypted export as roadmap, and omits the `balaur export` command and the whole
export/vault-mirror feature.

`internal/self/knowledge.md` (the running binary's self-description) is **current
and correct** — it is the truth source for every fix below. This plan only brings
the two human-facing docs back in line with it and the code. Pure prose edits; no
behavior changes.

## Current state (each wrong claim + the code that proves it wrong)

### DESIGN.md — the "Honesty ledger" (`## 3. Honesty ledger`)

1. **Collections list names memories/skills as collections** — `DESIGN.md:116`:
   > "PocketBase collections for conversations, messages, **memories, skills**, heads, audit log"
   Memory and skill are `type=memory`/`type=skill` rows in the unified `nodes`
   collection (plans 164–168). Proof: `internal/self/knowledge.md:85` ("…the old
   `tasks` collection is retired…" and the nodes-spine description).

2. **"No remote provider" — FALSE, the remote path shipped** — `DESIGN.md:119-120`:
   > "single local provider path (**no remote provider**, no Ollama — both removed in plan 074)"
   Proof it's wrong: `internal/llm/openai.go:33` `type OpenAIClient struct` (the
   consent-gated OpenAI-compatible remote client, plan 123). README itself already
   documents the opt-in cloud model (`README.md:34-36`). The "no Ollama / plan 074"
   part is still true — keep that; only "no remote provider" is wrong.

3. **`tasks` collection capture description — collection retired** — `DESIGN.md:131-133`:
   > "task capture from chat (`task_add` / …) into the **`tasks` collection** — tiny recurrence DSL, completions logged to the `entries` life log…"
   Proof: `migrations/1750000020_tasks_to_nodes.go:199` `app.Delete(tasksCol)` (Step 4
   "Drop the tasks collection"); `internal/self/knowledge.md:107` ("`type=task` nodes
   in the `nodes` collection"). Tasks are now `type=task` nodes. (Completion
   *entries* in the `entries` collection are still real — keep that detail; only the
   "`tasks` collection" claim is wrong.)

4. **Card registry list is stale (count + members)** — `DESIGN.md:176-179`:
   > "typed card registry — **13** parameterized … card resources under `/ui/cards/{type}`: today, quests, calendar, timeline, day, measure, lines, memory, skills, heads, habits, lifelog, settings"
   The actual registry (`internal/cards/cards.go`, the `Type:` values) currently has
   **21**: `today, quests, calendar, timeline, day, period, measure, lines, note,
   memory, skills, related, graph, network, heads, habits, lifelog, tasks, settings,
   review, chronicle`. Missing from DESIGN: period, note, related, graph, network,
   tasks, review, chronicle. (`internal/self/knowledge.md:347` references `period`.)

5. **Roadmap lists two SHIPPED features** — `DESIGN.md:185-187`:
   > "**Roadmap — do not state as shipped:** Johnny Decimal Markdown vault mirror (one-way export + git) · embedding recall · **encrypted export** · multi-human accounts · channel adapters"
   Both shipped: `internal/export/export.go:109` `func ExportMirror` (the vault
   mirror, plan 194) and `internal/export/encrypt.go:57` `func EncryptDir` (encrypted
   export, plan 195). `embedding recall`, `multi-human accounts`, and
   `channel adapters` are genuinely unshipped — KEEP those.

### README.md

6. **Non-existent ".gguf path" install form** — `README.md:31-33`:
   > "**Models:** Balaur runs local GGUF models in-process. Install one from the settings models section (**an absolute `.gguf` path**); it runs on CPU by default…"
   No such form exists. The web model paths are `POST /ui/model/download`
   (`downloadOfficialModel`, curated catalog) and `POST /ui/runtime/install`
   (`internal/web/web.go:191,197`; `internal/web/models_install.go`). Proof:
   `internal/self/knowledge.md:392` "There is no manual GGUF-path entry." The
   cloud-model sentence right after (`README.md:34-36`) is already correct — keep it.

7. **`tasks` collection** — `README.md:62`:
   > "Tasks live in the **`tasks` collection**; completions land in `entries`, the life-log substrate."
   Same retirement (proof as #3). Tasks are `type=task` nodes. (The "completions land
   in `entries`" half is still accurate — keep it; only "`tasks` collection" is wrong.)

8. **Roadmap lists shipped "Encrypted export"** — `README.md:426`:
   > "## Roadmap (not shipped — honesty ledger)\n- Embedding recall …\n- **Encrypted export**\n- Multi-human accounts …"
   Encrypted export shipped (proof as #5). Remove that bullet. KEEP "Embedding
   recall" and "Multi-human accounts" — genuinely unshipped. (README's roadmap does
   NOT list the vault mirror, so nothing to remove there.)

9. **CLI table omits `export` and `seed`** — `README.md` CLI table (~lines 296-313,
   rows from `balaur doctor` through `balaur ext`): there is no `export` row and no
   `seed` row. Proof both exist: `internal/cli/export.go:29` `Use: "export"` (flags
   `--encrypt`, `--out`, `--archive`), `internal/cli/seed.go:13` `Use: "seed"`.
   `internal/self/knowledge.md:321` lists `balaur export`.

10. **"Current shape" omits the export/vault-mirror feature** — the sovereignty
    export (plan 194) is absent from README's feature description (it's described at
    length in `internal/self/knowledge.md:321`). Add a short bullet so a reader can
    discover `balaur export` and the one-way Markdown mirror.

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Drift check | `git diff --stat ef9f2df..HEAD -- DESIGN.md README.md` | (compare excerpts if non-empty) |
| Build (unaffected) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Tours unaffected | `go test . -run TestTours -count=1` | PASS (set `TMPDIR=/home/alex/.cache/go-tmp` if `/tmp` OOMs) |
| Verify a claim | `grep -n "<phrase>" DESIGN.md README.md` | as specified per step |

## Scope

**In scope** (modify ONLY these):
- `DESIGN.md`
- `README.md`

**Out of scope** (do NOT touch):
- `internal/self/knowledge.md` — it is the CORRECT truth source; do not edit it.
- Any source code, tests, migrations, or `.tours/*` — this is a docs-only change.
- Roadmap items that are genuinely unshipped (embedding recall, multi-human,
  channel adapters) — leave them in the roadmaps.

## Git workflow

- Branch: `advisor/214-docs-truth-sync`
- Conventional-commit subject, e.g. `docs: truth-sync DESIGN.md + README to shipped reality`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

> Each edit is a targeted prose replacement. Before each, re-read the live line to
> confirm it still matches the "Current state" excerpt. After each file, run its
> verify grep.

### Step 1: DESIGN.md — fix the four ledger claims + roadmap

- **(1) collections list** (`DESIGN.md:116`): change
  `memories, skills` in the collection list so it reflects the nodes spine, e.g.
  `…conversations, messages, the unified \`nodes\` spine (memories, skills, tasks, notes, measures, day pages), heads, audit log…`. (Keep it concise; the point is memories/skills are NOT their own collections.)
- **(2) remote provider** (`DESIGN.md:119-120`): replace
  `single local provider path (no remote provider, no Ollama — both removed in plan 074)`
  with something like
  `local inference is the default path (no Ollama — removed in plan 074), with an opt-in, consent-gated EU/GDPR cloud provider available from the Models page (plan 123) — never the default, never auto-selected`.
- **(3) tasks collection** (`DESIGN.md:131-133`): replace `into the \`tasks\` collection` with `as \`type=task\` nodes in the \`nodes\` spine`. Keep "completions logged to the `entries` life log" — still true.
- **(4) card registry** (`DESIGN.md:176-179`): update the count `13` to match the
  actual registry and replace the enumerated list with the current registered types
  from `internal/cards/cards.go`. As of this writing that is **21** types:
  `today, quests, calendar, timeline, day, period, measure, lines, note, memory, skills, related, graph, network, heads, habits, lifelog, tasks, settings, review, chronicle`.
  **Re-derive the live list first** to avoid restating a stale set:
  `grep -oE 'Type:  "[a-z]+"' internal/cards/cards.go | sed 's/.*"\(.*\)"/\1/' | paste -sd, -`
  Use that command's output (and its count) verbatim.
- **(5) roadmap** (`DESIGN.md:185-187`): remove `Johnny Decimal Markdown vault mirror (one-way export + git)` and `encrypted export` from the "Roadmap — do not state as shipped" list (they shipped). KEEP `embedding recall`, `multi-human accounts`, `channel adapters`. Optionally move the two removed items into the "True today" ledger above as shipped features (one short clause each), matching how `internal/self/knowledge.md:321` describes them.

**Verify**:
- `grep -c "no remote provider" DESIGN.md` → `0`
- `grep -c "the \`tasks\` collection\|into the \`tasks\` collection" DESIGN.md` → `0`
- `grep -c "encrypted export" DESIGN.md` → `0` (the roadmap line is gone; if you moved it to "True today" rephrase so this exact lowercase roadmap phrase is gone — or adjust the grep to target only the roadmap line)
- `grep -n "period" DESIGN.md` → the card list now includes `period`

### Step 2: README.md — model form, tasks collection, roadmap, CLI table, current shape

- **(6) model install** (`README.md:32`): replace `(an absolute \`.gguf\` path)` with
  the curated-catalog reality, e.g. `(one click from the curated model catalog; the engine downloads and verifies it)`. Leave the surrounding CPU/Vulkan + cloud sentences intact.
- **(7) tasks collection** (`README.md:62`): replace `Tasks live in the \`tasks\` collection` with `Tasks are \`type=task\` nodes in the unified \`nodes\` collection`. Keep `completions land in \`entries\`, the life-log substrate.`
- **(8) roadmap** (`README.md:~426`): delete the `- Encrypted export` bullet under
  `## Roadmap (not shipped — honesty ledger)`. Keep the embedding-recall and
  multi-human bullets.
- **(9) CLI table** (`README.md` ~296-313): add two rows (match the existing table's
  3-column `| Command | What it does | Model? |` format), placed sensibly (e.g.
  `export` near `history`/`audit`, `seed` near the end):
  - `| \`balaur export [--encrypt --archive <path>]\` | Read-only Markdown mirror of active nodes (sovereign vault); \`--encrypt\` wraps it as a passphrase-protected archive. | no |`
  - `| \`balaur seed\` | Populate the dev/demo data set. | no |`
  Confirm the exact flags from `internal/cli/export.go:35-37` before writing the row.
- **(10) current shape** (README feature section, e.g. near the sovereignty/SQLite
  bullets): add one bullet so the export feature is discoverable, e.g.
  `**Your record, portable:** \`balaur export\` writes a one-way Johnny Decimal Markdown mirror of your active nodes (plan 194); \`--encrypt\` produces a passphrase-protected archive (plan 195).`

**Verify**:
- `grep -c "absolute \`.gguf\` path" README.md` → `0`
- `grep -c "Tasks live in the \`tasks\` collection" README.md` → `0`
- `grep -c "balaur export" README.md` → `≥ 1` (CLI table + current-shape bullet)
- In the `## Roadmap` section, `grep -A6 "## Roadmap" README.md | grep -c "Encrypted export"` → `0`

### Step 3: Sanity — docs only, nothing else moved

**Verify**:
- `git status --short` → only `DESIGN.md` and `README.md` modified
- `CGO_ENABLED=0 go build ./...` → exit 0 (unaffected; proves no stray code edit)
- `go test . -run TestTours -count=1` → PASS (no tour anchors these docs; this just confirms you didn't break anything)

## Test plan

- Docs-only change — no Go tests to add. The gates are the verify greps above plus
  a manual re-read of the edited ledger/README sections for tone (match the
  surrounding voice; DESIGN.md is terse and `·`-separated, README is prose).
- No tour or test references these specific doc lines (confirmed: tours anchor
  `.go` files, not `DESIGN.md`/`README.md`). If `go test . -run TestTours` somehow
  fails, STOP — something unexpected changed.

## Done criteria — ALL must hold

- [ ] `grep -c "no remote provider" DESIGN.md` → 0
- [ ] `grep -c "into the \`tasks\` collection" DESIGN.md README.md` → 0
- [ ] `grep -c "absolute \`.gguf\` path" README.md` → 0
- [ ] DESIGN.md card-registry count + list match `internal/cards/cards.go` (includes `period`, `note`, `tasks`, `chronicle`, …)
- [ ] DESIGN.md roadmap no longer lists the vault mirror or encrypted export; README roadmap no longer lists "Encrypted export"
- [ ] `grep -c "balaur export" README.md` ≥ 1 (CLI table row + current-shape bullet); `balaur seed` row present
- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go test . -run TestTours -count=1` passes
- [ ] `git status --short` shows ONLY `DESIGN.md` and `README.md`
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report (do not improvise) if:

- A "Current state" excerpt does NOT match the live doc line (the doc may already
  have been fixed since `ef9f2df`). Skip that specific edit, note it as
  "already correct", and continue with the rest.
- The proof code has changed — e.g. `internal/llm/openai.go` no longer has
  `OpenAIClient`, or `migrations/1750000020_tasks_to_nodes.go` no longer deletes the
  collection, or `internal/export/encrypt.go` no longer has `EncryptDir`. That means
  the feature's status flipped; report rather than write a now-wrong claim.
- The live card registry (`internal/cards/cards.go`) has a different set/count than
  the 21 listed here — use the LIVE list (re-derived via the grep in Step 1), and
  note the difference in your report.
- Editing would require touching any file other than `DESIGN.md`/`README.md`.

## Maintenance notes

- `internal/self/knowledge.md` is the truth source and is kept current in the same
  commit as architecture changes (AGENTS.md rule). When DESIGN.md/README drift
  again, diff them against `knowledge.md` first.
- Root cause of this drift: features shipped (remote model plan 123, tasks→nodes
  plan 167, export plans 194/195) updated `knowledge.md` but not the two
  human-facing docs. A reviewer merging a feature should check whether DESIGN.md's
  ledger or README's "Current shape"/roadmap/CLI table need the same edit.
- Card registry: DESIGN.md enumerates the card types by hand — it will re-drift when
  cards are added/removed. Consider (future) a note pointing readers at
  `internal/cards/cards.go` as the authoritative list instead of re-enumerating.
