# Sovereign export: a one-way Markdown vault mirror + encrypted backup (design)

> **Status**: design note for plan 192 (SPIKE). This documents the redaction
> boundary, the type → Johnny Decimal mapping, the Markdown file format, the
> answered open questions, and the phased plan. It ships alongside ONE thin,
> read-only prototype (`internal/export`, `note` type only) and a `balaur export`
> CLI stub. It is NOT the shipped feature — the README honesty ledger
> (`README.md:425-432`) stays "not shipped" until the real mirror (Phase 2) lands.

## Why

Balaur's **Sovereignty** pillar promises (`PRODUCT.md:61-62`) that "The life
lives on the owner's box, in inspectable SQLite **and exported Markdown** — never
in a vendor's database." Two direction bets make that promise concrete: the
**Johnny Decimal vault mirror** (one-way export of the life record to Markdown +
git, `PRODUCT.md:141-142`) and **Encrypted export** (safe off-box backup,
`PRODUCT.md:143-144`).

The hard part is NOT the rendering — it is the **redaction boundary**. A single
leaked secret (an OAuth token, a stored API key, a proposed-but-rejected node
surfaced as fact) violates the Sovereignty pillar. So this spike's job is to
**define and enforce that boundary up front** and prove the thin render slice
end-to-end, before anyone commits to a full exporter.

## Redaction boundary

This is the spec's spine. It is the rule the prototype enforces in code and the
`internal/export` test asserts.

### What IS exportable

Rows in the **`nodes` collection** with **`status = active`**, and nothing else.
For each such node, the export carries:

- `title`
- `body` (verbatim — it already contains `[[wikilinks]]`, see
  `internal/nodes/links.go:20-25`)
- `type`
- `created` / `updated`
- the node's `props` map (scalars in the Pareto slice; nested values deferred)

Edges are represented as the `[[wikilinks]]` **already present in the body**. The
Pareto slice does NOT invent new link syntax or walk the `edges` collection to
synthesize links.

### What is NEVER exported (the exclusion list)

- **Any node whose `status != active`** — proposed, rejected, or archived. This is
  the consent filter: a proposed or rejected node is never surfaced as fact (the
  consent boundary, `internal/nodes/nodes.go:6-10`, enforced by every active-only
  reader, e.g. `internal/nodes/query.go` `ActiveSubgraph`). The exporter loads
  nodes only through `nodes.ListByTypeStatus(app, typ, nodes.StatusActive)`.
- **`llm_providers.api_key`** and anything in `llm_providers` / `llm_models` —
  stored cloud API keys. `api_key` is a HIDDEN field
  (`migrations/1749600000_init.go:228` adds it; `:233-234` calls
  `f.SetHidden(true)`), redacted on every read in the Go layer (e.g.
  `internal/store/llm_settings.go:79`, `:104` set `cfg.APIKey = ""`). "Stored on
  this box only. Redacted from the UI and audit log; treat pb_data and backups as
  secret." (`internal/feature/modelcards/cloud.go`).
- **OAuth tokens and vault entries** living in `pb_data` —
  "Secrets (OAuth tokens, vault entries) live in the local PocketBase data
  directory and its backups. Treat `pb_data/` as secret." (`README.md:194-195`).
- **`extensions`, `owner_settings`, `audit_log`, `llm_settings`, conversations,
  and messages** — out of the v1 mirror scope entirely. The mirror is the
  *knowledge* record, not the runtime/secret state.

### The enforcement rule (the invariant the test asserts)

The exporter reads the **`nodes` collection ONLY**, through the
`internal/nodes` active-filtered readers. It has NO code path that opens any
other collection. This is an invariant:

> **`internal/export` may read exactly one collection (`nodes`) and exactly one
> status (`active`). It never opens `llm_providers`, `llm_models`, `extensions`,
> `owner_settings`, `audit_log`, or any token/secret/conversation collection.**

The load-bearing test seeds a real stored secret via `store.SaveCloudModel`
(which writes an `api_key` into `llm_providers`), runs the export, walks every
written file, and asserts the secret substring appears in NONE of them. If it
ever appears, the exporter read the wrong collection — a defect, not something to
paper over by string-filtering output.

## Type → Johnny Decimal area/category map

Johnny Decimal organizes into AREAS (`10-19`, `20-29`, …) containing CATEGORIES
(`11`, `12`, …). The proposal below is a **recommendation, not a commitment** —
it is recorded here so Phase 2 has a starting layout.

| node type | JD area          | JD category | folder                         |
|-----------|------------------|-------------|--------------------------------|
| note      | 10-19 Knowledge  | 11          | `10-19 Knowledge/11 Notes/`    |
| idea      | 10-19 Knowledge  | 12          | `10-19 Knowledge/12 Ideas/`    |
| person    | 20-29 People     | 21          | `20-29 People/21 People/`      |
| book      | 30-39 Library    | 31          | `30-39 Library/31 Books/`      |
| place     | 40-49 Places     | 41          | `40-49 Places/41 Places/`      |
| day       | 50-59 Journal    | 51          | `50-59 Journal/51 Days/`       |
| task      | 60-69 Tasks      | 61          | `60-69 Tasks/61 Tasks/`        |

**Open question — dynamic types.** Node types are not the static init list; they
live in the `node_types` registry (`internal/nodes/types.go`, read at runtime via
`nodes.TypeNames`/`nodes.OwnerAuthoredTypes`). Later migrations add `task` and
`day` and retire `journal` into `day`. So the map needs a **default bucket** for
any type without an explicit mapping.

