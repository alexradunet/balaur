# Plan 157: Rename `heads.tools` → `capability_groups` (disambiguate from `extensions.tools`)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 4a8c8c9..HEAD -- internal/heads/heads.go migrations/1749600000_init.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.
>
> **Precondition**: plan **156** (consolidated schema baseline) MUST be landed
> first — this rename edits the `heads` collection field defined in the rewritten
> `migrations/1749600000_init.go`. Verify:
> `grep -n 'Name: "tools"' migrations/1749600000_init.go` → matches the `heads`
> block (one line, in the `heads.Fields.Add(...)` call). If `migrations/` still
> has many files, 156 is not landed — STOP.

## Status

- **Priority**: P3 (optional polish; cosmetic clarity)
- **Effort**: S (≈20–30 min)
- **Risk**: LOW
- **Depends on**: 156
- **Category**: tech-debt
- **Planned at**: commit `4a8c8c9`, 2026-06-22

## Why this matters

Two collections store a JSON field named `tools`, with different shapes:
`heads.tools` is a JSON array of capability-group keys (`["memory","tasks",…]`)
that filters which tool groups a persona offers, while `extensions.tools` is a
JSON array of tool-definition objects (`[{name,description},…]`). The collision
is a maintenance smell — anyone reading the schema or grepping `"tools"` has to
disambiguate by collection. Renaming the heads field to `capability_groups`
matches the domain vocabulary already used in the code: `internal/heads/heads.go`
calls them "capability groups" / `Groups`, and the persona-design doc
(`docs/superpowers/specs/2026-06-14-heads-as-personas-design.md`) calls a head
"a capability filter, NOT a security boundary".

> **Scope trap:** `extensions.tools` (`internal/ext/ext.go:171`), the
> self-inventory `"tools"` key (`internal/self/self.go:125`,
> `internal/self/tool.go:80`), `internal/cli/ext.go:26`, and the LLM wire
> `"tools"` (`internal/llm/openai.go`, `internal/kronk/client.go`) are all
> DIFFERENT `tools` — **do not touch them.** This rename is only the `heads`
> DB column and its two accessors in `internal/heads/heads.go`.

## Current state

The `heads` field is declared in the consolidated baseline (after plan 156) at
`migrations/1749600000_init.go`, inside the `heads.Fields.Add(...)` block:
```go
		&core.JSONField{Name: "tools"},
```

`internal/heads/heads.go` is the only reader/writer of that column:

`Create` (line ~107):
```go
	rec.Set("tools", marshalGroups(groups))
```

`headFromRecord` (line ~124–128):
```go
func headFromRecord(r *core.Record) Head {
	var groups []string
	if raw := r.GetString("tools"); raw != "" {
		_ = json.Unmarshal([]byte(raw), &groups)
	}
	...
}
```

The HTML form field is also named `tools` (`internal/web/heads.go:60`
`e.Request.Form["tools"]`, `internal/feature/headscards/heads.go:188`
`h.Name("tools")`). **This is a transport key, not the DB column** — leave it as
`tools`. The handler reads the posted `tools` form values into `groups` and
`heads.Create` persists them into the renamed column. Renaming the form field is
unnecessary churn and out of scope.

## Commands you will need

| Purpose          | Command                                   | Expected on success     |
|------------------|-------------------------------------------|-------------------------|
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...`            | exit 0                  |
| Vet              | `go vet ./...`                            | exit 0                  |
| Tests            | `go test ./...`                           | all packages `ok`       |
| Dead-code gate   | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | exit 0, no output |
| Format check     | `gofmt -l .`                              | empty output            |
| Whitespace       | `git diff --check`                        | no output               |

(In a TLS-intercepting sandbox the Go commands need the GOPROXY shim — see
`docs/hyperagent-sandbox.md`.)

## Scope

**In scope** (the only files you may modify):
- `migrations/1749600000_init.go` — rename the `heads` JSON field.
- `internal/heads/heads.go` — the two accessors.
- `migrations/schema_test.go` — add an assertion for the new field name (optional
  but recommended).
- A `internal/heads` test only if one asserts the old `"tools"` column directly.

**Out of scope** (do NOT touch):
- `extensions.tools` and every other `"tools"` listed in the scope trap above.
- The `tools` HTML form field name (`internal/web/heads.go`,
  `internal/feature/headscards/heads.go`).

## Git workflow

- Branch off `origin/main` (executor worktree convention).
- One commit; conventional-commit subject, e.g.
  `refactor(heads): rename heads.tools column to capability_groups`.
- Do NOT push or merge unless the operator instructs it.

## Steps

### Step 1: Rename the schema field

In `migrations/1749600000_init.go`, in the `heads.Fields.Add(...)` block change:
```go
		&core.JSONField{Name: "tools"},
