# Plan 192: SPIKE — sovereign export: a one-way Markdown vault mirror + encrypted backup

> **Executor instructions**: This is a SPIKE / design plan, not a build-it-all
> plan. The deliverable is (1) a written design note (open questions answered
> with recommendations + a redaction boundary + a phased plan) committed as a
> doc, and (2) ONE thin, read-only prototype that renders a single node type to
> Markdown into a temp dir, behind a `balaur export` CLI verb stub. Do NOT build
> a full exporter, do NOT wire git, do NOT implement encryption, do NOT write to
> the real data directory. Follow this plan step by step. Run every verification
> command and confirm the expected result before moving to the next step. If
> anything in the "STOP conditions" section occurs, stop and report — do not
> improvise. When done, update the status row for this plan in `plans/README.md`
> — unless a reviewer dispatched you and told you they maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 12a48bf..HEAD -- internal/nodes/ internal/cli/ migrations/1749600000_init.go PRODUCT.md README.md internal/self/knowledge.md`
> If any of those changed since this plan was written, compare the "Current
> state" excerpts against the live code before proceeding; on a mismatch, treat
> it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: L
- **Risk**: LOW
- **Depends on**: none
- **Category**: direction
- **Planned at**: commit `12a48bf`, 2026-06-24

## Why this matters

Balaur's **Sovereignty** pillar promises (`PRODUCT.md:61-62`) that "The life
lives on the owner's box, in inspectable SQLite **and exported Markdown** —
never in a vendor's database." Two direction bets make that promise concrete:
the **Johnny Decimal vault mirror** — one-way export of the life record to
Markdown + git "so sovereignty is provable and portable, not just claimed"
(`PRODUCT.md:141-142`) — and **Encrypted export** — safe off-box backup
(`PRODUCT.md:143-144`). README's honesty ledger lists BOTH as not shipped
(`README.md:425-432`), and `README.md:194-195` warns "Secrets (OAuth tokens,
vault entries) live in the local PocketBase data directory … Treat `pb_data/`
as secret." Today **no export code exists anywhere in `internal/`** (verified:
`grep -rn -iE "func.*Export|func.*Backup|func.*Mirror|package export" internal/`
returns nothing). The uniform nodes/edges spine (`internal/nodes/nodes.go`) is a
walkable substrate a one-way exporter renders to Markdown directly.

The hard part of this feature is NOT the rendering — it is the **redaction
boundary**. A single leaked secret (an OAuth token, a stored API key, a
proposed-but-rejected node surfaced as fact) violates the Sovereignty pillar.
So the spike's job is to **define and enforce that boundary up front** and prove
the thin render slice end-to-end, before anyone commits to a full exporter. The
done criterion is a design note plus a one-type prototype whose test asserts the
redaction — never a finished exporter.

## Current state

### The substrate: the nodes/edges spine (the thing the exporter reads)

`internal/nodes/nodes.go:1-11` — package doc establishes the consent boundary:

```go
// Package nodes is Balaur's unified knowledge spine: every piece of knowledge —
// a note, a memory, a skill, a journal day, a typed object (person, book, …) —
// is one row in the `nodes` collection, distinguished by `type` and linked to
// other nodes through the `edges` collection.
//
// THE CONSENT BOUNDARY lives in `status`: owner-authored kinds (note, journal,
// typed objects) are born active and trusted; agent-proposed kinds (memory,
// skill) are born proposed and become active only on the owner's explicit
// approval. Graph traversal AND search filter to status=active so a proposed or
// rejected node is never surfaced as fact. Every mutation is audited.
```

Status constants (`internal/nodes/nodes.go:26-31`):

```go
const (
	StatusProposed = "proposed"
	StatusActive   = "active"
	StatusArchived = "archived"
	StatusRejected = "rejected"
)
```

Reading props off a node (`internal/nodes/nodes.go:49-69`):

```go
// Props reads a node's props json into a map. Returns an empty (non-nil) map
// when props is absent or malformed, so callers can index it unconditionally.
func Props(rec *core.Record) map[string]any {
	m := map[string]any{}
	if raw, ok := rec.Get("props").(map[string]any); ok {
		return raw
	}
	// props may round-trip as types.JSONRaw; decode defensively.
	if err := rec.UnmarshalJSONField("props", &m); err != nil {
		return map[string]any{}
	}
	return m
}

