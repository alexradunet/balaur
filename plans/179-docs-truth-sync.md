# Plan 179: Docs truth-sync — purge three stale, actively-misleading claims from README/AGENTS/knowledge.md

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 5dfb285..HEAD -- README.md AGENTS.md internal/self/knowledge.md`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: docs
- **Planned at**: commit `5dfb285`, 2026-06-25

> Note: the brief named `12a48bf` as the planning SHA, but the live tree at
> authoring time was `5dfb285`. The "Current state" excerpts below were pasted
> verbatim from `5dfb285`. Run the drift check against `5dfb285`.

## Why this matters

Three claims in the project's own docs are confirmed false today, and each one
actively misleads a reader who trusts them:

1. **Phantom env vars.** `README.md` documents `BALAUR_CHAT_MODEL` and
   `BALAUR_EMBED_MODEL` (env table + a worked `go run . serve` example), and
   `AGENTS.md` repeats the claim. **No Go code reads either variable** — a new
   owner who copies the example gets a server with no model and *no error*.
   The real path is the Models page / `llm_models` record (`store.SaveLocalModel`);
   `EnsureDefaultLLMConfig` seeds no model on purpose, and
   `internal/self/knowledge.md` already states "There is no manual GGUF-path
   entry." Deletion is the fix that makes the docs internally consistent.
2. **Stale collections.** `README.md`'s "Current shape" lists `memories` and
   `skills` as PocketBase collections. They were folded into the unified `nodes`
   collection (plans 164–168) and no longer exist. The real collection set is
   asserted by `migrations/schema_test.go` `TestSchemaBaseline`.
3. **Missing CLI commands.** The `README.md` CLI table and the
   `internal/self/knowledge.md` CLI list both omit the shipped `note` and
   `search` subcommands (both wired in `internal/cli/cli.go` `Register`).

Plus one deliberate-cost note to record: Balaur runs **two** SQLite engines on
purpose (PocketBase's `modernc.org/sqlite` + the FTS5 search sidecar's
`ncruces/go-sqlite3` wazero driver). That is intentional, not drift, and
belongs in AGENTS.md's "Known limitations & deferred work" so a future reader
doesn't "consolidate" it.

This is **docs-only**. Stale self-description makes Balaur lie about itself
(`AGENTS.md`: "A stale self-description makes Balaur lie about itself").

## Current state

### File roles

- `README.md` — public/dev README. Holds the stale env vars, the stale
  collection list, and the CLI table missing `note`/`search`.
- `AGENTS.md` — canonical agent instructions. Repeats the `BALAUR_CHAT_MODEL`
  claim in its "Known limitations" section; also the place to add the
  dual-SQLite note.
- `internal/self/knowledge.md` — the running binary's self-description,
  embedded via `//go:embed` (so editing it must still compile). Its CLI list
  omits `note`/`search`. **Its collection list is already correct** — do not
  touch it for collections.
- `migrations/schema_test.go` — source of truth for the real collection set
  (read-only reference, do NOT edit).
- `internal/cli/cli.go` — source of truth for the real CLI command set
  (read-only reference, do NOT edit).
- `internal/store/llm_settings.go` — confirms no model is seeded by default
  (read-only reference, do NOT edit).
- `internal/search/index.go` — header documents the FTS5 wazero driver
  (read-only reference for the dual-SQLite note; do NOT edit).

### (Reference) the real collection set — `migrations/schema_test.go:22-32`

```go
	// 1. All 14 app collections exist (+ built-in users). tasks is gone (plan 167).
	for _, name := range []string{
		"users", "heads", "conversations", "messages", "nodes", "edges",
		"audit_log", "summaries", "entries", "extensions",
		"llm_providers", "llm_models", "llm_settings", "owner_settings",
		"node_types",
	} {
		if _, err := app.FindCollectionByNameOrId(name); err != nil {
			t.Errorf("collection %q missing: %v", name, err)
		}
	}
```

