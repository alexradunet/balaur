# Plan 191: Let node_write accept typed props and add a node_edit verb

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 12a48bf..HEAD -- internal/tools/knowledge.go internal/tools/graph.go internal/nodes/nodes.go internal/nodes/schema.go internal/cli/knowledge.go internal/self/knowledge.md`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW
- **Depends on**: none
- **Category**: direction
- **Planned at**: commit `12a48bf`, 2026-06-24

## Why this matters

Balaur's "typed object" knowledge spine (plans 164–174) gives node types a
property schema (`book` has `author`/`year`, `memory` has `importance`, etc.),
and the `node_schema` tool literally tells the model "Read this before writing a
typed node **so you supply the right props**." But `node_write` has **no `props`
parameter** — it calls `nodes.Create(app, typ, title, body, StatusActive, nil)`
with a hard-coded `nil`. So every agent-authored person/book/idea/place is born
with **empty schema'd properties**, and there is **no `node_edit` verb** to set
them afterward (only memory/skill can be edited, via the consent-gated
`propose_edit`). The typed half of the typed-object spine is unreachable through
tools — this is nearly a bug: the model is instructed to gather props it cannot
persist.

`nodes.Create` already accepts and validates props against the type schema
(`internal/life/life.go:100` passes props to it). This plan closes the gap on
the **tool surface only**: add `props` to `node_write`, add a `node_edit` verb
(load active node → update title/body/props in place → validate → save → audit),
and mirror both in the CLI gateway for harness parity. Owner-authored types are
born active and trusted, so there is **no consent gate** — unlike memory/skill,
whose `propose_edit` path stays exactly as it is.

## Current state

### Files and roles

- `internal/tools/knowledge.go` — the model's knowledge verbs. Holds
  `nodeWriteTool` (lines 321–362) which passes `nil` props, and `proposeEditTool`
  (the consent-gated edit shape for memory/skill, lines 252–317). The `obj()` /
  `str()` JSON-schema spec helpers live in `internal/tools/os.go:59` and `:69`.
- `internal/tools/graph.go` — `nodeSchemaTool` (lines 182–226), the registry
  introspection that tells the model to supply props; and `nodeLinkTool`
  (lines 27–82), the canonical "load nodes, require active, mutate, audit"
  pattern to mirror.
- `internal/nodes/nodes.go` — `Create` (lines 89–138) does template-apply →
  `ValidateProps` → save → `store.Audit("owner", "node.create", …)`. There is
  **no `Update`/`Edit`/`Set` function** — node_edit must add one (see Step 1).
  `Get` (lines 141–147), `Props` (49–59), `PropString` (62–69), `StatusActive`
  constant (line 28).
- `internal/nodes/schema.go` — `ValidateProps(defs, props)` (lines 40–63),
  `TypeSchema(app, typ)` (67–84). `book`'s schema is `author` (text) + `year`
  (number, **not** required); only `memory` has a Required typed prop, and memory
  is consent-gated (not owner-authored), so the invalid-props test uses a **type
  mismatch** on `book`, not a missing-required field (see Test plan).
- `internal/cli/knowledge.go` — the CLI mirror. `noteCmd` (285–292) groups
  `note add/list/show/drop`; `noteAddCmd` (294–316) calls
  `nodes.Create(... nil)`. `ownerNodeTypes` (line 18) =
  `["note","person","book","idea","place"]`. `memoryEditCmd` (184–219) is the
  per-field `--flag`/`Changed` edit pattern to mirror.
- `internal/self/knowledge.md` — the running binary's self-description; the node
  verbs are enumerated at lines 180–191.

### Key excerpts (verbatim — confirm these before editing)

`internal/tools/knowledge.go:321-362` — `nodeWriteTool` as it stands (the `nil`
props and the missing `props` spec param):

```go
// nodeWriteTool creates an owner-authored node (note or typed object). Unlike
// remember/propose_skill these are born active — owner-voiced, trusted writes.
func nodeWriteTool(app core.App) agent.Tool {
	allowedTypes, err := nodes.OwnerAuthoredTypes(app)
	if err != nil || len(allowedTypes) == 0 {
		app.Logger().Warn("node_write: could not load owner-authored types from registry; falling back to [note]", "error", err)
		allowedTypes = []string{"note"}
	}
	return agent.Tool{
		Spec: agent.ToolSpecOf("node_write",
			"Write an owner-authored knowledge node — a note or a typed object (person, book, idea, place). "+
				"Born active (the owner's own, trusted). For things you want the owner to APPROVE as a memory, use remember instead.",
			obj(map[string]any{
				"type":  map[string]any{"type": "string", "enum": allowedTypes, "description": "Node type (default note)."},
				"title": str("Short title for the node."),
				"body":  str("The node's markdown body."),
			}, "title")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Type  string `json:"type"`
				Title string `json:"title"`
				Body  string `json:"body"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("node_write: bad arguments: %w", err)
			}
			if strings.TrimSpace(args.Title) == "" {
				return "", fmt.Errorf("node_write: title is required")
			}
			typ := args.Type
			if typ == "" {
				typ = "note"
			}
			if !slices.Contains(allowedTypes, typ) {
				return "", fmt.Errorf("node_write: type %q is not an owner-authored type", typ)
			}
			rec, err := nodes.Create(app, typ, args.Title, args.Body, nodes.StatusActive, nil)
			if err != nil {
				return "", fmt.Errorf("node_write: %w", err)
			}
			return fmt.Sprintf("Saved %s %q (id %s).", typ, args.Title, rec.Id), nil
		},
	}
}
```

`internal/nodes/nodes.go:89-138` — `Create` (the template→validate→save→audit
pattern `Update` must mirror):

```go
func Create(app core.App, typ, title, body, status string, props map[string]any) (*core.Record, error) {
	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("nodes: title is required")
	}
	ok, err := TypeExists(app, typ)
	if err != nil {
		return nil, fmt.Errorf("nodes: checking type: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("nodes: unknown type %q (not in node_types registry)", typ)
	}

	// Apply template defaults before validation so required-with-default fields pass.
	tmpl, err := TypeTemplate(app, typ)
	if err != nil {
		return nil, fmt.Errorf("nodes: loading template for %q: %w", typ, err)
	}
	if props == nil {
		props = map[string]any{}
	}
	body, props = ApplyTemplate(tmpl, body, props)

	// Validate props against the type's schema (empty schema = any props ok).
	defs, err := TypeSchema(app, typ)
	if err != nil {
		return nil, fmt.Errorf("nodes: loading schema for %q: %w", typ, err)
	}
	if err := ValidateProps(defs, props); err != nil {
		return nil, fmt.Errorf("nodes: invalid props for type %q: %w", typ, err)
	}

	col, err := app.FindCollectionByNameOrId("nodes")
	if err != nil {
		return nil, fmt.Errorf("finding nodes collection: %w", err)
	}
	rec := core.NewRecord(col)
	rec.Set("type", typ)
	rec.Set("title", title)
	rec.Set("body", body)
	rec.Set("status", status)
	if len(props) > 0 {
		rec.Set("props", props)
	}
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("saving node: %w", err)
	}
	store.Audit(app, "owner", "node.create", "nodes/"+rec.Id, true,
		map[string]any{"type": typ, "title": title})
	return rec, nil
}
```

`internal/cli/knowledge.go:294-316` — `noteAddCmd` (the CLI mirror that passes
`nil`):

```go
func noteAddCmd(app core.App) *cobra.Command {
	var typ, title, body string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create an owner-authored node (note or typed object), born active",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().StringVar(&typ, "type", "note", "note | person | book | idea | place")
	cmd.Flags().StringVar(&title, "title", "", "node title (required)")
	cmd.Flags().StringVar(&body, "body", "", "node markdown body")
	_ = cmd.MarkFlagRequired("title")
	cmd.RunE = run(app, "note.add", func(cmd *cobra.Command, args []string) (any, error) {
		if !slices.Contains(ownerNodeTypes, typ) {
			return nil, fmt.Errorf("type %q is not an owner-authored node type", typ)
		}
		rec, err := nodes.Create(app, typ, title, body, nodes.StatusActive, nil)
		if err != nil {
			return nil, err
		}
		return nodeJSON(rec), nil
	})
	return cmd
}
```

`internal/self/knowledge.md:180-191` — the node-verb prose to update:

```
  skill node's procedure. node_write creates owner-authored nodes — a note
  or a typed object (person, book, idea, place), born active; node_list,
  node_get, and node_drop list, read, and delete them. node_get now also
  returns the node's props and a one-line link summary (N outbound, M backlinks).
  Four graph verbs let you build and walk the object graph — all consent-filtered
  to active nodes only (proposed/rejected never surface):
  node_schema discovers registered types and their property schemas (read before
  writing a typed node); node_link asserts a typed relation between two active
  nodes (default relation "relates_to" — agent-asserted; "links" is reserved
  for wikilink-origin edges; idempotent); node_related returns 1-hop neighbours
  (direction=both/out/in); node_query searches active nodes by type and/or
```

### Conventions that apply here (with exemplars)

- **Domain packages own their PocketBase writes.** PB save+audit for a node
  belongs in `internal/nodes`, not the tool/CLI layer. `nodes.Create` is the
  exemplar; the new `nodes.Update` (Step 1) mirrors it. Tools/CLI call the nodes
  function — they do **not** call `app.Save` directly.
- **Audit STRICTLY AFTER a successful write**, never before — `store.Audit(app,
  "owner", "node.update", "nodes/"+rec.Id, true, …)` runs only after `app.Save`
  returns nil. See `Create` line 132–136 and `nodeLinkTool` → `AddEdge`
  (`internal/nodes/nodes.go:207-218`).
- **Errors are values**: `return "", fmt.Errorf("node_edit: %w", err)`, return
  early, no panics. Exemplar: every `Execute` in `knowledge.go`/`graph.go`.
- **gomponents/g.Text escaping rule** does not apply here (no UI change).
- **JSON-schema spec helpers**: `obj(props, required...)` and `str(desc)` from
  `internal/tools/os.go`. A free-form object param is declared as
  `map[string]any{"type": "object", "description": "…"}` (no fixed `properties`)
  — there is no existing `obj`-nested-object exemplar in this repo, so declare
  the `props` schema literally as shown in Step 2.
- **Structured logging only**: `app.Logger().Warn(...)` with key/value pairs
  (see `nodeWriteTool` line 324). No `fmt.Print*`.
- **CLI v1 envelope**: every command body returns `(any, error)` wrapped by
  `run(app, "<kind>", …)`; the kind string is `<command>.<subcommand>` e.g.
  `"note.edit"`. See `noteAddCmd` (`run(app, "note.add", …)`).
- This is a **tool/CLI surface change with one new nodes helper — NO migration,
  NO schema change.** The `props` JSONField on `nodes` and every type schema
  already exist. Do **not** add a migration file.

## Commands you will need

| Purpose          | Command                                  | Expected on success      |
|------------------|------------------------------------------|--------------------------|
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...`           | exit 0, no output        |
| Test (tools)     | `go test ./internal/tools/`              | `ok` / all pass          |
| Test (cli)       | `go test ./internal/cli/`                | `ok` / all pass          |
| Test (nodes)     | `go test ./internal/nodes/`              | `ok` / all pass          |
| Test (all)       | `go test ./...`                          | all pass                 |
| Vet              | `go vet ./...`                           | exit 0, no output        |
| Format check     | `gofmt -l .`                             | empty output (no files)  |
| Whitespace check | `git diff --check`                       | exit 0, no output        |

(`gofmt` is enforced by a PostToolUse hook + CI gate; `go vet`/`staticcheck`/
`govulncheck` gate CI. Tests are standard `testing`, table-driven where it
helps, **no** assertion frameworks, **no** `time.Sleep`. Use
`storetest.NewApp(t)` — `internal/storetest` — for a PocketBase-backed test app:
it boots the full migration chain so all node types and schemas are seeded.)

## Suggested executor toolkit

- Invoke the `go-standards` skill if available before writing Go — it covers the
  error-wrapping, slog, records-as-domain-model, audit-after-save, and testing
  idioms this plan relies on.
- Before reading any source file, the repo hook requires orienting with
  graphify first: `graphify query "node_write props node_edit"` then open the
  cited files to confirm the excerpts above.

## Scope

**In scope** (the only files you should modify):

- `internal/nodes/nodes.go` — add one new `Update` function (Step 1).
- `internal/tools/knowledge.go` — `props` on `node_write`; new `nodeEditTool`;
  register it in `KnowledgeTools` (Steps 2–3).
- `internal/tools/knowledge_test.go` — new test cases (Step 6).
- `internal/cli/knowledge.go` — `--props` on `note add`; new `note edit` verb
  (Steps 4–5).
- `internal/cli/cli_test.go` — new test case (Step 6).
- `internal/nodes/nodes_test.go` — new `Update` test (Step 6). Create this file
  if it does not exist; if it does, append.
- `internal/self/knowledge.md` — update the node-verb prose (Step 7).

**Out of scope** (do NOT touch, even though they look related):

- `internal/knowledge/*` and `proposeEditTool` — memory/skill keep their
  **consent-gated** `propose_edit` path. node_edit is for owner-authored,
  born-active types ONLY; it must never edit a memory or skill node.
- `internal/nodes/schema.go` validation internals — **reuse** `ValidateProps`
  and `TypeSchema` as-is; do not rewrite them.
- `migrations/*` — no migration. The `props` field and type schemas exist.
- `internal/nodes/types.go` (the node_types registry) — leave the registry
  alone.
- The web UI / note card / storybook — no rendering change in this plan.

## Git workflow

- Branch: `advisor/191-node-authoring-props-edit` (executors typically run in a
  worktree off `origin/main`).
- Commit per logical unit; conventional-commit subjects. Example from `git log`:
  `feat(knowledge): node_write accepts typed props + node_edit verb`. A
  reasonable split: one `feat(nodes)` commit for the `Update` helper, one
  `feat(tools)` for the tool surface + tests, one `feat(cli)` for CLI parity,
  one `docs(self)` for knowledge.md.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Add `nodes.Update` (load active node, update fields, validate, save, audit)

In `internal/nodes/nodes.go`, add a new exported function `Update` directly
**after `Create`** (after line 138). It is the in-place sibling of `Create` and
keeps the PB write + audit inside the nodes package (domain-owns-its-writes).

Target shape:

```go
// Update edits an existing ACTIVE node's title, body, and/or props in place and
// audits node.update after the write. Only non-nil arguments change a field:
// title/body are pointers (nil = leave unchanged, "" = clear); props, when
// non-nil, REPLACES the node's props (after template-apply + schema validation).
// Returns an error if the node is missing or not active — owner-authored typed
// nodes are born active, so a non-active node is not an editable owner object.
func Update(app core.App, id string, title, body *string, props map[string]any) (*core.Record, error) {
	rec, err := app.FindRecordById("nodes", strings.TrimSpace(id))
	if err != nil {
		return nil, fmt.Errorf("nodes: no node %q", id)
	}
	if rec.GetString("status") != StatusActive {
		return nil, fmt.Errorf("nodes: node %q is not active (status=%s)", id, rec.GetString("status"))
	}
	typ := rec.GetString("type")

	if title != nil {
		if strings.TrimSpace(*title) == "" {
			return nil, fmt.Errorf("nodes: title cannot be cleared")
		}
		rec.Set("title", *title)
	}
	if body != nil {
		rec.Set("body", *body)
	}
	if props != nil {
		// Validate the replacement props against the type's schema, mirroring
		// Create (template-apply first so required-with-default fields pass).
		tmpl, err := TypeTemplate(app, typ)
		if err != nil {
			return nil, fmt.Errorf("nodes: loading template for %q: %w", typ, err)
		}
		_, merged := ApplyTemplate(tmpl, rec.GetString("body"), props)
		defs, err := TypeSchema(app, typ)
		if err != nil {
			return nil, fmt.Errorf("nodes: loading schema for %q: %w", typ, err)
		}
		if err := ValidateProps(defs, merged); err != nil {
			return nil, fmt.Errorf("nodes: invalid props for type %q: %w", typ, err)
		}
		rec.Set("props", merged)
	}

	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("saving node: %w", err)
	}
	store.Audit(app, "owner", "node.update", "nodes/"+rec.Id, true,
		map[string]any{"type": typ})
	return rec, nil
}
```

Notes for the executor:
- `ApplyTemplate` is a pure function (does not mutate inputs) — see
  `internal/nodes/schema.go:114`. We pass the current body only to satisfy its
  signature; the merged props is what we keep.
- Required imports (`strings`, `fmt`, `store`, `core`) are already imported in
  this file — no new imports.

**Verify**: `CGO_ENABLED=0 go build ./internal/nodes/` → exit 0.

### Step 2: Add a `props` parameter to `node_write`

In `internal/tools/knowledge.go`, edit `nodeWriteTool` (lines 321–362):

1. Add a `props` field to the spec object passed to `obj(...)` — a free-form
   object (no fixed sub-schema, so the model may pass any key→value the type
   schema expects):

   ```go
   "props": map[string]any{"type": "object", "description": "Optional typed properties for the node, keyed by the type's schema (call node_schema first to learn the keys and value-types)."},
   ```

   Keep `"title"` as the only required key (third arg to `obj`).

2. Add `Props map[string]any` to the `args` struct with tag `` `json:"props"` ``.

3. Replace the `nil` in the `nodes.Create` call with `args.Props`, and when the
   create fails with a validation error, steer the model to `node_schema`:

   ```go
   rec, err := nodes.Create(app, typ, args.Title, args.Body, nodes.StatusActive, args.Props)
   if err != nil {
       return "", fmt.Errorf("node_write: %w — call node_schema %q to see the required props and value-types", err, typ)
   }
   ```

   (`nodes.Create` already runs `ValidateProps`; its error text reads
   `invalid props for type "book": …`, so the wrapped message points the model
   at the fix.)

**Verify**: `CGO_ENABLED=0 go build ./internal/tools/` → exit 0.

### Step 3: Add the `nodeEditTool` verb and register it

In `internal/tools/knowledge.go`, add a new tool `nodeEditTool` (place it right
after `nodeWriteTool`, before `nodeListTool`). It loads an active node and
updates title/body/props via `nodes.Update`. **No consent gate** — owner
types are born active and trusted (unlike `propose_edit`, which is for
memory/skill).

Target shape (note the pointer args so "omitted" differs from "set to empty"):

```go
// nodeEditTool updates an owner-authored node's title, body, and/or props in
// place. Owner types are born active and trusted, so there is no consent gate
// (unlike propose_edit, which parks memory/skill changes for approval). To
// revise a memory or skill, use propose_edit instead.
func nodeEditTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("node_edit",
			"Edit an existing owner-authored node (note or typed object) in place by id — "+
				"set its title, body, or typed props. Takes effect immediately (owner-authored, trusted). "+
				"Call node_schema first to learn a typed node's prop keys. To change a memory or skill, use propose_edit instead.",
			obj(map[string]any{
				"id":    str("Id of the active node to edit."),
				"title": str("Optional: new title."),
				"body":  str("Optional: new markdown body."),
				"props": map[string]any{"type": "object", "description": "Optional: replacement typed properties, keyed by the type's schema (call node_schema first)."},
			}, "id")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				ID    string          `json:"id"`
				Title *string         `json:"title"`
				Body  *string         `json:"body"`
				Props map[string]any  `json:"props"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("node_edit: bad arguments: %w", err)
			}
			id := strings.TrimSpace(args.ID)
			if id == "" {
				return "", fmt.Errorf("node_edit: id is required")
			}
			if args.Title == nil && args.Body == nil && args.Props == nil {
				return "", fmt.Errorf("node_edit: nothing to edit — pass title, body, or props")
			}
			rec, err := nodes.Update(app, id, args.Title, args.Body, args.Props)
			if err != nil {
				return "", fmt.Errorf("node_edit: %w", err)
			}
			return fmt.Sprintf("Updated %s %q (id %s).", rec.GetString("type"), rec.GetString("title"), rec.Id), nil
		},
	}
}
```

Then register it in `KnowledgeTools` (lines 24–35) immediately after
`nodeWriteTool(app),`:

```go
		nodeWriteTool(app),
		nodeEditTool(app),
		nodeListTool(app),
```

**Verify**: `CGO_ENABLED=0 go build ./internal/tools/` → exit 0.

### Step 4: Add `--props` to the CLI `note add`

In `internal/cli/knowledge.go`, edit `noteAddCmd` (lines 294–316) to accept a
`--props` JSON flag and pass the decoded map to `nodes.Create`:

1. Add a `propsJSON` string var and flag:
   ```go
   cmd.Flags().StringVar(&propsJSON, "props", "", `typed properties as a JSON object, e.g. '{"author":"Le Guin","year":1969}'`)
   ```
2. In the `RunE` body, after the `ownerNodeTypes` check, decode it (empty = nil):
   ```go
   props, err := decodeProps(propsJSON)
   if err != nil {
       return nil, err
   }
   rec, err := nodes.Create(app, typ, title, body, nodes.StatusActive, props)
   ```
3. Add a small shared helper near the top of the file (after `ownerNodeTypes`):
   ```go
   // decodeProps parses a --props JSON object flag. An empty string yields nil
   // (no props); anything else must be a JSON object.
   func decodeProps(s string) (map[string]any, error) {
       if strings.TrimSpace(s) == "" {
           return nil, nil
       }
       var m map[string]any
       if err := json.Unmarshal([]byte(s), &m); err != nil {
           return nil, fmt.Errorf("--props must be a JSON object: %w", err)
       }
       return m, nil
   }
   ```
   Add `"encoding/json"` and `"strings"` to the import block (currently `fmt`,
   `slices`, `strconv`, plus the pocketbase/cobra/internal imports — confirm
   `strings` and `encoding/json` are NOT already there; add only the missing
   ones).

**Verify**: `CGO_ENABLED=0 go build ./internal/cli/` → exit 0.

### Step 5: Add the CLI `note edit` verb

In `internal/cli/knowledge.go`, add `noteEditCmd` (model it on `memoryEditCmd`,
lines 184–219, but call `nodes.Update`), and register it in `noteCmd` (line
290):

```go
func noteEditCmd(app core.App) *cobra.Command {
	var title, body, propsJSON string
	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit an owner-authored node's title, body, or props in place",
		Args:  cobra.ExactArgs(1),
	}
	cmd.Flags().StringVar(&title, "title", "", "new title")
	cmd.Flags().StringVar(&body, "body", "", "new markdown body")
	cmd.Flags().StringVar(&propsJSON, "props", "", `replacement typed properties as a JSON object`)
	cmd.RunE = run(app, "note.edit", func(cmd *cobra.Command, args []string) (any, error) {
		var titlePtr, bodyPtr *string
		if cmd.Flags().Changed("title") {
			titlePtr = &title
		}
		if cmd.Flags().Changed("body") {
			bodyPtr = &body
		}
		var props map[string]any
		if cmd.Flags().Changed("props") {
			p, err := decodeProps(propsJSON)
			if err != nil {
				return nil, err
			}
			props = p
		}
		if titlePtr == nil && bodyPtr == nil && props == nil {
			return nil, fmt.Errorf("nothing to edit: pass --title, --body, or --props")
		}
		rec, err := nodes.Update(app, args[0], titlePtr, bodyPtr, props)
		if err != nil {
			return nil, err
		}
		return nodeJSON(rec), nil
	})
	return cmd
}
```

Register it in `noteCmd`:

```go
	cmd.AddCommand(noteAddCmd(app), noteListCmd(app), noteShowCmd(app), noteEditCmd(app), noteDropCmd(app))
```

Note: `--props` with `Changed("props")` and an empty value decodes to `nil`,
which `nodes.Update` treats as "do not touch props". That is intended: clearing
all props is not a CLI need here (YAGNI). The `nothing to edit` guard catches a
no-flag call.

**Verify**: `CGO_ENABLED=0 go build ./internal/cli/` → exit 0.

### Step 6: Write the tests

#### `internal/nodes/nodes_test.go` (create or append)

Add `TestUpdateValidatesAndAudits` and `TestUpdateRejectsNonActive`. Use
`storetest.NewApp(t)` (`github.com/alexradunet/balaur/internal/storetest`).

- **Happy path**: `Create` a `book` with `{"author":"A","year":1969.0}`
  (JSON numbers are float64; pass `1969.0` or `float64(1969)`), then `Update`
  with a new title pointer and props `{"author":"B","year":1970.0}`. Assert the
  returned record's title changed and `nodes.PropString(rec, "author") == "B"`.
  Then assert a `node.update` audit row exists:
  ```go
  audits, _ := store.ListAudit(app, "node.update", "owner", 10)
  if len(audits) == 0 { t.Fatal("expected a node.update audit row") }
  ```
  (import `github.com/alexradunet/balaur/internal/store`).
- **Non-active reject**: `Create` a `note` with `StatusProposed`, then `Update`
  it — expect an error whose text contains `"not active"`.
- **Invalid props reject**: `Create` a `book` (active), then `Update` with props
  `{"year":"not-a-number"}` — expect an error containing `"invalid props"`.

#### `internal/tools/knowledge_test.go` (append)

Model on the existing `TestNodeWriteToolCreatesActiveNode` (lines 44–66) and
`TestNodeGetIncludesPropsAndLinkSummary` (graph_test.go lines 227–250).

- `TestNodeWriteToolPersistsProps`: execute `nodeWriteTool` with
  `{"type":"book","title":"LHoD","body":"","props":{"author":"Le Guin","year":1969}}`,
  then load the node by filter and assert
  `nodes.PropString(rec, "author") == "Le Guin"`.
- `TestNodeWriteToolRejectsInvalidProps`: execute with
  `{"type":"book","title":"X","props":{"year":"nineteen"}}` (string where the
  schema wants number) and assert the returned `err` is non-nil and
  `strings.Contains(err.Error(), "node_schema")`.
- `TestNodeEditToolUpdatesProps`: `nodes.Create` an active `book`, then execute
  `nodeEditTool` with `{"id":"<id>","title":"New","props":{"author":"Z","year":2000}}`;
  reload and assert title == "New" and `PropString(rec,"author") == "Z"`.
- `TestNodeEditToolRejectsMissingNode`: execute `nodeEditTool` with
  `{"id":"nonexistentid","title":"x"}` and assert `err != nil`.
- Extend `TestKnowledgeToolsIncludesGraphVerbs` **or** add a focused test that
  `KnowledgeTools(app)` contains a tool named `node_edit`.

The `book` type is seeded by the migration chain (`schema_test.go:131` lists it),
so `storetest.NewApp(t)` has it. The invalid-props test must use a **type
mismatch** (string for the `year` number), NOT a missing-required field —
owner-authored types have no Required typed props (only `memory` does, and memory
is not owner-authored; see STOP conditions).

#### `internal/cli/cli_test.go` (append)

Model on `TestTaskLifecycle` (lines 89–136) and `TestMemoryConsentLifecycle`.

- `TestNoteAddAndEditWithProps`:
  - `execute(t, noteCmd(app), "add", "--type", "book", "--title", "LHoD",
    "--props", `{"author":"Le Guin","year":1969}`)` → capture `id`.
  - `execute(t, noteCmd(app), "edit", id, "--title", "The Left Hand of Darkness",
    "--props", `{"author":"Le Guin","year":1969}`)` → assert the returned title
    changed.
  - Reload via `app.FindRecordById("nodes", id)` and assert
    `nodes.PropString(rec, "author") == "Le Guin"`.
  - Optionally assert a `note.edit` audit row via
    `executeList(t, auditCmd(app), "--action", "node.update")` length ≥ 1.

**Verify**: `go test ./internal/nodes/ ./internal/tools/ ./internal/cli/` →
all pass, including the new cases.

### Step 7: Update self-knowledge

In `internal/self/knowledge.md`, edit the node-verb prose (lines 180–191) so it
reflects the new capability. Minimal change: in the sentence that currently ends
"…node_list, node_get, and node_drop list, read, and delete them.", state that
`node_write` now accepts typed props and that `node_edit` updates an
owner-authored node's title/body/props in place (born active, no consent gate;
memory/skill still go through `propose_edit`). Keep it to one or two sentences —
this file is injected into agent context, stay lean and high-signal.

Example insertion (adapt wording to the surrounding paragraph):

```
  ... node_write creates owner-authored nodes — a note or a typed object
  (person, book, idea, place), born active, and now accepts typed props
  validated against the type schema; node_edit updates an owner-authored
  node's title, body, or props in place (no consent gate — owner types are
  trusted; memory/skill changes still go through propose_edit); node_list,
  node_get, and node_drop list, read, and delete them. ...
```

**Verify**: `gofmt -l .` is empty (knowledge.md is not Go, but this confirms no
Go file regressed) and `git diff internal/self/knowledge.md` shows only the
node-verb prose changed.

### Step 8: Full validation

Run the whole gate.

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `gofmt -l .` → empty
- `git diff --check` → exit 0
- `go test ./...` → all pass (note: the full link can need
  `TMPDIR=/home/alex/.cache/go-tmp` if the box's `/tmp` is a small tmpfs and the
  link fails "No space left on device" — set it only if that error appears).

## Test plan

New tests, by file:

- `internal/nodes/nodes_test.go`: `TestUpdateValidatesAndAudits` (happy path +
  audit row), `TestUpdateRejectsNonActive`, plus an invalid-props case.
- `internal/tools/knowledge_test.go`: `TestNodeWriteToolPersistsProps`,
  `TestNodeWriteToolRejectsInvalidProps` (error mentions `node_schema`),
  `TestNodeEditToolUpdatesProps`, `TestNodeEditToolRejectsMissingNode`, and a
  registration assertion that `node_edit` is in `KnowledgeTools`.
- `internal/cli/cli_test.go`: `TestNoteAddAndEditWithProps`.

Structural patterns to copy:
- Tool tests → `TestNodeWriteToolCreatesActiveNode`
  (`internal/tools/knowledge_test.go:44`) and `TestNodeGetIncludesPropsAndLinkSummary`
  (`internal/tools/graph_test.go:227`).
- Nodes tests → the `Create(... map[string]any{...})` call shape used in
  `graph_test.go:229`.
- CLI tests → `TestTaskLifecycle` (`internal/cli/cli_test.go:89`) for the
  add→mutate→reload→audit arc.

Edge cases explicitly covered: invalid props is rejected and the error steers to
`node_schema`; editing a non-active/missing node errors cleanly; props round-trip
(`PropString` reads back what was written); a `node.update` audit row is written
only after the save.

Verification: `go test ./internal/nodes/ ./internal/tools/ ./internal/cli/` →
all pass, including the new cases; `go test ./...` → all pass.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `gofmt -l .` prints nothing
- [ ] `git diff --check` exits 0
- [ ] `go test ./...` passes, including the new tests in
  `internal/nodes/`, `internal/tools/`, and `internal/cli/`
- [ ] `node_write` spec has a `props` object param and passes it to
  `nodes.Create` (no hard-coded `nil`)
- [ ] `node_edit` is registered in `KnowledgeTools` (a typed node's props can be
  set via `node_write` AND changed later via `node_edit`)
- [ ] The CLI has `balaur note edit <id> --props …` and `note add --props …`
- [ ] `internal/self/knowledge.md` describes `node_write` props + `node_edit`
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- The code at the "Current state" locations does not match the excerpts (the
  codebase drifted) — in particular if `nodes.Create`'s signature is no longer
  `Create(app, typ, title, body, status string, props map[string]any)`, or
  `ValidateProps`/`TypeSchema`/`ApplyTemplate` have moved or changed signature.
  Re-read `internal/nodes/nodes.go` and `internal/nodes/schema.go` and adapt;
  if the props-validation contract genuinely differs, STOP.
- A `nodes.Update` (or `Edit`/`Set`) function **already exists** when you go to
  add it in Step 1 — reuse it instead of adding a duplicate, and note the
  divergence in your report.
- It turns out owner-authored typed nodes **require a consent gate** (i.e.
  `nodes.OwnerAuthoredTypes` no longer returns them as `born_status='active'`,
  or a reviewer says owner edits must be parked for approval). The whole premise
  of this plan is that owner types are born active and trusted, so `node_edit`
  mutates in place with **no** proposal. If that boundary is in doubt, STOP and
  confirm before letting `node_edit` bypass a consent path — do not silently
  route node edits through `propose_edit`.
- A step's verification fails twice after a reasonable fix attempt.
- The fix appears to require touching an out-of-scope file (especially
  `internal/knowledge/*`, `proposeEditTool`, `internal/nodes/schema.go`
  internals, or a new migration).

## Maintenance notes

For the human/agent who owns this code after the change lands:

- `nodes.Update` is now the single owner-side in-place node editor. If a future
  change adds per-type write rules (e.g. a "locked" status, or fields only the
  system may set), enforce them inside `nodes.Update`, not in the tool/CLI
  callers — keep the gateways thin.
- `node_edit` replaces props wholesale when `props` is provided (it does not
  merge into existing props beyond template-apply). If a future need arises for
  patch-style prop edits (set one key, keep the rest), that is a deliberate API
  extension — record it rather than silently changing the replace semantics,
  because the model's mental model is "props = the object's current props".
- The consent boundary is the thing to scrutinize in review: confirm
  `node_edit`/`note edit` are reachable **only** for owner-authored
  (born-active) types and never for memory/skill (those stay on
  `propose_edit`). The `status != active` guard in `nodes.Update` is the
  backstop — a proposed memory/skill node cannot be edited through this path.
- Deferred out of this plan: the web UI / note-card edit form already exists for
  notes (`/ui/show/note`); wiring a typed-prop edit form into that card is a
  separate UI plan and is intentionally not in scope here.
