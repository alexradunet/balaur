# Plan 206: Audit head mutations inside `internal/heads` via an `actor` param — closing the unaudited web head switch/create/delete gap

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 59a94e0..HEAD -- internal/heads/heads.go internal/tools/heads.go internal/web/heads.go internal/seed/seed.go` (expect EMPTY)
> If any in-scope file changed since this plan was written, compare the "Current
> state" excerpts against the live code before proceeding; on a mismatch, treat
> it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt (closes a real behavioral gap: an unaudited mutation)
- **Planned at**: commit `07fb4d6`, 2026-06-26 (re-verified against `59a94e0` 2026-06-29: heads/tools/web unchanged; `seed.go` shifted by 211 so the `heads.Create` call is now line **530**, not 525. No tour reconcile needed — tours 05/17/19 anchor `heads.go:27/:83`, `tools/heads.go:23`, `seed.go:70`, all ABOVE 206's changes.)

## Why this matters

Head switch/create/delete is audited **only at the tool layer**
(`internal/tools/heads.go` calls `store.Audit` after each `heads.*` call), while
the sibling domains (`tasks`, `life`, `knowledge`, `nodes`) audit **inside the
domain package**. The consequence is a live gap: the **web** head handlers
(`internal/web/heads.go`) call `heads.SetActive`/`Create`/`Delete` and do
**not** audit — so an owner switching, creating, or deleting a head from the UI
leaves no `audit_log` entry, even though the same actions via the model are
audited. Auditing belongs in `internal/heads` so every caller (tools, web, and
any future gateway) is covered once, consistently.

`AGENTS.md` rule: "Audit strictly AFTER the successful write, never before."
This plan keeps that — the audit goes after `app.Save`/`app.Delete`/
`SetOwnerSetting` succeeds.

## Current state

### `internal/heads/heads.go` — no auditing today

```go
// line 93
func SetActive(app core.App, id string) error {
	return store.SetOwnerSetting(app, activeHeadSetting, id)
}

// line 98
func Create(app core.App, name, purpose, avatar string, groups []string) (string, error) {
	col, err := app.FindCollectionByNameOrId("heads")
	if err != nil {
		return "", err
	}
	rec := core.NewRecord(col)
	rec.Set("name", name)
	rec.Set("purpose", purpose)
	rec.Set("balaur_avatar", avatar)
	rec.Set("capability_groups", marshalGroups(groups))
	if err := app.Save(rec); err != nil {
		return "", err
	}
	return rec.Id, nil
}

// line 116
func Delete(app core.App, id string) error {
	rec, err := app.FindRecordById("heads", id)
	if err != nil {
		return err
	}
	return app.Delete(rec)
}
```

`internal/heads/heads.go` already imports `store` (used by `Active`/`SetActive`).
The audit signature, from existing call sites:
`store.Audit(app core.App, actor, action, target string, success bool, details map[string]any)`.

### `internal/tools/heads.go` — currently audits after each call

```go
// line 50-54 (head_switch)
if err := heads.SetActive(app, id); err != nil {
	return "", fmt.Errorf("head_switch: %w", err)
}
store.Audit(app, "model", "head.switch", "heads/"+id, true, map[string]any{"name": head.Name})
return fmt.Sprintf("Active head set to %q — it takes effect on the next turn.", head.Name), nil

// line 91-96 (head_create)
id, err := heads.Create(app, args.Name, args.Purpose, args.Avatar, args.Groups)
if err != nil {
	return "", fmt.Errorf("head_create: %w", err)
}
store.Audit(app, "model", "head.create", "heads/"+id, true, map[string]any{"name": args.Name})

