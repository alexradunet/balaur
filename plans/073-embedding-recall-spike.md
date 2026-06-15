# Plan 073: prove an embedding rerank can ride the SearchActive seam — spike a flag-gated second ranking stage

> **Executor instructions**: This is a **DESIGN/SPIKE** plan. Its deliverable is
> a validated design, a documented API decision, a thin prototype behind a flag,
> and a written list of open questions — **NOT** a finished production feature.
> Do the investigation steps in order; at each decision point, record the answer
> in the "Spike findings" section you append to this file (Step 6). Run every
> verification command and confirm the expected result before moving on. If
> anything in "STOP conditions" occurs, stop and report — do not improvise or
> expand scope. When done, update the status row for this plan in
> `plans/readme.md` — unless a reviewer dispatched you and told you they maintain
> the index.
>
> **Drift check (run first)**: `git diff --stat 1f8f55e..HEAD -- internal/knowledge/ internal/llm/llm.go internal/llmtest/ internal/ollama/`
> If any in-scope or cited file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: L
- **Risk**: MED
- **Depends on**: none
- **Category**: direction (DESIGN/SPIKE)
- **Planned at**: commit `1f8f55e`, 2026-06-15

## Why this matters

Balaur's per-turn memory recall is purely lexical today: FTS5 bm25 with a LIKE
fallback (`internal/knowledge/knowledge.go` `SearchActive`). The honesty ledger
in `README.md` (lines 444–445) reserves a second stage:

> - Embedding recall (FTS5 lexical recall shipped; `Embed()` seam reserved
>   for a second ranking stage behind the same SearchActive call)

Every plumbing piece already exists: `llm.Client.Embed(ctx, texts) ([][]float32, error)`
is implemented on the shared `OpenAIClient` (`internal/llm/openai.go:207-238`), a
local embed model is configured by default (`internal/ollama/presets.go`:
`DefaultEmbedModel = "embeddinggemma"`, wired into the client in
`internal/ollama/client.go`). What is *missing* is the design: where the rerank
slots in given that `SearchActive` has no `llm.Client` parameter, where candidate
embeddings get cached, how big K is, how to keep private memory off remote
providers, and how to default the feature OFF.

This spike answers those questions with a thin, reversible prototype — a cosine
helper plus a flag-gated `RerankByEmbedding` function over the *existing*
candidate set, tested with the `llmtest` fake `Embed` — and a written list of the
follow-up work (chiefly: persisting candidate embeddings). It does **not** build
production recall. The value is a de-risked, decided design so the real feature
is a known quantity, not a guess.

## Current state

### The recall chokepoint — `internal/knowledge/knowledge.go:228-289`

`SearchActive` is the single recall function called per turn. Its signature has
**no `llm.Client`** — this is the central design constraint:

```go
// SearchActive finds active memories matching any of the given terms.
// When a FTS5 sidecar index is available in app.Store() (key
// search.StoreKey), results are bm25-ranked by the index and the LIKE path
// is skipped. On any error, a missing index, or zero FTS results, it falls
// through to the plain LIKE body unchanged — deterministic, offline-safe.
func SearchActive(app core.App, terms []string, limit int) ([]*core.Record, error) {
	// --- FTS5 fast path ---
	if raw, ok := app.Store().GetOk(search.StoreKey); ok {
		if ix, ok := raw.(*search.Index); ok && ix != nil {
			ids, err := ix.Query(terms, limit)
			...
			// returns active records in FTS rank order, capped at limit
		}
	}
	// --- LIKE fallback (unchanged) ---
	...
}
```

It returns `[]*core.Record` in rank order, already capped at `limit`. A rerank
would reorder this returned slice — it does **not** need to touch the FTS or LIKE
query bodies.

### The per-turn caller — `internal/knowledge/context.go:30-72`

`BuildContext(app core.App, userMessage string)` is the only per-turn caller of
interest. It **also has no `llm.Client`** and calls `SearchActive` with a fixed
`recallLimit = 6`:

```go
const (
	upfrontLimit = 12
	recallLimit  = 6
)

func BuildContext(app core.App, userMessage string) (string, []*core.Record) {
	...
	recalled, err := SearchActive(app, recallTerms(userMessage), recallLimit)
	...
}
```

The package comment on `recallTerms` already anticipates this work: *"Replaced
wholesale when real ranking lands; callers won't notice."* (`context.go:88`).

