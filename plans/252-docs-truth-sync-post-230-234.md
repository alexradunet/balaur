# Plan 252: Truth-sync the maintained docs and tours to the shipped 230–234 reality

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, do **NOT** update `plans/README.md` —
> the reviewer who dispatched you maintains the index. Do not touch
> `plans/README.md` at all.
>
> **Drift check (run first)**:
> `git diff --stat 077318a..HEAD -- AGENTS.md internal/self/knowledge.md .tours/00-orientation.tour .tours/07-the-web-gateway.tour .tours/10-the-cli-api.tour .tours/15-sovereign-export.tour .tours/19-bootstrapping.tour docs/first-run-design.md`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition. Exception: plan 236 (a dependency)
> edits AGENTS.md's Known-limitations section — a diff in AGENTS.md that does
> NOT touch the vault-mirror bullet quoted below is expected and fine.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW
- **Depends on**: plans/236-turn-guard-honesty-scope.md (that plan corrects the
  turn-guard comments in `internal/web/messenger.go` and `internal/cli/chat.go`
  to honest per-process wording; this plan writes the same honest claim into
  `internal/self/knowledge.md`, so 236 must land first). Also note: a later
  plan (253, dx) touches AGENTS.md — land this plan before it to avoid merge
  friction.
- **Category**: docs
- **Planned at**: commit `077318a`, 2026-07-01

## Why this matters

Five features landed same-day (plans 224/225/230/231+233/232/234: the `balaur
restore` verb, day-journal export, the first-run onboarding banner, the
messenger API endpoint, the single-instance launcher guard, and the
cross-surface turn guard) without the repo's own "update the docs in the same
change" rules being applied. The cost is concrete: `AGENTS.md` is injected
into every agent session as law and currently forbids claiming a feature that
shipped months of work ago; `internal/self/knowledge.md` is embedded in the
binary and served through the self tool — a stale copy makes Balaur lie about
itself; the `.tours/` files are the onboarding path and their lint
(`tours_test.go`) only catches missing files and out-of-range line anchors,
never false prose or anchors that drifted onto the wrong (but in-range) line.
This plan re-syncs every verified drift to what the code actually does. It
changes zero behavior — Markdown and `.tour` JSON only.

## Current state

Every excerpt below was read from the live file at commit `077318a`.
Re-verify each before editing (the drift check above).

### 1. AGENTS.md claims the vault mirror is unshipped

`AGENTS.md:244-245` (in "Known limitations & deferred work"):

```
- The Johnny Decimal Markdown vault mirror (one-way export + git) is
  roadmap, not shipped. Do not claim it in user-facing copy until real.
```

Reality: the mirror shipped. `internal/export/export.go:59-81`:

```go
var jdFolder = map[string]string{
	"note":   "10-19 Knowledge/11 Notes",
	"idea":   "10-19 Knowledge/12 Ideas",
	"person": "20-29 People/21 People",
	"book":   "30-39 Library/31 Books",
	"place":  "40-49 Places/41 Places",
	"day":    "50-59 Journal/51 Days",
	// "task" has a JD folder in the design (60-69 Tasks/61 Tasks) but is
	// DEFERRED — see deferredTypes. It is intentionally NOT exported until its
	// content redaction pass lands (a future plan).
}
...
var deferredTypes = map[string]bool{
	"task": true,
}
```

`internal/export/encrypt.go:21` documents the `--encrypt` envelope (plan 195,
scrypt + AES-256-GCM), and `internal/cli/restore.go:14-17`:

```go
// restoreCmd decrypts an archive produced by `export --encrypt` into a
// plaintext directory tree the owner can read. It does NOT re-import into the
// live database — that is a separate, later capability. The passphrase comes
// from BALAUR_EXPORT_PASSPHRASE (never a flag), matching the export convention.
```

`README.md:143` is already truthful (do NOT edit README.md; this is the
spot-check baseline):

```
- **Your record, portable:** `balaur export` writes a one-way Johnny Decimal Markdown mirror of your active nodes (plan 194); `--encrypt` produces a passphrase-protected archive (plan 195).
```

### 2. internal/self/knowledge.md — four stale spots

(a) The `internal/turn` bullet says nothing about the in-flight guard.
`internal/self/knowledge.md:74-77`:

```
- internal/turn owns one owner turn: context assembly (system prompt,
  present moment, today block, knowledge block, recent window), the
  agent loop, the verify honesty check with one self-repair pass, and
  persistence. It also resolves the active model choice.
```

Reality: `internal/turn/guard.go` `TryBegin()` (line 24) is acquired by the
web chat handler (`internal/web/chat.go:40-48`):

```go
	end, ok := turn.TryBegin()
	if !ok {
		// Open a minimal SSE connection just to deliver the toast; no #chat
		// mutation, no user bubble.
		sse := datastar.NewSSE(e.Response, e.Request)
		emitToast(sse, "warn", "One message is still being answered — try again in a moment.")
		return nil
	}
	defer end()
```

and by the messenger endpoint (`internal/web/messenger.go:83-87`):