// line 122-125 (head_delete)
if err := heads.Delete(app, id); err != nil {
	return "", fmt.Errorf("head_delete: %w", err)
}
store.Audit(app, "model", "head.delete", "heads/"+id, true, map[string]any{"name": head.Name})
```

(The tool keeps using `head.Name`/`args.Name` for its model-facing response
strings and `store.ValidBalaurAvatarKey` for validation, so the `heads.Find`
lookups and the `store` import STAY in `tools/heads.go`.)

### `internal/web/heads.go` — currently UNAUDITED

```go
// line 20 (setActiveHead)
if err := heads.SetActive(h.app, id); err != nil { ... }
// line 61 (createHead)
if _, err := heads.Create(h.app, name, purpose, avatar, groups); err != nil { ... }
// line 76 (deleteHead, switch-back before delete)
_ = heads.SetActive(h.app, heads.MainKey)
// line 78 (deleteHead)
if err := heads.Delete(h.app, id); err != nil { ... }
```

### `internal/seed/seed.go:530` — one Create call

```go
if _, err := heads.Create(app, name, "tends the garden plan and seasonal chores; practical and seasonal", "balaur-16", []string{"tasks", "life", "memory"}); err != nil {
```

## Commands you will need

| Purpose   | Command                                                  | Expected         |
|-----------|----------------------------------------------------------|------------------|
| Build     | `CGO_ENABLED=0 go build ./...`                           | exit 0           |
| Vet       | `go vet ./...`                                            | exit 0           |
| Test pkg  | `go test ./internal/heads/... ./internal/tools/... ./internal/web/...` | PASS |
| Full test | `go test ./...`                                           | all pass         |
| gofmt     | `gofmt -l internal/heads internal/tools internal/web internal/seed` | prints nothing |

> If `go test ./...` fails the link step with "No space left on device", set
> `TMPDIR=/home/alex/.cache/go-tmp` and retry.

## Scope

**In scope**:
- `internal/heads/heads.go` (add `actor` param + audit-after-write to `SetActive`/`Create`/`Delete`)
- `internal/tools/heads.go` (pass `actor="model"`; remove the now-duplicated `store.Audit` calls)
- `internal/web/heads.go` (pass `actor="owner"`)
- `internal/seed/seed.go` (pass `actor="seed"` at line 530)

**Out of scope** (do NOT touch):
- The `heads.Find`/`BuiltIn` gating in tools and web — unchanged.
- The model-facing response strings in tools — unchanged.
- Any other `heads.*` function (`List`, `Active`, `Find`, etc.) — unchanged.

## Git workflow

- Branch: `advisor/206-heads-in-domain-audit`
- Conventional-commit subject, e.g. `refactor(heads): audit head mutations in-domain via actor param`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Add `actor` param + audit-after-write in `internal/heads/heads.go`

Rewrite the three functions (audit AFTER the successful write; resolve the head
name for the detail map):

```go
// SetActive persists the active head id/key and audits the switch.
func SetActive(app core.App, actor, id string) error {
	if err := store.SetOwnerSetting(app, activeHeadSetting, id); err != nil {
		return err
	}
	name := id
	if h, ok := Find(app, id); ok {
		name = h.Name
	}
	store.Audit(app, actor, "head.switch", "heads/"+id, true, map[string]any{"name": name})
	return nil
}

// Create adds a custom head, returns its record id, and audits the creation.
func Create(app core.App, actor, name, purpose, avatar string, groups []string) (string, error) {
	col, err := app.FindCollectionByNameOrId("heads")
	if err != nil {
		return "", err
	}
	rec := core.NewRecord(col)
	rec.Set("name", name)
	rec.Set("purpose", purpose)
	rec.Set("balaur_avatar", avatar)
	rec.Set("capability_groups", marshalGroups(groups))
	if err := app.Save(rec); err != nil {
		return "", err
	}
	store.Audit(app, actor, "head.create", "heads/"+rec.Id, true, map[string]any{"name": name})
	return rec.Id, nil
}

// Delete removes a custom head record and audits the deletion. Built-ins never
// reach here — callers gate on BuiltIn first.
func Delete(app core.App, actor, id string) error {
	rec, err := app.FindRecordById("heads", id)
	if err != nil {
		return err
	}
	name := rec.GetString("name")
	if err := app.Delete(rec); err != nil {
		return err
	}
	store.Audit(app, actor, "head.delete", "heads/"+id, true, map[string]any{"name": name})
	return nil
}
```

**Verify**: `go build ./internal/heads/...` → exit 0

### Step 2: Update `internal/tools/heads.go` — pass `actor="model"`, drop the duplicated audits

- `head_switch`: change the call to `heads.SetActive(app, "model", id)` and
  **delete** the `store.Audit(...)` line that followed it (line 53). Keep the
  `head.Name` response string.
- `head_create`: change to `heads.Create(app, "model", args.Name, args.Purpose, args.Avatar, args.Groups)` and **delete** the following `store.Audit` line (95).
- `head_delete`: change to `heads.Delete(app, "model", id)` and **delete** the following `store.Audit` line (125).

`store` is still imported (used by `store.ValidBalaurAvatarKey` in
`head_create`) — keep it. Confirm with `go vet`.

**Verify**:
- `grep -n "store.Audit" internal/tools/heads.go` → no matches
- `go build ./internal/tools/...` → exit 0; `go vet ./internal/tools/...` → exit 0

### Step 3: Update `internal/web/heads.go` — pass `actor="owner"`

- line 20: `heads.SetActive(h.app, "owner", id)`
- line 61: `heads.Create(h.app, "owner", name, purpose, avatar, groups)`
- line 76: `_ = heads.SetActive(h.app, "owner", heads.MainKey)` (the switch-back
  when deleting the active head — it is an owner-initiated consequence, so audit
  it as the owner too; this adds a `head.switch` audit entry alongside the
  `head.delete` when an owner deletes the active head, which is correct)
- line 78: `heads.Delete(h.app, "owner", id)`

**Verify**: `go build ./internal/web/...` → exit 0

### Step 4: Update `internal/seed/seed.go:530` — pass `actor="seed"`

```go
if _, err := heads.Create(app, "seed", name, "tends the garden plan and seasonal chores; practical and seasonal", "balaur-16", []string{"tasks", "life", "memory"}); err != nil {
```

(Decision recorded explicitly: seed-created heads ARE audited, with
`actor="seed"`, so the demo box's audit log truthfully shows the seed wrote
them. If the drift check shows additional `heads.Create`/`SetActive`/`Delete`
callers beyond the four files in scope, update each: gateways → `"owner"`,
model/tool paths → `"model"`, seed/demo → `"seed"`.)

**Verify**:
- `grep -rn "heads.SetActive\|heads.Create\|heads.Delete" internal/ --include=*.go | grep -v _test` → every call now passes an actor as the 2nd arg
- `go build ./...` → exit 0

### Step 5: Full verification

**Verify**:
- `gofmt -l internal/heads internal/tools internal/web internal/seed` → prints nothing
- `go vet ./...` → exit 0
- `go test ./internal/heads/... ./internal/tools/... ./internal/web/...` → PASS
- `go test ./...` → all pass

## Test plan

- Add `TestSetActiveAudits` / `TestCreateAudits` / `TestDeleteAudits` in
  `internal/heads/heads_test.go` (create the file if absent, modeled on an
  existing store-backed domain test such as `internal/tasks` or
  `internal/knowledge` tests that assert audit rows):
  - Call `heads.Create(app, "owner", "Gardener", "...", "", nil)`, then query
    the `audit_log` collection for an entry with `action == "head.create"` and
    `actor == "owner"` and `success == true`. Assert exactly one matching row.
  - Similarly for `SetActive(app, "owner", id)` → `head.switch`, and
    `Delete(app, "owner", id)` → `head.delete`.
  - To read the audit log, use the same approach the existing audit-asserting
    tests use (find by `store`/`tests` helper). If you can't find the audit
    collection name, it is `audit_log`.
- Update any existing `tools/heads_test.go` that asserted the tool emitted the
  audit — the audit now comes from `internal/heads`, so the assertion still
  holds (one `head.*` row per action) but the actor for tool-driven actions is
  `"model"`. Adjust the asserted actor if a test pins it.
- Verification: `go test ./internal/heads/...` → PASS including the new audit
  tests.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `grep -n "store.Audit" internal/heads/heads.go` returns 3 matches (one per mutation)
- [ ] `grep -n "store.Audit" internal/tools/heads.go` returns no matches
- [ ] `grep -rn "heads.SetActive\|heads.Create\|heads.Delete" internal/ --include=*.go | grep -v _test` shows every call passing an actor string
- [ ] `go test ./...` exits 0; new `internal/heads` audit tests pass
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- There are `heads.SetActive`/`Create`/`Delete` callers beyond the four in-scope
  files (the `grep` in step 4 surfaces a fifth) and you can't confidently
  classify its actor — report it rather than guess.
- An existing test asserts the EXACT audit action string or actor for a
  head mutation and the new value differs in a way you can't reconcile — report
  which test.
- `store.Audit`'s signature differs from the 6-arg shape shown above — confirm
  against an existing caller and report.

## Maintenance notes

- After this, head mutations audit once, in-domain — matching `tasks`/`life`/
  `knowledge`/`nodes`. Any new head-mutation gateway inherits auditing for free
  by passing its `actor`.
- Reviewer: confirm every audit is AFTER the successful write (never before), and
  that deleting the active head from the UI now produces both a `head.switch`
  (switch-back to main) and a `head.delete` entry.
- The actor vocabulary in use here: `"owner"` (UI), `"model"` (agent tools),
  `"seed"` (demo seeding) — matches the actors other domains record.
