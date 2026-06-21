# Plan 118: Documentation reflects the real single-page UI, and two dead symbols are gone

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat ce2ba72..HEAD -- main.go README.md AGENTS.md internal/web/web.go internal/feature/knowledgecards/knowledgefocus.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: docs + tech-debt (dead code)
- **Planned at**: commit `ce2ba72`, 2026-06-21

## Why this matters

After plan 089 retired the top-nav and the `/focus/*` pages for a single-page
panel UI, the user-facing docs were never fully resynced, and a recent
`clean docs` commit deleted the entire `docs/` tree — including two files that
`AGENTS.md` and `README.md` still link to. The result: `main.go` calls the UI
"HTMX" (it is Datastar), the README points users at routes that now 302-redirect
to `/`, mentions a `boards` feature that was deleted, and links two missing docs.
Separately, two symbols are dead weight (zero callers). Fixing all of this makes
the repo honest about itself — the single most damaging kind of staleness.

## Current state

Files and the exact text to change:

- `main.go:2` — package header, wrong UI framework:
  ```go
  // PocketBase (data, auth, migrations), an HTMX web UI, and local LLM
  ```
  Every other doc says Datastar (`README.md:10`, `DESIGN.md`, `AGENTS.md`).

- `README.md` — four references to retired `/focus/*` routes. The single-page
  IA replaced them with the SSE summon-door `/ui/show/{type}`, which `DESIGN.md`'s
  honesty ledger already documents (e.g. DESIGN.md:137 "the quests card (opens in
  the right panel at `/ui/show/quests`)"). The legacy `/focus/*` and flat routes
  now 302→`/` (DESIGN.md:110-113). The four sites:
  - `README.md:84-86` quests focus → `/focus/quests`
  - `README.md:93` lifelog → `/focus/lifelog`
  - `README.md:97-98` day → `/focus/day?date={date}`
  - `README.md:178` settings → `/focus/settings?section=models`

- `README.md:428` — project layout lists `boards`, a collection/feature dropped
  in plan 089:
  ```
  internal/web/      Datastar gateway: dock chat, boards, cards & focuses, recap
  ```

- **Deleted-but-referenced docs.** The `clean docs` commit removed the whole
  `docs/` directory. Two of those files are still linked by live docs:
  - `AGENTS.md:30` → "run the GOPROXY shim per `docs/hyperagent-sandbox.md`"
  - `README.md:201` and `README.md:244` → `[docs/netbird.md](docs/netbird.md)`
  Both files still exist in git history at commit `8c58aad` (the commit just
  before the deletion). They carry real operational info (the sandbox GOPROXY
  shim; NetBird access without VPN code) and are referenced, so restore them
  rather than scrubbing the references.

- `internal/feature/knowledgecards/knowledgefocus.go:20-22` — dead package var,
  zero callers (the comment claims it backs a title helper, but the helper
  hardcodes its categories):
  ```go
  // focusMemoryCategories mirrors the migration constant and the web-side list.
  // Kept here for the title helper below.
  var focusMemoryCategories = []string{"fact", "preference", "person", "project", "context"}
  ```

- `internal/web/web.go:302-307` — dead function, zero callers (it is the last
  function in the file):
  ```go
  // isDatastarRequest reports whether the request is a Datastar @get/@post fetch
  // (which expects an SSE patch stream) rather than a full document load. A
  // Datastar fetch advertises Accept: text/event-stream.
  func isDatastarRequest(e *core.RequestEvent) bool {
  	return strings.Contains(e.Request.Header.Get("Accept"), "text/event-stream")
  }
  ```
  `strings` and `core` remain used elsewhere in `web.go`, so removing this does
  not orphan an import (verified by `go vet` in the steps below).

Documented vocabulary to honor (from `DESIGN.md`): cards "open in the right
panel" via `/ui/show/{type}`; the layout surface is the **panel**, not a
"focus" page. Match that wording.

## Commands you will need

| Purpose   | Command                                  | Expected on success |
|-----------|------------------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`           | exit 0              |
| Vet       | `go vet ./...`                           | exit 0, no errors   |
| Format    | `gofmt -l main.go internal/web/web.go internal/feature/knowledgecards/knowledgefocus.go` | empty output |
| Tests     | `go test ./internal/web/... ./internal/feature/... ./...` | all `ok`            |
| Diff hygiene | `git diff --check`                    | no output           |

(In a TLS-intercepting sandbox, Go commands may need a GOPROXY shim; GOSUMDB
stays on.)

## Scope

**In scope** (the only files you should modify):
- `main.go`
- `README.md`
- `internal/web/web.go`
- `internal/feature/knowledgecards/knowledgefocus.go`
- `docs/hyperagent-sandbox.md` and `docs/netbird.md` (restored via `git checkout`, step 4)

**Out of scope** (do NOT touch, even though they look related):
- `README.md:430` (`web/  embedded templates and static assets`) — the
  `templates` wording belongs to the `html/template` retirement owned by plan
  117; leave it.
- `DESIGN.md`, `internal/self/knowledge.md` — already accurate for the
  single-page IA; do not edit.
- Any `// Ports {{define ...}} from web/templates/...` provenance comment —
  owned by plan 117.