```go
	end, ok := turn.TryBegin()
	if !ok {
		return e.JSON(http.StatusTooManyRequests, map[string]string{"error": "busy"})
	}
	defer end()
```

The guard is a package-level mutex — **per-process**. A separate `balaur chat`
process takes a different mutex in a different address space and is NOT
covered. Plan 236 (the dependency) corrects the source comments to this honest
wording; knowledge.md must state the same honest claim, not the inflated
"every surface" one.

(b) The launcher paragraphs omit the single-instance guard and the onboarding
banner. `internal/self/knowledge.md:420-426`:

```
- main.go — wire-up: PocketBase app, migrations, CLI, routes, crons.
  A bare `balaur` (no args) is the no-terminal launcher: it boots a
  loopback UI on the XDG data dir, prefers a stable default port
  (8099, falling back to a free port if taken), and opens the browser.
- internal/launch — the no-args loopback launcher helpers (XDG data dir,
  stable default port + free-port fallback, first-run stat, browser-open);
  fires only on a bare argv and never constructs a non-loopback address
```

Reality in `main.go:53-95` (abbreviated):

```go
	if launch.IsLauncherInvocation(os.Args[1:]) {
		// Single-instance guard (plan 232): ... FAIL-OPEN: any error from
		// RunningInstance → proceed to start. Stale locks ... handled by the
		// TCP probe ...
		if addr, alive := launch.RunningInstance(launch.DataDir()); alive {
			...
			return
		}
		isFirstRun = launch.IsFirstRun(launch.DataDir())
		port, err := launch.SelectPort()
		...
		if err := launch.WriteInstanceLock(launch.DataDir(), addr); err != nil { ... }
		os.Args = append(os.Args[:1], "serve", "--http", addr, "--dir", launch.DataDir())
		...
	}
```

and the banner: `internal/web/home.go:217-222` — `onboardingBannerNode`
renders "the first-run onboarding banner — an info Alert with a link to model
setup. Shown only when first-run AND no active model; dismissible via the
client-side $firstRunDismissed signal". `main.go:111` stashes the flag:
`se.App.Store().Set("balaur_first_run", isFirstRun)`.

(c) Navigation lists are missing Chronicle and Graph.
`internal/self/knowledge.md:262`:

```
one icon per primary destination (Quests, Life, Memory, Skills, Settings), and a
```

`internal/self/knowledge.md:268-269`:

```
surfaces fire GET /ui/show/{type}; the full destination set is Quests, Life,
Memory, Review, Skills, and the three settings sections (Profile, Models, Heads).
```

Reality — `internal/web/home.go:133-148` `navDestinations()` lists (in order):
Quests, Life, **Chronicle**, Memory, Review, Skills, **Graph**, Profile,
Models, Heads, plus the "Compact today" action; `home.go:160-169`
`navRailPrimary()` lists Quests, Life, **Chronicle**, Memory, Skills,
Settings. (knowledge.md:304-305 already mentions the Chronicle destination, so
today the file contradicts itself.)

(d) Roadmap wording to KEEP: knowledge.md:72-73 says "Future gateways
(messengers) follow the same rule." The messenger endpoint
(`POST /api/messenger/turn`) is a consent-gated, token-authed bridge API — a
settled decision keeps messenger *products* as roadmap. Do NOT document it as
a shipped messenger surface; mention it only as an API endpoint/turn caller.

### 3. .tours/15-sovereign-export.tour — day is no longer deferred

Steps 15.1 / 15.2 / 15.4 (file lines 10 / 16 / 28) still say:

- 15.1 (line 10): "Notice that `day` and `task` are absent — they belong in
  the JD design (`50-59 Journal`, `60-69 Tasks`) but are listed in
  `deferredTypes` instead." — and its snippet omits the `"day"` map entry.
- 15.2 (line 16): "Skips `deferredTypes` (`day`, `task`) entirely — they are
  never opened, never read, never written."
- 15.4 (line 28): "**Type deferral**: `day` and `task` are in `deferredTypes`
  and skipped before any read attempt."

Reality (excerpt in §1 above): `jdFolder` maps `day` to
`50-59 Journal/51 Days`; `deferredTypes` holds only `task`. Day was
un-deferred in plan 225 behind the leak test `TestDayJournalExportLeakTest`
(`internal/export/mirror_test.go:289`, referenced at `mirror_test.go:156`).
The rule sentence "**Do not add a type here without simultaneously removing it
from `deferredTypes` and landing a leak test.**" must be KEPT.

### 4. .tours/10-the-cli-api.tour — command count is 19, code has 20

Tour description (file line 4): "the full command roster (19 commands across
chat, tasks, knowledge, life, export, and infra)". Step 10.2 (file line 16):
its snippet ends `exportCmd(app), doctorCmd(app), seedCmd(app),` (no
`restoreCmd`), the prose says "`Register` mounts 19 top-level commands", and
the "Infra / ops" group omits `restore`.

Reality — `internal/cli/cli.go:53-76` registers **20** commands;
`cli.go:72` is `restoreCmd(app),` between `exportCmd` and `doctorCmd`.

