# Plan 121: Delete the unwired embedding-rerank spike

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat ce2ba72..HEAD -- internal/knowledge/`
> If `internal/knowledge/rerank.go` or `rerank_test.go` changed since this plan
> was written, OR any new file references `RerankByEmbedding`, treat it as a
> STOP condition.

## Why this matters

`internal/knowledge/rerank.go` is a validated prototype from the plan-073
embedding-recall spike: it was deliberately left **unwired** ("standalone and
unwired" by design). At HEAD nothing in production calls `RerankByEmbedding`,
`cosine`, or `rerankEnabled` — the only callers are `rerank_test.go`. It is the
sole reason the `internal/knowledge` package imports `internal/llm`. The owner
has chosen to remove this dead weight now and rebuild "sharper recall" from the
plan-073 design later if/when it is actually pursued, rather than carry an
unexercised feature path in the tree. Deleting it removes ~105 lines of
production code plus its test, with zero behavioral change (the lexical
`SearchActive` recall is untouched and was always the live path).

## Current state

- `internal/knowledge/rerank.go` (105 lines) — defines `rerankEnabled()` (reads
  `BALAUR_EMBED_RERANK`), `cosine(a,b []float32) float64`, and the exported
  `RerankByEmbedding(ctx, client llm.Client, query string, candidates []*core.Record) []*core.Record`.
  Its package doc references `plans/073`.
- `internal/knowledge/rerank_test.go` — `TestRerankByEmbedding` (the only caller
  of the above symbols).

Proof it is dead (run these yourself in the drift check):
- `grep -rn "RerankByEmbedding" --include='*.go' internal/ main.go` → only
  `rerank.go` (definition) and `rerank_test.go`.
- `grep -rn "BALAUR_EMBED_RERANK" --include='*.go' --include='*.md' . | grep -v '^plans/'`
  → only `rerank.go` + `rerank_test.go` (no docs claim the feature).
- `grep -rln "llm\." internal/knowledge/*.go | grep -v _test | grep -v rerank.go`
  → empty (the `llm` import in `internal/knowledge` is used ONLY by `rerank.go`).

What stays (do not touch): `plans/073-embedding-recall-spike.md` (the design doc,
historical record); `PRODUCT.md`'s "sharper recall" direction bet (a future bet,
not a shipped claim); the `llm.Client.Embed` method and any non-knowledge users
of embeddings.

## Commands you will need

| Purpose   | Command                                   | Expected on success |
|-----------|-------------------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`            | exit 0              |
| Vet       | `go vet ./...`                            | exit 0              |
| Test (knowledge)| `go test ./internal/knowledge/...`  | `ok`                |
| Full tests | `go test ./...`                          | all `ok`            |
| Diff hygiene | `git diff --check`                      | no output           |

(In a TLS-intercepting sandbox, Go commands may need a GOPROXY shim; GOSUMDB
stays on.)

## Scope

**In scope** (delete these two files entirely):
- `internal/knowledge/rerank.go`
- `internal/knowledge/rerank_test.go`

**Out of scope** (do NOT touch):
- Any other file in `internal/knowledge/` (knowledge.go, etc.) — they do not use
  the rerank symbols; the package's `llm` import lives only in `rerank.go` and
  disappears with the file.
- `plans/073-embedding-recall-spike.md`, `PRODUCT.md`, `internal/llm/*` — leave
  them.
- `BALAUR_EMBED_MODEL` anywhere — that env var configures the embed model
  generally and is unrelated to the rerank flag; do not remove it.

## Git workflow

- Land on `main`; if dispatched, base off `origin/main`. Conventional-commit
  subject, e.g. `refactor(knowledge): delete the unwired embedding-rerank spike (plan 073 prototype)`. Commit/push only when the operator instructs.

## Steps

### Step 1: Re-confirm the symbols are dead (do this before deleting)

Run the three greps from "Current state". If any shows a live (non-test,
non-definition) caller of `RerankByEmbedding`/`cosine`/`rerankEnabled`, or a doc
that advertises `BALAUR_EMBED_RERANK` as a shipped feature, STOP and report —
the spike is no longer dead and deletion would remove a used path or make a doc
lie.

### Step 2: Delete the two files

```
git rm internal/knowledge/rerank.go internal/knowledge/rerank_test.go
```
(or delete them with your file tools).

**Verify**: `ls internal/knowledge/rerank.go internal/knowledge/rerank_test.go 2>&1` → "No such file".

### Step 3: Build, vet, test

Run build, `go vet ./...`, `go test ./internal/knowledge/...`, then full
`go test ./...`.

**Verify**: all green. The `internal/knowledge` package must still compile — the
now-unused `llm` import vanished with `rerank.go`, so there is no orphaned
import. `go vet` confirms this.

## Test plan

No new tests. Deletion only. The guard is that the full suite stays green and
`go vet ./...` reports no orphaned import in `internal/knowledge`. Removing
`rerank_test.go` removes the only tests of the deleted code — that is intended.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `internal/knowledge/rerank.go` and `rerank_test.go` do not exist
- [ ] `grep -rn "RerankByEmbedding\|rerankEnabled\|BALAUR_EMBED_RERANK" --include='*.go' internal/ main.go` → empty
- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` all `ok`
- [ ] `git diff --check` → no output
- [ ] `plans/073-embedding-recall-spike.md` and `PRODUCT.md` are unchanged
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back if:

- Step 1 finds any live caller or a doc that claims the rerank feature ships.
- Deleting the files causes a build error anywhere outside `internal/knowledge`
  (means an external caller existed) — STOP; do not start deleting callers.
- `go vet` flags a problem you cannot resolve by the deletion alone.

## Maintenance notes

- The plan-073 design doc is retained intentionally; if "sharper recall" is
  pursued later, rebuild from that design and wire it into `knowledge.BuildContext`
  / `SearchActive` as a real, tested integration (the 4-caller change the spike
  documented) — not as another parked prototype.
- A reviewer should confirm `internal/knowledge` no longer imports `internal/llm`
  and that lexical recall behavior is unchanged.
