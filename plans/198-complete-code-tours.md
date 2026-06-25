# Plan 198: A complete code tour of the whole Balaur codebase

> **Executor instructions**: Follow this plan step by step. This is a
> **WRITING-heavy, code-read-only** plan: you create and edit VS Code CodeTour
> files under `.tours/` ONLY. You must NEVER modify product code (anything under
> `internal/`, `main.go`, `migrations/`). For every tour step you write, **open
> the file you are anchoring to and confirm the symbol is really there** — never
> write a step from memory of what "should" be in a file. Run every verification
> command and confirm the expected result before moving on. If a "STOP
> condition" occurs, stop and report — do not improvise. When done, update this
> plan's status row in `plans/README.md`.
>
> **Line numbers in this plan are APPROXIMATE LEADS, not gospel.** They were
> accurate at the planning commit but code drifts. The binding contract for
> every step is: anchor to the **named symbol/declaration** (find it with grep),
> put the step's `line` on that declaration, then let `go test . -run TestTours`
> prove the anchor is valid. A wrong line number is not a STOP condition; a
> missing symbol is (see STOP conditions).
>
> **Drift check (run first)**:
> `git diff --stat 7642f92..HEAD -- .tours internal main.go migrations`
> If `.tours/` changed since `7642f92`, re-read the existing tours before editing.
> If a package you are about to tour was heavily refactored, re-confirm its
> symbols with grep before writing its steps.

## Status

- **Priority**: P3 · **Effort**: L (almost entirely prose; ~10 new tours + a
  refresh pass over 10 existing tours)
- **Risk**: LOW (docs only — `.tours/*.tour` are JSON data files; the only Go
  file in scope is the optional cosmetic message in `tours_test.go`)
- **Depends on**: none
- **Category**: docs
- **Planned at**: commit `7642f92`, 2026-06-25

## Why this matters

`.tours/` holds VS Code CodeTour files — guided, anchored walkthroughs of the
codebase that double as onboarding and as a living architecture map. Today there
are **10 tours covering ~12 of ~40 packages**. They teach the agent loop, the
web gateway, cards, recall, and the CLI — but say nothing about the **shared
turn pipeline** (`internal/turn`, the spine every gateway calls), the **embedded
Kronk inference engine**, the **unified `nodes` object spine**, the
**tasks/life/recap companion domain**, the **sovereign export + encryption**, the
**goja extension sandbox**, the **gomponents component system + storybook**, or
**how the binary boots**. A newcomer (human or agent) cannot learn the whole
system from the tours.

This plan makes the tour set **complete**: every internal package is taught by at
least one tour. It does two things — (A) refreshes the 10 existing tours to
shipped reality (they have prose drift the lint test cannot catch — e.g.
orientation claims "29 migration files" when there are 10), and (B) adds 10 new
tours so the whole codebase is covered. The `tours_test.go` lint gate keeps every
anchor honest forever after.

## Current state

### The tour files (`.tours/*.tour`)

10 files exist, all currently passing `tours_test.go`:

```
00-orientation              01-packages-structs-interfaces   02-goroutines-channels
03-agent-loop               04-testing-fakes-closures        06-memory-and-self-evolution
07-the-web-gateway          08-hateoas-cards-and-boards      09-recall-and-search
10-the-cli-api
```

Number `05` is **free** (the old `05-the-security-boundary.tour` was deleted when
heads became personas, not a security boundary). This plan reuses `05` for the
turn pipeline.

### The tour file format (study an existing tour before writing)

Each tour is a JSON file. Open `.tours/00-orientation.tour` and read it fully —
it is your style and structure exemplar. Shape:

```json
{
  "$schema": "https://aka.ms/codetour-schema",
  "title": "05 — The Turn Pipeline: One Spine for Every Gateway",
  "description": "One-paragraph summary of what this tour teaches.",
  "steps": [
    {
      "file": "internal/turn/turn.go",
      "line": 69,
      "title": "5.1 — Where a turn begins",
      "description": "Markdown. Explain INTENT, not just what the code says. Use **bold**, bullet lists, and short ```go fenced blocks```. Cross-reference other tours: 'Tour 03 covers the loop this drives.' End load-bearing steps with the AGENTS.md rule they embody."
    }
  ]
}
```

Rules the lint test (`tours_test.go`) enforces — **all must hold or the suite
fails**:
- Valid JSON.
- `title` non-empty.
- Every step with a `file` → file exists at the repo-root-relative path, and
  `line` (when present) is `>= 1` and `<= the file's line count`.