### 5. .tours/00-orientation.tour — stale launcher snippet + "two surfaces" + 19 commands

Step 0.2 (file line ~14, anchored at `main.go:47`) shows:

```go
if launch.IsLauncherInvocation(os.Args[1:]) {
    _ = launch.IsFirstRun(launch.DataDir())
    ...
}
```

— stale: `main.go:68` now assigns `isFirstRun = launch.IsFirstRun(...)`, and
the `RunningInstance`/`WriteInstanceLock` single-instance guard (plan 232)
surrounds it. The anchor `main.go:47` now lands inside a comment block; the
launcher branch starts at `main.go:53`.

Step 0.3 (file line ~20) is anchored at `main.go:75`, which is now
`url := "http://" + addr + "/"` *inside* the launcher block — it should anchor
`app := pocketbase.New()` at `main.go:97`.

Step 0.10 (file line ~64) says `internal/web` is "one of two surfaces over
`internal/turn`" and lists the CLI commands ending "`export`, `doctor`,
`seed`" (no `restore`). Reality: `turn.Run` now has a third caller —
`internal/web/messenger.go:49` `messengerTurn` (`POST /api/messenger/turn`),
a consent-gated bridge API, not a product surface.

### 6. .tours/19-bootstrapping.tour — false "discarded result" prose + four shifted anchors

Step 19.1 (file line ~10, anchored `main.go:47`) says:

```
`IsFirstRun` is called here but its result is discarded (`_`): the seam exists
for Phase 2 onboarding without gating the browser-open, which happens on every
no-args boot.
```

— false since plan 230 (`main.go:68` assigns it; `main.go:111` stashes it for
the banner). Its snippet is the same stale one as tour 00 step 0.2.

Shifted anchors (all `main.go`; the launcher block grew by ~25 lines):

| Tour step | tour file line | anchored today | should anchor | symbol at target |
|---|---|---|---|---|
| 19.3 | ~20 | `main.go:75` | `main.go:97` | `app := pocketbase.New()` |
| 19.5 | ~32 | `main.go:126` | `main.go:151` | `func registerKronkEngine(` |
| 19.6 | ~38 | `main.go:136` | `main.go:161` | `func scheduleJob(` |
| 19.7 | ~44 | `main.go:226` | `main.go:251` | `func registerSearchIndex(` |

(Verified at 077318a; **re-grep on the day you execute** — line numbers move.)
Two more staleness spots in 19.5's prose/snippet: "The paired `OnTerminate`
hook (line 101)" — it is at `main.go:126` now — and the snippet calls
`resolveProcessor(app)` while the code (`main.go:152`) calls
`turn.ResolveProcessor(app)`.

### 7. .tours/07-the-web-gateway.tour — missing busy guard + stale nudge filter

Step 7.3 (file line ~22, anchored `internal/web/chat.go:28`): the snippet and
the 6-item walkthrough go straight from `readChatMessage` to building the
chatStream — the `turn.TryBegin` busy-guard that is now the handler's first
act (`chat.go:40-48`, excerpt in §2a above) is absent.

Step 7.7 (file line ~44, anchored `internal/web/tasks.go:186`): the snippet
shows the old filter `"origin != '' && created > {:since}"` and the prose
claims "Chat turns never flow through here, so polling cannot duplicate
them." Reality — `internal/web/tasks.go:192-193`:

```go
	recs, err := h.app.FindRecordsByFilter("messages",
		"(origin = 'nudge' || origin = 'briefing') && created > {:since}", "@rowid", 20, 0,
```