// PropString reads one string field out of props (empty when absent).
func PropString(rec *core.Record, key string) string {
	if v, ok := Props(rec)[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
```

The active-only readers the exporter should reuse (`internal/nodes/nodes.go:162-167`):

```go
// ListByTypeStatus returns nodes of one type in one status, newest first.
func ListByTypeStatus(app core.App, typ, status string) ([]*core.Record, error) {
	return app.FindRecordsByFilter("nodes",
		"type = {:t} && status = {:s}", "-created", 0, 0,
		dbx.Params{"t": typ, "s": status})
}
```

The whole-graph reader, active-only by construction (`internal/nodes/query.go:59-89`):

```go
// ActiveSubgraph returns the whole active graph: up to limit active nodes
// (most-recently-updated first) and every edge whose BOTH endpoints are in that
// set. status=active is non-negotiable — proposed and rejected nodes are never
// returned and never reachable through an edge (the consent spine). Edges to a
// node beyond the cap are dropped so no endpoint dangles.
func ActiveSubgraph(app core.App, limit int) ([]*core.Record, []Edge, error) {
```

The `[[wikilink]]` regex the body uses (the exporter renders edges back to this
syntax) — `internal/nodes/links.go:20-25`:

```go
// wikilinkRe matches [[Target]] and [[Target|alias]]. The target is group 1
// (everything up to an optional pipe); the alias (group 2) is display-only and
// does not affect resolution. ...
var wikilinkRe = regexp.MustCompile(`\[\[([^\[\]|]+?)(?:\|([^\[\]]*))?\]\]`)
```

### The `nodes` collection schema (the export source) — `migrations/1749600000_init.go:88-103`

```go
nodes := core.NewBaseCollection("nodes")
setOwnerRules(nodes, owner)
nodes.Fields.Add(
	&core.SelectField{Name: "type", Required: true, MaxSelect: 1, Values: []string{"note", "memory", "skill", "journal", "person", "book", "idea", "place"}},
	&core.TextField{Name: "title", Required: true, Max: 300},
	&core.TextField{Name: "body", Max: 100000},
	&core.SelectField{Name: "status", Required: true, MaxSelect: 1, Values: []string{"proposed", "active", "archived", "rejected"}},
	&core.JSONField{Name: "props"},
	&core.AutodateField{Name: "created", OnCreate: true},
	&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
)
```

**Type registry note**: the `node_types` collection is the live registry; later
migrations add `task` (`migrations/1750000020_tasks_to_nodes.go`) and `day`
(`migrations/1750000040_day_type.go`), and retire `journal` into `day`
(`migrations/1750000050_unify_journal_into_day.go`). Read types at runtime via
`nodes.TypeNames(app)` (`internal/nodes/types.go:31-42`) and the owner-authored
subset via `nodes.OwnerAuthoredTypes(app)` (`internal/nodes/types.go:69-82`) —
do NOT hardcode the init list above.

### The secret boundary (what must NEVER leave the box)

The redaction boundary is REAL and has clean markers — secrets live in separate
collections, not interleaved with exportable node content:

- **Stored API keys**: `api_key` is a HIDDEN field on the `llm_providers`
  collection (`migrations/1749600000_init.go:228` adds the field;
  `:233-234` calls `f.SetHidden(true)`). The Go layer redacts it on read
  everywhere (e.g. `internal/store/llm_settings.go:79` sets `cfg.APIKey = ""`
  before returning lists). The UI/audit copy makes the rule explicit —
  `internal/feature/modelcards/cloud.go:18`: "Stored on this box only. Redacted
  from the UI and audit log; treat pb_data and backups as secret."
- **OAuth tokens / vault entries**: README's secret warning,
  `README.md:194-195`: "Secrets (OAuth tokens, vault entries) live in the local
  PocketBase data directory and its backups. Treat `pb_data/` as secret."
- **The consent filter**: proposed/rejected/archived nodes are NOT fact. Every
  graph reader in `internal/nodes` filters to `status = active` (see
  `Query`/`ActiveSubgraph` above). The exporter MUST do the same.

The exporter walks the `nodes` collection ONLY. It must never read
`llm_providers`, `llm_models`, `extensions`, `owner_settings`, or any auth/token
collection. That is the redaction boundary: **node body + node props + active
status, and nothing else.**

### Repo conventions the prototype must follow

- **CLI verbs**: every command is a thin cobra wrapper that prints one JSON
  envelope and lives under `internal/cli`. Register new top-level commands in
  `cli.Register` (`internal/cli/cli.go:53-74`). The closest exemplar to copy is
  the `note` verb group over `internal/nodes`, in **`internal/cli/knowledge.go`**:
  - `noteCmd` (`internal/cli/knowledge.go:285-292`) builds the group;
  - `noteListCmd` (`:318-338`) shows the `run(app, "note.list", func(...)...)`
    body wrapper, `cmd.Flags().StringVar`, and returning a `[]map[string]any`
    that `emit` JSON-encodes;
  - `nodeJSON` (`:20-29`) shows reading fields off a `*core.Record`.

  The `run` wrapper that gives the v1 envelope contract is
  `internal/cli/cli.go:83-101`; `emit` is `:110-115`.

- **Tests**: standard `testing`, table-driven where it helps, NO assertion
  frameworks, NO `time.Sleep`. Boot a PocketBase-backed test app with
  `storetest.NewApp(t)` (`internal/storetest/storetest.go:18-26`) — it runs the
  full migration chain into `t.TempDir()`. Use `t.TempDir()` for the export
  output directory. Exemplar test file shape: `internal/nodes/nodes_test.go:1-32`
  (`package nodes_test`, `app := storetest.NewApp(t)`, `nodes.Create(...)`,
  assert with `t.Fatalf`/`t.Errorf`).

- **Errors are values**: wrap with `fmt.Errorf("doing x: %w", err)`, return
  early, no panics in library code.

- **gofmt is law** (a PostToolUse hook + CI gofmt gate enforce it); `go vet`,
  `staticcheck`, `govulncheck` all gate CI.

- **Self-knowledge**: `internal/self/knowledge.md` is the running binary's own
  description of its architecture. Since this spike adds a new (stub) capability
  surface, note the spike's existence there in the same commit (one line under
  the relevant capability list — see Step 6).

## Commands you will need

| Purpose          | Command                                   | Expected on success            |
|------------------|-------------------------------------------|--------------------------------|
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...`            | exit 0, no output              |
| Test one package | `go test ./internal/export/`              | `ok` (after Step 4 creates it) |
| Test the CLI     | `go test ./internal/cli/`                 | `ok`                           |
| Test all         | `go test ./...`                           | all `ok`                       |
| Vet              | `go vet ./...`                            | exit 0, no output              |
| Format check     | `gofmt -l .`                              | empty output (no files listed) |
| Diff whitespace  | `git diff --check`                        | no output                      |
| Update graph     | `graphify update .`                       | regenerates graphify-out       |

(Exact commands for this repo — verified during recon.)

> **Note on `TMPDIR`**: the full `go test ./...` link can fail with "No space
> left on device" on a small tmpfs `/tmp`. If you hit that, set
> `TMPDIR=$HOME/.cache/go-tmp` (mkdir it first) for the test run.

## Suggested executor toolkit

- Invoke the `go-standards` skill before writing Go — it covers the error
  wrapping, structured logging, gomponents-vs-text rules, and the testing idioms
  (table-driven, `t.TempDir`, `storetest.NewApp`) this plan relies on.
- Orient with graphify before reading source: `graphify explain "nodes spine"`,
  `graphify query "cli verb registration"`. The repo hook mandates it.

## Scope

This is a SPIKE. The point is a design + a thin slice, not a feature.

**In scope** (the only files you should create/modify):
- `docs/superpowers/specs/2026-06-25-sovereign-export-design.md` (create) — the
  design note: redaction boundary, the answered open questions, the phased plan.
- `internal/export/export.go` (create) — the thin, read-only, ONE-type
  Markdown writer (library function; writes to a caller-supplied dir).
- `internal/export/export_test.go` (create) — the redaction-asserting test.
- `internal/cli/export.go` (create) — the `balaur export` CLI verb stub that
  calls the library function.
- `internal/cli/cli.go` (modify) — register `exportCmd(app)` in `Register`.
- `internal/cli/export_test.go` (create) — a thin CLI-level test (optional but
  recommended; mirror `internal/cli/cli_test.go` structure).
- `internal/self/knowledge.md` (modify) — one line noting the export spike stub.

**Out of scope** (do NOT touch, even though they look related):
- Git integration / `git init` / committing the mirror — DEFERRED to the real
  implementation. The prototype writes plain files to a temp dir, nothing more.
- Encryption (filippo.io/age, stdlib crypto, key handling) — DESIGN ONLY in this
  spike. Do NOT add a crypto dependency or write any encrypt/decrypt code.
- Exporting more than ONE node type — the prototype does `note` only. Multi-type,
  edges-as-wikilinks beyond what's already in the body, frontmatter for every
  prop type, recap/summary transcripts: all DEFERRED, listed in the design note.
- Writing into the real data directory (`app.DataDir()`), or any code that
  mutates the record. The prototype is strictly read-only on PocketBase and
  writes only to a caller-supplied directory.
- Any migration. This spike adds NO schema. Do not create a migration file.
- Any change that reads `llm_providers`, `llm_models`, `extensions`,
  `owner_settings`, or any token/secret collection.

## Git workflow

- Branch: executor worktree off `origin/main` (per repo convention). If working
  directly, branch `advisor/192-spike-sovereign-export`.
- Commit per logical unit; conventional-commit subjects. Examples from this
  repo's log: `feat(ui): …`, `refactor(knowledge): …`, `docs(storybook): …`.
  Suggested subjects here:
  `docs(export): sovereign-export design note + open questions (192)` and
  `feat(export): read-only one-type Markdown export prototype + CLI stub (192)`.
- Do NOT push or open a PR unless the operator instructed it. Gate any push on a
  green `go test ./...`.

## Steps

### Step 1: Enumerate the exportable record and write the redaction boundary section of the design note

Create `docs/superpowers/specs/2026-06-25-sovereign-export-design.md`. Start with
a **Redaction boundary** section that is the spec's spine. State, explicitly:

- **What IS exportable**: rows in the `nodes` collection with `status = active`.
  For each such node: its `title`, `body`, `type`, `created`/`updated`, and its
  `props` map. Edges are represented as `[[wikilinks]]` already present in the
  body (the mirror does not invent new link syntax in the Pareto slice).
- **What is NEVER exported (the exclusion list)**:
  - Any node whose `status != active` (proposed / rejected / archived) — the
    consent filter (cite `internal/nodes/nodes.go:8-10` and `query.go:18`).
  - The entire `llm_providers.api_key` hidden field and anything in
    `llm_providers` / `llm_models` (stored API keys — cite
    `migrations/1749600000_init.go:228,233-234`).
  - OAuth tokens and vault entries living in `pb_data` (cite
    `README.md:194-195`).
  - `extensions`, `owner_settings`, `audit_log`, `llm_settings`, conversations,
    and messages — out of the v1 mirror scope entirely (the mirror is the
    *knowledge* record, not the runtime/secret state).
- **The enforcement rule**: the exporter reads the `nodes` collection ONLY,
  through `internal/nodes` active-filtered readers. It has NO code path that
  opens any other collection. State this as an invariant the test asserts.

**Verify**: the file exists and contains a "Redaction boundary" heading with an
explicit exclusion list naming `api_key`, OAuth tokens, vault entries, and
non-active nodes.
`grep -c "Redaction boundary" docs/superpowers/specs/2026-06-25-sovereign-export-design.md` → `1` (or more).

### Step 2: Design the type → Johnny Decimal mapping and the Markdown file format (in the design note)

Add two sections to the design note.

**(a) Type → Johnny Decimal area/category map.** Johnny Decimal organizes into
`AREAS (10-19, 20-29, …)` containing `CATEGORIES (11, 12, …)`. Propose a concrete
map from node `type` to an area/category, and RECOMMEND one. A sane starting
proposal (record it, mark it a recommendation, not a commitment):

| node type | JD area | JD category | folder |
|-----------|---------|-------------|--------|
| note      | 10-19 Knowledge | 11 | `10-19 Knowledge/11 Notes/` |
| idea      | 10-19 Knowledge | 12 | `10-19 Knowledge/12 Ideas/` |
| person    | 20-29 People    | 21 | `20-29 People/21 People/` |
| book      | 30-39 Library   | 31 | `30-39 Library/31 Books/` |
| place     | 40-49 Places    | 41 | `40-49 Places/41 Places/` |
| day       | 50-59 Journal   | 51 | `50-59 Journal/51 Days/` |
| task      | 60-69 Tasks     | 61 | `60-69 Tasks/61 Tasks/` |

Note the open question: types are dynamic (`node_types` registry) — the map needs
a default bucket (e.g. `90-99 Unsorted/91 Other/`) for any type without an
explicit mapping. Recommend: a hardcoded map for the known types + an "Unsorted"
fallback, revisited when a new type is added. The Pareto slice exports
`note`/`idea`/typed objects first; **defer `day` recap transcripts and `task`**.

**(b) Markdown file format.** Specify: one file per node, named from a slugified
title (collision-resolved by appending the node id), with:
- **YAML frontmatter** carrying `type`, `status`, `created`, `updated`, and each
  `props` key/value (scalars only in the Pareto slice; nested props deferred).
- **Body** = the node `body` verbatim (it already contains `[[wikilinks]]` —
  cite `internal/nodes/links.go:20-25`), under an `# {title}` H1.

Show a worked example block in the design note (a `note` node → its `.md` text).

**Verify**: `grep -cE "Johnny Decimal|frontmatter" docs/superpowers/specs/2026-06-25-sovereign-export-design.md` → ≥ `2`.

### Step 3: Answer the remaining open questions in the design note (each with a recommendation)

Add an **Open questions** section. Each question gets a short discussion and a
**Recommendation:** line.

1. **JD numbering scheme** — answered in Step 2(a). Recommendation: the table
   above + an Unsorted fallback.
2. **Redaction / exclusion boundary** — answered in Step 1. Recommendation:
   active-nodes-only, `nodes` collection only, never any secret/token collection;
   enforced by the test.
3. **Incremental vs full re-export.** Recommend: **full re-export** for the
   Pareto slice (deterministic, simplest correct; git diffs show what changed).
   Incremental (mtime/updated-based) is a later optimization, noted as deferred.
4. **Key handling for encrypted export (B).** Lay out the trade-off: owner
   passphrase (the owner remembers it; **lost passphrase = lost backup**, state
   this bluntly) vs a generated key (must itself be stored/backed up somewhere,
   which re-poses the sovereignty problem). Recommend: a single owner-supplied
   passphrase, KDF-stretched, with a loud "if you lose this, the backup is
   unrecoverable" warning at the point of creation. No key escrow, no cloud.
5. **How (A) and (B) compose.** Options: encrypt the whole Markdown mirror
   directory into one archive, vs a separate encrypted blob of the SQLite record.
   Recommend: **(B) encrypts the (A) Markdown mirror** (one `age`/archive over
   the mirror tree) — one artifact, and the plaintext mirror stays the
   inspectable, sovereign default; encryption is the opt-in off-box carry.

**Verify**: `grep -c "Recommendation:" docs/superpowers/specs/2026-06-25-sovereign-export-design.md` → ≥ `5`.

### Step 4: Prototype a thin, read-only nodes→Markdown writer for ONE type (`note`)

Create `internal/export/export.go`. Package doc must state the invariant: this
package is read-only on the record, writes only to a caller-supplied dir, exports
ONLY `status = active` nodes, and reads ONLY the `nodes` collection.

Implement a single library function (no git, no encryption, one type):

```go
// Package export is Balaur's sovereign-export spike (plan 192): a one-way,
// read-only renderer of the active knowledge record to Markdown. It is the thin
// slice of the Johnny Decimal vault mirror — ONE node type, no git, no
// encryption. The redaction boundary is hard: it reads ONLY the `nodes`
// collection, ONLY status=active rows, and never touches any secret/token
// collection (api_key, OAuth tokens, vault entries — see the design note at
// docs/superpowers/specs/2026-06-25-sovereign-export-design.md).
package export

// ExportType renders every ACTIVE node of one type to a Markdown file under
// destDir (one file per node: YAML frontmatter from props + an H1 title + the
// node body, which already carries [[wikilinks]]). It returns the relative file
// paths written. It is read-only on PocketBase: it calls only the active-filtered
// nodes.ListByTypeStatus reader and writes only under destDir. It never reads
// llm_providers, extensions, owner_settings, or any token/secret collection.
func ExportType(app core.App, typ, destDir string) ([]string, error)
```

Implementation requirements (keep it small — target well under 100 lines):
- Load nodes with `nodes.ListByTypeStatus(app, typ, nodes.StatusActive)` — this
  is the consent filter; do NOT add any other status.
- For each node, build YAML frontmatter from `nodes.Props(rec)` plus `type`,
  `status`, `created`, `updated`. Render scalars only; if a prop value is a
  map/slice, JSON-encode it inline (or skip with a comment — pick one, document
  it). Use `gopkg.in/yaml.v3` ONLY if it is already a dependency
  (`grep yaml.v3 go.mod`); otherwise hand-write the simple `key: value` lines
  (titles/values that contain `:` or newlines must be quoted — keep the Pareto
  slice to quoting with `%q`). Do NOT add a new dependency for the spike.
- Slugify the title for the filename; on collision append `-<rec.Id>`.
- Write with `os.MkdirAll(destDir, 0o755)` then `os.WriteFile(path, data, 0o644)`.
- Wrap every error: `fmt.Errorf("export: writing %s: %w", path, err)`.
- Structured logging only if you log at all (`app.Logger()`); the library
  function should just return errors.

**Verify**:
- `CGO_ENABLED=0 go build ./internal/export/` → exit 0.
- `gofmt -l internal/export/export.go` → empty.

### Step 5: Write the redaction-asserting test for the prototype

Create `internal/export/export_test.go` (`package export_test`). Use
`storetest.NewApp(t)` and `t.TempDir()`. Cover, at minimum:

1. **Happy path / frontmatter + wikilink**: create an ACTIVE `note` via
   `nodes.Create(app, "note", "My Note", "Body with [[Other Note]] link.",
   nodes.StatusActive, map[string]any{"tag": "demo"})`, run
   `ExportType(app, "note", dir)`, read the written file, and assert it contains
   the title, the `type: note` / `status: active` frontmatter, the `tag` prop,
   AND the literal `[[Other Note]]` wikilink (wikilinks survive verbatim).
2. **Redaction — non-active excluded**: create a `note` with
   `nodes.StatusProposed` (or a rejected one) titled e.g. "SECRET-PROPOSAL",
   export, and assert NO written file contains that title. (Cite: this is the
   consent boundary.)
3. **Redaction — no secret leakage**: this is the load-bearing assertion. Seed a
   stored secret via the real path — call
   `store.SaveCloudModel(app, "TestProvider", "https://example.test", "sk-SECRET-TOKEN-DO-NOT-LEAK", "Test", "test-model", "")`
   (signature: `internal/store/llm_settings.go:158`) so an `api_key` exists in
   `llm_providers`. Then run `ExportType(app, "note", dir)`, walk every written
   file under `dir`, and assert NONE contains the substring
   `sk-SECRET-TOKEN-DO-NOT-LEAK`. (Use a non-secret placeholder string literal in
   the test — never a real credential.)

Read every file under the temp dir with `filepath.WalkDir` + `os.ReadFile` and
assert on the bytes. Model the structure after `internal/nodes/nodes_test.go:1-32`.

**Verify**: `go test ./internal/export/` → `ok` (all cases pass). If the secret
appears in any output, that is a STOP condition (see below) — do not "fix" it by
filtering the output string; the leak means the exporter read the wrong
collection.

### Step 6: Add the `balaur export` CLI verb stub and register it

Create `internal/cli/export.go`. Mirror `noteCmd`/`noteListCmd` in
`internal/cli/knowledge.go:285-338` and the `run`/`emit` envelope contract in
`internal/cli/cli.go:83-115`. The verb is a STUB in the sense that it exercises
ONLY the one-type prototype — it is not a full exporter:

```go
func exportCmd(app core.App) *cobra.Command
```

- `Use: "export"`, short description noting this is the spike's one-type Markdown
  export (read-only, no git, no encryption).
- One flag `--type` (default `"note"`) and one flag `--out` (required: the
  destination directory — NEVER default to `app.DataDir()`).
- Body: `paths, err := export.ExportType(app, typ, out)`; on success return
  `map[string]any{"type": typ, "out": out, "files": paths}` so `emit` prints the
  v1 envelope. Wrap errors via the existing `run` wrapper (it handles failJSON).

Register it in `internal/cli/cli.go` `Register` (`:53-74`), adding
`exportCmd(app),` to the `root.AddCommand(...)` list (keep gofmt alignment).

Optionally add `internal/cli/export_test.go` mirroring `internal/cli/cli_test.go`
to assert the command prints a `{"v":1,"kind":"export", ...}` envelope and writes
files under a `t.TempDir()`.

**Verify**:
- `go test ./internal/cli/` → `ok`.
- `CGO_ENABLED=0 go build ./...` → exit 0.
- Manual smoke (optional): `go run . --dir $(mktemp -d) export --type note --out $(mktemp -d)`
  prints a `"kind":"export"` JSON envelope (zero files on an empty box is fine).

Then update `internal/self/knowledge.md`: add one line under the relevant
capability/CLI list noting that a sovereign-export spike stub (`balaur export`,
one-type Markdown, read-only) exists, with a pointer to the design note. Keep it
to one or two sentences — this is a spike, not a shipped feature.

### Step 7: Write the phased recommendation and final checks

Close the design note with a **Phased plan** section:
- **Phase 1 (this spike)**: design + one-type read-only prototype + CLI stub +
  redaction test. ✅ delivered here.
- **Phase 2 (mirror, future plan)**: all owner-authored types, the JD folder
  layout, full re-export, git init/commit under the data dir, the storybook/UI
  surface if any. One-way + additive + owner-initiated + offline.
- **Phase 3 (encryption, future plan)**: `age` (or stdlib) pure-Go envelope over
  the Phase-2 mirror, owner-passphrase KDF, the lost-passphrase warning UX.

State the hard constraints as a standing guard for future phases: **strictly
one-way, additive, owner-initiated, offline; the redaction boundary is enforced
in code and asserted by tests; a single leaked secret violates the Sovereignty
pillar.**

**Verify** (run all, all must pass):
- `gofmt -l .` → empty.
- `go vet ./...` → exit 0.
- `go test ./...` → all `ok`.
- `git diff --check` → no output.
- `git status` → only the in-scope files changed.
- `graphify update .` → regenerates the graph (so the new package is indexed).

## Test plan

- New file `internal/export/export_test.go` (`package export_test`), using
  `storetest.NewApp(t)` + `t.TempDir()`, covering:
  - **happy path**: active `note` → file with frontmatter (`type`/`status`/prop)
    + H1 title + verbatim `[[wikilink]]` in the body.
  - **consent redaction**: a `proposed`/`rejected` node is NOT written.
  - **secret redaction (load-bearing)**: after `store.SaveCloudModel` seeds an
    `api_key`, no exported file contains the secret placeholder substring.
- Optional `internal/cli/export_test.go` mirroring `internal/cli/cli_test.go`:
  the `export` command emits a `{"v":1,"kind":"export",...}` envelope and writes
  files to a temp dir.
- Structural pattern to copy: `internal/nodes/nodes_test.go` for the library
  test; `internal/cli/cli_test.go` for the CLI test.
- Verification: `go test ./internal/export/ ./internal/cli/` → all pass,
  including the three new export cases.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `docs/superpowers/specs/2026-06-25-sovereign-export-design.md` exists with:
      a "Redaction boundary" section (exclusion list names `api_key`, OAuth
      tokens, vault entries, non-active nodes), a type→Johnny Decimal map, a
      Markdown frontmatter/wikilink format, an "Open questions" section with ≥ 5
      `Recommendation:` lines (incl. key handling + how A/B compose), and a
      "Phased plan" (mirror first, encryption second).
- [ ] `internal/export/export.go` exists: a read-only, ONE-type Markdown writer
      that reads only via `nodes.ListByTypeStatus(..., StatusActive)` and writes
      only under a caller-supplied dir. No git, no encryption, no new dependency.
- [ ] `internal/export/export_test.go` exists and `go test ./internal/export/`
      passes, INCLUDING the assertion that no exported file contains the seeded
      `api_key` secret and that proposed/rejected nodes are excluded.
- [ ] `internal/cli/export.go` adds `exportCmd`, registered in `cli.Register`;
      `go test ./internal/cli/` passes.
- [ ] `internal/self/knowledge.md` notes the export spike stub (one or two lines).
- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go vet ./...` exits 0;
      `gofmt -l .` is empty; `git diff --check` is clean.
- [ ] `go test ./...` exits 0.
- [ ] No files outside the in-scope list are modified (`git status`).
- [ ] `plans/README.md` status row updated.

## STOP conditions

Stop and report back (do not improvise) if:

- **The redaction boundary cannot be cleanly defined or enforced.** If you find
  that secrets (OAuth tokens, stored keys, vault content) are interleaved INTO
  node bodies/props with no reliable marker — i.e. exportable content and secret
  content share the same `nodes` rows and you cannot exclude the secret without
  excluding legitimate content — STOP. Do not ship an exporter that might leak.
  Report what you found. (The recon says secrets live in SEPARATE collections, so
  this should NOT happen; if it does, the schema drifted and the spike's premise
  is broken.)
- **The secret-redaction test fails** (an exported file contains the seeded
  secret). Do NOT patch it by string-filtering the output — that masks a real
  read of the wrong collection. Treat it as a defect in `ExportType` (it must be
  reading something beyond the active `nodes` rows) and, if you cannot make it
  pass by restricting reads to `nodes`, STOP and report.
- The code at the locations in "Current state" doesn't match the excerpts (the
  codebase drifted since 12a48bf) — re-run the drift check and report the diff.
- A step's verification fails twice after a reasonable fix attempt.
- The work appears to require touching an out-of-scope file (a migration, a
  crypto dependency, git wiring, `app.DataDir()` writes, or any
  secret/token/provider collection) — that means scope is creeping past the
  spike; STOP and report rather than expanding it.
- You find yourself implementing encryption, git, or multi-type export — that is
  Phase 2/3, explicitly out of scope. STOP.

## Maintenance notes

For the human/agent who owns this after the spike lands:

- This is a SPIKE. `internal/export` is intentionally one-type and git-less. The
  next plan (Phase 2) should generalize to all owner-authored types
  (`nodes.OwnerAuthoredTypes`), lay out the JD folders, do a full re-export, and
  add `git init`/commit under the data dir — re-reading the design note's
  recommendations and re-validating the redaction boundary as the surface grows.
- The redaction boundary is the thing a reviewer must scrutinize hardest: the
  ONLY collection `internal/export` may read is `nodes`, and ONLY `status=active`
  rows. Any future change that makes the exporter read another collection MUST
  add a corresponding "this secret never appears in output" test, or it
  regresses the Sovereignty pillar. The secret-redaction test is the canary —
  never delete or weaken it.
- Encryption (Phase 3) must stay CGO-free (the repo builds with
  `CGO_ENABLED=0`): prefer `filippo.io/age` or stdlib crypto, never a CGO
  binding. The "lost passphrase = lost backup" warning is a product requirement,
  not a nicety — surface it at creation time.
- Deferred out of this spike (and why): git history (keeps the prototype a pure
  read→write slice), incremental export (full re-export is simpler and
  deterministic for now), recap/`day` transcripts and `task` export (Pareto:
  notes/typed objects first), nested-prop frontmatter (scalars cover the Pareto
  slice), and all of encryption (design-only until the mirror is real).
- When this lands, flip the matching lines in README's honesty ledger
  (`README.md:425-432`) only once the REAL mirror ships — the spike stub is NOT
  the shipped feature, so leave the ledger's "not shipped" lines as-is for now.
