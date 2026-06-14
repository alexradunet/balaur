# Plan 060: `make fmt` and CI print which files need gofmt (not just exit 1)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 7b16063..HEAD -- Makefile .github/workflows/ci.yml`
> If either file changed since this plan was written, compare the "Current
> state" excerpts against the live files before proceeding; on a mismatch,
> treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: dx
- **Planned at**: commit `7b16063`, 2026-06-14

## Why this matters

Both the local `make fmt` gate and the CI `gofmt` step fail with a bare exit
code and **no output** when files need formatting — they run `test -z "$(gofmt
-l .)"`, which prints nothing. A developer (or an executor agent) sees "make
lint failed" with no clue *which* files are unformatted, and has to re-run
`gofmt -l .` by hand to find out. This is gratuitous friction on the repo's
primary verification gate (`make lint` = fmt + vet + test), which every plan in
this backlog runs. Making the gate name the offending files is a tiny, low-risk
DX win that pays off on every future change. It is in the spirit of the repo's
tooling investment (plan 006 added the CI race + harness jobs).

## Current state

`Makefile` (as of `7b16063`), the `fmt` target:

```make
fmt:
	@[ "$(shell gofmt -l .)" = "" ]
```

`gofmt -l .` lists files whose formatting differs from `gofmt`'s. The current
recipe captures that list via `$(shell ...)`, compares it to empty, and exits 1
if non-empty — but never prints the list. `lint` depends on it:

```make
lint: fmt vet test
```

`.github/workflows/ci.yml` (as of `7b16063`), the `gofmt` step in the `check` job:

```yaml
      - name: gofmt
        run: test -z "$(gofmt -l .)"
```

Same problem: fails silently with no file list.

Repo conventions: the Makefile uses tab-indented recipes and `@`-prefixed quiet
commands; `$$` escapes a literal `$` inside a recipe (Make consumes single `$`).
Keep the change POSIX-sh compatible (CI runs `ubuntu-latest` default shell; the
Makefile recipe runs under `/bin/sh`).

## Commands you will need

| Purpose   | Command                  | Expected on success |
|-----------|--------------------------|---------------------|
| Drift     | `git diff --stat 7b16063..HEAD -- Makefile .github/workflows/ci.yml` | empty |
| Fmt gate (clean tree) | `make fmt` | exit 0, no output |
| Fmt gate sees a bad file | (see Step 3) | prints the file, exits 1 |
| Vet       | `go vet ./...`           | exit 0 |
| Tests     | `go test ./...`          | all pass |
| Full lint | `make lint`              | exit 0 |

## Scope

**In scope** (the only files you should modify):
- `Makefile` — the `fmt` target recipe only
- `.github/workflows/ci.yml` — the `gofmt` step only

**Out of scope** (do NOT touch):
- Any `.go` file — this plan must not reformat or change source. The tree is
  already gofmt-clean; keep it that way.
- Any other Makefile target (`vet`, `test`, `lint`, build/service targets) or
  any other CI step/job.
- The behavior contract: a clean tree must still exit 0; an unformatted tree
  must still exit non-zero. Only the *output on failure* changes.

## Git workflow

- Branch: `improve/060-lint-show-unformatted`
- One commit; conventional-commit style: e.g.
  `build(lint): make fmt + CI gofmt print which files need formatting`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Make the `Makefile` `fmt` target print offenders

Replace the `fmt` target recipe so it captures the list, prints it on failure,
and preserves the exit-0-when-clean / exit-1-when-dirty contract. Target shape:

```make
fmt:
	@unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "These files need gofmt -w:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi
```

Notes:
- Use `$$` (Make → shell `$`).
- It is one logical recipe line continued with `\`; each physical line except
  the last ends with `\ `. Keep the leading tab on every recipe line.
- Do not use `$(shell ...)` here — capture the list **at recipe runtime** in the
  shell variable so the output reflects the tree when `make fmt` actually runs.

**Verify** (tree is currently gofmt-clean):
```
make fmt
```
→ exits 0, prints nothing.

### Step 2: Make the CI `gofmt` step print offenders

In `.github/workflows/ci.yml`, replace the `gofmt` step's `run:` with a
multi-line script that prints the list on failure:

```yaml
      - name: gofmt
        run: |
          unformatted="$(gofmt -l .)"
          if [ -n "$unformatted" ]; then
            echo "These files need gofmt -w:"
            echo "$unformatted"
            exit 1
          fi
```

Keep the step `name: gofmt` and its position (first step after checkout/setup-go
in the `check` job) unchanged. Only the `run:` body changes. (Single `$` here —
this is a YAML/GitHub-Actions shell block, not a Makefile recipe.)

**Verify**: the YAML still parses and the indentation matches the surrounding
steps (2-space step indent, `run: |` block body indented under it). Confirm with:
```
grep -n 'name: gofmt' .github/workflows/ci.yml
grep -n 'need gofmt' .github/workflows/ci.yml   # → 2 hits total across both files in this plan
```

### Step 3: Prove the failure path actually prints the file (then revert)

Create a deliberately misformatted temp file, confirm `make fmt` now names it,
then delete it so the tree is clean again:

```
printf 'package web\nfunc  X(){}\n' > internal/web/zz_fmt_probe.go
make fmt; echo "exit=$?"
rm -f internal/web/zz_fmt_probe.go
```

**Verify**: the `make fmt` run prints `internal/web/zz_fmt_probe.go` under
"These files need gofmt -w:" and reports `exit=1`. After `rm`, run `make fmt`
again → exit 0, no output. Confirm `git status --porcelain` lists ONLY `Makefile`
and `.github/workflows/ci.yml` (the probe file is gone).

> If you cannot remove the probe file for any reason, STOP — do not commit with
> it present.

### Step 4: Full lint sanity

```
make lint
```

**Verify**: exits 0 (fmt clean, vet clean, tests pass).

## Test plan

No Go tests — this changes build tooling only. The verification is behavioral:
clean tree → exit 0 silent (Step 1/4); dirty tree → prints offenders, exit 1
(Step 3). Both must be demonstrated. There is no test framework for the Makefile
or CI YAML in this repo; do not add one.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `make fmt` on the (clean) tree exits 0 with no output
- [ ] The Step-3 probe demonstrates `make fmt` prints the offending filename and exits 1, and the probe file is deleted afterward
- [ ] `make lint` exits 0
- [ ] `grep -c 'need gofmt' Makefile` → `1` and `grep -c 'need gofmt' .github/workflows/ci.yml` → `1`
- [ ] `git status --porcelain` shows only `Makefile` and `.github/workflows/ci.yml` modified (no `.go` files, no probe file)
- [ ] `plans/readme.md` status row for 060 updated (unless your reviewer maintains it)

## STOP conditions

Stop and report back (do not improvise) if:

- The `Makefile` `fmt` target or the CI `gofmt` step does not match the "Current
  state" excerpts (the files drifted).
- After your Makefile change, `make fmt` does NOT exit 0 on the clean tree (you
  broke the contract — the tree is gofmt-clean, so a non-zero exit means the
  recipe is wrong).
- The Step-3 probe leaves any stray file you cannot remove.

## Maintenance notes

- The two changes must keep the same pass/fail contract as before — only the
  diagnostic output is added. A reviewer should confirm the clean-tree exit is
  still 0 (a common mistake is a recipe that always exits 1, or that loses the
  failure exit).
- `make lint` is the gate every other plan in this backlog relies on; this makes
  its first stage self-explaining.