- Every step with a `directory` → directory exists.

The lint **cannot** catch prose that is wrong-but-anchored-to-a-real-line. That
is why the contract is "open the file, confirm the symbol, then write the prose."

### Voice (match the existing tours exactly)

- Teaching, plain, calm. No marketing, no hype.
- Step titles use the format `"N.M — Short title"` (tour number . step number).
- Each step teaches **intent and trade-offs**, names the **AGENTS.md invariant**
  it embodies where one applies, and cross-references sibling tours.
- 6–11 steps per tour. Anchor to **declarations** (func/type/const) rather than
  mid-body lines, so anchors survive edits.

### The verified package → tour coverage matrix (target end state)

| Package | Covered by tour(s) |
|---|---|
| `main.go`, `migrations/`, `launch`, `seed` | 00, **19** |
| `internal/agent` | 03, 04, **05** |
| `internal/turn`, `internal/verify`, `internal/heads` | **05** |
| `internal/llm`, `internal/kronk`, `internal/kronk/modelget` | 00, 01, 02, **11** |
| `internal/conversation`, `internal/store` | **12** |
| `internal/tasks`, `internal/life`, `internal/recap` | **13** |
| `internal/nodes` | **14** |
| `internal/knowledge`, `internal/search` | 06, 09 (refresh) |
| `internal/export` | **15** |
| `internal/ext` | **16** |
| `internal/tools` | 06, 08, **17** |
| `internal/ui`, `internal/ui/chat`, `internal/ui/shell`, `internal/feature/*cards`, `internal/feature/storybook` | **18** |
| `internal/web` | 00, 07, 08 (refresh) |
| `internal/cli` | 00, 10 |

Bold = created by this plan. End state: **20 tours, every package covered.**

### Architecture facts you must teach correctly (verified at `7642f92`)

These are the truths the tours must reflect. Several contradict what the *old*
tours imply — fix them in the refresh pass:

- **One binary.** PocketBase embedded as a library; gomponents+Datastar UI from
  `embed.FS`; local inference in-process via the embedded **Kronk** SDK
  (llama.cpp through yzma, CGO-free, native lib `dlopen`'d at runtime).
- **Gateways adapt, never re-implement.** `internal/web` and `internal/cli` both
  call `turn.Run` in `internal/turn`; behavior lives below the gateway line.
- **No `html/template`.** UI is typed gomponents over Datastar (SSE + signals);
  no Node build. The storybook (`/storybook`) is the source of truth.
- **The `nodes` object spine.** `tasks`, `measures`, `journal`, and `day` were
  folded into the unified `nodes` collection (migrations `1750000020`–`50`).
  Node `status` (proposed/active/archived/rejected) is the consent boundary;
  graph traversal filters `status=active`.
- **10 migration `.go` files** (`migrations/*.go`, excluding 3 `_test.go`), not
  29. The init baseline creates ~14 collections; later files fold the domain
  into `nodes`.
- **app.Save bypasses API rules by design** (trusted owner code); **audit
  strictly after a successful write**; **owner-timezone cron math** resolved
  per-call (no globals).

## Commands you will need

| Purpose | Command | Expected on success |
|---|---|---|
| Tour lint only | `go test . -run TestTours -v` | `PASS` |
| Full suite (pre-push gate) | `go test ./...` | `ok` (all packages) |
| Confirm a symbol's real line | `grep -nE 'func Foo\(' path/to/file.go` | the line to anchor |
| Count a file's lines (anchor ≤ this) | `wc -l path/to/file.go` | N |
| JSON validity of one tour | `python3 -m json.tool .tours/NN-x.tour >/dev/null` | exit 0 |
| No stale engine terms | `grep -rli 'llamafile\|ollama\|html/template' .tours/` | no matches |

> `TMPDIR` note: if `go test ./...` fails to link with "No space left on
> device", set `TMPDIR=/home/alex/.cache/go-tmp` (the repo's `/tmp` is a small
> tmpfs). The `-run TestTours` lint alone does not need this.

## Scope

**In scope** (the only files you may create or modify):
- `.tours/05-the-turn-pipeline.tour` (create)
- `.tours/11-local-inference-kronk.tour` (create)
- `.tours/12-persistence-and-store.tour` (create)
- `.tours/13-companion-domain.tour` (create)
- `.tours/14-nodes-and-graph.tour` (create)
- `.tours/15-sovereign-export.tour` (create)
- `.tours/16-extension-sandbox.tour` (create)
- `.tours/17-the-tool-surface.tour` (create)
- `.tours/18-component-system-and-storybook.tour` (create)
- `.tours/19-bootstrapping.tour` (create)
- `.tours/00-orientation.tour` … `.tours/10-the-cli-api.tour` (refresh — Stage A)
- `tours_test.go` (OPTIONAL: only the cosmetic `"expected at least 11"` message
  string may be updated to `20`. Do NOT change the test logic.)

**Out of scope** (do NOT touch, even though tours reference them):
- ALL product code: everything under `internal/`, `main.go`, `migrations/`,
  `internal/web/assets/`. You read these to anchor steps; you never edit them.
- `AGENTS.md` — it already carries the `.tours/` maintenance contract (the line
  beginning "`.tours/` is a maintained artifact …"). Confirm it exists; do not
  rewrite it.
- `DESIGN.md`, `README.md`, `internal/self/knowledge.md` — no capability
  changes here.
- `plans/` — except this file's status row in `plans/README.md`.

## Git workflow

- Work on the branch you were dispatched on. Commit message:
  `docs(tours): complete code-tour coverage — refresh 10 + add 05,11–19`.
- Commit in logical units is fine (e.g. one commit for Stage A, one per new
  tour, or a few grouped) — but do **not** push or merge; the reviewer does
  that after re-running the gates.

---

## Steps

> Recommended order: **Stage A** (refresh, smallest risk, re-grounds you in the
> voice) → **Stage B** new tours in listed order → **Stage C** (wire-up). After
> **each** tour file you create or edit, run `go test . -run TestTours -v` and
> confirm `PASS` before starting the next. That keeps every anchor honest
> incrementally instead of debugging 20 tours at once.

### Stage A — Refresh the 10 existing tours

For **each** existing tour: open every step's anchored file, confirm the `line`
still lands on the intended symbol (fix it if it drifted), and **rewrite any
prose that no longer matches shipped reality**. Add a closing cross-reference to
the relevant new tour where natural (e.g. orientation should point at all the new
tours). Known, specific drift to fix:

| Tour | Known fix (verify, then correct) |
|---|---|
| `00-orientation` | "**29 migration files**" → there are **10** `migrations/*.go` (+3 tests). The `drop_boards` narrative is stale (boards are gone). The package map must add `internal/turn`, `internal/nodes`, `internal/export`, `internal/ext`, `internal/ui` + `feature/*`, `internal/launch`, and point each to its new tour (05, 11–19). Confirm `main.go` line anchors (the no-args launcher block exists now — see Tour 19). |
| `01-packages-structs-interfaces` | Confirm the `internal/llm` file list (`llm.go`, `env.go`, `openai.go`) and the `Client` interface incl. `Embed`. Confirm kronk references resolve. |
| `02-goroutines-channels` | Confirm concurrency anchors are live (Kronk engine mutex/streaming, the chat streaming writer). No `llamafile` references may remain. |
| `03-agent-loop` | Largely stable. Confirm `Loop.Run`/`executeCall` anchors and the event-kind list. Add a cross-ref to Tour 05 (the pipeline that drives this loop). |
| `04-testing-fakes-closures` | Confirm the fake `llm.Client` test pattern anchors. |
| `06-memory-and-self-evolution` | Confirm it reflects the `nodes`-based knowledge spine and FTS5 recall; cross-ref Tour 14 (nodes) and Tour 17 (knowledge tools). |
| `07-the-web-gateway` | Confirm it is gomponents+Datastar (no `html/template`, no HTMX, no `web/templates/`). Cross-ref Tour 18 (components) and Tour 05 (it adapts `turn.Run`). |
| `08-hateoas-cards-and-boards` | Boards were removed; confirm the title/prose do not promise a live board system that no longer exists. Card stack via `ui.RegisterCard` + `/ui/cards/{type}`. Cross-ref Tour 18. |
| `09-recall-and-search` | Confirm FTS5 sidecar anchors (`internal/search`, boot rebuild in `main.go`). |
| `10-the-cli-api` | Confirm the v1 envelope + command list against `internal/cli`. |

**Verify (Stage A)**:
- `go test . -run TestTours -v` → `PASS`
- `grep -rli 'llamafile\|ollama\|html/template\|web/templates' .tours/` → no matches
- `grep -c '29 migration' .tours/00-orientation.tour` → `0`

### Stage B — Write the 10 new tours

Each sub-section gives: the tour's purpose, the verified anchor files, and a step
outline (`title — file → symbol (~approx line) — what to teach`). **Confirm each
symbol with grep before writing its step**, set `line` to the symbol's real
declaration, and write 1–3 short paragraphs of teaching prose per step in the
established voice. Aim for the step counts shown (±2 is fine).

#### Tour 05 — `05-the-turn-pipeline.tour` · "The Turn Pipeline: One Spine for Every Gateway"

Purpose: the shared pipeline every gateway calls — context assembly, the loop,
the honesty check, persistence, and personas. This is the most important new
tour; it ties the others together.

| Step | Anchor | Teach |
|---|---|---|
| 5.1 The turn boundary | `internal/turn/turn.go` → file header / `func Run` (~L69) | "Gateways adapt, never re-implement." Web + CLI both call `Run`; behavior lives here, once. |
| 5.2 Context assembly | `internal/turn/turn.go` → context-building block (~L75–103) + `const RecentTurnWindow` (~L32) | Persistence ≠ context: full history in SQLite, only the last 20 turns + summary + today block enter the model. |
| 5.3 Driving the loop | `internal/agent/agent.go` → `func (l *Loop) Run` (~L84) | How `turn` hands history to the agent loop and consumes its `Event` stream. Cross-ref Tour 03. |
| 5.4 Composing the tool set | `internal/turn/tools.go` → `func Tools` (~L22) + `func ToolsForHead` (~L69) | How tool groups assemble; collision guard; head filtering. Cross-ref Tour 17. |
| 5.5 The honesty check | `internal/verify/verify.go` → `func CaptureSucceeded` (~L46) + `func ClaimsCapture` (~L90) | Runtime distrust of small-model claims; one self-repair pass; owner-facing note. |
| 5.6 Persistence & audit | `internal/turn/turn.go` → persistence block (~L124–164) | Append every turn; `OriginUncommitted` for unbacked claims; audit-after-write. |
| 5.7 Personas, not sandboxes | `internal/heads/heads.go` → `type Head` (~L27) + `func Active` (~L83) | Heads = identity + capability filter, **not** a security boundary; switching is reversible + audited. |

#### Tour 11 — `11-local-inference-kronk.tour` · "Local Inference: The Embedded Kronk Engine"

Purpose: how a turn gets a model — the `llm.Client` seam, provider resolution,
the in-process Kronk engine, runtime/model install, and the opt-in EU remote.

| Step | Anchor | Teach |
|---|---|---|
| 11.1 The one seam | `internal/llm/llm.go` → `type Client interface` (~L44) | `ChatStream` + `Embed`; everything above is provider-agnostic. |
| 11.2 What models exist | `internal/turn/models.go` → `func ModelChoices` (~L61) | Enumerate + resolve the active choice. |
| 11.3 Choice → client | `internal/turn/models.go` → `type ClientSource` (~L168) + `clientForConfig` (~L207) | Routes local GGUF → Kronk, explicit `openai` → remote. |
| 11.4 The resident engine | `internal/kronk/engine.go` → `func NewEngine` (~L52) + ensureInit/`Close` | Lazy `kronk.Init` on first inference, one resident chat + embed model, dlopen, `CGO_ENABLED=0` holds. |
| 11.5 Bridging to llm.Client | `internal/kronk/client.go` → `func (c *Client) ChatStream` (~L41) | Kronk `ChatResponse` → `llm.Chunk`; timeout injection. |
| 11.6 Runtime install (fail-closed) | `internal/kronk/librt.go` → `func InstallRuntime` (~L32) + `internal/kronk/runtime.go` `RuntimeInstalled` | Pinned llama.cpp b9664; sha256 manifest verify; engine never downloads on boot; CPU default vs `BALAUR_PROCESSOR=vulkan`. |
| 11.7 The EU catalog | `internal/kronk/officialmodel.go` → `func OfficialModels` (~L31) | Curated, pinned GGUF URLs+sha256; changing a model is a reviewed code change. |
| 11.8 Model download | `internal/kronk/modelget/modelget.go` → `func Fetch` (~L37) | HTTP Range resume, sha256 verify, atomic rename, Bearer token never logged. Cross-ref the opt-in remote (`internal/llm/openai.go`); embeddings stay local; key on-box never logged. |

#### Tour 12 — `12-persistence-and-store.tour` · "Persistence & the Store Seam"

Purpose: records-as-domain-model, the master conversation, and what
`internal/store` is allowed to be (cross-cutting only).

| Step | Anchor | Teach |
|---|---|---|
| 12.1 Persistence is not context | `internal/conversation/conversation.go` → file header + `CompactedThrough` (~L73) | The split: full record persisted, bounded window into context. |
| 12.2 The master singleton | `internal/conversation/conversation.go` → `func Master` (~L44) | Create-or-find with lost-race retry. |
| 12.3 Appending turns by origin | `internal/conversation/conversation.go` → `func Append`/`AppendOrigin` (~L81–96) | Origins (`nudge`, `briefing`, `check`, `uncommitted`) and why they exist. |
| 12.4 Context extraction | `internal/conversation/conversation.go` → `func RecentTurns` (~L134) | Reverse chronology, exclude runtime origins + tool rounds. |
| 12.5 Audit, fire-and-forget | `internal/store/audit.go` → `func Audit` (~L14) | Append-only, silent-fail by design, called AFTER the write. |
| 12.6 Owner settings (the only KV) | `internal/store/owner_settings.go` → `func SetOwnerSetting` (~L47) | UNIQUE-key retry-once; avatar rosters as single source of truth. |
| 12.7 Owner timezone | `internal/store/time.go` → `func OwnerLocation` (~L22) | Resolved per-call, no globals; anchors all period math. |
| 12.8 What store is NOT | `internal/store/llm_settings.go` → `EnsureDefaultLLMConfig`/`ListLLMModels` | Store = cross-cutting only (audit/settings/LLM config/time). Domain logic talks to PocketBase in its own package — it must NOT route through store. |

#### Tour 13 — `13-companion-domain.tour` · "The Companion Domain: Quests, Life & Recaps"

Purpose: the personal-companion features built on `nodes` — recurring tasks, the
life log, and the telescoping recap.

| Step | Anchor | Teach |
|---|---|---|
| 13.1 The recur DSL (pure) | `internal/tasks/recur.go` → `func Parse` (~L34) + `func Next` (~L85) | `daily`/`weekly`/`monthly`/`every:Nd`; occurrence math is derived, only `due` is persisted. |
| 13.2 Create a quest | `internal/tasks/tasks.go` → `func Create` (~L30) | Stored as node props; calendar-rule snap-forward. |
| 13.3 Completion | `internal/tasks/tasks.go` → `func Done` (~L189) | One-off closes; recurring logs a completion + bumps `due`. |
| 13.4 Nudge & briefing crons | `internal/tasks/nudge.go` → `func Nudge` (~L88) + `internal/tasks/briefing.go` → `func Briefing` (~L46) | Idempotency derived from origin tags / `nudged_at`, not a table row. |
| 13.5 The life log | `internal/life/life.go` → `func Log` (~L67) | Owner-defined measure kinds; `noted_at` is the chronology truth. |
| 13.6 A day, assembled | `internal/life/day.go` → `func Day` (~L30) + `internal/life/journal.go` → `func JournalWrite` (~L23) | Join done tasks + measures + journal for a calendar day. |
| 13.7 The period telescope | `internal/recap/periods.go` → `Period` + `Day/Week/Month/...` builders (~L11–86) | day→week→month→quarter→year; owner-tz bounds. |
| 13.8 Idempotent catch-up | `internal/recap/generate.go` → `func EnsureSummaries` (~L273) | High-water mark + existence short-circuit; resumes safely. |
| 13.9 Manual compaction | `internal/recap/compact.go` → `func DraftToday` (~L31) + `func CommitToday` (~L60) | Mid-day fold; appends, never rewrites; advances the boundary. |

#### Tour 14 — `14-nodes-and-graph.tour` · "The Knowledge Spine: Nodes & the Graph"

Purpose: the generic typed-object system that everything else now sits on.
Distinct from Tour 06/09 (which teach the memory/skill *domain* and recall).

| Step | Anchor | Teach |
|---|---|---|
| 14.1 Status is consent | `internal/nodes/nodes.go` → status constants (~L24–31) | `proposed`/`active`/`archived`/`rejected`; traversal filters `active` so proposals never surface as fact. |
| 14.2 Creating a node | `internal/nodes/nodes.go` → `func Create` (~L89) + `internal/nodes/types.go` `OwnerAuthoredTypes` (~L72) | Type contract; one row per object distinguished by `type`. |
| 14.3 Typed properties | `internal/nodes/schema.go` → `func ValidateProps` (~L40) + `ApplyTemplate` (~L114) | Per-type prop schemas + templates. |
| 14.4 Active-only queries | `internal/nodes/nodes.go` → `ListByTypeStatus` (~L229) + `Backlinks` (~L312) | How reads stay consent-scoped. |
| 14.5 Wikilinks become edges | `internal/nodes/links.go` → `func ParseLinks` (~L38) + `func SyncLinks` (~L89) | `[[wikilink]]` → `edges` rows; stub creation. |
| 14.6 Day pages | `internal/nodes/day.go` → `DayNode`/`LinkOnDay` (~L33–56) | `on_day` edges tie every node to its creation day. |

#### Tour 15 — `15-sovereign-export.tour` · "Sovereign Export & Encryption"

Purpose: the one-way Johnny Decimal Markdown mirror + passphrase-encrypted
archive, and the redaction boundary that makes it safe.

| Step | Anchor | Teach |
|---|---|---|
| 15.1 The Johnny Decimal tree | `internal/export/export.go` → `jdFolder`/`jdFolderFor` (~L59–91) | How node types map to numbered folders. |
| 15.2 The mirror writer | `internal/export/export.go` → `func ExportMirror` (~L109) | One-way mirror of active nodes; git commit of the export repo. |
| 15.3 Render (deterministic) | `internal/export/export.go` → `render` (~L223) | YAML front-matter + Markdown; byte-identical re-export (no `time.Now()` leak). |
| 15.4 The redaction boundary | `internal/export/export.go` → package header (~L1–9) | Reads ONLY `nodes` where `status=active`; never touches providers/secrets/settings. `day`/`task` are deferred. |
| 15.5 Passphrase encryption | `internal/export/encrypt.go` → `func EncryptDir` (~L57) + `ErrBadPassphrase` (~L49) | scrypt + AES-256-GCM, CGO-free via `x/crypto`; "lost passphrase = lost backup." |
| 15.6 The canary test | `internal/export/mirror_test.go` → `TestMirrorNeverLeaksStoredSecret` (~L229) | The regression that proves a stored secret never reaches the mirror — this invariant is sacred. |

#### Tour 16 — `16-extension-sandbox.tour` · "The Extension Sandbox: Untrusted JS, Safely"

Purpose: how balaur-extensions add verbs without privileges — goja, the consent
ledger, sha256 pinning, the tiny surface, and the regression suite that guards it.

| Step | Anchor | Teach |
|---|---|---|
| 16.1 Discovery ≠ service | `internal/ext/ext.go` → `func Sync` (~L61) | New files become proposals; nothing loads unapproved. |
| 16.2 Approval pins sha256 | `internal/ext/ext.go` → `func Approve` (~L150) | Approval hashes the file now and pins it; any later change re-proposes. |
| 16.3 Tools join the loop | `internal/ext/ext.go` → `func Tools` (~L115) | Bridged into the agent loop; can never shadow built-ins. |
| 16.4 The VM surface | `internal/ext/vm.go` → `func newVM` (~L121) | `balaur.registerTool`; the deliberately tiny surface (no fs/shell/db). |
| 16.5 Load-time side effects forbidden | `internal/ext/vm.go` → http binding gated (~L147–150) | `balaur.http` exists only inside a handler call, never at load — proves no egress until invoked. |
| 16.6 Egress guarded | `internal/ext/vm.go` → `deniedEgressIP` (~L56) + `extHTTPClient` (~L85) | Metadata/link-local denied by default; redirects not followed. |
| 16.7 Every call audited | `internal/ext/ext.go` → audit in tool bridge (~L141) | Per-invocation audit. |
| 16.8 The propose verb | `internal/ext/propose.go` → `func ProposeTool` (~L25) | The model's `propose_extension`; goja compile syntax-check. |
| 16.9 The sandbox regression suite | `internal/ext/ext_test.go` → `TestLoadTimeSideEffectsAreForbidden` (~L210), `TestMetadataEgressDeniedByDefault` (~L273), `TestTamperReproposes...` (~L116) | Why a goja version bump is a gated act (these tests must pass). |

#### Tour 17 — `17-the-tool-surface.tour` · "The Tool Surface: The Agent's Verbs"

Purpose: the full built-in verb catalog, the marker protocol, OS access, and the
consent/audit rules. Tour 05 showed how tools are *composed*; this shows what
they *are*.

| Step | Anchor | Teach |
|---|---|---|
| 17.1 Knowledge & graph verbs | `internal/tools/knowledge.go` → `func KnowledgeTools` (~L23) + `internal/tools/graph.go` `GraphTools` (~L17) | `remember`/`propose_skill` create **proposals**, not mutations (consent boundary lives in `internal/knowledge`). |
| 17.2 Task & life & journal verbs | `internal/tools/tasks.go` `TaskTools` + `internal/tools/life.go` `LifeTools` + `internal/tools/journal.go` `JournalTools` | Direct, audited mutations; journal stores the owner's words verbatim. |
| 17.3 The marker protocol | `internal/tools/ui.go` → `UICardMarker` (~L25) + `choices.go` `ChoicesMarker` + `artifact.go`/`refresh.go` | `\x00balaur-*` markers are inert to the model; the web layer reads them to render cards/choices and re-render live. |
| 17.4 Persona & profile verbs | `internal/tools/heads.go` `HeadsTools` + `internal/tools/profile.go` `ProfileTools` | Always-on (a scoped head can switch back); reversible config. |
| 17.5 OS access, gated | `internal/tools/os.go` → `func OSAccess` (~L50) | `read`/`write`/`edit`/`bash` ship disabled; opt-in `BALAUR_OS_ACCESS`; every call audited; secrets redacted. |

#### Tour 18 — `18-component-system-and-storybook.tour` · "The Component System & Storybook"

Purpose: the gomponents atomic design system over Datastar, and the storybook as
its source of truth. (Recommended as one ~11-step tour.)

| Step | Anchor | Teach |
|---|---|---|
| 18.1 Hearthwood tokens | `internal/web/assets/static/basm.css` → tokens block (~L1–71) | The canonical token source; the layout-token scale. |
| 18.2 An atom | `internal/ui/button.go` → `func Button` (~L36) | Typed gomponents primitive; `h "...html"` alias; attribute order. |
| 18.3 Card chrome & the escaping firewall | `internal/ui/components.go` → `func CardHead` (~L28) | `g.Text` escapes user/model text; `g.Raw` only for already-trusted HTML. |
| 18.4 A chat organism | `internal/ui/chat/dock.go` → `func Dock` (~L52) | Atoms composed into the chat chrome; Datastar `@post`/signals. |
| 18.5 The page shell | `internal/ui/shell/shell.go` → `func Page` (~L30) | The single place that emits `<!doctype html>`. |
| 18.6 A story's anatomy | `internal/feature/storybook/story.go` → `type Story` (~L27) | Variants, Props table, Do/Don't — the contract. |
| 18.7 The story registry | `internal/feature/storybook/story.go` → `func Page` (~L151) / registry slice | One source of truth for every component. |
| 18.8 Storybook routes | `internal/web/storybook.go` → `renderStorybook`/`sidebarFor` (~L47–64) | `/storybook` + `/storybook/{id}` wired through the shell. |
| 18.9 A domain card (tile + focus) | `internal/feature/taskcards/quests.go` → `QuestsCard` (~L24) | Domain cards compose `ui` atoms; one card type, two renderers. |
| 18.10 The rule | (prose, anchor to `story.go`) | Storybook is the source of truth: for any UI change, check it first, reuse/extend a component, and add/update its story in the same change. |

#### Tour 19 — `19-bootstrapping.tour` · "Bootstrapping: From `balaur` to a Running Companion"

Purpose: how the binary boots — the no-args launcher, the wiring phases, the
schema baseline, dev seeding, and the background jobs.

| Step | Anchor | Teach |
|---|---|---|
| 19.1 No-args launch | `main.go` → `func main` launcher block (~L40–73) + `internal/launch/launch.go` `IsLauncherInvocation` (~L55) | Bare `balaur` → pick a port → open the browser → rewrite argv. |
| 19.2 Port selection (loopback) | `internal/launch/launch.go` → `func SelectPort` (~L90) + `DefaultPort` (~L82) | 8099 stable default, free-port fallback; always 127.0.0.1. |
| 19.3 Wiring phases | `main.go` → PocketBase wiring (~L75–99) | New → migrations → CLI → `OnServe` hooks (lazy, post-migration). |
| 19.4 Schema baseline | `migrations/1749600000_init.go` → `InitCollections` (~L10+) | The ~14 collections; later files fold the domain into `nodes` (cross-ref Tour 14). |
| 19.5 The Kronk engine hook | `main.go` → `registerKronkEngine` (~L126) | Engine created at serve, no model loaded at boot (cross-ref Tour 11). |
| 19.6 Cron choreography | `main.go` → `scheduleJob` (~L157) + recap/nudge/briefing | Single-flight + catch-up; owner-tz timing (cross-ref Tour 13). |
| 19.7 The search sidecar | `main.go` → `registerSearchIndex` (~L247) | FTS5 `search.db` rebuilt on boot, corrupt-retry, per-collection sync hooks (cross-ref Tour 09). |
| 19.8 Dev seeding | `internal/seed/seed.go` → `func Run` (~L70) + `func Reset` (~L143) | Marker-tagged, idempotent, reversible; dev-seeded / prod-empty. |

**Verify (per new tour, after writing each)**:
`go test . -run TestTours -v` → `PASS` (the count of checked tour files goes up
by one each time).

### Stage C — Wire-up & final pass

1. **Orientation cross-links**: ensure `00-orientation.tour`'s package map names
   every new tour (05, 11–19) so a reader can navigate the whole set from step 1.
2. **Maintenance contract**: confirm `AGENTS.md` still contains the line starting
   "`.tours/` is a maintained artifact". Do not edit it if present. If it is
   absent (drift), that is a STOP condition — report it.
3. **Optional cosmetic**: in `tours_test.go`, the `t.Fatalf("...expected at least
   11")` message may be updated to `20`. Logic unchanged.

**Verify (final)**:
- `ls .tours/*.tour | wc -l` → `20`
- `go test . -run TestTours -v` → `PASS`
- `go test ./...` → `ok` (no product code changed, so nothing else should move)
- `git diff --check` → clean
- `git status` → only `.tours/*.tour` (and at most `tours_test.go`,
  `plans/README.md`) modified.

## Test plan

The lint test **is** the test — there is no new test to author. Coverage is
proven by:
- `go test . -run TestTours -v` passing over **20** tour files (each step's
  file exists and every line anchor is in range).
- The grep gates in Stage A (no stale engine/template terms; no "29 migration").
- A human spot-read (see Maintenance notes) for prose accuracy the lint can't
  see.

## Done criteria

ALL must hold:

- [ ] `ls .tours/*.tour | wc -l` → `20` (10 refreshed + 10 new: 05, 11–19)
- [ ] `go test . -run TestTours` → PASS over all 20 files
- [ ] `go test ./...` → `ok` for every package
- [ ] `grep -rli 'llamafile\|ollama\|html/template\|web/templates' .tours/` → no matches
- [ ] `grep -rl '29 migration' .tours/` → no matches
- [ ] Every new tour has 6+ steps, each anchored to a real symbol declaration
- [ ] `00-orientation.tour` links to every new tour
- [ ] `git status` shows only in-scope files changed
- [ ] `plans/README.md` status row for plan 198 updated

## STOP conditions

Stop and report (do not improvise) if:

- A symbol named in this plan does not exist in its file (grep finds nothing) —
  the codebase drifted; report the package + symbol so the plan can be corrected.
  Do **not** invent a different anchor to "make it work."
- A whole package in the coverage matrix has been renamed or removed since
  `7642f92`.
- The `AGENTS.md` `.tours/` maintenance line is missing.
- You find yourself wanting to edit any file under `internal/`, `main.go`, or
  `migrations/` to make a tour read better — never. Tours describe the code; the
  code does not bend to the tour.
- `go test . -run TestTours` fails twice on the same tour after a genuine fix
  attempt (likely a JSON syntax error or an out-of-range line).

## Maintenance notes

- **The lint cannot catch prose drift** — a line that moved but still exists, or
  a description that is now subtly wrong. Mitigation already baked in: anchor to
  declarations, not mid-body lines. A reviewer should spot-read 2–3 steps per new
  tour against the live code and confirm the voice matches the existing tours
  (plain, teaching, no hype).
- **When a future change moves an anchored symbol**, `tours_test.go` fails the
  suite — fix the tour in the same commit (the AGENTS.md contract). When a change
  alters what a tour *teaches* (not just where), update the prose too.
- **Deferred**: this plan does not add a `.tours/README.md` index or renumber the
  existing tours into a strict curriculum — numbers stay stable to avoid churn and
  broken external references. If a curriculum index is later wanted, add it as a
  new doc, not by renumbering.
