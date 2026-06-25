# Balaur — Modularity & Coupling Audit

> Read-only architecture audit focused on reducing coupling / tangled code so any
> one part of the system can be understood without tracing a tangled path across
> many files. Generated against `main` near commit `07fb4d6`. Findings were
> produced by deep per-subsystem readers, then each was adversarially verified
> against the repo's KISS/YAGNI/suckless doctrine (44 raw findings → 24 verified
> real tangles; 20 dropped as justified or over-engineering).

## 1. Verdict

Balaur is a **clean small-core design, not a ball of mud** — the dependency graph
is overwhelmingly a DAG with stable, zero-efferent hubs (`store`, `ui`, `llm`,
`nodes`) at the bottom and gateways at the top, exactly as the doctrine
prescribes. The central architectural thesis holds: `internal/turn` is a real
shared pipeline, and the gateways mostly delegate to it rather than re-running the
loop. The damage is concentrated and shallow: a handful of **god-files that
crossed the 500-LOC decompose line** (chiefly `web/models.go` at 550 LOC, which
alone surfaced in five clusters) and a scatter of **small verbatim duplications**
where two callers re-derive the same domain fact instead of asking the owning
package. THE biggest comprehension tangle is `web/models.go`: it fuses the
chat-dock view-model, model selection, cloud-consent, and two long-lived SSE
background-job orchestrators (with `app.Store()` cancel-func sidecars) into one
file, so understanding any one of those four flows forces you to read past the
other three. Nothing here needs new abstraction — every fix is
extract/split/move/merge, and the riskiest mistake would be "fixing" coupling that
is deliberate medium-adaptation.

## 2. The coupling map

The **stable foundation** behaves correctly: `store` (Ca=16, Ce=0), `ui` (Ca=12,
Ce=0), `llm` (Ca=11, Ce=0), and `nodes` (Ca=13, Ce=1) are high-fan-in /
low-or-zero-fan-out sinks — many depend on them, they depend on almost nothing.
That is the shape you want at the bottom of a DAG. `store` having Ce=0 confirms it
stayed a leaf cross-cutting seam (audit/settings/LLM config/time) and did not
accrete domain logic.

The **orchestrators** layer correctly on paper: `turn` (Ce=12) sits below the
gateways and above the domains; `web` (Ce=28) and `cli` (Ce=17) sit at the top.
The KEY HYPOTHESIS — that `web` and `cli` re-reach into every domain (`agent`,
`conversation`, `knowledge`, `nodes`, `recap`, `tasks`, `tools`, `life`) that
`turn` already orchestrates — **is mostly disproven for the hot path**: `turn.Run`
is the single owner of the loop, and both gateways call it (`cli/chat.go:105`,
`turn/turn.go:114`). The gateways' wide fan-out is overwhelmingly *rendering* (web
→ cards/ui) and *non-turn surfaces* (model management, export, seed, recap
browsing), which legitimately live above the gateway line. So the graph is a clean
DAG at the package level.

Where it **stops being clean is intra-package and at a few domain edges**, not in
the core import graph:

- **`tools` (Ce=9)** is the one orchestrator that fans into every domain *and*
  leaks two domains' internals back out: `tools/knowledge.go:483` hand-rolls
  `recap`'s summaries query, and the "task is a row in `nodes`" storage fact is
  encoded in `tools`, `cli`, and `web` simultaneously.
- **`life/day.go`** is a genuine cross-domain aggregator (imports `nodes`,
  `recap`, `store`, AND `tasks`) that re-derives `tasks`' done-task semantics by
  hand.
- **`heads.Groups` ↔ `turn/tools.go`** is a string-coupled contract across two
  packages that don't import each other's constant — compile-unchecked.

None of these are cycles; they are knowledge-leaks where a consumer re-implements
a fact the owning package should expose.

## 3. Top tangles, ranked

Ordered by comprehension-pain × cheapness-to-fix. Gateway-duplication thesis
leads.