with the explaining comment at `tasks.go:184-186`: "Runtime artifacts the
honesty check writes during a turn (origin \"uncommitted\"/\"check\") are
deliberately excluded — the streamed turn already renders those, so polling
must not re-append them." (the plan-212 duplicate-artifact fix).

### 8. docs/first-run-design.md — "not built" items that shipped

The doc is cited as a current reference by `internal/launch/launch.go:90`
("acceptable for a localhost launcher (see docs/first-run-design.md)"), so its
"not built" claims mislead. Three spots:

- Lines 49-54: "**First-run detection (recommendation, not built here):** ...
  This spike does not implement the stat — it is noted for Phase 2 so
  onboarding has a trigger." — the stat shipped in plan 193
  (`launch.IsFirstRun`), the banner consuming it in plan 230.
- Lines 140-142: "**Single-instance guard** — a second double-click should
  focus the running instance (or refuse), not start a second server on a
  second random port. Not built; needs a lockfile or a fixed-port probe." —
  built in plan 232 (`launch.RunningInstance` + `launch.WriteInstanceLock`:
  lock file + TCP liveness probe, fail-open on stale locks).
- Lines 146-151 (Phase 1 hardening list): "a single-instance guard; a
  **stable default port with fallback** (try a fixed port like 8090 first ...)
  ...; the first-run stat from Q1." — all shipped: guard (plan 232), stable
  default port 8099 + first-run stat (plan 193; `launch.DefaultPort` at
  `internal/launch/launch.go:83`).

The historical reasoning must stay intact — annotate, do not rewrite.

### Conventions that apply here

- `.tours/` are maintained artifacts. `tours_test.go` (repo root) fails the
  suite when a tour references a missing file or an out-of-range line — it
  tolerates in-range-but-wrong anchors, so after editing you must also check
  each edited step's anchor lands on its named symbol by eye.
- `.tour` files are JSON — every `description` is one JSON string. After
  editing, validate with `python3 -m json.tool` (verification in each step).
- `internal/self/knowledge.md` is the binary's self-description; it is data
  for `internal/self`, not rendered as Go — no build implications.
- No Go code changes anywhere in this plan. Conventional-commit subjects
  (`docs:` here throughout).

## Commands you will need

| Purpose | Command | Expected on success |
|---|---|---|
| Full test gate (merge gate) | `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` | exit 0, all packages ok |
| Tours lint | `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` | `ok` |
| Vet | `go vet ./...` | exit 0 |
| Format | `gofmt -l .` | empty output |
| Staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | no output, exit 0 |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| JSON validity of a tour | `python3 -m json.tool < .tours/<file>.tour > /dev/null` | exit 0, no output |

(The host `/tmp` is a small tmpfs; the Go linker OOMs there — always set
`TMPDIR=$HOME/.cache/go-tmp` for `go test`. Use `-count=1`: the uncached form
is the gate.)

## Scope

**In scope** (the only files you may modify):

- `AGENTS.md` (one bullet)
- `internal/self/knowledge.md`
- `.tours/00-orientation.tour`
- `.tours/07-the-web-gateway.tour`
- `.tours/10-the-cli-api.tour`
- `.tours/15-sovereign-export.tour`
- `.tours/19-bootstrapping.tour`
- `docs/first-run-design.md`

**Out of scope** (do NOT touch, even though they look related):

- `internal/web/messenger.go`, `internal/web/messenger_test.go`,
  `internal/cli/chat.go` — plan 236 owns the guard-comment corrections there.
- `README.md` and `DESIGN.md` — spot-check both for the same vault-mirror
  drift (`README.md:143` was verified already truthful); if you find NEW
  drift there, report it in your final summary — do not fix it.
- `plans/README.md` — the reviewer maintains the index.
- Any `.go` file, `graphify-out/` (do not run `graphify update` — the
  reviewer regenerates the graph at merge; executor worktrees pollute it).
- Tours other than the five listed.

## Git workflow

- You run in an isolated git worktree branched from `origin/main`.
- Branch: `advisor/252-docs-truth-sync-post-230-234`
- Conventional-commit subjects (`feat`/`fix`/`docs`/`refactor`/`style`/
  `test`/`chore`) — everything here is `docs:`.
- Commit per logical unit (one commit per step below) with **explicit
  pathspecs** (`git add AGENTS.md`, never `git add -A` — the main checkout is
  shared by parallel agents; stage only your own files).
- **NEVER push.** The reviewer merges.

## Steps

### Step 0: Confirm dependency and baseline

Run the drift check from the header blockquote. Then confirm plan 236 landed
(it corrects the source-comment guard claims this plan's knowledge.md wording
must match):

**Verify**: `grep -c "never written concurrently from any surface" internal/web/messenger.go` → `0`
(If it prints `1`, plan 236 has not landed — STOP.)

**Verify**: `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → `ok` (green baseline before you touch tours).

### Step 1: AGENTS.md — the vault mirror shipped

In `AGENTS.md`, replace exactly the two lines quoted in Current-state §1
(lines 244-245 at 077318a) with:

```
- The Johnny Decimal Markdown vault mirror SHIPPED (plans 194/195/224/225):
  `balaur export` writes the one-way Markdown mirror with a local git
  history, `--encrypt` wraps it in a passphrase-protected archive
  (scrypt + AES-256-GCM), and `balaur restore` decrypts an archive back to
  a readable tree. Only the `task` type stays deferred pending its content
  redaction pass; `day` journal bodies export behind a leak test. Do not
  claim task export in user-facing copy until that redaction pass lands.
```

Touch nothing else in the file (plan 236 may have added its own bullet
nearby — leave it alone).

**Verify**: `grep -c "vault mirror (one-way export + git) is" AGENTS.md` → `0`
**Verify**: `grep -c "vault mirror SHIPPED" AGENTS.md` → `1`
**Commit**: `git add AGENTS.md && git commit -m "docs(agents): vault mirror shipped — truth-sync the known-limitations bullet"`

### Step 2: internal/self/knowledge.md — guard, launcher, nav lists

Four edits:

**(a)** In the `internal/turn` bullet (Current-state §2a, lines 74-77 at
077318a), append after "It also resolves the active model choice.":

```
  A process-wide in-flight guard (turn.TryBegin) admits one turn at a
  time on the master conversation across the serve process's callers —
  the web chat handler and the consent-gated messenger API endpoint
  (and any other in-process caller). A busy web turn gets the "One
  message is still being answered" toast; a busy messenger POST gets
  HTTP 429. The guard is per-process: a separate `balaur chat` process
  is not covered by it.
```

Do NOT describe the messenger endpoint as a messenger product surface
anywhere — messenger products stay roadmap (keep line 72-73's "Future
gateways (messengers) follow the same rule." as is).

**(b)** Replace the two launcher bullets (Current-state §2b, lines 420-426 at
077318a) with:

```
- main.go — wire-up: PocketBase app, migrations, CLI, routes, crons.
  A bare `balaur` (no args) is the no-terminal launcher: it boots a
  loopback UI on the XDG data dir, prefers a stable default port
  (8099, falling back to a free port if taken), and opens the browser.
  A single-instance guard (lock file + TCP liveness probe) makes a
  second bare launch open the already-running instance and exit instead
  of starting a second server; stale locks fail open and the launch
  proceeds. On the first boot of a fresh data dir, a dismissible
  onboarding banner routes the owner to model setup — it never gates
  chat or the browser-open.
- internal/launch — the no-args loopback launcher helpers (XDG data dir,
  stable default port + free-port fallback, single-instance lock +
  liveness probe, first-run stat, browser-open); fires only on a bare
  argv and never constructs a non-loopback address
```

**(c)** Line 262: change `(Quests, Life, Memory, Skills, Settings)` to
`(Quests, Life, Chronicle, Memory, Skills, Settings)`.
Lines 268-269: change

```
surfaces fire GET /ui/show/{type}; the full destination set is Quests, Life,
Memory, Review, Skills, and the three settings sections (Profile, Models, Heads).
```

to

```
surfaces fire GET /ui/show/{type}; the full destination set is Quests, Life,
Chronicle, Memory, Review, Skills, Graph, and the three settings sections
(Profile, Models, Heads).
```

**Verify**: `grep -c "Chronicle, Memory" internal/self/knowledge.md` → `2`
**Verify**: `grep -c "HTTP 429" internal/self/knowledge.md` → `1`
**Verify**: `grep -c "single-instance" internal/self/knowledge.md` → at least `1`
**Verify**: `grep -c "per-process" internal/self/knowledge.md` → `1`
**Commit**: `git add internal/self/knowledge.md && git commit -m "docs(self): turn guard, launcher guard + onboarding banner, nav destinations"`

### Step 3: .tours/15-sovereign-export.tour — only task is deferred

Edit the three descriptions (steps 15.1 / 15.2 / 15.4; file lines 10/16/28):

- **15.1**: in the snippet, add `\n    \"day\":    \"50-59 Journal/51 Days\",`
  after the `"place"` entry (match `export.go:65`). Replace the "Notice that
  `day` and `task` are absent…" sentence pair with: "Notice that `task` is
  absent — it belongs in the JD design (`60-69 Tasks`) but is listed in
  `deferredTypes` instead: its content needs its own redaction pass before it
  is safe to export. `day` was un-deferred in plan 225: it exports the
  owner-authored journal body behind a dedicated leak test (recap/transcript
  text lives in the separate `summaries` collection and never reaches the
  node body)." KEEP the bolded rule sentence "**Do not add a type here without
  simultaneously removing it from `deferredTypes` and landing a leak test.**"
  and the closing YAGNI/KISS sentence.
- **15.2**: change item 3 from "Skips `deferredTypes` (`day`, `task`) entirely
  — they are never opened, never read, never written." to "Skips
  `deferredTypes` (now only `task` — `day` was un-deferred in plan 225 behind
  a leak test) entirely — a deferred type is never opened, never read, never
  written."
- **15.4**: change the "**Type deferral**" bullet to: "**Type deferral**:
  `task` is in `deferredTypes` and skipped before any read attempt — its body
  content needs its own redaction pass. `day` graduated out of the defer list
  in plan 225: its journal body exports behind the leak test
  `TestDayJournalExportLeakTest` (`internal/export/mirror_test.go`)."
  Adjust the following "Their content…" wording accordingly.

Anchors (export.go:59, :109, :223, :1; encrypt.go:57; mirror_test.go:229) are
still correct — do not change them, but eyeball each against the live file.

**Verify**: `python3 -m json.tool < .tours/15-sovereign-export.tour > /dev/null` → exit 0
**Verify**: `grep -c '`day`, `task`' .tours/15-sovereign-export.tour` → `0`
**Verify**: `grep -c "TestDayJournalExportLeakTest" .tours/15-sovereign-export.tour` → `1`
**Commit**: `git add .tours/15-sovereign-export.tour && git commit -m "docs(tours): export tour — day exports behind leak test, only task deferred"`

### Step 4: .tours/10-the-cli-api.tour — 20 commands, restore in the roster

- Tour `description` (file line 4): "19 commands across chat, tasks,
  knowledge, life, export, and infra" → "20 commands across chat, tasks,
  knowledge, life, export/restore, and infra".
- Step 10.2: in the snippet, add `\n        restoreCmd(app),` between
  `exportCmd(app),` and `doctorCmd(app),` (match `cli.go:72`). Change
  "`Register` mounts 19 top-level commands" → "…20 top-level commands".
  Change the "Infra / ops" group to: "`export`, `restore`, `doctor`, `seed`,
  `verify`".

**Verify**: `python3 -m json.tool < .tours/10-the-cli-api.tour > /dev/null` → exit 0
**Verify**: `grep -c "19 top-level commands\|19 commands" .tours/10-the-cli-api.tour` → `0`
**Verify**: `grep -c "restoreCmd" .tours/10-the-cli-api.tour` → `1`
**Commit**: `git add .tours/10-the-cli-api.tour && git commit -m "docs(tours): CLI tour — 20 commands, add restore to roster and grouping"`

### Step 5: .tours/00-orientation.tour — launcher snippet, anchors, third caller, restore

- **Step 0.2** (file line ~14): repoint `"line": 47` → the current line of
  `if launch.IsLauncherInvocation(os.Args[1:]) {` in `main.go` (53 at
  077318a; re-grep: `grep -n "IsLauncherInvocation(os.Args" main.go`).
  Replace the snippet with the current shape:

  ```go
  if launch.IsLauncherInvocation(os.Args[1:]) {
      if addr, alive := launch.RunningInstance(launch.DataDir()); alive {
          _ = launch.OpenBrowser("http://" + addr + "/") // already running: open it, exit
          return
      }
      isFirstRun = launch.IsFirstRun(launch.DataDir())
      port, err := launch.SelectPort()
      addr := fmt.Sprintf("127.0.0.1:%d", port)
      _ = launch.WriteInstanceLock(launch.DataDir(), addr)
      os.Args = append(os.Args[:1], "serve", "--http", addr, "--dir", launch.DataDir())
      go launch.OpenAfterReady(addr)
  }
  ```

  Update the prose: keep the argv-rewrite explanation; extend the helper list
  to "`IsLauncherInvocation`, `RunningInstance` + `WriteInstanceLock` (the
  single-instance guard: lock file + TCP liveness probe, fail-open on stale
  locks — a second bare launch opens the running instance and exits),
  `IsFirstRun` (feeds the first-run onboarding banner via `app.Store()`),
  `SelectPort` (stable default 8099, free-port fallback), and
  `OpenAfterReady`".