- The `/ui/show` route handler or any Go behavior — this plan changes docs and
  deletes dead symbols only.

## Git workflow

- Land on `main` per `AGENTS.md` (no PR gate). If dispatched into a worktree,
  base off `origin/main`.
- Conventional-commit subjects, e.g. `docs: resync README/main.go to single-page UI; restore referenced docs` and `refactor: drop two dead symbols`. Commit per logical unit is fine.
- Do NOT push or merge unless the operator instructs it.

## Steps

### Step 1: Fix the `main.go` UI-framework label

In `main.go:2`, replace `an HTMX web UI` with `a Datastar web UI`.

**Verify**: `grep -n "HTMX" main.go` → no output.

### Step 2: Resync the README route references

Make these exact replacements in `README.md`:

1. quests (lines 84-86): change the bold heading `**The quests focus —` to
   `**The quests card —`, change `lives in the quests card's focus` to
   `opens as the quests card in the right panel`, and change `` (`/focus/quests`); ``
   to `` (`/ui/show/quests`); ``.
2. lifelog (line 93): `` (`/focus/lifelog`) `` → `` (`/ui/show/lifelog`) ``.
3. day (lines 97-98): `the `` `day` `` card's focus` → `the `` `day` `` card`, and
   `` (`/focus/day?date={date}`) `` → `` (`/ui/show/day?date={date}`) ``.
4. settings (line 178): `` (`/focus/settings?section=models`) `` →
   `` (`/ui/show/settings?section=models`) ``.

**Verify**: `grep -n "/focus/" README.md` → no output.

### Step 3: Drop the stale `boards` mention in the README layout

In `README.md:428`, change:
```
internal/web/      Datastar gateway: dock chat, boards, cards & focuses, recap
```
to:
```
internal/web/      Datastar gateway: dock chat, cards & panels, recap
```

**Verify**: `grep -n "boards" README.md` → no output.

### Step 4: Restore the two deleted-but-referenced docs

Run: `git checkout 8c58aad -- docs/hyperagent-sandbox.md docs/netbird.md`

This restores exactly those two files (the rest of the old `docs/` tree stays
deleted). Do not edit the restored content.

**Verify**: `test -f docs/hyperagent-sandbox.md && test -f docs/netbird.md && echo OK` → `OK`.
Then confirm the links now resolve: `grep -n "docs/hyperagent-sandbox.md" AGENTS.md` and `grep -n "docs/netbird.md" README.md` should each still match, and the target files now exist.

### Step 5: Delete the dead `focusMemoryCategories` var

In `internal/feature/knowledgecards/knowledgefocus.go`, delete lines 20-22 (the
two comment lines and the `var focusMemoryCategories = ...` declaration) plus the
blank line that follows, so the file goes straight from the import block to the
`KnowledgeFocusView` doc comment.

**Verify**: `grep -rn "focusMemoryCategories" internal/` → no output.

### Step 6: Delete the dead `isDatastarRequest` function

In `internal/web/web.go`, delete lines 302-307 (the doc comment and the
`isDatastarRequest` function — the last function in the file).

**Verify**: `grep -rn "isDatastarRequest" internal/` → no output.

### Step 7: Build, vet, format, test

Run the four commands from the table.

**Verify**: build exit 0; `go vet ./...` exit 0 (this is what proves no import
was orphaned by the deletions); `gofmt -l ...` empty; `go test ./...` all `ok`.

## Test plan

No new tests. This is a docs + dead-code change; the existing suite is the
guard. `go vet ./...` catches an orphaned import from the two deletions, and
`go test ./...` confirms nothing referenced the deleted symbols. Confirm the
full suite is green (40 packages `ok`) before declaring done.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -n "HTMX" main.go` → empty
- [ ] `grep -n "/focus/" README.md` → empty
- [ ] `grep -n "boards" README.md` → empty
- [ ] `docs/hyperagent-sandbox.md` and `docs/netbird.md` exist
- [ ] `grep -rn "focusMemoryCategories\|isDatastarRequest" internal/` → empty
- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` all `ok`
- [ ] `git diff --check` → no output
- [ ] Only the in-scope files are modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- The "Current state" excerpts don't match the live code (the codebase drifted
  since `ce2ba72`).
- `git checkout 8c58aad -- docs/...` fails because the commit or files are not
  found (the history may have been rewritten) — report; do not hand-recreate the
  docs.
- Deleting `isDatastarRequest` or `focusMemoryCategories` causes a build error
  (means a caller exists that the audit missed) — STOP; the symbol is not dead.
- Any verification fails twice after a reasonable fix attempt.

## Maintenance notes

- If the owner truly wants `docs/hyperagent-sandbox.md` / `docs/netbird.md` gone
  rather than restored, the alternative is to remove the references in
  `AGENTS.md:30` and `README.md:201,244` instead of restoring — but that loses
  the GOPROXY-shim and NetBird-access instructions, so restore is the default.
- `README.md:430`'s "templates" wording and the feature-card "Ports {{define}}"
  comments are intentionally left for plan 117 (the `html/template` retirement).
- A reviewer should confirm no `/focus/` route is reintroduced and that
  `go vet` (not just build) passed — vet is the real guard for the deletions.