**1. `web/models.go` — 550-LOC god-file fusing four unrelated flows (HIGH pain,
cheap).**
The file holds the chat-dock view-model (`homeData`/`chatbar`/`patchChatbar`/
`refreshDockChrome`, 27–142), model selection + cloud consent (`selectModel`/
`saveCloudModel`/`confirmCloudModel`/`deleteCloudModel`, 339–482), and **two
long-lived SSE background-job orchestrators** — `downloadOfficialModel` (196–300)
and `installRuntime` (492–550) — each parking a `context.CancelFunc` on
`app.Store()` (`downloadStoreKey`, `runtimeInstallStoreKey`). To edit how the
dock's model label refreshes you scroll past 100+ lines of HuggingFace download
streaming; to trace a download you filter out dock chrome and consent dialogs. It
crosses the repo's own decompose threshold and the file's *name* hides that it
owns the home page's view-model.
**Smallest move:** pure same-package relocation, zero new types. (a) Move
`homeData`/`chatbar`/`patchChatbar`/`refreshDockChrome` + the `headChoice`/
`homeData` structs into `home.go`, where their renderers (`chatBarNode`/
`composerNode`) already live. (b) Move both SSE orchestrators + their shared
single-flight helpers (`claimInFlight`, both `*StoreKey`, `formatProgress`,
`humanBytes`, the injectable `kronk*` seams) into `models_install.go` — they are
the same single-flight-progress shape and the only place the cancel sidecars live.
`models.go` drops to ~150 LOC of model-selection + cloud-consent + `setProcessor`/
`modelsPanel`. **Do NOT** wrap the two SSE handlers behind a shared "progress
orchestrator" interface — that ~15 lines of single-flight boilerplate is not worth
a single-impl seam. **Effort: medium** (mechanical, but the largest relocation).

**2. Tool output is a NUL-delimited side-channel every gateway re-decodes (HIGH
pain, but LEAVE — see §5).**
`agent.Event.Text` (`agent/agent.go:32`) smuggles five NUL-prefixed markers
(`UICardMarker`, `ChoicesMarker`, `ProposalMarker`, `RefreshMarker`,
`ArtifactMarker`); the agent loop forwards them blind (`agent.go:132`), and each
gateway re-decodes the set in its own ladder — `web/chatstream.go:201` does all
five, `cli/chat.go:77` only two. The encoding contract lives nowhere and is
enforced nowhere; a sixth marker silently degrades on the CLI. This is real latent
drift. **Smallest *defensible* move:** a single exhaustiveness/round-trip test
colocated with the markers in `internal/tools` asserting every `Marker` constant
has a `Parse*` that round-trips and returns `ok=false` for every *other* marker's
output, so adding a marker forces a test edit. **Do NOT** add `Event.Card
*ToolResult` or `tools.ClassifyResult` — both relocate the per-medium branching
without removing it (the divergence is correct medium-adaptation). **Effort:
small** (test only).

