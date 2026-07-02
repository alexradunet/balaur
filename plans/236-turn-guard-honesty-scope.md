# Plan 236: Scope the turn-guard claim honestly to one process (and record the CLI gap as a known limitation)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, do **NOT** update `plans/README.md` —
> the reviewer who dispatched you maintains the index. Do not touch
> `plans/README.md` at all.
>
> **Drift check (run first)**:
> `git diff --stat 077318a..HEAD -- internal/cli/chat.go internal/web/chat.go internal/web/messenger.go internal/web/messenger_test.go AGENTS.md`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `077318a`, 2026-07-01

## Why this matters

`internal/turn/guard.go` implements the "one turn at a time on the master
conversation" guard as a package-level `sync.Mutex` — by construction it only
serializes turns **within a single OS process** (guard.go's own comment says
"process-wide", which is accurate). But three comments elsewhere claim more:
`internal/cli/chat.go` says "web + CLI + messenger share the same guard",
`internal/web/chat.go` says "exactly one turn runs on the master conversation
at a time (web + CLI + messenger)", and the header of
`internal/web/messenger.go` says the master conversation "is never written
concurrently from any surface". All three claims are **false across
processes**: `balaur chat` is a *separate process* that opens the same SQLite
data directory directly (the CLI's `run` wrapper in `internal/cli/cli.go`
applies pending migrations itself precisely because it runs outside `serve`;
SQLite WAL permits multi-process access). Its `turn.TryBegin()` takes a
*different* mutex in a different address space, so running `balaur chat` while
`balaur serve` is up can interleave two turns on the master conversation — the
exact race the guard was built to close. Nothing crashes today, but future work
will be built on this false invariant unless the comments and docs are
corrected. This plan is **honesty-only**: fix the comments, fix one stale
adjacent doc claim in the same header, and record the cross-process gap as a
known limitation in `AGENTS.md`. A real cross-process guard (a DB lease with a
TTL) is deliberately **not** built here.

## Current state

Relevant files and their roles:

- `internal/turn/guard.go` — the guard itself (honest; **not modified** by this
  plan, quoted only as evidence).
- `internal/cli/chat.go` — the `balaur chat` gateway; contains the overclaiming
  comment (lines 65–66).