**Design consequence (decide in Step 2):** because neither `SearchActive` nor
`BuildContext` carries a client, the spike prototype MUST be a **new function**
that takes the client and candidate slice explicitly — do **not** widen
`SearchActive`'s signature in the spike. Threading a client all the way down to
`BuildContext` is a real refactor and is an **open question / follow-up**, not
spike work.

### All `SearchActive` callers (so you know the blast radius of any signature change — and why you must NOT change it)

- `internal/knowledge/context.go:45` — per-turn `BuildContext` (no client in scope)
- `internal/cli/knowledge.go:156` — CLI search subcommand
- `internal/tools/knowledge.go:122` — the model's `recall` tool

Three callers, none of which pass a client. Changing the signature is a
4-file ripple; the spike avoids it by adding a separate rerank function.

### The embed seam is fully built — `internal/llm/openai.go:207-238`

```go
func (c *OpenAIClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	model := c.EmbedModel
	if model == "" {
		model = c.Model
	}
	resp, err := c.post(ctx, "/embeddings", map[string]any{
		"model": model,
		"input": texts,
	})
	...
	vecs := make([][]float32, len(out.Data))
	for _, d := range out.Data {
		if d.Index < 0 || d.Index >= len(vecs) {
			return nil, fmt.Errorf("embedding index %d out of range", d.Index)
		}
		vecs[d.Index] = d.Embedding
	}
	return vecs, nil
}
```

`Embed` is on the `llm.Client` interface (`internal/llm/llm.go:48-49`):

```go
	// Embed returns one embedding vector per input text.
	Embed(ctx context.Context, texts []string) ([][]float32, error)
```

It returns `[][]float32`, one vector per input text, index-aligned to the input
slice. **This is the exact contract the spike's cosine + rerank code consumes.**

### A local embed model is already configured — `internal/ollama/presets.go:15,37` + `internal/ollama/client.go`

```go
const (
	...
	DefaultEmbedModel    = "embeddinggemma"
	...
)

// EmbedModel is the dedicated embedding tag (BALAUR_EMBED_MODEL or the default).
func EmbedModel() string {
	if m := os.Getenv("BALAUR_EMBED_MODEL"); m != "" {
		return m
	}
	return DefaultEmbedModel
}
```

`ollama.NewClient` wires it in (`client.go`): `EmbedModel: EmbedModel()`. So a
*local* embed path exists today with no new config. The local-vs-remote privacy
question (Step 4) is about which client the rerank is allowed to use, not about
whether embedding works.

### The `llmtest` fake already stubs `Embed` — `internal/llmtest/llmtest.go:78-80`

```go
func (f *ScriptedClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, nil
}
```

It currently returns `(nil, nil)` — it satisfies the interface but cannot script
vectors. **The spike's first code task (Step 1) is to make this fake scriptable**
so the rerank test can drive a deterministic reorder. The brief's STOP condition
"llmtest has no way to fake Embed" is already true in the weak sense (it returns
nil); Step 1 resolves it by adding a scriptable hook. If extending `llmtest`
turns out to require changes that ripple into other packages' tests, STOP and
report.

### Test helpers (use these; never hit a real model or daemon)

- `internal/storetest.NewApp(t) core.App` (`internal/storetest`, `func NewApp(t *testing.T) core.App` at line 18) — temp-dir PocketBase app with all migrations applied. Used by every test in `internal/knowledge/knowledge_test.go`.
- `internal/llmtest` — the scripted `llm.Client` fake. Tests never construct a real `OpenAIClient`.
- Model `github.com/alexradunet/balaur`, Go 1.26.4, PocketBase embedded, `CGO_ENABLED=0` required.

### Conventions that bind this plan

- `gofmt` is law (a PostToolUse hook reformats on edit, but verification still requires `gofmt -l .` to print nothing).
- Errors are values: `fmt.Errorf("doing x: %w", err)`, return early, no panics in library code.
- Deterministic, offline, free behavior is the default; the network/LLM path is **opt-in** and the trade-off is documented in a comment (AGENTS.md KISS/YAGNI rule). The rerank must therefore be OFF unless explicitly enabled and must degrade cleanly to pure-FTS on any embed error / missing model / disabled flag.
- **No new dependency.** Cosine similarity is ~15 lines of stdlib `math`; do not import a vector library.
- Threat model is single-owner, loopback-first, full-trust — do NOT add malicious-input defenses. The *only* privacy concern in scope is the README/AGENTS safety rule: keep secrets out of content sent to **remote** providers.
- Tests: standard `testing`, table-driven where it helps, no assertion framework. Model new tests after `internal/knowledge/knowledge_test.go` (e.g. `TestSearchActiveFTSPath` at line 265).

