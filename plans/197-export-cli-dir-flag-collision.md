# Plan 197: Fix the `balaur export` `--dir` flag collision (rename the dest flag to `--out`)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If a
> STOP condition occurs, stop and report. When done, update this plan's row in
> `plans/README.md` — unless a reviewer dispatched you and told you they maintain
> the index.
>
> **Drift check (run first)**: `git diff --stat 0a71f99..HEAD -- internal/cli/export.go internal/cli/export_test.go`
> If either in-scope file changed since this plan was written, compare the
> "Current state" excerpt below against the live code; on a mismatch, STOP.
>
> **Set `TMPDIR` before any `go` command** (tmpfs link bug): `export TMPDIR=/home/alex/.cache/go-tmp; mkdir -p "$TMPDIR"`. If `go build`/`go test` print `error obtaining VCS status: exit status 128`, your shell has a leaked `GIT_*` env var from a worktree — run with `env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE go …` or unset them; it is environmental, not a code defect.

## Status

- **Priority**: P2 (a just-shipped feature is broken for its real use case)
- **Effort**: S
- **Risk**: LOW
- **Depends on**: 194 (done)
- **Category**: bug
- **Planned at**: commit `0a71f99`, 2026-06-25

## Why this matters

Plan 194 shipped `balaur export` (the Johnny Decimal Markdown vault mirror). Its
destination flag is named `--dir` (`internal/cli/export.go:26`), which **collides
with PocketBase's global persistent `--dir`** — the data-dir flag every `balaur`
subcommand inherits. cobra binds the single `--dir` token to both, so
`balaur --dir <box> export` sets the export **dest** to `<box>` (the data dir)
instead of the default `<box>/export`.

**Confirmed on disk:** `balaur --dir <box> note add …` then `balaur --dir <box>
export` writes the mirror to `<box>/10-19 Knowledge/11 Notes/smoke-note.md` —
**inside the data dir**, next to `data.db`/`auxiliary.db`. Writing the sovereign
vault *into* `pb_data` (which `README.md:194` says to "treat as secret") is
structurally wrong and defeats the "separate, inspectable mirror" intent. The
194 unit tests pass because they call `export.ExportMirror` directly via
`storetest` and never exercise the real CLI flag path, so the collision is
invisible to them.

The fix is a one-flag rename: the export dest flag must be `--out` (distinct from
the global data-dir `--dir`) — which is what the plan-192 stub used before 194
renamed it.

## Current state

`internal/cli/export.go` (the whole `exportCmd`, lines 17–39):

```go
func exportCmd(app core.App) *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "One-way Johnny Decimal Markdown mirror of active nodes (+ git history)",
		Args:  cobra.NoArgs,
	}
	// Default: an "export" dir under the data dir. Resolved lazily in RunE so a
	// custom --dir wins and we never capture a stale path at construction time.
	cmd.Flags().StringVar(&dir, "dir", "", "destination directory (default: <data dir>/export)")
	cmd.RunE = run(app, "export", func(cmd *cobra.Command, args []string) (any, error) {
		dest := dir
		if dest == "" {
			dest = filepath.Join(app.DataDir(), "export")
		}
		files, err := export.ExportMirror(app, dest)
		if err != nil {
			return nil, fmt.Errorf("export: %w", err)
		}
		return map[string]any{"files": files, "dest": dest}, nil
	})
	return cmd
}
```

Line 26 is the bug: the flag is named `"dir"`. PocketBase registers `--dir`
(data dir) as a persistent flag on the root command (see how `internal/cli`
commands all accept `--dir`); a local `--dir` on `export` collides with it.

The existing `internal/cli/export_test.go` exercises `exportCmd` via the envelope
harness and may reference the `--dir` flag — read it and update any use.

### Conventions

- Standard Go, gofmt is law, `go vet`/`staticcheck` gate CI. Tests are
  table-driven, no assertion frameworks, `t.TempDir()` for I/O, `storetest`/the
  existing CLI harness for app wiring. Match the style already in
  `internal/cli/export_test.go` and its neighbors (`cli_test.go`).

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| CLI tests | `go test ./internal/cli/` | ok |
| Full suite | `go test ./...` | all pass |
| Vet / fmt | `go vet ./...` ; `gofmt -l internal/cli/` | exit 0 / empty |

