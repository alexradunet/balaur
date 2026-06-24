# Plan 165: Per-type property schemas, templates, and write-path validation

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If a
> "STOP conditions" item occurs, stop and report — do not improvise. When done,
> update this plan's status row in `plans/README.md`.
>
> **Drift check (run first)**:
> `git diff --stat 1c094a7..HEAD -- migrations/ internal/nodes/ internal/self/knowledge.md`
> Also confirm plan 164 is DONE (its `node_types` collection must already exist).
> If plan 164 is not landed, STOP — this plan depends on it.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: MED (adds a validation gate on the node write path)
- **Depends on**: `plans/164-object-type-registry.md`
- **Category**: architecture / migration
- **Planned at**: commit `1c094a7`, 2026-06-24

## Why this matters

Plan 164 made object *types* extensible, but a type is still just a name —
`props` is freeform JSON with no schema. Capacities/Anytype's strongest feature
is **typed properties per object type**: a `Book` has `author` (text) and `year`
(number); a `Person` has `birthday` (date). Without a schema, "objects" are
notes with a tag, and an AI can't reliably read or write structured fields. This
plan attaches a **property schema** and an optional **template** to each
`node_types` row, and validates a node's `props` against its type's schema on
write. This is what turns the spine into a real object database the model can
populate predictably — and it is the prerequisite for folding tasks (167) and
measures (168) in as first-class typed objects with `due`/`status`/`value_num`
as declared properties rather than ad-hoc JSON.

## Current state

- After plan 164: `node_types` collection holds `name`, `label`, `icon`,
  `born_status`, `system`, timestamps; `nodes.type` is an open `TextField`
  validated against the registry in `nodes.Create`
  (`internal/nodes/nodes.go`), and `internal/nodes/types.go` has
  `TypeExists` / `TypeNames` / `BornStatus` / `OwnerAuthoredTypes`.
- `internal/nodes/nodes.go` — props helpers exist (`Props`, `PropString`,
  `PropInt`, lines 47–81); `Create` (line 85) accepts `props map[string]any`
  and stores it on the `props` JSONField. There is **no validation** of props
  shape today.
- The node write path: nodes are created via `nodes.Create`, and edited via
  domain code (e.g. `internal/knowledge.UpdateFields`) and potentially the
  PocketBase dashboard. There is no PocketBase `OnRecordCreate`/`OnRecordUpdate`
  (pre-save) hook for nodes today — only post-success hooks in `main.go`
  (FTS upsert at lines 234–257, link sync at 266–275). Validation must run
  BEFORE the write to reject bad data.
- `internal/knowledge/knowledge.go` already hand-codes memory/skill props
  (`category`, `importance`, `when_to_use`, `description`) in `ProposeMemory`
  (line 117) / `ProposeSkill` (line 145). This plan does NOT rewrite those — it
  records their existing shape as the `memory`/`skill` type schemas so the
  registry documents reality, but the knowledge package keeps writing props as
  it does (validation must accept what it already writes).

### Conventions to match

- Migration: a new incremental file, same style as plan 164's
  (`m.Register(up, down)`, timestamp `> 164's`). Do NOT edit the baseline or
  164's migration — add a new one that ALTERs `node_types` to add columns and
  backfills the built-in schemas.
- Go: `gofmt`, `%w` error wrap, `app.Logger()` structured logs, table-driven
  tests. See `go-standards`.
- Validation should be a small pure function over (schema, props) so it is unit
  testable without a DB (`go-standards`: cover pure helpers directly).

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Pkg tests | `go test ./internal/nodes/ ./migrations/ ./internal/knowledge/` | PASS |
| Full suite | `go test ./...` | all ok |
| Format | `gofmt -l migrations/ internal/` | no output |

## Scope