## Commands you will need

| Purpose | Command | Expected on success |
|---|---|---|
| Drift | `git diff --stat 1f8f55e..HEAD -- internal/knowledge/ internal/llm/llm.go internal/llmtest/ internal/ollama/` | empty |
| Vet (package) | `go vet ./internal/knowledge/ ./internal/llmtest/` | exit 0 |
| Package tests | `go test ./internal/knowledge/ ./internal/llmtest/` | all pass |
| Whole-tree tests | `go test ./...` | all pass |
| Host build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Format check | `gofmt -l .` | prints nothing |
| Whitespace check | `git diff --check` | no output |
| Confirm flag is OFF by default | `grep -n "BALAUR_EMBED_RERANK" internal/knowledge/*.go` | the flag is read in exactly one place, default disabled |

## Scope

**In scope** (prototype only — the code slice is deliberately tiny and reversible):

- `internal/llmtest/llmtest.go` — make the fake `Embed` scriptable (a settable hook or vectors queue). MODIFY.
- `internal/knowledge/rerank.go` — **new file**: the cosine helper + a flag-gated `RerankByEmbedding` function over an existing candidate slice. CREATE.
- `internal/knowledge/rerank_test.go` — **new file**: table-driven test using the scripted `Embed` proving (a) the reorder happens and (b) absent/empty/erroring embed leaves FTS order unchanged. CREATE.
- This plan file itself — append a "Spike findings" section (Step 6).

**Out of scope** (do NOT touch — these are the follow-up / the parts a spike deliberately defers):

- **Persistence/caching of candidate embeddings** — the big follow-up. The spike recomputes per query for top-K only; it does NOT add a sidecar vector column, a schema migration, or any cache. Listing this as the #1 open question IS the deliverable; building it is NOT.
- **`SearchActive`'s signature** (`internal/knowledge/knowledge.go`) and **`BuildContext`'s signature** (`internal/knowledge/context.go`) — do NOT add an `llm.Client` parameter to either in the spike. Wiring the rerank into the live per-turn path is a follow-up; the spike's `RerankByEmbedding` is a standalone function proven by test only.
- The three `SearchActive` callers (`internal/cli/knowledge.go`, `internal/tools/knowledge.go`, `internal/knowledge/context.go`) — unchanged.
- **Any remote-embed-by-default path.** If the prototype needs to choose a client, it must prefer the LOCAL embed client; never send memory content to a remote provider by default (README/AGENTS safety rule). The spike does not actually call a network embed — it uses the fake — so this is a *documented decision*, not code.
- Schema migrations, the `internal/search` FTS index, the `ollama`/`llm` packages themselves (read-only references).
- `plans/readme.md` content beyond your own status row.

## Git workflow

- Branch: `improve/073-embedding-recall-spike`
- Commit style: conventional commits, matching `git log` (e.g. `spike(knowledge): flag-gated embedding rerank over the FTS candidate set`). One commit for the prototype + findings is fine.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

The spike order is: **make the test seam → define the API → prototype → measure/verify → decide the open questions → write findings.** Each step has a decision point or a verification.

### Step 1 — Make the `llmtest` fake `Embed` scriptable (the test seam)

The fake currently returns `(nil, nil)` for `Embed` (`internal/llmtest/llmtest.go:78-80`), which cannot drive a deterministic reorder. Add a scriptable hook so a test can return chosen vectors per input.

Add a field to `ScriptedClient` and rewrite `Embed`:

```go
// EmbedFunc, when non-nil, produces the embedding vectors for Embed.
// Lets a test script a deterministic reorder without a real model.
EmbedFunc func(texts []string) ([][]float32, error)
```

```go
func (f *ScriptedClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if f.EmbedFunc != nil {
		return f.EmbedFunc(texts)
	}
	return nil, nil // default: no embeddings scripted
}
```

Keep the default `(nil, nil)` path so existing callers that don't set `EmbedFunc`
are unchanged. Do not touch `ChatStream` or the reply queue.