And `schema_test.go:34-39` confirms the retired ones (`memories`, `skills`,
`tasks`, `boards`, `grants`) must NOT exist.

### (Reference) the real CLI command set — `internal/cli/cli.go:53-72`

```go
func Register(app core.App, root *cobra.Command) {
	root.AddCommand(
		chatCmd(app),
		taskCmd(app),
		memoryCmd(app),
		skillCmd(app),
		noteCmd(app),
		searchCmd(app),
		lifeCmd(app),
		journalCmd(app),
		dayCmd(app),
		recapCmd(app),
		historyCmd(app),
		auditCmd(app),
		verifyCmd(app),
		modelCmd(app),
		selfCmd(app),
		extCmd(app),
		doctorCmd(app),
		seedCmd(app),
```

So the shipped set includes `note` and `search`, which the docs omit.

### (Reference) no model is seeded by default — `internal/store/llm_settings.go:40-44`

```go
// EnsureDefaultLLMConfig makes sure the "Local model" provider exists. It seeds
// ... a fresh box never reports a model as ready.
func EnsureDefaultLLMConfig(app core.App, dataDir string) error {
```

(The real registration path is `SaveLocalModel(app, path, embedPath)` —
`internal/store/llm_settings.go:131-143` — driven by the Models page.)

### (Reference) the dual-SQLite cost — `internal/search/index.go:1-6`

```go
// Package search provides the FTS5 knowledge recall index — a rebuildable
// sidecar SQLite database backed by the ncruces/go-sqlite3 wazero driver
// (CGO-free, FTS5 included). It indexes all active node types (note, memory,
// skill, journal, and typed objects), keyed by kind. The index is disposable:
// deleting pb_data/search.db is always safe; it is rebuilt on the next boot.
package search
```

PocketBase itself uses the `modernc.org/sqlite` pure-Go driver; the search
sidecar adds `ncruces/go-sqlite3` (wazero) because FTS5 is needed. Two engines,
intentional.

### Stale claim (a): README env vars — FIVE references in three regions

`grep -n 'BALAUR_CHAT_MODEL\|BALAUR_EMBED_MODEL' README.md` currently returns:

`README.md:32` (inside "Current shape" → Models bullet):

```
  settings models section (an absolute `.gguf` path) or via `BALAUR_CHAT_MODEL`;
```

`README.md:177-180` (inside "Local inference"):

```
- a GGUF model file — install it from the settings models section
  (`/ui/show/settings?section=models`) or pin one with `BALAUR_CHAT_MODEL`.
  That page can also fetch Balaur's official curated model in one click
  (owner-initiated download into `BALAUR_MODELS_DIR`; plan 086)
```

`README.md:182-187` (the worked example):

```
```bash
# Run a local GGUF on a Vulkan GPU:
BALAUR_LIB_PATH=~/.local/share/balaur/kronk/lib \
BALAUR_PROCESSOR=vulkan \
BALAUR_CHAT_MODEL=/models/qwen3.gguf go run . serve
```
```

`README.md:204-205` (the env table rows):

```
| `BALAUR_CHAT_MODEL` | (unset) | Absolute path to a local `.gguf` chat model |
| `BALAUR_EMBED_MODEL` | (unset) | Absolute path to a local `.gguf` embedding model |
```

> NOTE / DRIFT vs. brief: the brief cited only lines ~186, ~204, ~205. The live
> file has TWO more references (lines 32 and 177–178). All five must be fixed
> for the README's grep to come back clean.

### Stale claim (a): AGENTS.md repeats it — `AGENTS.md:236-238`

```
- Local inference is embedded (`internal/kronk`, the Kronk SDK). GGUF model files
  are runtime assets, owner-supplied via `BALAUR_CHAT_MODEL` or the Models page;
  the engine never downloads anything on boot. CPU is the default;
```

### Stale claim (b): README collections — `README.md:24-27`

