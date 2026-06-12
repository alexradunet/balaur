# Plan 014: Resolve the duplicate migration timestamp 1750200000 (rename + applied-box guard)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on.
> If anything in "STOP conditions" occurs, stop and report. When done,
> update this plan's row in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat c4fce47..HEAD -- migrations/`
> On drift — especially if either 1750200000 file was already renamed —
> re-verify everything below before touching anything.

## Status

- **Priority**: P3
- **Effort**: S (code) — but read the whole plan; the risk is all in migration identity
- **Risk**: MED (a wrong rename re-runs a migration on existing boxes; the guard in Step 3 is mandatory)
- **Depends on**: none (coordinate numbering with plans 002 and 010, which add migrations at 1750700000/1750710000)
- **Category**: tech-debt
- **Planned at**: commit `c4fce47`, 2026-06-12
- **Issue**: https://github.com/alexradunet/balaur/issues/29

## Why this matters

Two migration files share the timestamp prefix `1750200000`:
`1750200000_extensions.go` and `1750200000_llm_model_config.go`. The
numbering convention (unique unix-timestamp prefixes, ordered) is what
makes migration ordering self-evident; a duplicated prefix is a trap for
the next contributor — the order between the two is decided by full-string
sort (`_e…` < `_l…`), which nobody reading the directory listing can be
expected to rely on, and a third `1750200000_*` file would wedge itself
between them invisibly. The two current migrations happen to be mutually
independent (extensions creates the `extensions` collection;
llm_model_config builds `llm_providers`/`llm_models` and replaces
`llm_settings`), so TODAY this is hygiene — fix it while the deployment
count is near zero, because the fix gets harder with every applied box.

The catch: PocketBase tracks applied Go migrations by NAME. Renaming a file
changes the migration's identity — an existing box that already ran
`1750200000_llm_model_config` will see the renamed
`1750205000_llm_model_config` as NEW and re-run it. Its `Up` deletes and
recreates `llm_settings` (dropping the active-model pointer) and re-creates
collections that already exist (which errors). The plan therefore pairs the
rename with an idempotency guard, and verifies BOTH fresh-box and
upgraded-box paths with real binaries.

## Current state

- `migrations/1750200000_extensions.go` — `init()` calls
  `m.Register(extensionsUp, extensionsDown)`; creates the `extensions`
  collection (full file read at planning time; 45 lines).
- `migrations/1750200000_llm_model_config.go` — `m.Register(llmModelConfigUp,
  llmModelConfigDown)`; creates `llm_providers` + `llm_models`, then
  DELETES and recreates `llm_settings` (lines ~49-60: `if old, err :=
  app.FindCollectionByNameOrId("llm_settings"); err == nil { app.Delete(old) }`
  followed by `core.NewBaseCollection("llm_settings")`).
- Migration identity mechanics — VERIFY, do not trust this plan: PocketBase
  v0.39.3's `m.Register` (package `github.com/pocketbase/pocketbase/migrations`)
  derives the migration name from the registering FILE. Find the mechanism:

```bash
grep -rn "func Register" $(go env GOMODCACHE)/github.com/pocketbase/pocketbase@v0.39.3/migrations/migrations.go
grep -rn "runtime.Caller\|filepath.Base" $(go env GOMODCACHE)/github.com/pocketbase/pocketbase@v0.39.3/migrations/*.go $(go env GOMODCACHE)/github.com/pocketbase/pocketbase@v0.39.3/tools/migrate/*.go
```

  Expected: the registered name is the caller's base filename. Applied
  names live in the `_migrations` table of `pb_data/data.db`. If reality
  differs (e.g. names come from an explicit argument), STOP and report.
- Repo conventions: migrations are append-only; numbering by timestamp;
  errors wrapped with `%w`.

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Gates | `gofmt -l .` / `go vet ./...` / `go test -p 1 ./...` | clean / 0 / ok |
| Build OLD binary (pre-rename) | `git stash && CGO_ENABLED=0 go build -o /tmp/balaur-old . && git stash pop` | exit 0 |
| Build NEW binary | `CGO_ENABLED=0 go build -o /tmp/balaur-new .` | exit 0 |
| Inspect applied names | `sqlite3 <box>/data.db "SELECT file FROM _migrations ORDER BY file" \| tail -8` (column may be named differently — check schema first with `.schema _migrations`) | the migration name list |

Sandbox note: TLS failures → `docs/hyperagent-sandbox.md`. If `sqlite3` is
unavailable, read the table via a 10-line Go snippet using the app, or via
`balaur` itself — any read of `_migrations` works.

## Scope

**In scope**:
- `migrations/1750200000_llm_model_config.go` → renamed to
  `migrations/1750205000_llm_model_config.go` (keep BELOW 1750300000, the
  next existing migration) with the guard added
- `AGENTS.md` — one sentence added to the migrations-related rules:
  "Migration timestamp prefixes must be unique and strictly increasing."

**Out of scope** (do NOT touch):
- `1750200000_extensions.go` — renaming ONE file resolves the collision;
  touching both doubles the blast radius.