**Decision point**: confirm no other test in the tree sets `Embed` behavior on
`ScriptedClient` in a way this breaks: `grep -rn "\.EmbedFunc\|ScriptedClient" --include=*.go .`
should show only `internal/llmtest` and your new test. If another package relies
on the old nil behavior, the field is additive and default-off, so it stays
green — but verify.

**Verify**:
```
go vet ./internal/llmtest/
go test ./internal/llmtest/ ./...   # whole tree still green; the field is additive
gofmt -l internal/llmtest/llmtest.go   # prints nothing
```

### Step 2 — Define the rerank API (decision, write it as the file's doc comment)

Decide and record the signature. The constraint from "Current state" is that
neither `SearchActive` nor `BuildContext` carries a client, so the spike API is a
**standalone function** that takes the candidate slice and a client explicitly:

```go
// RerankByEmbedding reorders an already-ranked candidate slice (the FTS/LIKE
// output of SearchActive) by cosine similarity between the query and each
// candidate's text. It is a SECOND ranking stage: candidates is assumed to be
// the small top-K from the lexical stage, NOT the whole memory store.
//
// It is opt-in and degrades cleanly to the input order unchanged when:
//   - the rerank flag is off (rerankEnabled()),
//   - client is nil,
//   - Embed returns an error or a vector count that doesn't match,
//   - candidates has 0 or 1 element.
// This preserves the deterministic-offline default: absent embeddings, recall
// behaves exactly as the lexical stage alone.
//
// NETWORK: Embed may hit the configured embed model. By design the caller
// passes the LOCAL embed client; memory content must not be sent to a remote
// provider by default (README/AGENTS safety rule).
func RerankByEmbedding(ctx context.Context, client llm.Client, query string, candidates []*core.Record) []*core.Record
```

Note the function returns the slice (no error): a spike rerank that *fails*
silently falls back to input order — a failed rerank must never break recall.
Record any deviation from this signature in Step 6.

**Decision point — text to embed per candidate**: choose the candidate text used
for the embedding. Reuse the same fields the FTS index uses
(`internal/search/index.go`: `title, content, when_to_use, category`). The
simplest correct choice is `title + ": " + content` (mirrors
`writeMemoryLine` in `context.go:74-84`). Record the choice.

**Decision point — K**: the candidate set is already capped by `SearchActive`'s
`limit` (per-turn `recallLimit = 6` in `context.go:24`). The spike does NOT
introduce a separate larger K; it reranks whatever it is handed. Record "K = the
caller's existing limit; a larger K to widen the lexical net before reranking is
an open question" in findings.

**Verify**: no command yet — this step is the documented API decision that Step 3
implements.

### Step 3 — Prototype: cosine helper + flag-gated `RerankByEmbedding` in `internal/knowledge/rerank.go`

Create `internal/knowledge/rerank.go`. Three pieces:

1. **Flag gate** (default OFF — the opt-in rule):
   ```go
   // rerankEnabled reports whether the embedding rerank second stage is on.
   // OFF by default: deterministic lexical recall is the baseline; the
   // embedding path is opt-in (BALAUR_EMBED_RERANK=1) and documented as a
   // network path. See plans/073.
   func rerankEnabled() bool { return os.Getenv("BALAUR_EMBED_RERANK") == "1" }
   ```

2. **Cosine similarity** (~15 lines, stdlib `math` only — NO new dependency):
   ```go
   // cosine returns the cosine similarity of two equal-length vectors, in
   // [-1,1]. Returns 0 for empty, length-mismatched, or zero-magnitude inputs
   // so a degenerate vector can never reorder above a real match.
   func cosine(a, b []float32) float64 {
   	if len(a) != len(b) || len(a) == 0 {
   		return 0
   	}
   	var dot, na, nb float64
   	for i := range a {
   		dot += float64(a[i]) * float64(b[i])
   		na += float64(a[i]) * float64(a[i])
   		nb += float64(b[i]) * float64(b[i])
   	}
   	if na == 0 || nb == 0 {
   		return 0
   	}
   	return dot / (math.Sqrt(na) * math.Sqrt(nb))
   }
   ```