- `internal/web/chat.go` — the web chat gateway; its guard comment block
  (lines 35–39) opens with the same overclaim (lines 35–36: "Cross-surface
  in-flight guard: exactly one turn runs on the master conversation at a time
  (web + CLI + messenger)").
- `internal/web/messenger.go` — `POST /api/messenger/turn` gateway; contains
  the overclaiming header paragraph (lines 29–31), an overclaiming call-site
  comment (lines 81–82), and a stale claim about how the token is set
  (lines 21–23).
- `internal/web/messenger_test.go` — comment at line 278 references a symbol
  (`messengerMu`) that no longer exists.
- `internal/web/messenger_settings.go` — proof the settings-UI "follow-up"
  already shipped (evidence only; **not modified**).
- `AGENTS.md` — "Known limitations & deferred work" section (heading at
  line 239) gains one bullet.

### The guard is process-local by construction

`internal/turn/guard.go:5-9`:

```go
// turnMu is the process-wide in-flight guard. TryBegin acquires it with
// TryLock (immediate "busy" — never blocking). One guard for v1's single
// master conversation; key by conversation id if multiple turn-bearing
// conversations are ever added.
var turnMu sync.Mutex
```

Note it honestly says "process-wide". `TryBegin` (guard.go:24) is called from
three gateways: web (`internal/web/chat.go:40`), messenger
(`internal/web/messenger.go:83`), and CLI (`internal/cli/chat.go:67`). Web and
messenger run inside the `balaur serve` process; the CLI does not.

### Overclaim 1 — `internal/cli/chat.go:65-71`

```go
		// Cross-surface in-flight guard: one turn at a time on the master
		// conversation (web + CLI + messenger share the same guard).
		end, ok := turn.TryBegin()
		if !ok {
			return nil, errors.New("busy: a turn is already in progress")
		}
		defer end()
```

"web + CLI + messenger share the same guard" is false: `balaur chat` is its own
process. Evidence that the CLI runs outside the server —
`internal/cli/cli.go:78-79`:

```go
// run wraps a command body with the CLI contract. Pending migrations apply
// first: PocketBase runs app migrations in serve (apis.Serve) or `migrate`,
```

### Overclaim 2 — `internal/web/messenger.go:29-31` (header) and `:81-82` (call site)

Header:

```go
// In-flight guard: turn.TryBegin (internal/turn) provides a cross-surface
// guard — web, CLI, and messenger all acquire it before running a turn so the
// master conversation is never written concurrently from any surface.
```

Call site (`internal/web/messenger.go:81-83`):

```go
	// 4. Cross-surface in-flight guard — one turn at a time on the master
	//    conversation (shared with the web and CLI gateways via turn.TryBegin).
	end, ok := turn.TryBegin()
```

"never written concurrently from any surface" and "shared with the … CLI" are
false across processes for the same reason.

### Overclaim 3 — `internal/web/chat.go:35-39` (guard comment; lines 35–36 overclaim)

```go
	// Cross-surface in-flight guard: exactly one turn runs on the master
	// conversation at a time (web + CLI + messenger). Acquire before any
	// medium setup so a busy response never paints a user bubble or opens
	// the stream. TryLock is intentional — at v1 a second concurrent turn
	// is always a race, never a work queue.
	end, ok := turn.TryBegin()
```

"Cross-surface … (web + CLI + messenger)" is the same cross-process overclaim:
the CLI is a separate process. Only lines 35–36 are wrong; the "Acquire before
any medium setup…" and TryLock rationale (lines 36–39) are accurate and must be
kept.

### Staleness 4 — `internal/web/messenger.go:20-23` (same header block)

```go
//  2. Consent-gated / fail-closed — DISABLED unless owner_settings key
//     "messenger_token" is non-empty. No token → 403, no turn run. The owner
//     sets the token via the PocketBase admin engine room; a settings-UI
//     toggle is a natural follow-up.
```

The "natural follow-up" already shipped: `POST /ui/settings/messenger-token` is
handled by `saveMessengerToken` in `internal/web/messenger_settings.go:16` and
wired in `internal/web/web.go:227`
(`se.Router.POST("/ui/settings/messenger-token", h.saveMessengerToken)`); the
owner control renders in Settings → Capabilities
(`MessengerGatewaySection` in
`internal/feature/settingscards/settingsfocus_capabilities.go:143-145`).

### Dead symbol reference 5 — `internal/web/messenger_test.go:278`

```go
	// Wait until the handler has entered ChatStream (and thus holds messengerMu).
	<-bc.started
```

`messengerMu` no longer exists anywhere in the repo (it was replaced by
`turn.TryBegin`). The synchronization itself is correct — only the comment is
stale. Do **not** change any test code, only this comment.

### AGENTS.md target — heading at `AGENTS.md:239`

```markdown
## Known limitations & deferred work
```

The section is a flat bullet list (first bullet: "Multi-human multi-user is
FUTURE work…"). One new bullet is added at the **end** of the list, before the
`## Safety` heading.

### Conventions that apply here

- This is a comments/docs-only change: no behavior, no new tests. The full
  suite must stay green.
- "Comments explain non-obvious intent, trade-offs, or constraints — never
  narrate what the code already says" (AGENTS.md, Coding style). Keep the
  rewritten comments tight.
- `gofmt` is law; a PostToolUse hook may format edited Go files, but verify
  with `gofmt -l .` regardless.
- `.tours/` are maintained artifacts: `tours_test.go` fails when a tour
  references a missing file or out-of-range line.
  `.tours/10-the-cli-api.tour` anchors `internal/cli/chat.go` at **line 45**
  (the `chatCmd` function); your edit is at lines 65–66, *below* the anchor.
  `.tours/07-the-web-gateway.tour` anchors `internal/web/chat.go` at
  **line 28**; your edit is at lines 35–39, also below the anchor. Both anchors
  are unaffected even if the comments change line count — but you must still
  run the tours test (anchored files changed) and it must pass without touching
  any `.tours/` file.
- `internal/self/knowledge.md` does **not** need updating: it contains no
  mention of the turn guard, `TryBegin`, or any cross-surface serialization
  claim (verified via `grep -n "guard\|in-flight\|TryBegin\|one turn"
  internal/self/knowledge.md` → no matches at the planned-at commit), and this
  change alters no user-visible architecture or capability.

## Commands you will need

Run all commands from the repo root (the worktree root).

| Purpose | Command | Expected on success |
|---|---|---|
| Full test gate (merge gate) | `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` | exit 0, all packages `ok` |
| Targeted tests | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run TestMessenger -count=1` | exit 0 |
| Vet | `go vet ./...` | exit 0, no output |
| Format | `gofmt -l .` | empty output |
| Staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | no output, exit 0 |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Tours lint | `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` | `ok` |

(The host `/tmp` is a small tmpfs; the Go linker OOMs there — always set
`TMPDIR=$HOME/.cache/go-tmp` for test commands.)

## Scope

**In scope** (the only files you may modify):

- `internal/cli/chat.go` (comment at lines 65–66 only)
- `internal/web/chat.go` (comment lines 35–36 only — the first two lines of the
  guard comment block at 35–39)
- `internal/web/messenger.go` (header comment lines 20–31 and call-site comment
  lines 81–82 only)
- `internal/web/messenger_test.go` (comment at line 278 only — no code)
- `AGENTS.md` (one new bullet in "Known limitations & deferred work")

**Out of scope** (do NOT touch, even though they look related):

- `internal/turn/guard.go` — its wording is already accurate ("process-wide");
  no code or comment change there.
- Any actual cross-process locking (DB lease, flock, PID file, etc.) — the fix
  is documentation honesty, not a new mechanism.
- `internal/web/messenger_settings.go`, `internal/web/web.go`,
  `internal/feature/settingscards/*` — evidence only.
- `plans/README.md` — the reviewer maintains the plan index; never edit it.
- `.tours/*.tour` — the anchors into `internal/cli/chat.go` (line 45) and
  `internal/web/chat.go` (line 28) are above the edited lines and stay valid;
  if the tours test fails, that is a STOP condition, not a license to edit
  tours.
- `internal/self/knowledge.md` — no guard claim exists there (see Current
  state).

## Git workflow

- You run in an isolated git worktree branched from `origin/main`.
- Branch: `advisor/236-turn-guard-honesty-scope`.
- Conventional-commit subjects (`feat`/`fix`/`docs`/`refactor`/`style`/`test`/`chore`).
  This change is a single logical unit; one commit is appropriate, e.g.:
  `docs(turn): scope the in-flight-guard claim to one process; record CLI gap`
- Commit with **explicit pathspecs** (the main checkout is shared by parallel
  agents — stage only your own files):
  `git add internal/cli/chat.go internal/web/chat.go internal/web/messenger.go internal/web/messenger_test.go AGENTS.md`
- **NEVER push.** The reviewer merges.

## Steps

### Step 1: Fix the CLI guard comment in `internal/cli/chat.go`

Replace the two comment lines at `internal/cli/chat.go:65-66`:

```go
		// Cross-surface in-flight guard: one turn at a time on the master
		// conversation (web + CLI + messenger share the same guard).
```

with an honest version. Target shape (adjust wording freely, but it must state
all three facts — per-process scope, the live-server gap, and why the acquire
is kept):

```go
		// Turn in-flight guard — per-process only. `balaur chat` is its own
		// process, so when a server is running on the same data dir this does
		// NOT serialize against the server's turns (known limitation — see
		// AGENTS.md "Known limitations & deferred work"). The acquire is kept
		// for symmetry with the other gateways and for any future in-process
		// caller; cobra runs one command per process, so it never contends
		// today.
```

Do not change the `turn.TryBegin()` call, the error, or anything else in the
file.

**Verify**:
`grep -n "web + CLI + messenger share the same guard" internal/cli/chat.go`
→ no matches (exit 1).
`grep -n "per-process" internal/cli/chat.go` → one match in the new comment.

### Step 2: Fix the web guard comment in `internal/web/chat.go`

Replace only the first two lines of the guard comment
(`internal/web/chat.go:35-36`):

```go
	// Cross-surface in-flight guard: exactly one turn runs on the master
	// conversation at a time (web + CLI + messenger). Acquire before any
```

with:

```go
	// In-flight guard: exactly one turn runs on the master conversation at
	// a time within this process (web + messenger; a separate `balaur chat`
	// process is NOT serialized against this server — see AGENTS.md "Known
	// limitations & deferred work"). Acquire before any
```

The replacement must end with "Acquire before any" so it flows into the
untouched line 37 ("medium setup so a busy response…"). Keep lines 37–39 (the
medium-setup ordering and the TryLock rationale) exactly as they are; do not
change `turn.TryBegin()`, the busy toast, or anything else in the file. The
one tour anchor into this file (`.tours/07-the-web-gateway.tour`, line 28) is
above this comment and is unaffected by the line-count change.

**Verify**:
`grep -n "web + CLI + messenger" internal/web/chat.go` → no matches (exit 1).
`grep -n "within this process" internal/web/chat.go` → one match in the new
comment.
`grep -n "never a work queue" internal/web/chat.go` → one match (the tail of
the block was preserved).

### Step 3: Fix the messenger header and call-site comments in `internal/web/messenger.go`

Three edits in one file, comments only:

**3a — header, constraint 2 (lines 20–23).** Replace:

```go
//  2. Consent-gated / fail-closed — DISABLED unless owner_settings key
//     "messenger_token" is non-empty. No token → 403, no turn run. The owner
//     sets the token via the PocketBase admin engine room; a settings-UI
//     toggle is a natural follow-up.
```

with (keep constraint numbering and the fail-closed statement intact):

```go
//  2. Consent-gated / fail-closed — DISABLED unless owner_settings key
//     "messenger_token" is non-empty. No token → 403, no turn run. The owner
//     sets the token in Settings → Capabilities (POST
//     /ui/settings/messenger-token, messenger_settings.go); the PocketBase
//     admin engine room remains a fallback.
```

**3b — header, in-flight-guard paragraph (lines 29–31).** Replace:

```go
// In-flight guard: turn.TryBegin (internal/turn) provides a cross-surface
// guard — web, CLI, and messenger all acquire it before running a turn so the
// master conversation is never written concurrently from any surface.
```

with:

```go
// In-flight guard: turn.TryBegin (internal/turn) is a process-wide mutex —
// within the serve process, web and messenger turns are serialized. It does
// NOT reach across processes: a separate `balaur chat` process on the same
// data dir takes its own copy of the mutex and is not serialized against a
// running server (known limitation — see AGENTS.md).
```

**3c — call-site comment (lines 81–82).** Replace:

```go
	// 4. Cross-surface in-flight guard — one turn at a time on the master
	//    conversation (shared with the web and CLI gateways via turn.TryBegin).
```

with:

```go
	// 4. In-flight guard — one turn at a time on the master conversation
	//    within this process (shared with the web gateway via turn.TryBegin).
```

**Verify**:
`grep -n "never written concurrently from any surface" internal/web/messenger.go`
→ no matches (exit 1).
`grep -n "natural follow-up" internal/web/messenger.go`
→ no matches (exit 1). (The full sentence spans a line break in the live file —
line 22 ends "…a settings-UI", line 23 starts "toggle is a natural follow-up."
— so grep only for the single-line fragment "natural follow-up", which exists
today at line 23 and must be gone after edit 3a.)
`grep -n "shared with the web and CLI gateways" internal/web/messenger.go`
→ no matches (exit 1).
`grep -cn "process" internal/web/messenger.go` → at least 2 (new wording present).

### Step 4: Fix the dead-symbol comment in `internal/web/messenger_test.go`

At `internal/web/messenger_test.go:278`, replace:

```go
	// Wait until the handler has entered ChatStream (and thus holds messengerMu).
```

with:

```go
	// Wait until the handler has entered ChatStream (and thus holds the turn
	// in-flight guard, turn.TryBegin).
```

Change nothing else in the test.

**Verify**: `grep -rn "messengerMu" internal/` → no matches (exit 1).

### Step 5: Record the cross-process gap in `AGENTS.md`

In the `## Known limitations & deferred work` section (heading at
`AGENTS.md:239`), append one bullet at the **end of the bullet list** (after
the last existing bullet — the one about the two SQLite engines — and before
the `## Safety` heading):

```markdown
- The turn in-flight guard (`internal/turn/guard.go`) is per-process: a
  package-level mutex serializing web + messenger turns inside the serve
  process. A separate `balaur chat` process on the same data dir is NOT
  serialized against a running server — two turns can interleave on the master
  conversation. If cross-process serialization is ever needed, the deferred
  design is a DB lease: an `owner_settings` marker with a TTL, acquired via an
  atomic retry-on-conflict write — not another in-memory mutex.
```

**Verify**:
`grep -n "turn in-flight guard" AGENTS.md` → one match inside the
"Known limitations & deferred work" section.
`awk '/## Known limitations/,/## Safety/' AGENTS.md | grep -c "per-process"`
→ `1`.

### Step 6: Run the full gate

Run, in order:

1. `gofmt -l .` → empty output.
2. `go vet ./...` → exit 0.
3. `CGO_ENABLED=0 go build ./...` → exit 0.
4. `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output, exit 0.
5. `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → `ok`
   (two anchored files changed: `internal/cli/chat.go` — anchor at line 45 —
   and `internal/web/chat.go` — anchor at line 28; both anchors are above the
   edits and must still resolve).
6. `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0, all `ok`.

**Verify**: all six commands succeed with the stated output.

### Step 7: Commit

```
git add internal/cli/chat.go internal/web/chat.go internal/web/messenger.go internal/web/messenger_test.go AGENTS.md
git commit -m "docs(turn): scope the in-flight-guard claim to one process; record CLI gap"
```

**Verify**: `git status --porcelain` → empty (nothing unstaged, nothing
untracked that you created); `git show --stat HEAD` lists exactly the five
in-scope files.

## Test plan

- **No new tests.** This change touches only comments and Markdown; the
  existing suites are the regression net:
  - `internal/turn/guard_test.go` (TryBegin semantics) must stay green.
  - `internal/web/messenger_test.go` (including the in-flight 429 test whose
    comment Step 4 edits) must stay green — Step 4 must not alter the
    `<-bc.started` synchronization or any assertion.
  - `TestTours` must stay green (Step 6.5).
- Verification: `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -rn "web + CLI + messenger" internal/` → no matches (covers both
      `internal/cli/chat.go:66` and `internal/web/chat.go:36`)
- [ ] `grep -rn "never written concurrently from any surface" internal/` → no matches
- [ ] `grep -rn "shared with the web and CLI gateways" internal/` → no matches
- [ ] `grep -rn "natural follow-up" internal/` → no matches (single-line
      fragment; the full sentence spans a line break in the pre-edit file, and
      today's only hit is `internal/web/messenger.go:23`)
- [ ] `grep -rn "messengerMu" internal/` → no matches
- [ ] `awk '/## Known limitations/,/## Safety/' AGENTS.md | grep -c "per-process"` → `1`
- [ ] `gofmt -l .` → empty; `go vet ./...` → exit 0; `CGO_ENABLED=0 go build ./...` → exit 0
- [ ] `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output, exit 0
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0
- [ ] `git show --stat HEAD` (and `git status --porcelain` → empty) shows changes to exactly:
      `internal/cli/chat.go`, `internal/web/chat.go`, `internal/web/messenger.go`,
      `internal/web/messenger_test.go`, `AGENTS.md` — and nothing else
      (in particular NOT `plans/README.md`, NOT `internal/turn/guard.go`,
      NOT any `.tours/` file)

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows any in-scope file changed since `077318a` AND the
  quoted excerpts in "Current state" no longer match the live lines (in
  particular: if `internal/turn/guard.go` no longer uses a package-level
  `sync.Mutex`, or if someone has already built a cross-process lease, this
  plan's premise is gone).
- `internal/cli/chat.go:65-66`, `internal/web/chat.go:35-36`, or
  `internal/web/messenger.go:20-31` / `:81-82` do not contain the quoted
  comments (already reworded elsewhere).
- `TestTours` fails after your edits — do NOT edit `.tours/` files to make it
  pass; report instead (it means an anchor context in `internal/cli/chat.go`
  or `internal/web/chat.go` shifted in a way this plan did not predict).
- The full suite fails on anything, even a package you did not touch (the
  checkout is shared; a red suite may not be yours to fix — report it).
- Completing the change appears to require touching any out-of-scope file.

## Maintenance notes

- **What a reviewer should scrutinize**: that no Go statement changed — only
  comments and Markdown (`git diff` on the four `.go` files must show only
  `//` lines); and that the new wording nowhere claims cross-process
  serialization.
- **Future interaction**: if a messenger bridge or any owner tooling starts
  driving `balaur chat` against a live server routinely, the deferred DB-lease
  design in the new AGENTS.md bullet becomes real work — an `owner_settings`
  TTL marker acquired with an atomic retry-on-conflict write (per the repo
  rule: single-flight/check-then-act uses an atomic primitive, never
  read-modify-write). Keep the busy-response contract of `turn.TryBegin`
  (immediate reject, never block) when that lands.
- **Explicitly deferred**: any cross-process locking mechanism; keying the
  guard by conversation id (guard.go already documents that as the multi-
  conversation follow-up); rows in the plans ledger for earlier plans that
  repeat the overclaim are the advisor's to amend, not this plan's.