- **Step 0.3** (file line ~20): repoint `"line": 75` → the current line of
  `app := pocketbase.New()` (97 at 077318a). Leave its abbreviated snippet
  as is except: add `se.App.Store().Set("balaur_first_run", isFirstRun)` as
  the first line inside the `OnServe` func to match `main.go:111`.
- **Step 0.10** (file line ~64): add `restore` to the command list (between
  `export` and `doctor`), and reword "one of two surfaces over
  `internal/turn`" to: "one of the two owner-facing gateways over
  `internal/turn` (`turn.Run` has a third caller: the consent-gated,
  token-authed `POST /api/messenger/turn` bridge endpoint in
  `internal/web/messenger.go` — an API for the owner's own bridge process,
  not a product surface; messenger products remain roadmap)".

**Verify**: `python3 -m json.tool < .tours/00-orientation.tour > /dev/null` → exit 0
**Verify**: `grep -c "result is discarded\|_ = launch.IsFirstRun" .tours/00-orientation.tour` → `0`
**Verify**: `grep -c "RunningInstance" .tours/00-orientation.tour` → at least `1`
**Verify**: `sed -n "$(grep -n 'IsLauncherInvocation(os.Args' main.go | head -1 | cut -d: -f1)p" main.go` shows the `if launch.IsLauncherInvocation` line, and that number equals step 0.2's `"line"` value.
**Commit**: `git add .tours/00-orientation.tour && git commit -m "docs(tours): orientation — launcher guard snippet, repoint main.go anchors, third turn caller, restore"`