> **Recommendation:** a hardcoded map for the known types plus an **Unsorted**
> fallback (`90-99 Unsorted/91 Other/`), revisited when a new type is added. The
> Pareto slice exports `note`/`idea`/typed objects first; **`day` recap
> transcripts and `task` are DEFERRED** (recap/transcript content needs its own
> redaction pass).

## Markdown file format

One file per node:

- **Filename**: a slugified title (lowercase, non-alphanumerics collapsed to `-`).
  On a slug collision within a folder, append `-<rec.Id>` so two nodes never
  clobber each other. Empty/odd titles fall back to the node id.
- **YAML frontmatter** carrying `type`, `status`, `created`, `updated`, and each
  `props` key/value. Scalars only in the Pareto slice; a map/slice prop value is
  JSON-encoded inline (nested-prop frontmatter is deferred). Values that contain
  `:`, quotes, or newlines are quoted with Go `%q` so the YAML stays well-formed.
  The spike hand-writes the `key: value` lines — `gopkg.in/yaml.v3` is NOT a
  dependency (only `go.yaml.in/yaml/v2` appears as an indirect dep), and the
  spike adds no new dependency.
- **Body** = the node `body` verbatim (already carries `[[wikilinks]]`,
  `internal/nodes/links.go:20-25`), under an `# {title}` H1.

### Worked example

An active `note` node titled "My Note", body `Body with [[Other Note]] link.`,
with `props = {"tag":"demo"}`, renders to `my-note.md`:

```markdown
---
type: "note"
status: "active"
created: "2026-06-25T09:00:00Z"
updated: "2026-06-25T09:00:00Z"
tag: "demo"
---

# My Note

Body with [[Other Note]] link.
```

The `[[Other Note]]` wikilink survives verbatim — that is what makes the export a
faithful, walkable mirror.

## Open questions

Each question gets a short discussion and a recommendation.

### 1. JD numbering scheme

Discussed in *Type → Johnny Decimal map* above.

**Recommendation:** the table above (areas `10-19`…`60-69`) plus an
`90-99 Unsorted/91 Other/` fallback for any unmapped registry type, revisited as
types are added.

### 2. Redaction / exclusion boundary

Discussed in *Redaction boundary* above.

**Recommendation:** active-nodes-only, the `nodes` collection only, never any
secret/token collection. Enforced in code (the exporter has no other read path)
and asserted by the `internal/export` secret-redaction test (the canary).

### 3. Incremental vs full re-export

Incremental export (skip files whose source `updated` predates the last run)
saves writes on a large vault but adds state — a manifest of last-export
timestamps, stale-file detection when a node is deleted or archived, and a
subtle correctness surface. Full re-export rewrites every file every run:
deterministic, simplest correct, and git diffs already show exactly what changed.

**Recommendation:** **full re-export** for the Pareto slice. Incremental
(`updated`/mtime-based) is a later optimization once the vault is large enough to
feel the cost — deferred.

### 4. Key handling for encrypted export (B)

Two shapes. An **owner passphrase** is something the owner remembers, with no
extra artifact to store — but **a lost passphrase means a lost backup**, and
that has to be said bluntly at creation time. A **generated key** removes the
"weak passphrase" risk but must itself be stored/backed up somewhere, which
re-poses the sovereignty problem (where does the key live? if in `pb_data`, the
encrypted backup is no safer than `pb_data` itself).

**Recommendation:** a single **owner-supplied passphrase, KDF-stretched**, with a
loud **"if you lose this passphrase, the backup is unrecoverable"** warning at the
point of creation. No key escrow, no cloud, no generated-key-on-disk. This keeps
the secret in the owner's head, where sovereignty wants it.

### 5. How (A) the mirror and (B) encryption compose

Option 1: encrypt the whole Markdown mirror directory into one archive. Option 2:
a separate encrypted blob of the raw SQLite record. Option 2 re-introduces the
opaque-blob problem the mirror exists to avoid, and would need its own redaction
pass over the DB.

**Recommendation:** **(B) encrypts the (A) Markdown mirror** — one `age`/archive
over the mirror tree. One artifact; the plaintext Markdown mirror stays the
inspectable, sovereign default; encryption is the opt-in off-box carry layered on
top, not a parallel export of secret state.

## Phased plan

- **Phase 1 (this spike)** — ✅ delivered here: this design note + a one-type,
  read-only `internal/export` prototype (`note` only, YAML frontmatter + H1 +
  verbatim body) + a `balaur export` CLI stub + the redaction-asserting test
  (active-only, no-secret-leak, consent-excluded). No git, no encryption, no new
  dependency, no writes to the real data dir.
- **Phase 2 (mirror, future plan)** — all owner-authored types
  (`nodes.OwnerAuthoredTypes`), the JD folder layout above, **full re-export**
  (per question 3), `git init`/commit under the data dir, and the storybook/UI
  surface if any. One-way + additive + owner-initiated + offline. Re-read this
  note's recommendations and re-validate the redaction boundary as the surface
  grows: any new collection read MUST come with its own "this secret never appears
  in output" test.
- **Phase 3 (encryption, future plan)** — a pure-Go `age` (or stdlib) envelope
  over the Phase-2 mirror tree (per question 5), an owner-passphrase KDF (per
  question 4), and the "lost passphrase = lost backup" warning UX surfaced at
  creation time. Must stay CGO-free (`CGO_ENABLED=0`) — no CGO crypto binding.

## Standing constraints (a guard for every future phase)

The mirror is **strictly one-way, additive, owner-initiated, and offline**. The
**redaction boundary is enforced in code and asserted by tests** — a single
leaked secret violates the Sovereignty pillar, so the secret-redaction test is a
canary that must never be deleted or weakened. Any change that makes the exporter
read a collection beyond `nodes` must add a corresponding leak test in the same
change, or it regresses sovereignty.