```
- **One binary:** `balaur` — web UI, database, migrations, agent loop.
- **Data:** PocketBase collections — `conversations`, `messages`,
  `memories`, `skills`, `heads`, `audit_log` — in plain SQLite
  under `pb_data/`.
```

`memories` and `skills` are the stale entries. (knowledge.md's own list at
`internal/self/knowledge.md:102-105` is already correct and is the altitude to
mirror — node spine + relational sidecars.)

### Stale claim (c): README CLI table is missing rows — `README.md:303-307`

```
| `balaur task add/list/done/snooze/drop` | Commitments, directly. | no |
| `balaur memory propose/list/recall/approve/reject/archive/edit` | Memory lifecycle across the consent boundary. | no |
| `balaur skill propose/list/show/approve/reject/archive` | Skill lifecycle. | no |
| `balaur life log/series/kinds/drop` | The owner-defined life log. | no |
| `balaur journal write`, `balaur day <date>` | Keep a journal line verbatim; read one day (journal, log, done, recap). | no |
```

No `balaur note` row, no `balaur search` row.

### Stale claim (c): knowledge.md CLI list is missing them — `internal/self/knowledge.md:309-312`

```
machine-facing
CLI (doctor, chat, task, memory, skill, life, journal, day, recap, history,
audit, verify, model, self, ext, seed) printing v1 JSON envelopes
`{"v":1,"kind":"<cmd>","data":{…}}` for external harnesses — `balaur doctor`
```

`note` and `search` are absent from this parenthetical list.

### Repo doc conventions to honor

- These three files are Markdown prose, not Go. Match the surrounding voice
  (terse, declarative). No new headings unless the template below says so.
- Use backtick code spans for command/collection/env-var names, as the existing
  lines do.
- `AGENTS.md` is "injected into agent context, so keep it lean and high-signal."
  Add exactly one bullet, not a paragraph.

## Commands you will need

| Purpose        | Command                                                                                  | Expected on success            |
|----------------|------------------------------------------------------------------------------------------|--------------------------------|
| Drift check    | `git diff --stat 5dfb285..HEAD -- README.md AGENTS.md internal/self/knowledge.md`         | empty (no drift) — else STOP   |
| Confirm phantom env unused | `grep -rn 'BALAUR_CHAT_MODEL\|BALAUR_EMBED_MODEL' internal main.go --include='*.go'` | NO output, exit 1 — else STOP |
| README env grep (after) | `grep -n 'BALAUR_CHAT_MODEL\|BALAUR_EMBED_MODEL' README.md`                      | NO output after Step 1         |
| Build (knowledge.md is embedded) | `CGO_ENABLED=0 go build ./...`                                          | exit 0                         |
| Full test suite | `go test ./...`                                                                         | all pass (unchanged)           |
| Schema test    | `go test ./migrations/`                                                                   | PASS (sanity for the set)      |
| fmt check (Go untouched) | `gofmt -l .`                                                                    | empty                          |
| Whitespace check | `git diff --check`                                                                      | no output                      |

> If `go test ./...` link-fails with "No space left on device" on a tmpfs
> `/tmp`, set `TMPDIR=/home/alex/.cache/go-tmp` and retry (known box quirk).

## Scope

**In scope** (the only files you may modify):
- `README.md`
- `AGENTS.md`
- `internal/self/knowledge.md`
- `plans/README.md` (your status row only)

**Out of scope** (do NOT touch, even though they look related):
- **ALL Go code.** Do not wire `BALAUR_CHAT_MODEL`/`BALAUR_EMBED_MODEL` into
  any package. The decision is to DELETE the doc claim, not implement the var.
- `internal/search/*` — read-only reference for the dual-SQLite note; changing
  it is out of scope.
- `migrations/schema_test.go`, `internal/cli/cli.go`,
  `internal/store/llm_settings.go` — reference sources of truth, read-only.
- The collection list in `internal/self/knowledge.md` (lines ~102-105) — it is
  already correct; do not "fix" it.