3. **`RerankByEmbedding`** per the Step 2 signature. Behavior, in order:
   - If `!rerankEnabled() || client == nil || len(candidates) < 2`, return `candidates` unchanged.
   - Build the embed input: `[]string{query}` followed by one text per candidate (the field choice from Step 2). One `Embed` call with the whole batch (query at index 0).
   - On `err != nil` OR `len(vecs) != len(candidates)+1`, return `candidates` unchanged (clean degrade — log nothing private).
   - Compute `cosine(vecs[0], vecs[i+1])` for each candidate; **stable-sort** candidates by descending score (use `sort.SliceStable` so equal scores keep lexical order).
   - Return the reordered slice. Do not re-cap; the input was already capped.

   Keep it small. No caching, no persistence, no goroutines.

**Verify**:
```
gofmt -l internal/knowledge/rerank.go            # prints nothing
go vet ./internal/knowledge/                      # exit 0
CGO_ENABLED=0 go build ./...                      # exit 0
grep -n "BALAUR_EMBED_RERANK" internal/knowledge/rerank.go   # flag read once, default off
```

### Step 4 — Decision point: local-only embed by default (privacy), written down

No code in the spike sends real network traffic (the test uses the fake). But the
design decision must be recorded because it gates the follow-up: **the rerank
must use the LOCAL embed client by default**; memory content must not flow to a
remote provider unless the owner explicitly opts in. The local client already
exists (`ollama.NewClient` with `EmbedModel: embeddinggemma`). Record in findings:
"When wired into the live path (follow-up), `RerankByEmbedding` must be passed the
local Ollama embed client, never the active remote chat client, unless a future
explicit opt-in setting says otherwise (README safety rule: keep secrets out of
content sent to remote providers)."

**Verify**: this is a written decision (Step 6), no command.

### Step 5 — Prototype test: prove reorder + clean fallback (`internal/knowledge/rerank_test.go`)

Model the test after `internal/knowledge/knowledge_test.go` (table-driven, no
assertion framework, `storetest.NewApp(t)` for records). Create active memory
records, then call `RerankByEmbedding` with a scripted `llmtest` client.

Cover at least these cases:

1. **Reorder happens (flag on)**: set `t.Setenv("BALAUR_EMBED_RERANK", "1")`.
   Build 3 candidates in a known lexical order. Script `EmbedFunc` to return
   vectors such that the *last* candidate is closest to the query vector (e.g.
   query `[1,0]`; candidates `[0,1]`, `[0.5,0.5]`, `[1,0]`). Assert the returned
   order is reranked (the `[1,0]` candidate first), differing from input order.
2. **Flag off → unchanged**: do NOT set the env var (or set `"0"`). Same inputs;
   assert the returned slice is byte-identical order to the input. Prove the
   feature is truly off by default.
3. **Embed error → unchanged**: flag on, `EmbedFunc` returns `(nil, errors.New("boom"))`.
   Assert input order is preserved (clean degrade — recall never breaks).
4. **Vector-count mismatch → unchanged**: flag on, `EmbedFunc` returns one fewer
   vector than `len(candidates)+1`. Assert input order preserved.
5. **Nil client / <2 candidates → unchanged**: quick guards.

Also add a direct `cosine` unit test (table-driven): identical vectors → ~1.0,
orthogonal → 0, length mismatch → 0, zero vector → 0. Use a small epsilon compare
for floats.

**Verify**:
```
go test ./internal/knowledge/ -run 'Rerank|Cosine' -v   # new tests pass
go test ./internal/knowledge/                            # whole package green
gofmt -l internal/knowledge/rerank_test.go               # prints nothing
```

### Step 6 — Write the "Spike findings" section into this file

Append a `## Spike findings (filled by executor)` section to THIS plan file
(`plans/073-embedding-recall-spike.md`) capturing the decisions and the open
questions the spike surfaced. It MUST answer/record:

- **Final `RerankByEmbedding` signature** as implemented (and any deviation from Step 2).
- **Candidate text choice** (which fields embedded) and **why**.
- **K decision**: confirmed the spike reranks the caller's existing limit; whether widening K before rerank is worth it (open question).
- **Caching — the #1 follow-up**: recomputing candidate embeddings per query is the cost. State the options the real feature must choose between: (a) a sidecar vector column alongside the FTS5 index in `internal/search` (embed on `Upsert`/`Rebuild`, store the vector, cosine in SQL or in Go); (b) recompute only top-K per query (what the spike does — cheap to build, expensive per turn); (c) an in-process LRU. Recommend one with a one-line rationale. Do NOT build it.
- **Wiring into the live path — the #2 follow-up**: `SearchActive`/`BuildContext` need an `llm.Client` threaded in (a 4-caller ripple: `context.go`, `cli/knowledge.go`, `tools/knowledge.go`). State the scope.
- **Local-vs-remote decision** (Step 4) restated as a hard requirement for the follow-up.
- **Measurement note**: record that the spike has NOT measured real embed latency or recall quality against a live model (tests use the fake). The follow-up must measure: per-turn added latency with `embeddinggemma` locally, and whether rerank actually improves recall on a small labeled set — measurement-first before shipping.

**Verify**: `grep -n "Spike findings" plans/073-embedding-recall-spike.md` → one match.

## Test plan

- New tests in `internal/knowledge/rerank_test.go`:
  - `TestRerankByEmbedding` (table-driven): reorder-on, flag-off-unchanged, embed-error-unchanged, vector-mismatch-unchanged, nil-client/short-input guards.
  - `TestCosine` (table-driven): identical→~1, orthogonal→0, length-mismatch→0, zero-vector→0.