**In scope**:
- `migrations/1750000010_node_type_schemas.go` (create — adjust timestamp to be
  `>` plan 164's file; keep `_node_type_schemas` suffix)
- `migrations/schema_test.go` (assert new `node_types` columns)
- `internal/nodes/schema.go` (create — property-schema types + validation)
- `internal/nodes/schema_test.go` (create)
- `internal/nodes/nodes.go` (call validation in `Create`; add a
  `CreateValidated`/template-applying entry point if needed — see Step 3)
- `internal/self/knowledge.md` (document property schemas + templates)

**Out of scope**:
- Relation/edge property types and AI link tools — that is plan 166.
- Folding tasks/measures in — plans 167/168 (they will *use* this schema).
- A UI for editing schemas — dashboard-only for v1 (note in maintenance).
- Rewriting `internal/knowledge`'s memory/skill prop handling — leave as-is;
  just ensure validation accepts its current output.
- Validating writes that come directly through the PocketBase REST/dashboard
  (those bypass `nodes.Create`). v1 validates the Go write path only; note the
  gap in maintenance. (A pre-save record hook is a possible later hardening.)

## Property model (the design to implement)

A property definition (one entry in a type's schema):

```go
// internal/nodes/schema.go
type PropType string
const (
    PropText   PropType = "text"
    PropNumber PropType = "number"
    PropDate   PropType = "date"   // RFC3339 string
    PropBool   PropType = "bool"
    PropSelect PropType = "select" // value must be one of Options
)

type PropDef struct {
    Key      string   `json:"key"`              // props map key, e.g. "author"
    Label    string   `json:"label"`            // display
    Type     PropType `json:"type"`
    Required bool     `json:"required"`
    Options  []string `json:"options,omitempty"` // for select
}
```

A type's `properties` column is `[]PropDef` (JSON). A type's `template` column
is `map[string]any` (JSON) — default `props` (and optionally a default `body`
under a reserved `"_body"` key) applied when creating a node of that type with
missing fields.

**Validation rule** (`ValidateProps(defs []PropDef, props map[string]any) error`):
- Every `Required` def must have a non-empty value in `props`.
- Every present value whose key matches a def must satisfy the def's type:
  number → `float64`/`int`; bool → `bool`; date → parseable RFC3339 string;
  select → string in `Options`; text → string.
- **Unknown keys are allowed** (forward-compatible; props may carry extras like
  `use_count`). Do NOT reject extra keys — `internal/knowledge` writes
  `use_count`/`last_used` that aren't declared.
- A type with an **empty** schema accepts any props (so `note`, `journal`, and
  any type the owner hasn't given properties, behave exactly as today).

This "empty schema = anything" rule is the safety valve that keeps every
existing node valid after this plan lands.

## Steps

### Step 1: Add `properties` + `template` columns to `node_types`

New migration `up`:
- `col, err := app.FindCollectionByNameOrId("node_types")` (STOP if not found —
  plan 164 isn't landed).
- Add two `JSONField`s: `properties` and `template`. `app.Save(col)`.
- `down` removes both fields (best-effort).

**Verify**: `go test ./migrations/ -run TestSchemaBaseline` compiles and runs
(assertions updated in Step 5).

### Step 2: Backfill built-in type schemas

In the same migration, after adding columns, set `properties` on the built-in
types to document their real shape (so the registry tells the truth and 167/168
have a pattern to copy). Load each `node_types` row by name and set
`properties`:

- `memory`: `[{key:"category",type:"select",options:["fact","preference","person","project","context"],required:true},{key:"importance",type:"number",required:true},{key:"when_to_use",type:"text"},{key:"source",type:"text"}]`
  (mirrors `knowledge.ProposeMemory`, `internal/knowledge/knowledge.go:117`).
- `skill`: `[{key:"description",type:"text"},{key:"when_to_use",type:"text"}]`
  (mirrors `ProposeSkill`, line 145).
- `note`, `journal`, `person`, `book`, `idea`, `place`: leave `properties`
  empty/null for now (empty schema = accepts anything; the owner can enrich
  `book`/`person` later via the dashboard). Optionally seed a couple of obvious
  `book` props (`author` text, `year` number) as a demonstration — keep it
  minimal and note it.

Use `app.Save(rec)` per row. Do not fail the migration if a built-in row is
missing (log a warning) — be defensive.

**Verify**: after `go test ./migrations/`, a quick check that `memory`'s
`properties` is non-empty (add an assertion in Step 5).

### Step 3: Implement schema types + validation in `internal/nodes/schema.go`

- Define `PropType`, `PropDef`, the consts, and:
  - `ValidateProps(defs []PropDef, props map[string]any) error` — pure, as
    specified above. Return a clear error naming the offending key and reason.
  - `TypeSchema(app core.App, typ string) ([]PropDef, error)` — load the
    `node_types` row for `typ`, unmarshal `properties` (empty slice if null).
  - `TypeTemplate(app core.App, typ string) (map[string]any, error)` — load and
    unmarshal `template` (nil if absent).
  - `ApplyTemplate(tmpl map[string]any, body string, props map[string]any) (string, map[string]any)`
    — fill missing `props` keys from `tmpl`, and if `body` is empty and the
    template has `"_body"`, use it. Pure function.

Wire validation into `nodes.Create` (`nodes.go:85`): after the existing type
check, load the schema and call `ValidateProps`; on error, return it
(`fmt.Errorf("nodes: invalid props for type %q: %w", typ, err)`). Apply the
template before validation so required-with-default fields pass.

> Keep `Create`'s signature. If applying templates needs the body too, mutate
> the local `body`/`props` before the `rec.Set` calls — do not add params.

**Verify**: `go test ./internal/nodes/ -v` → pass (including schema_test.go).

### Step 4: Confirm the knowledge package still writes valid props

`internal/knowledge.ProposeMemory`/`ProposeSkill` call `nodes.Create`, which now
validates. Run the knowledge tests:

**Verify**: `go test ./internal/knowledge/ -v` → all pass. If a memory/skill
proposal now fails validation, the seeded schema in Step 2 is wrong — fix the
schema to match what `knowledge` actually writes (the code is the source of
truth; do NOT change `knowledge`). This is the key compatibility gate.

### Step 5: Update `migrations/schema_test.go`

- Assert `node_types` now has `properties` and `template` fields.
- Assert the `memory` type row has a non-empty `properties` (load it, unmarshal,
  len > 0).

**Verify**: `go test ./migrations/ -v` → PASS.

### Step 6: Update `internal/self/knowledge.md`

Add a sentence to the nodes section: each registered type may declare a typed
**property schema** (text/number/date/bool/select) and an optional **template**;
node writes validate `props` against the type's schema (empty schema accepts
anything), bringing Balaur in line with Capacities/Anytype typed objects.

**Verify**: `grep -n "property schema\|template" internal/self/knowledge.md` →
hit.

### Step 7: Full gate

**Verify**:
```
CGO_ENABLED=0 go build ./... && go vet ./... && go test ./... && gofmt -l migrations/ internal/ && git diff --check
```
All exit 0; no files listed by `gofmt -l`; `git diff --check` empty.

## Test plan

- `internal/nodes/schema_test.go` (pure-function tests, no DB needed for
  `ValidateProps`/`ApplyTemplate`):
  - `ValidateProps` happy path: a book with `author` string, `year` 2020 passes.
  - Required missing → error naming the key.
  - Wrong type (year = `"twenty"`) → error.
  - Select value not in options → error.
  - Unknown extra key (`use_count`) → allowed (no error).
  - Empty schema → any props pass.
  - `ApplyTemplate` fills a missing key, leaves a present key untouched, fills
    empty body from `_body`.
- DB-backed (use `storetest.NewApp(t)`): create a `book` node missing a required
  prop → `nodes.Create` errors; with the prop → succeeds.
- **Wikilink stub regression** (the interaction to guard): `[[Some Title]]` in a
  node body auto-creates a `type=note` stub via `resolveOrCreateStub`
  (`internal/nodes/links.go`). `note` has an empty schema → must still validate
  and save fine. Add/confirm a test that a body with a `[[wikilink]]` still syncs
  (creates the stub) after validation lands. If `note` ever gets a required
  property, stub creation would break — that's why `note` must keep an empty
  schema (note this in maintenance).
- `migrations/schema_test.go`: Step 5.
- Compatibility: `go test ./internal/knowledge/` proves memory/skill still write.
- Verification: `go test ./...` → all pass including new tests.

## Done criteria

ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0; new `schema_test.go` tests exist and pass
- [ ] `internal/knowledge` tests pass unchanged (compatibility proven)
- [ ] Creating a typed node with a required prop missing returns an error
      (validation works); empty-schema types still accept any props
- [ ] `migrations/1749600000_init.go` and plan 164's migration are unmodified
- [ ] `gofmt -l migrations/ internal/` prints nothing
- [ ] `plans/README.md` status row for 165 updated

## STOP conditions

- Plan 164 is not landed (`node_types` collection absent).
- A memory/skill proposal fails validation and the fix would require changing
  `internal/knowledge` (it must not — adjust the seeded schema instead; if the
  real shape can't be expressed by the PropDef model, report it).
- Drift-check mismatch with the "Current state" excerpts.
- A verification fails twice after a reasonable fix.

## Maintenance notes

- **Plans 167/168** register `task`/`measure` types with a full property schema
  (status/due/recur, and value_num/unit/kind respectively) — copy the Step 2
  backfill pattern.
- **Deferred**: validating REST/dashboard-direct writes (a pre-save
  `OnRecordCreate`/`OnRecordUpdate` hook on `nodes`) — v1 validates only the Go
  `nodes.Create` path. If the owner starts editing `props` via the dashboard and
  wants the same guard, add the pre-save hook then.
- **Deferred**: an owner-facing schema editor UI; the dashboard (`/_/`) edits
  `node_types.properties` JSON for now.
- **Reviewer should scrutinize**: the "empty schema accepts anything" rule (it
  is what keeps every pre-existing node valid) and the `internal/knowledge`
  compatibility (memory/skill must keep writing without error).