- Any other migration file; any schema content change inside
  llmModelConfigUp beyond the guard.

## Git workflow

- Branch: `advisor/014-migration-timestamp`
- Commit style: `fix(migrations): unique timestamp for llm_model_config + re-run guard` with a body explaining the identity mechanics. Use `git mv` for the rename so history follows. No push/PR unless instructed.

## Steps

### Step 1: Verify the identity mechanism

Run the two greps from "Current state". Confirm: name == base filename, and
applied names are stored per-box. Record the exact finding in the commit
body.

**Verify**: you can point at the line in PocketBase source where the name
is derived. If not → STOP.

### Step 2: Capture an "old box" BEFORE the rename

```bash
CGO_ENABLED=0 go build -o /tmp/balaur-old .
OLDBOX=/tmp/balaur-oldbox && rm -rf $OLDBOX && mkdir -p $OLDBOX
/tmp/balaur-old --dir $OLDBOX task list >/dev/null   # applies all migrations at c4fce47-era names
```

**Verify**: the `_migrations` inspection shows BOTH `1750200000_*` names
applied.

### Step 3: Rename + guard

`git mv migrations/1750200000_llm_model_config.go migrations/1750205000_llm_model_config.go`,
then add the guard as the FIRST statement of `llmModelConfigUp`:

```go
	// Re-run guard: boxes migrated before this file was renamed from
	// 1750200000_llm_model_config.go already hold this schema under the old
	// migration name. The work is keyed on its outcome, not the name.
	if _, err := app.FindCollectionByNameOrId("llm_providers"); err == nil {
		return nil
	}
```

(Function names stay `llmModelConfigUp`/`Down` — only the file name and the
registered identity change.)

**Verify**: `gofmt -l .` empty; `go vet ./...` → 0; `go test -p 1 ./...` →
ok (storetest boots a fresh box through the renamed file).

### Step 4: Prove both upgrade paths with binaries

```bash
CGO_ENABLED=0 go build -o /tmp/balaur-new .
# Fresh box on the new binary:
NEWBOX=/tmp/balaur-newbox && rm -rf $NEWBOX && mkdir -p $NEWBOX
/tmp/balaur-new --dir $NEWBOX task list            # expect: exit 0
# Old box upgraded to the new binary (the dangerous path):
/tmp/balaur-new --dir $OLDBOX task list            # expect: exit 0, no error
```

Then assert the old box kept its data shape: `_migrations` now ALSO lists
`1750205000_llm_model_config` (it ran and no-op'd via the guard), and
`llm_providers` still exists exactly once.

**Verify**: both commands exit 0; the guard line was exercised on the old
box (the collections were not recreated — check `llm_settings` still has
its rows if any were present; on an empty box, absence of errors is the
signal).

### Step 5: AGENTS.md sentence

Add the uniqueness rule sentence near the migrations guidance (AGENTS.md's
"Prefer PocketBase-native mechanisms" area or the architecture section —
read the file and place it where migrations are discussed).

**Verify**: `grep -n "timestamp prefixes must be unique" AGENTS.md` → 1 match.

## Test plan

- The binary-level old-box/new-box procedure in Steps 2/4 IS the test —
  scripted, reproducible, and stronger than any unit test for migration
  identity.
- Unit suite: `go test -p 1 ./...` (storetest covers the fresh path on
  every package).
- Optional: a `migrations/` test asserting no two files share a timestamp
  prefix (10 lines: glob, parse prefixes, fail on duplicates) — include it;
  it makes the convention self-enforcing. Use the external test package
  pattern (`package migrations_test`, see plan 002).

## Done criteria

- [ ] `ls migrations/ | grep -c "^1750200000"` → `1`
- [ ] Fresh-box and old-box runs (Step 4) both exit 0
- [ ] Duplicate-prefix test exists and passes (`go test ./migrations/...`)
- [ ] `go test -p 1 ./...` exit 0; `gofmt -l .` empty
- [ ] Diff: one `git mv` + guard, AGENTS.md sentence, one test file (plus `plans/README.md`)
- [ ] `plans/README.md` status row updated

## STOP conditions

- Step 1 finds migration names do NOT come from filenames — the whole
  approach changes; report.
- Step 4's old-box run errors (the guard did not fire, or fired too late
  after partial work) — restore the box from Step 2 (rebuild it), report
  the exact error; do NOT hand-edit `_migrations` rows as a workaround.
- Plans 002/010 landed migrations with timestamps that now sort BETWEEN the
  renamed file and its old position in a way that breaks their assumptions
  (they shouldn't — they're additive index/field changes — but if their
  `Up` reads llm collections, re-check ordering).

## Maintenance notes

- The old box's `_migrations` table permanently lists BOTH the old name
  (1750200000_llm_model_config, from before the rename) and the new one
  (no-op'd). Harmless; do not clean it.
- Any future migration touching `llm_providers`/`llm_models`/`llm_settings`
  must assume boxes reached this schema by EITHER name.
- The duplicate-prefix test makes recurrence impossible to merge silently —
  that test is the actual fix; the rename is cosmetics plus convention
  repair.