- The other env-table rows in README (`BALAUR_LIB_PATH`, `BALAUR_PROCESSOR`,
  `BALAUR_MODELS_DIR`, `BALAUR_HF_TOKEN`, `BALAUR_OS_ACCESS`, …) — those ARE
  read by Go code; leave them.

## Git workflow

- Branch: executor worktree off `origin/main` (or `docs/179-docs-truth-sync` if
  branching by hand).
- One commit is fine; conventional-commit subject, e.g.
  `docs: purge phantom BALAUR_CHAT_MODEL + stale collections/CLI (179)`.
- Do NOT push or open a PR unless the operator instructed it. (Land-on-main repo,
  no PR gate, but pushing is the owner's call.)

## Steps

### Step 1: Delete the phantom env vars from README.md

Make all five references go away. Specifically:

**1a. Line 32** — drop the "or via `BALAUR_CHAT_MODEL`" clause. Change:

```
  settings models section (an absolute `.gguf` path) or via `BALAUR_CHAT_MODEL`;
```
to:
```
  settings models section (an absolute `.gguf` path);
```

**1b. Lines 177-178** — drop the "or pin one with `BALAUR_CHAT_MODEL`" clause.
Change:

```
- a GGUF model file — install it from the settings models section
  (`/ui/show/settings?section=models`) or pin one with `BALAUR_CHAT_MODEL`.
```
to:
```
- a GGUF model file — install it from the settings models section
  (`/ui/show/settings?section=models`).
```

**1c. Lines 182-187** — remove the phantom line from the worked example so it
runs as written. Change the fenced block from:

```bash
# Run a local GGUF on a Vulkan GPU:
BALAUR_LIB_PATH=~/.local/share/balaur/kronk/lib \
BALAUR_PROCESSOR=vulkan \
BALAUR_CHAT_MODEL=/models/qwen3.gguf go run . serve
```
to (model comes from the Models page, so the example only sets the two real
vars):
```bash
# Run on a Vulkan GPU (install the GGUF from the Models page, not an env var):
BALAUR_LIB_PATH=~/.local/share/balaur/kronk/lib \
BALAUR_PROCESSOR=vulkan go run . serve
```

**1d. Lines 204-205** — delete both env-table rows entirely
(`BALAUR_CHAT_MODEL` and `BALAUR_EMBED_MODEL`). The row above
(`BALAUR_PROCESSOR`) and below (`BALAUR_MODELS_DIR`) stay; just remove the two
phantom rows so the table closes up.

**Verify**: `grep -n 'BALAUR_CHAT_MODEL\|BALAUR_EMBED_MODEL' README.md` → no
output (exit 1).

### Step 2: Fix the AGENTS.md `BALAUR_CHAT_MODEL` clause

In `AGENTS.md:236-237`, change:

```
- Local inference is embedded (`internal/kronk`, the Kronk SDK). GGUF model files
  are runtime assets, owner-supplied via `BALAUR_CHAT_MODEL` or the Models page;
```
to (drop the phantom var, keep the real path):
```
- Local inference is embedded (`internal/kronk`, the Kronk SDK). GGUF model files
  are runtime assets, owner-supplied via the Models page;
```

Leave the rest of that bullet (CPU default, `BALAUR_PROCESSOR=vulkan`, plans
086/087, checksum manifest) unchanged — those are accurate.

**Verify**: `grep -n 'BALAUR_CHAT_MODEL\|BALAUR_EMBED_MODEL' AGENTS.md` → no
output (exit 1).

### Step 3: Correct the README "Current shape" collection list

In `README.md:25-27`, replace the stale list (which says `memories`, `skills`)
with the real set at README altitude — mirror knowledge.md's framing (node
spine + relational sidecars), not all 15 names. Change:

```
- **Data:** PocketBase collections — `conversations`, `messages`,
  `memories`, `skills`, `heads`, `audit_log` — in plain SQLite
  under `pb_data/`.
```
to:
```
- **Data:** PocketBase collections — `conversations`, `messages`, `nodes`
  (the unified knowledge spine: memories, skills, notes, and typed objects),
  `edges`, `heads`, `audit_log` — in plain SQLite under `pb_data/`.
```

This drops the non-existent `memories`/`skills` collections, names the real
`nodes`/`edges` spine, and keeps the bullet short. Do NOT enumerate all 15
collections here — README "Current shape" is a high-level summary.

**Verify**: `grep -n 'memories.*skills.*PocketBase\|`memories`, `skills`' README.md`
returns nothing in the "Current shape" section; the new line contains `nodes`
and `edges` and does NOT list `memories`/`skills` as collections. Cross-check
the names you keep against `migrations/schema_test.go:23-28` (all must be in
that set: `conversations`, `messages`, `nodes`, `edges`, `heads`, `audit_log`).

### Step 4: Add `note` and `search` rows to the README CLI table

After the `balaur skill …` row (`README.md:305`) and before/around the existing
rows, add two rows. Place them where they read naturally — `note` near `memory`
(both are knowledge writes), `search` near it too. Suggested insertion right
after the `balaur skill …` row:

```
| `balaur note add/list/show` | Owner-authored notes as `type=note` nodes. | no |
| `balaur search <terms>` | Cross-type FTS5 recall over approved knowledge nodes. | no |
```

> Adjust the subcommand list / description to match the real help if it differs
> — run `go run . note --help` and `go run . search --help` to confirm the
> exact subcommands before finalizing the wording. The "Model? = no" column is
> correct (neither calls a model). Keep the table's column count (3) and the
> markdown alignment consistent with the surrounding rows.

**Verify**:
`grep -nE '\| `balaur note|\| `balaur search' README.md` → two matching rows.

### Step 5: Add `note` and `search` to the knowledge.md CLI list

In `internal/self/knowledge.md:310-311`, the parenthetical command list reads:

```
CLI (doctor, chat, task, memory, skill, life, journal, day, recap, history,
audit, verify, model, self, ext, seed) printing v1 JSON envelopes
```

Insert `note` and `search` so the list matches `cli.go` Register order
(`note`/`search` sit right after `skill`):

```
CLI (doctor, chat, task, memory, skill, note, search, life, journal, day, recap,
history, audit, verify, model, self, ext, seed) printing v1 JSON envelopes
```

(Wrap the line where it naturally breaks; exact wrapping is cosmetic.)

**Verify**: `grep -n 'memory, skill, note, search, life' internal/self/knowledge.md`
→ one match.

### Step 6: Add the dual-SQLite deferred-work note to AGENTS.md

In `AGENTS.md`'s "Known limitations & deferred work" section, append ONE bullet
after the last existing bullet there (the memory-category-axis one ending at
`AGENTS.md:261`, immediately before the blank line + `## Safety` at line 263):

```
- Balaur runs two SQLite engines on purpose: PocketBase's `modernc.org/sqlite`
  pure-Go driver for the main database, and `ncruces/go-sqlite3` (wazero) for
  the rebuildable FTS5 search sidecar (`pb_data/search.db`) because FTS5 is
  needed for recall (see `internal/search/index.go` header). Both stay CGO-free.
  This is a deliberate cost, not drift — do not "consolidate" to one driver
  without re-checking FTS5 support.
```

Keep it to one bullet (AGENTS.md must stay lean). Do not reword neighboring
bullets.

**Verify**: `grep -n 'two SQLite engines' AGENTS.md` → one match, inside the
"Known limitations & deferred work" section (before `## Safety`).

### Step 7: Build, format, and run the full suite

`internal/self/knowledge.md` is embedded via `//go:embed`, so a malformed edit
could break the build. Run:

- `CGO_ENABLED=0 go build ./...` → exit 0
- `gofmt -l .` → empty (no Go files were touched, so this should already pass)
- `go test ./...` → all pass, unchanged from baseline
- `git diff --check` → no whitespace errors