```
to:
```go
		&core.JSONField{Name: "capability_groups"},
```

**Verify**: `grep -n 'Name: "capability_groups"' migrations/1749600000_init.go` → one match (heads block); `grep -n 'Name: "tools"' migrations/1749600000_init.go` → still one match, in the **extensions** block (untouched).

### Step 2: Update the two accessors

In `internal/heads/heads.go`:
- `Create`: `rec.Set("capability_groups", marshalGroups(groups))`
- `headFromRecord`: `if raw := r.GetString("capability_groups"); raw != "" {`

**Verify**: `grep -n '"tools"' internal/heads/heads.go` → no match; `go build ./internal/heads/` → exit 0.

### Step 3: Update the schema contract test (recommended)

In `migrations/schema_test.go`, add `heads` to the field-presence checks:
```go
		{"heads", []string{"name", "purpose", "balaur_avatar", "capability_groups"}, []string{"tools"}},
```
(So the test asserts the rename: `capability_groups` present, `tools` absent on
`heads`.)

**Verify**: `go test ./migrations/...` → `ok`.

### Step 4: Reconcile heads tests + full gate

Run `go test ./internal/heads/...`. If a test references the old `"tools"`
column directly, update it to `capability_groups`. Then run the whole
verification set from "Commands you will need". All must pass.

## Test plan

No new behavior — a column rename plus its two Go accessors. The contract is:
a custom head created with capability groups round-trips through
`heads.Create` → `heads.List`/`Find` → `headFromRecord` with the same `Groups`.
If `internal/heads` already has a create-then-read test, it is the regression
net; otherwise the storybook/headscards tests exercise the path. `go test ./...`
covers the rest.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -rn '"tools"' internal/heads/` → no matches.
- [ ] `grep -n 'Name: "capability_groups"' migrations/1749600000_init.go` → one match.
- [ ] `grep -rn '"tools"' internal/ext/ internal/self/ internal/cli/ext.go` → STILL present (scope held — extensions/self untouched).
- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go vet ./...` exits 0.
- [ ] `go test ./...` — all packages `ok`.
- [ ] `staticcheck ./...` exits 0 with no output.
- [ ] `gofmt -l .` empty; `git diff --check` empty.
- [ ] No file outside the in-scope list is modified (`git status`).
- [ ] `plans/readme.md` status row for 157 updated.

## STOP conditions

Stop and report back (do not improvise) if:
- Plan 156 is not landed (the field lives in the old `heads_as_personas`
  migration, not the baseline) — land 156 first.
- A consumer of `heads.tools` exists outside `internal/heads/heads.go` (grep
  `GetString("tools")` / `Set("tools"` across `internal/` and confirm every hit
  is `extensions`/`self`/wire, not `heads`) — handle or report.
- You are about to edit `extensions.tools` or the form field name — stop; wrong
  `tools`.

## Maintenance notes

- After this, `grep '"tools"'` in the repo unambiguously means extension tool
  metadata or the LLM wire field — never persona capability groups.
- The HTML form input stays named `tools`; if a future pass wants full
  consistency, rename the form field + `e.Request.Form["tools"]` +
  `h.Name("tools")` together (a separate, larger UI change).
- Reviewer: confirm a custom head's selected groups still persist and re-render
  (the round-trip through the renamed column).