**3. "Load a task node from `nodes` + Hydrate" is encoded in three callers (MEDIUM
pain, cheap).**
The fact "a task is a row in the `nodes` collection that must be hydrated before
use" leaks into `cli/task.go:42`, `web/tasks.go:19`, and `tools/tasks.go:243/414/
455` — and `tools/tasks.go` even has its own `findTask` (447) that two sibling
tools re-inline anyway. `tasks` exposes `Hydrate` but no getter.
**Smallest move:** add `tasks.Get(app, id) (*core.Record, error)` next to
`Hydrate` (mirroring the already-blessed `nodes.Get`, `nodes.go:207`) —
`FindRecordById("nodes", strings.TrimSpace(id))` + `Hydrate`, nothing more.
Collapse all five sites; each gateway keeps its own user-facing error string by
wrapping `%w`. Pass an already-decoded id, not raw JSON, so JSON decoding stays in
the tool layer. **Effort: small.**

**4. `node_get` re-implements recap's summary lookup + reaches into the
`summaries` schema (MEDIUM pain, cheap).**
`tools/knowledge.go:483` hand-rolls `FindFirstRecordByFilter("summaries",
"conversation = … && period_type = 'day' && period_start ~ …")` against a
collection `recap` owns — the *only* non-test use of `conversation` and
`summaries` in `internal/tools`. `recap.Find(app, convID, recap.Day(t))`
(`recap/generate.go:26`) already does the exact-match version; the tool copies a
prefix-match (`~ 'd%'`) variant, so a recap-side column rename breaks `node_get`
with no compile error.
**Smallest move:** replace the inline block with `recap.Find(app, conv.Id,
recap.Day(day))`, parsing `day` in the **owner's** location to match `recap.Day`'s
truncation (not host `time.Local`, or you reintroduce the skew). Read content via
the returned record. Deletes the `"summaries"` literal and the filter from
`tools`. **Effort: small.**

**5. `taskViewOf`/`questGroup` view-model duplicated between `taskcards` and the
`web` gateway (MEDIUM pain, cheap).**
Two LIVE copies: `taskcards/quests.go:90` and `web/tasks.go:44`+`106` build the
same `TaskView` (overdue/due/recur logic via `tasks.DueLine`/`tasks.Describe`);
`web/tasks.go:66 questGroup` is byte-for-byte `taskcards/questsfocus.go:94
questGroupName`. The card package already owns the canonical superset mapper;
`web` re-derives a view the card layer is the source of truth for, and
`web/tasks.go:64` even comments that the two paths must behave identically.
**Smallest move:** export `taskcards.TaskViewOf(rec, now)`, have `web`'s
single-card route call it, delete `web`'s `taskView`/`taskViewOf`/
`taskCardViewOf`. **Delete `web.questGroup` + `TestQuestGroup` outright** — it is
dead code shadowing `taskcards.questGroupName`. Layering stays correct (`web →
taskcards`, never the reverse). **Effort: small** (~50 lines removed).

**6. `cardShowTool` / `showCardsTool` duplicate the card-registry vocabulary
builder verbatim (MEDIUM pain, trivial — CLEAR WIN).**
`tools/ui.go:71` and `tools/artifact.go:70` contain a byte-for-byte ~20-line
`cards.All()` → `"type (label) — params…"` block; only the lead-in prose differs.
**Smallest move:** one unexported `cardRegistryVocab() string` in `internal/tools`;
each tool keeps its own intro and appends it. **Keep it in `tools`, NOT `cards`** —
pushing it down would couple the domain registry to agent-loop prose. **Effort:
small.**

**7. `life/day.go` re-derives `tasks`' done-task semantics (MEDIUM pain, cheap).**
`life/day.go:88` calls `tasks.Hydrate` on raw `nodes.ListByTypeStatus("task")`
rows then re-implements completion (`status != "done"`, `done_at` in range) —
knowledge that belongs to `tasks`. A change to how completion is recorded breaks
`life` silently.
**Smallest move:** add `tasks.DoneBetween(app, start, end) ([]*core.Record,
error)`, the symmetric sibling of the existing `tasks.OpenTasks`. `life.Range`
calls it for the task half. **Keep the completion-*entries* half (`day.go:104`) in
`life`** — those are `entries`-collection rows, not task nodes, and `life` rightly
owns assembling the cross-source `RangeData`. **Effort: small.**

**8. `knowledge.go` — 637-LOC god-file, five concerns (MEDIUM pain, cheap).**
Hydration, proposal lifecycle, the parked-edit envelope, management listing, and
two search surfaces in one file (`cache.go`/`context.go` already split out
cleanly; this one didn't).
**Smallest move:** extract `edit.go` (parked-edit envelope, ~135 LOC) and
`search.go` (search trio, ~130 LOC), leaving ~370 LOC. **Keep `matchesQuery` in
`knowledge.go`** — both `FilterActive` (management) and the search fallback use it;
relocating it creates a management→search-file dependency. If you want the
absolute minimum, extract `edit.go` only (drops to ~500, highest-confidence
slice). **Effort: small.**

**9. `recap.go` (404 LOC) bundles the recap telescope with the shared chat-history
renderer (LOW pain, cheap).**
`messageView`/`renderMessages`/`chatBodyHTML`/`messageViews` (129–306) are core
chat infrastructure imported by home/dock/tasks/compact, but live in a file named
`recap.go`, discoverable only by following `dockData`.
**Smallest move:** extract those four symbols into `internal/web/history.go` next
to `chatstream.go` (live + reload renderers together). Same package, no import
changes, tests compile untouched. **Effort: small.**

**10. `turn.Tools` / `turn.ToolsForHead` duplicate the assembly tail + self-tool
naming ritual (LOW pain, cheap).**
Both (`turn/tools.go:22` and `:69`) repeat the always-on core, the OS gate, the
`taken` collision-guard, and the names-collection + `self.Tool` append; the
comment at `:107` literally says "mirroring Tools(app)."
**Smallest move:** extract one helper `finalize(app, ts, withExtensions bool)`
owning the collision-guard, conditional `ext.Tools` append, and `self.Tool` tail.
**Critical:** the `ext.Tools` append is *unconditional* in `Tools` but *gated on
`sel["extensions"]`* in `ToolsForHead` — a naive shared helper that always appends
would leak approved extensions into heads that didn't select them (a
capability-filter regression). The `withExtensions` param is non-speculative. Skip
the separate `coreTools` extraction (saves 4 lines, borderline YAGNI). **Effort:
small.**

**11. `heads` tools audit at the tool layer; siblings audit in-domain (LOW pain,
cheap — fixes a live gap).**
`tools/heads.go:53/95/125` call `store.Audit` *after* `heads.SetActive/Create/
Delete`; `internal/heads` has no `Audit` call, unlike `tasks`/`life`/`knowledge`/
`nodes`. Consequence: the **web** head switch/create/delete (`web/heads.go`) is
currently **unaudited**.
**Smallest move:** add an `actor string` param to the three `heads` funcs,
audit-after-save inside them (mirroring `tasks.Done`); pass `actor="model"` from
tools, `actor="owner"` from web — closing the live UI gap. Decide `seed.go:525`
explicitly (`actor="seed"` or no audit). **Effort: small.**

**12. `heads.Groups` ↔ `turn/tools.go` string-coupled contract (LOW pain,
cheap).**
`heads.go:38 var Groups` and `turn/tools.go:68`'s switch must stay in sync by
hand; a typo yields a head whose group maps to no tools, compile-unchecked.
**Smallest move:** a table-driven guard test *in package `turn`* (already imports
`heads`): for each `g` in `heads.Groups`, assert `ToolsForHead(app, {g})` yields
strictly more tools than core-only. `os` is env-gated — set `BALAUR_OS_ACCESS` or
special-case it. **Do NOT** add `turn.ToolGroups()` for `heads` to reference —
that inverts a domain→pipeline dependency and risks a cycle. **Effort: small**
(~15 lines, no production change).

**13. Two parallel `Transition` implementations (LOW pain, cheap).**
`nodes.go:237` and `knowledge.go:167` both load/validate-against-
`ValidTransitions`/`Set`/`Save`/`Audit`; the knowledge copy only differs by audit
prefix (`knowledge.*` vs `node.*`) and a trailing cache-invalidate. The plan-183
comment already *declares* knowledge "wraps nodes.Transition" — the implementation
diverged.
**Smallest move:** make `knowledge.Transition` delegate to `nodes.Transition` with
a passed-in audit-action prefix, then layer `invalidateContextCache` + hydrate.
The prefix is asserted by tests, so it must be a parameter, not collapsed.
**Effort: small.**

**14. CLI engine bootstrap drops the owner's saved processor (LOW pain, cheap —
fixes a real fork).**
`cli/chat.go:25` builds the engine with `kronk.Processor()` (env/cpu default);
`main.go:127` uses `resolveProcessor(app)`, which honors
`owner_settings.llm_processor`. So `balaur chat` can run inference on a *different*
processor than the server.
**Smallest move:** extract `resolveProcessor`'s body into
`turn.ResolveProcessor(app)` — `turn/models.go` already imports both `kronk` and
`store`, and both `main.go` and `cli/chat.go` import `turn`, so **zero new package
edges**. **Do NOT** put it in `kronk` (forces a forbidden `kronk → store` edge,
pushing owner-settings policy into the dlopen engine). **Effort: small.**

**15. `settingscards` reaches into the raw kronk SDK for runtime-install state
(MEDIUM pain, cheap).**
`settingsfocus_models.go:13` imports
`github.com/ardanlabs/kronk/sdk/tools/libs` directly (calling `libs.IsSupported`,
`libs.ReadVersionFile`) — the **only** SDK import outside `internal/kronk`. A UI
view-builder now depends on the SDK's library-layout contract; a kronk upgrade
ripples into a `feature/` package.
**Smallest move:** add `kronk.RuntimeStatus(processor) (supported bool, version
string)` wrapping `libs.IsSupported` + `InstallDirFor` + `ReadVersionFile` inside
`internal/kronk` (computing the triple from `runtime.GOOS/GOARCH` itself).
`settingscards` calls it and maps the two scalars to its own `modelcards.Status*`
constants. **Return facts, not a presentation string** — keep UI vocabulary in the
UI. **Effort: small.**

**16. `seed/world.go` repeats create→backdate→reload→link ~9 times (LOW pain,
cheap).**
The "must reload after `backdate` or `LinkOnDay` resolves the wrong day node"
invariant (`world.go:324`) is duplicated nine times; one missed reload is a silent
wrong-day bug.
**Smallest move:** extract `backdateAndReload(app, rec, at) (*core.Record, error)`
(the backdate+reload pair, *not* the link), owning the gotcha once. Seed/demo code
— no domain concern. **Effort: small.**

**17. `cli/verify.go` hand-rolls a second honesty verdict from records (LOW pain,
cheap — marginal).**
`cli/verify.go:50` re-derives the verdict over `messages` rows (`tool_name`/
`origin` columns + an inline `"error:"`-prefix check + the hand-written `honest =
!claims || captured`), while the canonical path uses `verify.CaptureSucceeded`
over `[]llm.Message`. Two implementations of "honest" that can drift.
**Smallest move:** add primitive helpers to `internal/verify` — `ToolSucceeded(name,
content string) bool` and optionally `Honest(claims, captured bool) bool`
(booleans/strings in, bool out). **Do NOT** add a `Verdict` type or make `verify`
import `core.Record` — coupling the pure honesty rule to PocketBase to save one
line is the wrong trade. Genuinely marginal; defensible to leave. **Effort:
small.**

**18. `tasks`/`life` hydrate-alias dance + duplicated PB-datetime format (LOW
pain, trivial).**
`tasks.go:417` and `life.go:44` each define a private `fmtTime` with the same
`"2006-01-02 15:04:05.000Z"` literal; both `SetRaw`-alias node props.
**Smallest move:** lift only the format constant — add `nodes.PBTimeString(t)` (+
optional `ParsePBTime`) where `nodes` already owns prop access, delete the two
`fmtTime`. **Do NOT** add `nodes.SetPropAlias` — wrapping a one-line `SetRaw` buys
nothing and the per-type alias maps rightly stay in their domains. **Effort:
small.**

## 4. Quick wins (do first)

Small-effort, high-clarity, no new abstractions:

- [ ] **#6** Extract `cardRegistryVocab()` in `tools` — delete ~20-line verbatim
  dup (CLEAR WIN).
- [ ] **#5** Export `taskcards.TaskViewOf`, delete `web`'s three copies + dead
  `web.questGroup`/`TestQuestGroup` (~50 lines gone).
- [ ] **#3** Add `tasks.Get(app, id)`; collapse five find-task-node sites across
  `cli`/`web`/`tools`.
- [ ] **#4** Route `node_get`'s day-summary lookup through `recap.Find`
  (owner-local `recap.Day`); delete the `"summaries"` literal from `tools`.
- [ ] **#7** Add `tasks.DoneBetween`; move the done-task branch out of
  `life/day.go`.
- [ ] **#9** Move the chat-history renderer (`messageView`/`renderMessages`/
  `chatBodyHTML`/`messageViews`) into `web/history.go`.
- [ ] **#8** Extract `knowledge/edit.go` (and optionally `search.go`), keeping
  `matchesQuery` in place.
- [ ] **#11** Audit head mutations in-domain with an `actor` param — closes the
  unaudited UI head-switch gap.
- [ ] **#12** Add the `heads.Groups` ↔ `ToolsForHead` guard test in package
  `turn`.
- [ ] **#14** Extract `turn.ResolveProcessor`; make `balaur chat` honor the
  owner's processor choice (zero new edges).
- [ ] **#15** Add `kronk.RuntimeStatus`; drop the `ardanlabs/kronk` import from
  `settingscards`.
- [ ] **#16** Extract `seed.backdateAndReload`; kill the wrong-day footgun at all
  sites.
- [ ] **#18** Lift the PB-datetime format into `nodes.PBTimeString`; delete the
  two private `fmtTime`.

## 5. Deliberate / leave alone

Coupling that looks bad but is correct — **do not "fix":**

- **Per-medium marker decoding (#2).** The CLI decoding 2 of 5 markers is correct
  medium-adaptation, not drift — a `uicard` has no terminal rendering. The five
  markers render differently per medium *and* per render-path (live SSE morph vs
  reload snapshot vs CLI JSON), so a typed `Event.Card` or `tools.ClassifyResult`
  only relocates the branching; every site still re-switches. The agent loop
  forwarding tool output *verbatim* is by design (the model sees what the caller
  sees). Ship only the exhaustiveness test.
- **`web`/`cli` wide fan-out.** Their breadth is rendering + non-turn surfaces
  (models, export, recap browsing), which legitimately sit above the gateway line.
  The turn hot path *is* shared via `turn.Run`. Don't manufacture a "service
  layer" to shrink Ce.
- **`store` at Ce=0.** It stayed a leaf cross-cutting seam. Do not route new domain
  logic through it.
- **The `"nodes"` collection name across ~40 sites.** A deliberate repo-wide
  idiom, not a leak. `tasks.Get` (#3) collapses the *find+Hydrate pairing*, not the
  collection name — don't frame it as "hiding nodes."
- **`nodes.Transition` with no production callers (#13).** Keep `nodes` as the
  lifecycle home and delegate to it; deleting it to collapse into `knowledge` only
  works if no future non-knowledge node type ever needs transitions.

**Over-engineering temptations to refuse:** a `Verdict` type or `core.Record`
import in `verify` (#17); `Event.Card`/`tools.ClassifyResult` (#2); a shared
"progress orchestrator" interface over the two SSE handlers (#1); folding
`ext.Tools` into the shared tool tail (#10, capability-filter regression);
`kronk.ResolveProcessor`/`kronk.EngineFromStore` forcing a `kronk → store` edge
(#14); `turn.ToolGroups()` inverting heads→pipeline (#12); `nodes.SetPropAlias`
wrapping a one-liner (#18); pushing `cardRegistryVocab` into `cards` (#6). Every
one trades a real one-binary/suckless boundary for cosmetic line savings.

## 6. Suggested sequencing

Ordered to minimize churn and let each step unblock the next:

1. **Pure deletions & verbatim-dup collapses first (#6, #5).** No behavior change,
   no dependencies — they shrink the surface and confirm the test suite is green
   before riskier moves. #5 removes dead code (`web.questGroup`).
2. **Domain-getter extractions (#3 → #4, #7).** Land `tasks.Get` first; it's the
   shared primitive that simplifies the `tools`/`cli`/`web` task sites. Then
   `recap.Find` routing (#4) and `tasks.DoneBetween` (#7) — together these pull all
   three domain-internal leaks (`summaries` schema, done-task semantics,
   find+Hydrate) behind their owning packages, leaving the gateways thinner
   *before* you split their files.
3. **God-file splits (#9, #8, then #1).** Do the small pure-move splits
   (`web/history.go`, `knowledge/edit.go`) first to build confidence, then the big
   one — `web/models.go` (#1) — last, after #3/#5 have already lightened
   `web/tasks.go` and the task view-model. Splitting `models.go` after the domain
   cleanup means you relocate less code.
4. **Audit & policy consistency (#11, #14, #15).** Independent of the splits; each
   closes a real behavioral gap (unaudited UI head-switch, CLI processor fork, SDK
   leak) and can land anytime, but grouping them keeps the "boundary correctness"
   review in one pass.
5. **Guard tests & low-priority tidies (#2 test, #12, #13, #10, #16, #18, #17).**
   Land the exhaustiveness (#2) and group-wiring (#12) guard tests once the code
   they protect is stable, then fold the remaining small dedups (#10, #13, #16,
   #18, #17) into the next change that already touches each file — none warrants a
   standalone churn commit.

Run `go test ./...`, `go vet ./...`, `staticcheck`, and `CGO_ENABLED=0 go build
./...` after each numbered step; gate every push on a green full suite per
doctrine.