### Step 6: .tours/19-bootstrapping.tour — false prose, snippet, four anchors

- **Step 19.1** (file line ~10): repoint `"line": 47` → the
  `IsLauncherInvocation` line (53 at 077318a). Replace the snippet with the
  same current-shape snippet as Step 5 above. Replace the "IsFirstRun is
  called here but its result is discarded (`_`)…" paragraph with: "Two later
  additions frame the rewrite. `launch.RunningInstance` (plan 232) is the
  **single-instance guard**: it reads the instance lock file and TCP-probes
  the recorded address; if a live instance answers, this second bare launch
  opens it in the browser and exits. It is FAIL-OPEN — any error, or a stale
  lock whose probe gets no answer, means proceed to a normal start.
  `launch.IsFirstRun` (plans 193/230) now feeds the **first-run onboarding
  banner**: the flag is stashed in `app.Store()` under `balaur_first_run` at
  OnServe, and the home handler renders a dismissible model-setup banner on a
  fresh box — it never gates the browser-open, which happens on every no-args
  boot." Keep the safety-rule paragraph ("`IsLauncherInvocation` returns true
  only when `len(args) == 0`…") unchanged.
- **Repoint the four shifted anchors** per the table in Current-state §6
  (19.3 → `pocketbase.New()` line; 19.5 → `func registerKronkEngine(` line;
  19.6 → `func scheduleJob(` line; 19.7 → `func registerSearchIndex(` line).
  Get the real lines on the day you execute:
  `grep -n "app := pocketbase.New()\|^func registerKronkEngine\|^func scheduleJob\|^func registerSearchIndex" main.go`