**Verify**: all four commands succeed.

## Test plan

No new tests — this is docs-only and no behavior changes. The existing
`migrations/schema_test.go` `TestSchemaBaseline` already pins the collection set
this plan documents; running it confirms the names you wrote into README are the
real ones:

- `go test ./migrations/` → PASS (sanity: the set you mirrored is the live set).
- `go test ./...` → PASS (nothing regressed; knowledge.md still embeds/compiles).

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -n 'BALAUR_CHAT_MODEL\|BALAUR_EMBED_MODEL' README.md` → no output.
- [ ] `grep -n 'BALAUR_CHAT_MODEL\|BALAUR_EMBED_MODEL' AGENTS.md` → no output.
- [ ] README "Current shape" no longer lists `memories`/`skills` as
      collections; it names `nodes` and `edges`; every collection it names is in
      `migrations/schema_test.go:23-28`.
- [ ] README CLI table has a `balaur note …` row and a `balaur search …` row.
- [ ] `grep -n 'memory, skill, note, search, life' internal/self/knowledge.md`
      → one match.
- [ ] `grep -n 'two SQLite engines' AGENTS.md` → one match in "Known limitations
      & deferred work".
- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go test ./...` passes (unchanged from baseline).
- [ ] `gofmt -l .` empty; `git diff --check` clean.
- [ ] No files outside `README.md`, `AGENTS.md`, `internal/self/knowledge.md`,
      and your `plans/README.md` row are modified (`git status`).
- [ ] `plans/README.md` status row for plan 179 updated.

## STOP conditions

Stop and report back (do not improvise) if:

- **`BALAUR_CHAT_MODEL` or `BALAUR_EMBED_MODEL` IS actually read by Go code on
  re-check** — i.e. `grep -rn 'BALAUR_CHAT_MODEL\|BALAUR_EMBED_MODEL' internal main.go --include='*.go'`
  returns ANY match. Then the premise is wrong: the fix flips from "delete the
  doc claim" to "document the var correctly." Stop and report; do not delete.
- The drift check shows `README.md`, `AGENTS.md`, or `internal/self/knowledge.md`
  changed since `5dfb285` and the "Current state" excerpts no longer match the
  live lines (e.g. the env rows, collection bullet, or CLI list moved or were
  already edited by another session).
- `migrations/schema_test.go`'s collection set differs from the names in
  "Current state" §"the real collection set" — re-derive the README list from
  the live test before writing, and note the difference.
- `go build ./...` fails after editing `internal/self/knowledge.md` (the
  `//go:embed` broke) and a re-read of the file doesn't reveal an obvious
  Markdown/encoding issue.
- Any step's verification fails twice after a reasonable fix attempt.
- A fix appears to require touching an out-of-scope file (especially any Go
  file).

## Maintenance notes

For the human/agent who owns these docs after this lands:

- The three files now agree that there is **no model env var** — the only way to
  pin a local GGUF is the Models page (`store.SaveLocalModel`). If a future plan
  re-introduces a `BALAUR_CHAT_MODEL` env path in Go, re-add the docs in the same
  change (README env table + the worked example + AGENTS.md) — keep code and
  docs in lockstep.
- The README "Current shape" collection list is intentionally a summary, not the
  full 15-collection set. The authoritative set lives in
  `migrations/schema_test.go` `TestSchemaBaseline`; when a migration adds/retires
  a collection, that test is the source of truth to reconcile against.
- The CLI tables (README + knowledge.md) must track `internal/cli/cli.go`
  `Register`. When a new subcommand is wired there, add a row/word in both docs.
- The dual-SQLite note exists to stop a well-meaning "use one driver" refactor;
  PocketBase's `modernc.org/sqlite` lacks FTS5, which is why the wazero sidecar
  exists. A reviewer should scrutinize any future change that removes the
  `ncruces/go-sqlite3` dependency.