- New/changed test seam in `internal/llmtest/llmtest.go`: the `EmbedFunc` hook (Step 1), exercised by the rerank test.
- Structural pattern to copy: `internal/knowledge/knowledge_test.go` (`TestSearchActiveFTSPath` for record seeding via `storetest.NewApp(t)`; `TestFilterActive` for the table-driven shape).
- No test hits a real model or daemon — `EmbedFunc` returns scripted vectors.
- Verification: `go test ./internal/knowledge/ ./internal/llmtest/` → all pass, including the new tests; `go test ./...` stays green.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `internal/llmtest/llmtest.go` has a scriptable `EmbedFunc` hook; the default (`nil`) still returns `(nil, nil)` so existing callers are unchanged.
- [ ] `internal/knowledge/rerank.go` exists with `cosine`, `rerankEnabled` (reads `BALAUR_EMBED_RERANK`, default OFF), and `RerankByEmbedding`; no new dependency (stdlib `math`/`sort`/`os`/`context` only).
- [ ] `internal/knowledge/rerank_test.go` exists with the cases listed; `go test ./internal/knowledge/ -run 'Rerank|Cosine'` passes.
- [ ] `grep -n "BALAUR_EMBED_RERANK" internal/knowledge/*.go` shows the flag read in exactly one place, default disabled.
- [ ] `SearchActive` and `BuildContext` signatures are **unchanged** (`git diff 1f8f55e..HEAD -- internal/knowledge/knowledge.go internal/knowledge/context.go` shows no signature change).
- [ ] `go vet ./...` exits 0; `go test ./...` passes; `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `gofmt -l .` prints nothing; `git diff --check` shows nothing.
- [ ] This file has a "Spike findings" section answering the open questions in Step 6 (chiefly: the caching follow-up and the client-threading follow-up).
- [ ] `git status --porcelain` shows only: `internal/llmtest/llmtest.go` (modified), `internal/knowledge/rerank.go` (new), `internal/knowledge/rerank_test.go` (new), `plans/073-embedding-recall-spike.md` (modified), and your `plans/readme.md` status row.
- [ ] `plans/readme.md` status row for 073 updated (unless your reviewer maintains it).

## STOP conditions

Stop and report back (do not improvise) if:

- **Drift**: the excerpts in "Current state" for `internal/knowledge/knowledge.go` (`SearchActive`), `internal/knowledge/context.go` (`BuildContext`/`recallLimit`), `internal/llm/llm.go` (the `Client` interface / `Embed`), `internal/llm/openai.go` (`Embed` returns `[][]float32`), or `internal/llmtest/llmtest.go` (the nil `Embed`) do not match the live files.
- **The signature assumption breaks**: you find you cannot prototype a rerank without changing `SearchActive`'s or `BuildContext`'s signature (e.g. a caller you didn't expect already threads a client, or `RerankByEmbedding` as a standalone function genuinely cannot be tested in isolation). Report the *refactor scope* (which of the 4 callers must change, and why) as an open question — do NOT perform the refactor in this spike.
- **The test seam can't be made**: extending `llmtest`'s `Embed` to be scriptable forces changes that ripple into other packages' tests or break the existing `(nil, nil)` contract. Report what broke; the spike then defines that seam as the single deliverable of Step 1 and stops.
- A verification command fails twice after a reasonable fix attempt.
- The work appears to need an out-of-scope file: a schema migration, the `internal/search` index, or any of the three `SearchActive` callers.
- You find yourself building caching/persistence of embeddings — that is explicitly the deferred follow-up, not spike work. Stop and record it in findings instead.

## Maintenance notes

For whoever picks up the real feature after this spike:

- The prototype's `RerankByEmbedding` is **standalone and unwired** — nothing calls it in the live per-turn path yet. Promoting it means threading an `llm.Client` (the LOCAL embed client) into `BuildContext` → `SearchActive`, a 4-caller change; that ripple is the deliberate cost the spike deferred.
- The hot cost the spike did NOT solve is **recomputing candidate embeddings per query**. The findings section recommends a caching strategy; revisit it with a measurement (per-turn latency under `embeddinggemma` locally) before shipping — measurement-first.
- A reviewer should scrutinize: (1) the flag truly defaults OFF and the lexical path is byte-identical when off; (2) every failure mode of `Embed` degrades to input order, never an error to the caller; (3) no memory content can reach a remote provider — the follow-up must pass the local embed client.
- Keep this in sync with `README.md:444-445` (the honesty ledger) and `internal/self/knowledge.md` when the feature actually ships — until then, recall is still lexical-only and the docs must say so.

## Spike findings (executed 2026-06-15 on branch `improve/073-embedding-recall-spike`; advisor-reviewed)

- **Final signature** (no deviation from the design): `func RerankByEmbedding(ctx context.Context, client llm.Client, query string, candidates []*core.Record) []*core.Record` — standalone in `internal/knowledge/rerank.go`, takes the client + candidates explicitly (does NOT widen `SearchActive`), returns the slice with no error (failures silently restore input order).
- **Candidate text embedded**: `title + ": " + content` (falls back to `title` when content is empty/equal) — mirrors `writeMemoryLine` (`context.go:74-84`) and the FTS5-indexed fields, keeping parity with the lexical stage.
- **K**: the spike reranks the caller's existing `recallLimit = 6`; no separate larger K. *Open question:* widening the FTS net (fetch ~24, rerank to 6) could improve quality at ~4× embed cost — measure before deciding.
- **Caching (the #1 follow-up, NOT built):** recompute-per-query is what the spike does (cheap to build, cost scales per turn). Recommended for the real feature: **(a) a sidecar vector column in `internal/search`** — embed on `Upsert`/`Rebuild`, persist the vector, cosine in Go after FTS retrieval — amortizing cost at write time and fitting the existing index seam. Rejected: (b) per-query recompute (only viable if local embed latency is sub-10ms), (c) in-process LRU (no advantage over (a)).
- **Wiring (the #2 follow-up):** neither `SearchActive` nor `BuildContext` carries an `llm.Client`; threading one is a 4-caller ripple (`context.go:45`, `cli/knowledge.go:156`, `tools/knowledge.go:122`, and `SearchActive` itself). Cleanest path: keep `SearchActive` unchanged, add the optional post-step in `BuildContext` only, threading the **local** embed client down from `internal/turn`; CLI/tool callers need no rerank on first ship.
- **Local-only (hard requirement):** `RerankByEmbedding` must always receive the LOCAL Ollama embed client (`embeddinggemma`), never the remote chat client — memory content must not leave the box without explicit owner opt-in (README/AGENTS safety rule). Design-level today (the prototype is unwired and tests use the `llmtest` fake).
- **Measurement (before shipping):** the spike measured neither real embed latency nor recall-quality lift (fake `Embed` only). The follow-up must measure (1) per-turn added latency under `embeddinggemma` locally — if >50ms, caching option (a) is mandatory — and (2) whether rerank actually improves recall on a small labeled set, before promoting it into `BuildContext`.

Code delivered (verified green: build/vet/`go test ./...`/gofmt all pass; flag `BALAUR_EMBED_RERANK` defaults OFF; `SearchActive`/`BuildContext` signatures unchanged): `internal/knowledge/rerank.go` (cosine + flag-gated rerank), `internal/knowledge/rerank_test.go` (reorder + 3 clean-fallback cases + cosine unit test), `internal/llmtest/llmtest.go` (scriptable `EmbedFunc` hook, default `(nil,nil)` preserved).