- **Step 19.5 prose/snippet**: change "The paired `OnTerminate` hook (line
  101)" to the current line of `app.OnTerminate().BindFunc` (126 at 077318a;
  `grep -n "OnTerminate" main.go`), and in the snippet change
  `resolveProcessor(app)` → `turn.ResolveProcessor(app)` to match
  `main.go:152`.
- **Step 19.3 snippet**: add `se.App.Store().Set("balaur_first_run", isFirstRun)`
  as the first line inside the `OnServe` func (matches `main.go:111`).

**Verify**: `python3 -m json.tool < .tours/19-bootstrapping.tour > /dev/null` → exit 0
**Verify**: `grep -c "result is discarded" .tours/19-bootstrapping.tour` → `0`
**Verify**: for each of the four repointed anchors, print the anchored line and
confirm it shows the named symbol, e.g.
`sed -n "97p" main.go` → `	app := pocketbase.New()` (use the numbers you set).
**Verify**: `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → `ok`
**Commit**: `git add .tours/19-bootstrapping.tour && git commit -m "docs(tours): bootstrapping — first-run flag is used, instance guard, repoint main.go anchors"`

### Step 7: .tours/07-the-web-gateway.tour — busy guard + nudge filter

- **Step 7.3** (file line ~22): in the snippet, insert after the
  `if msg == "" { return e.BadRequestError("empty message", nil) }` line:

  ```go
  end, ok := turn.TryBegin() // in-flight guard: one turn at a time
  if !ok {
      sse := datastar.NewSSE(e.Response, e.Request)
      emitToast(sse, "warn", "One message is still being answered — try again in a moment.")
      return nil
  }
  defer end()
  ```

  Insert a new item 2 into the numbered walkthrough: "Acquires the
  process-wide turn guard (`turn.TryBegin`, `internal/turn/guard.go`) before
  any medium setup — a busy turn gets only a warn toast over a minimal SSE
  connection (no user bubble, no `#chat` mutation); `defer end()` releases
  it" — and renumber the following items.
- **Step 7.7** (file line ~44): in the snippet, change the filter string to
  `"(origin = 'nudge' || origin = 'briefing') && created > {:since}"`.
  Replace "Chat turns never flow through here, so polling cannot duplicate
  them." with: "The filter is an explicit origin allowlist (`nudge`,
  `briefing`): chat turns (empty origin) never match, and the runtime
  artifacts the honesty check writes during a turn (origin
  `uncommitted`/`check`) are deliberately excluded because the streamed turn
  already renders them — re-appending them here was the plan-212
  duplicate-artifact bug." Also update the step's opening sentence
  ("Agent-initiated messages (origin != \"\")…") to name the two origins.

Anchors `chat.go:28` and `tasks.go:186` still land on the right
comment-above-function — leave them; eyeball both.

**Verify**: `python3 -m json.tool < .tours/07-the-web-gateway.tour > /dev/null` → exit 0
**Verify**: `grep -c "origin != ''" .tours/07-the-web-gateway.tour` → `0`
**Verify**: `grep -c "TryBegin" .tours/07-the-web-gateway.tour` → at least `1`
**Commit**: `git add .tours/07-the-web-gateway.tour && git commit -m "docs(tours): web gateway — busy guard in chat handler, nudge origin allowlist"`

### Step 8: docs/first-run-design.md — dated shipped-status annotations

Leave all historical reasoning intact; add annotations only.

- After the header blockquote (below line 7), add a dated status block (use
  the date you execute):

  ```
  > **Status (<YYYY-MM-DD>):** this is the plan-190 spike record. Several
  > items marked "Phase 2" / "not built" below have since shipped: the stable
  > default port 8099 and the first-run stat (plan 193), the first-run
  > onboarding banner consuming that stat (plan 230), and the single-instance
  > guard (plan 232). Per-item annotations below mark what shipped; the
  > original reasoning is preserved unedited.
  ```

- After the first-run-detection paragraph (lines 49-54), append:
  `*Shipped since: the stat landed as `launch.IsFirstRun` (plan 193); plan 230 consumes it for the onboarding banner — still never gating the browser-open.*`
- After the single-instance-guard bullet (lines 140-142), append:
  `*Shipped since: plan 232 — `launch.WriteInstanceLock` + `launch.RunningInstance` (lock file + TCP liveness probe, fail-open on stale locks).*`
- After the Phase-1 hardening paragraph (lines 146-151), append:
  `*Shipped since: all four hardening items landed — stable default port 8099 + fallback and the first-run stat (plan 193), the friendlier stderr URL message (main.go), the single-instance guard (plan 232).*`

**Verify**: `grep -c "plan 232" docs/first-run-design.md` → at least `2`
**Verify**: `grep -c "Shipped since" docs/first-run-design.md` → `3`
**Commit**: `git add docs/first-run-design.md && git commit -m "docs: annotate first-run design note with shipped status (plans 193/230/232)"`