## Scope

**In scope** (only these):
- `internal/cli/export.go` — rename the dest flag `"dir"` → `"out"`.
- `internal/cli/export_test.go` — update any `--dir` reference; add the regression guard.

**Out of scope**: `internal/export/*` (the mirror logic is correct), any other
CLI command, the global `--dir` (data dir) flag itself, the `README`/docs.

## Steps

### Step 1: Rename the export dest flag to `--out`

In `internal/cli/export.go`, change line 26 from `"dir"` to `"out"`, and rename
the local variable `dir` → `out` for clarity (so it reads cleanly and no longer
shadows the data-dir concept):

```go
	var out string
	...
	// Default: an "export" dir under the data dir. Named --out (NOT --dir) so it
	// does not collide with the global PocketBase --dir data-dir flag — see plan 197.
	cmd.Flags().StringVar(&out, "out", "", "destination directory (default: <data dir>/export)")
	cmd.RunE = run(app, "export", func(cmd *cobra.Command, args []string) (any, error) {
		dest := out
		if dest == "" {
			dest = filepath.Join(app.DataDir(), "export")
		}
		...
```

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 2: Update the test + add the collision regression guard

In `internal/cli/export_test.go`:
- Change any `--dir` used for the export DEST to `--out`.
- Add a regression guard that would FAIL on the old `--dir` name. The simplest
  robust structural assertion (no need to wire the full persistent-flag tree):

```go
func TestExportFlagDoesNotCollideWithDataDir(t *testing.T) {
	app := storetest.NewApp(t)
	cmd := exportCmd(app)
	if cmd.Flags().Lookup("dir") != nil {
		t.Fatal("export must NOT define a local --dir flag: it collides with the global PocketBase --dir (data dir), causing the mirror to be written into pb_data")
	}
	if cmd.Flags().Lookup("out") == nil {
		t.Fatal("export dest flag must be --out")
	}
}
```

- Keep/repair the existing default-dest behavior test (the dest defaults to
  `filepath.Join(app.DataDir(), "export")`).

**Verify**: `go test ./internal/cli/` → ok, including the new test; confirm the
new test FAILS if you temporarily rename the flag back to `"dir"` (then restore).

### Step 3: Manual smoke (optional but recommended)

With `TMPDIR` set and a clean git env:
```
BOX=$(mktemp -d)
go run . --dir "$BOX" note add --title "Smoke" --body "hi" >/dev/null 2>&1
go run . --dir "$BOX" export >/dev/null 2>&1
find "$BOX/export" -name '*.md'   # mirror is under <box>/export, NOT under <box> itself
ls "$BOX" | grep -q '^10-19' && echo "BUG: mirror leaked into data dir" || echo "OK: data dir clean"
```
Expected: the `.md` is under `$BOX/export/...`, and `$BOX` itself has no
`10-19 …` folder.

## Done criteria

- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go test ./internal/cli/` passes incl. `TestExportFlagDoesNotCollideWithDataDir`.
- [ ] `go test ./...` passes.
- [ ] `grep -n '"dir"' internal/cli/export.go` returns nothing (the flag is `"out"`).
- [ ] `go vet ./...` exit 0; `gofmt -l internal/cli/` empty.
- [ ] Only `internal/cli/export.go` and `internal/cli/export_test.go` modified.
- [ ] `plans/README.md` row updated.

## STOP conditions

- The "Current state" excerpt doesn't match live code (drift).
- Another `export`-reachable flag already uses `--out` (collision of a different
  kind) — then use `--dest` instead and note it.
- Renaming reveals that the data-dir flag is NOT actually named `--dir` upstream
  (so there was no collision) — STOP and report; the premise would be wrong
  (but the on-disk evidence in "Why this matters" says otherwise).

## Maintenance notes

- The real lesson: CLI gateway tests must exercise flag parsing, not only the
  underlying function. The new structural guard is cheap insurance; a fuller
  test would drive the whole root command with `--dir <box> export` and assert
  the mirror lands under `<box>/export`.
