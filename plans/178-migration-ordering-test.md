# Plan 178: Assert migration prefixes are strictly increasing, not merely unique

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 12a48bf..HEAD -- migrations/timestamp_uniqueness_test.go`
> If that file changed since this plan was written, compare the
> "Current state" excerpt against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: migration
- **Planned at**: commit `12a48bf`, 2026-06-24

## Why this matters

`AGENTS.md` mandates: "Migration timestamp prefixes must be unique and
strictly increasing — duplicate prefixes sort by full filename, which is not a
reliable ordering contract." PocketBase applies migrations in filename sort
order on a fresh DB, so an out-of-order (numerically smaller) prefix added
later would run **before** a migration it depends on — a silent ordering bug.

The existing test, `migrations/timestamp_uniqueness_test.go`
`TestMigrationTimestampsAreUnique`, only builds a seen-prefix map and flags
**duplicates**; it never asserts monotonic (strictly-increasing) ordering. So
the test promises to catch an out-of-order migration but does not. This plan
adds an assertion that the prefixes, in filename sort order, strictly increase
— closing the gap between the documented contract and what the test enforces.

All ten current prefixes are already strictly increasing (see "Current state"),
so the new assertion passes today and only fails when someone adds a migration
out of order.

## Current state

- `migrations/timestamp_uniqueness_test.go` — the only test in package
  `migrations_test`; scans `.` for `*.go` (excluding `*_test.go`) and checks
  prefix uniqueness. Verbatim, the whole file as it exists today:

```go
1	package migrations_test
2
3	import (
4		"os"
5		"strings"
6		"testing"
7	)
8
9	func TestMigrationTimestampsAreUnique(t *testing.T) {
10		entries, err := os.ReadDir(".")
11		if err != nil {
12			t.Fatal(err)
13		}
14		seen := map[string]string{}
15		for _, e := range entries {
16			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
17			continue
18		}
19		parts := strings.SplitN(e.Name(), "_", 2)
20		if len(parts) < 2 {
21			continue
22		}
23		prefix := parts[0]
24		if prev, ok := seen[prefix]; ok {
25			t.Errorf("duplicate timestamp prefix %s: %s and %s", prefix, prev, e.Name())
26		}
27		seen[prefix] = e.Name()
28	}
29 }
```

  (Note: lines 16–28 are tab-indented in the real file — the body above is
  reproduced for content, not whitespace. Do NOT hand-copy whitespace; you will
  edit the live file in place, so gofmt governs indentation.)

- The migration files present today (non-test `*.go` in `migrations/`), in
  filename sort order, with their 10-digit prefixes:

  ```
  1749600000_init.go
  1750000000_node_types.go
  1750000010_node_type_schemas.go
  1750000020_tasks_to_nodes.go
  1750000030_measures_to_nodes.go
  1750000040_day_type.go
  1750000050_unify_journal_into_day.go
  1750000060_node_type_icons.go
  1750000070_conversation_compaction.go
  1750000080_drop_memory_category.go
  ```

  Every prefix is a 10-digit integer; the sequence is strictly increasing.
  (Verified at plan time: all prefixes length 10, numerically ascending.)

### Repo conventions that apply here

- **Tests are standard `testing`, table-driven where it helps, NO assertion
  frameworks** (no testify). Use `t.Errorf` / `t.Fatalf` exactly like the
  existing test in this same file. This test is filesystem-only — it does NOT
  need a PocketBase app (`storetest.NewApp`), an `llm.Client` fake, or
  `t.TempDir()`; it reads the real `migrations/` directory via `os.ReadDir(".")`
  (the test's working directory is the package directory).
- **Errors are values**: on an unexpected `os.ReadDir` failure, `t.Fatal(err)`
  (matches the existing test, line 12). Do not panic.
- **gofmt is law** — a PostToolUse hook and CI gate run gofmt; after editing,
  the file must satisfy `gofmt -l .` (empty output). Run `go vet ./...` before
  declaring done.
- **Filename sort order is the contract**: PocketBase loads migrations in the
  lexical order of their filenames. `os.ReadDir` already returns entries sorted
  by filename, so iterating its result in order IS the application order — but
  do NOT rely on that implicitly; collect the qualifying filenames into a slice
  and `slices.Sort` (or `sort.Strings`) it explicitly so the test documents and
  enforces the order it checks. (Prefer `slices.Sort` from the modern stdlib.)

## Commands you will need

| Purpose        | Command                          | Expected on success            |
|----------------|----------------------------------|--------------------------------|
| Build          | `CGO_ENABLED=0 go build ./...`   | exit 0, no output              |
| Test (package) | `go test ./migrations/`          | `ok ... github.com/alexradunet/balaur/migrations` |
| Test (verbose) | `go test -v -run TestMigration ./migrations/` | both tests PASS    |
| Test (all)     | `go test ./...`                  | all packages `ok` / no `FAIL`  |
| Vet            | `go vet ./...`                   | exit 0, no output              |
| Fmt check      | `gofmt -l .`                     | empty output                   |
| Diff check     | `git diff --check`              | empty output                   |

## Scope

**In scope** (the only file you may modify):
- `migrations/timestamp_uniqueness_test.go`

**Out of scope** (do NOT touch):
- Any migration `*.go` file in `migrations/` — this plan only changes the
  test, never the schema. The consolidated baseline
  `migrations/1749600000_init.go` is especially off-limits.
- `plans/README.md` content other than your own status row.
- Any file outside `migrations/`.

## Git workflow

- This repo lands directly on `main`; executors typically work in a worktree
  off `origin/main`. Branch name (if you create one): `advisor/178-migration-ordering-test`.
- One commit. Conventional-commit subject, e.g.:
  `test(migrations): assert timestamp prefixes are strictly increasing`
- Do NOT push or open a PR unless the operator explicitly tells you to.

## Steps

### Step 1: Add a strictly-increasing-order test alongside the uniqueness test

Edit `migrations/timestamp_uniqueness_test.go`. Add a **new sibling test
function** named `TestMigrationTimestampsAreStrictlyIncreasing` (keep the
existing `TestMigrationTimestampsAreUnique` unchanged). Reuse the same directory
walk and the same prefix extraction (`strings.SplitN(name, "_", 2)[0]`).

The new test must:

1. `os.ReadDir(".")`; on error `t.Fatal(err)`.
2. Collect the qualifying filenames into a `[]string` — same skip conditions as
   the existing test: skip directories, non-`.go` files, and `*_test.go` files;
   also skip any name where `strings.SplitN(name, "_", 2)` yields fewer than 2
   parts.
3. `slices.Sort(files)` (add `"slices"` to the import block) so iteration is in
   PocketBase's filename application order, explicitly.
4. Walk the sorted slice, parsing each prefix with `strconv.Atoi` (add
   `"strconv"` to imports). For each file:
   - **Assert the prefix is exactly 10 digits.** If `len(prefix) != 10`,
     `t.Errorf(...)` naming the file and prefix. (This is the documented
     assumption that prefixes are comparable 10-digit timestamps.)
   - If `strconv.Atoi(prefix)` returns an error, `t.Errorf(...)` naming the file
     and prefix (a non-integer prefix breaks the ordering contract — see STOP
     conditions).
   - Compare against the previous parsed prefix: if `cur <= prev`, `t.Errorf`
     with a message naming both filenames and both prefixes, e.g.
     `"migration prefixes not strictly increasing: %s (%d) <= previous %s (%d)"`.
   - Track `prev` and the previous filename for the next iteration. Initialize
     `prev` to `-1` so the first file always passes the `cur <= prev` check.

Target shape (illustrative — match the existing file's style; gofmt will fix
spacing):

```go
func TestMigrationTimestampsAreStrictlyIncreasing(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		if len(strings.SplitN(e.Name(), "_", 2)) < 2 {
			continue
		}
		files = append(files, e.Name())
	}
	slices.Sort(files)

	prev := -1
	prevName := ""
	for _, name := range files {
		prefix := strings.SplitN(name, "_", 2)[0]
		if len(prefix) != 10 {
			t.Errorf("migration prefix %q in %s is not 10 digits", prefix, name)
			continue
		}
		cur, err := strconv.Atoi(prefix)
		if err != nil {
			t.Errorf("migration prefix %q in %s is not an integer: %v", prefix, name, err)
			continue
		}
		if cur <= prev {
			t.Errorf("migration prefixes not strictly increasing: %s (%d) <= previous %s (%d)", name, cur, prevName, prev)
		}
		prev = cur
		prevName = name
	}
}
```

Add `"slices"` and `"strconv"` to the existing import block (currently `os`,
`strings`, `testing`). Keep imports alphabetically grouped (gofmt enforces this:
`os`, `slices`, `strconv`, `strings`, `testing`).

**Verify**: `gofmt -l migrations/timestamp_uniqueness_test.go` → empty output
(file already formatted). Then `go vet ./...` → exit 0.

### Step 2: Run the test and confirm both tests pass

**Verify**: `go test -v -run TestMigration ./migrations/` →
both `TestMigrationTimestampsAreUnique` and
`TestMigrationTimestampsAreStrictlyIncreasing` print `--- PASS` and the package
reports `ok`.

### Step 3: Prove the new assertion actually fails on an out-of-order prefix (do NOT commit this)

This is a throwaway sanity check to confirm the test is not a no-op. In a scratch
location, NOT inside `migrations/`, create a temporary `*.go` file whose name has
an out-of-order prefix and confirm the assertion would fire — OR, more simply,
reason through it: with files sorted lexically, a later-added smaller prefix sorts
earlier, so the larger prefix that precedes it will trigger `cur <= prev`.

Concrete proof without touching `migrations/`: temporarily, in a throwaway copy of
the test (e.g. `/tmp` scratch), replace `os.ReadDir(".")` with a hardcoded slice
like `[]string{"1750000080_a.go", "1750000040_b.go"}` and run it — it must report
`not strictly increasing`. Discard the scratch file. Do **not** add any out-of-order
file to the real `migrations/` directory, and do **not** leave scratch code in the
committed test.

**Verify**: the scratch experiment errors as expected; the real
`migrations/timestamp_uniqueness_test.go` contains no scratch slice —
`git diff migrations/timestamp_uniqueness_test.go` shows only the new test
function and the two added imports.

### Step 4: Full validation

**Verify**:
- `go test ./...` → no `FAIL` lines; `migrations` package `ok`.
- `CGO_ENABLED=0 go build ./...` → exit 0.
- `gofmt -l .` → empty.
- `git diff --check` → empty.
- `git status` → only `migrations/timestamp_uniqueness_test.go` (and your
  `plans/README.md` status row) modified.

## Test plan

- **New test**: `TestMigrationTimestampsAreStrictlyIncreasing` in
  `migrations/timestamp_uniqueness_test.go`. Cases it must cover, by virtue of
  walking the real `migrations/` directory:
  - **Happy path**: all ten current prefixes strictly increase → PASS today.
  - **Regression it guards** (proven via the Step 3 scratch experiment, not
    committed): a later-added numerically smaller prefix that sorts earlier
    lexically → the larger preceding prefix trips `cur <= prev` → FAIL.
  - **Edge — equal prefixes**: caught by `cur <= prev` (the `<=`, not `<`), so
    it overlaps the existing duplicate check but on the numeric value.
  - **Edge — non-10-digit prefix**: `len(prefix) != 10` → `t.Errorf`.
  - **Edge — non-integer prefix**: `strconv.Atoi` error → `t.Errorf`.
- **Structural pattern to copy**: the existing
  `TestMigrationTimestampsAreUnique` in the same file — same `os.ReadDir(".")`
  walk, same skip conditions, same `t.Fatal`/`t.Errorf` style, no assertion
  framework.
- **Verification**: `go test -v -run TestMigration ./migrations/` → 2 tests,
  both PASS.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `migrations/timestamp_uniqueness_test.go` contains a new
      `TestMigrationTimestampsAreStrictlyIncreasing` function that sorts the
      migration filenames, parses each prefix as an integer, and `t.Errorf`s
      when a prefix is `<=` its predecessor.
- [ ] The new test also asserts every prefix is exactly 10 digits and parses as
      an integer.
- [ ] `TestMigrationTimestampsAreUnique` is unchanged.
- [ ] `go test ./migrations/` → `ok`.
- [ ] `go test -v -run TestMigration ./migrations/` → both tests PASS.
- [ ] `go test ./...` → no `FAIL`.
- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `gofmt -l .` empty; `go vet ./...` exit 0; `git diff --check` empty.
- [ ] No file outside `migrations/timestamp_uniqueness_test.go` is modified
      (besides your `plans/README.md` status row).
- [ ] `plans/README.md` status row for plan 178 updated.

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows `migrations/timestamp_uniqueness_test.go` changed since
  commit `12a48bf` and the live code no longer matches the "Current state"
  excerpt.
- **Any current migration prefix is NOT a 10-digit integer.** This means the
  ordering contract's core assumption (comparable 10-digit timestamps) is
  already violated by an existing file — report the offending filename(s) and
  stop, because the numeric comparison this plan installs would then flag a
  pre-existing file rather than a future mistake. (At plan time all ten
  prefixes were valid 10-digit integers; if that changed, do not "fix" the
  migration file — it is out of scope.)
- The new test FAILS against the current `migrations/` directory — that means
  the existing migrations are genuinely out of order (or a prefix is malformed);
  report which pair tripped the assertion. Do NOT rename or edit any migration
  file to make the test pass (migration files are out of scope).
- A verification command fails twice after a reasonable fix attempt.

## Maintenance notes

For the human/agent who owns this after it lands:

- This test hard-codes the assumption that **every migration prefix is exactly
  10 digits**. If the project ever adopts a different timestamp width (e.g. a
  longer epoch-millis prefix), the `len(prefix) != 10` assertion must be
  revisited — at that point a pure `strconv.Atoi` numeric compare without the
  width check is the more robust choice.
- The new test and the existing uniqueness test now both walk `migrations/`.
  They could be merged into one walk later, but they are kept separate for
  clarity (one asserts uniqueness, one asserts ordering); merging is an optional
  follow-up, not required here.
- A reviewer should confirm: (1) no migration file was touched, (2) the
  comparison is `<=` (strictly increasing), not `<`, and (3) the imports stay
  gofmt-grouped.