### Step 9: Final gates

Run, in order:

**Verify**: `gofmt -l .` → empty output (no Go files were touched)
**Verify**: `go vet ./...` → exit 0
**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0
**Verify**: `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → `ok`
**Verify**: `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0, all ok
**Verify**: `git status --short` → only the eight in-scope files appear in the
branch's commits; working tree clean.

Finally, the anchor eyeball pass required because `tours_test` tolerates
in-range-but-wrong lines: for EVERY tour step you edited or repointed, print
the anchored line (`sed -n '<line>p' <file>`) and confirm it shows the symbol
named in that step's title/snippet. State in your final report: "anchor
eyeball pass done — N anchors checked".

## Test plan

No new tests — this is a docs/tours truth-sync with zero behavior change.
The gates are:

- `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → `ok` after
  every tour edit (Steps 3-7) and at the end (this catches out-of-range
  anchors and missing files introduced by the edits).
- `python3 -m json.tool` on each edited `.tour` file → exit 0 (JSON validity;
  the tours test may not catch a malformed description string early).
- Full suite `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0
  (guards against any accidental Go-file touch and is the merge gate).
- The manual anchor eyeball pass in Step 9 (tours_test cannot verify anchor
  *correctness*, only range).

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` exits 0
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → `ok`
- [ ] `gofmt -l .` prints nothing; `go vet ./...` exits 0;
      `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `grep -c "roadmap, not shipped. Do not claim it in user-facing copy" AGENTS.md` → `0`
- [ ] `grep -c "Chronicle, Memory" internal/self/knowledge.md` → `2` and
      `grep -c "HTTP 429" internal/self/knowledge.md` → `1`
- [ ] `grep -c '`day`, `task`' .tours/15-sovereign-export.tour` → `0`
- [ ] `grep -rc "19 top-level commands" .tours/10-the-cli-api.tour` → `0`
- [ ] `grep -c "result is discarded" .tours/19-bootstrapping.tour` → `0`
- [ ] `grep -c "origin != ''" .tours/07-the-web-gateway.tour` → `0`
- [ ] `grep -c "Shipped since" docs/first-run-design.md` → `3`
- [ ] `git diff --name-only origin/main...HEAD` lists ONLY the eight in-scope
      files (no out-of-scope file modified)
- [ ] Anchor eyeball pass reported done for every edited/repointed tour step

## STOP conditions

Stop and report back (do not improvise) if:

- Plan 236 has not landed: `grep -c "never written concurrently from any surface" internal/web/messenger.go` → `1` (Step 0).
- Any cited line's current content does not match the excerpt quoted in
  "Current state" (e.g. AGENTS.md's vault-mirror bullet was already rewritten,
  `deferredTypes` no longer equals `{"task": true}`, the nudge filter changed
  again, or knowledge.md's lines 262/268-269 already mention Chronicle/Graph)
  — the drift means someone else fixed or moved the target.
- A `main.go` symbol you must anchor (`pocketbase.New()`,
  `registerKronkEngine`, `scheduleJob`, `registerSearchIndex`,
  `IsLauncherInvocation`) does not exist or `grep -n` finds it more than once
  as a definition.
- `TestTours` or `python3 -m json.tool` fails twice on the same tour file
  after a fix attempt.
- The fix appears to require touching an out-of-scope file (in particular:
  you feel the need to edit `internal/web/messenger.go` comments — that is
  plan 236's territory).
- You find the messenger endpoint documented anywhere as a shipped messenger
  *product* surface and the plan's wording seems to require matching that —
  it does not; messenger products are roadmap by settled decision. Report the
  conflict instead of escalating either claim.

## Maintenance notes

- **Recurring failure mode**: five same-day plans landed without the "docs in
  the same change" rules (AGENTS.md: "Self-knowledge is part of the change";
  ".tours/ is a maintained artifact"). Reviewers of future feature merges
  should grep `internal/self/knowledge.md` and `.tours/` for the touched
  symbols before merging — `tours_test` will not catch false prose.
- **What a reviewer should scrutinize here**: (1) the knowledge.md guard
  sentence must stay per-process-honest and must match the (post-236) source
  comments in `internal/web/chat.go` / `messenger.go`; (2) tour anchors — run
  the `sed -n '<line>p'` spot-checks yourself, since line numbers may have
  moved again between plan execution and merge; (3) no messenger-as-product
  claim crept into knowledge.md or tour 00.
- **Interactions**: a later dx plan (253) also edits AGENTS.md — merge this
  first. If `main.go` grows again, tours 00 and 19 anchors shift again; the
  anchor table in Current-state §6 documents the symbol-to-step mapping so
  the next sync is mechanical.
- **Deferred (reported, not fixed)**: if the README.md/DESIGN.md spot-check
  (Scope section) surfaces new drift, it goes into the executor's report for
  a follow-up docs plan — deliberately out of this plan's scope to keep the
  diff reviewable.
- The graph (`graphify-out/`) is regenerated by the reviewer at merge;
  executors must not run `graphify update` in the worktree.
