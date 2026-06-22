# Plan 137: Tighten `kronk.Embed` to error instead of returning silent nil vectors

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If a
> STOP condition occurs, stop and report. When done, update the status row for
> this plan in `plans/readme.md`.
>
> **Drift check (run first)**: `git diff --stat 0c06da8..HEAD -- internal/kronk/client.go`

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug (latent)
- **Planned at**: commit `0c06da8`, 2026-06-22

## Why this matters

`kronk.Embed` pre-sizes its result slice to `len(texts)` and fills an entry only
when the model returned data for that input; otherwise the slot stays `nil` and
the function returns `(out, nil)` as success. A caller then gets a
`[][]float32` of the right length with `nil` holes and no error — a downstream
cosine/index op on a nil vector silently scores zero or panics. This is currently
latent (no production code routes recall through this `Embed` — recall is FTS5
lexical search, and the embedding-rerank spike was deleted in plan 121), so the
fix is a cheap contract-tightening that prevents a future footgun rather than
fixing a live bug. The OpenAI client's `Embed` already validates its index range;
this brings the local client to the same standard.

## Current state

`internal/kronk/client.go` — `Embed` (68-84):
```go
func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	krn, err := c.eng.embedModel(ctx, c.embedPath)
	if err != nil {
		return nil, err
	}
	out := make([][]float32, len(texts))
	for i, t := range texts {
		resp, err := krn.Embeddings(ctx, model.D{"input": t})
		if err != nil {
			return nil, fmt.Errorf("local embed: %w", err)
		}
		if len(resp.Data) > 0 {
			out[i] = resp.Data[0].Embedding
		}
	}
	return out, nil
}
```

## Commands you will need

| Purpose | Command                          | Expected |
|---------|----------------------------------|----------|
| Build   | `CGO_ENABLED=0 go build ./...`   | exit 0   |
| Vet     | `go vet ./...`                   | exit 0   |
| Tests   | `go test ./internal/kronk/`      | all pass |
| Lint    | `make lint`                      | exit 0   |

## Steps

### Step 1: Error on an empty embedding response

Replace the `if len(resp.Data) > 0 { out[i] = ... }` block with a fail-closed
guard:
```go
		if len(resp.Data) == 0 {
			return nil, fmt.Errorf("local embed: model returned no vector for input %d", i)
		}
		out[i] = resp.Data[0].Embedding
```

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0;
`grep -n "no vector for input" internal/kronk/client.go` → one match.

### Step 2: Full gate

**Verify**: `go vet ./...` → exit 0; `go test ./internal/kronk/` → all pass;
`make lint` → exit 0.

## Test plan

- **Testability note**: `Embed` calls the native `dlopen`'d llama.cpp engine
  (`c.eng.embedModel` → `krn.Embeddings`), which cannot run in a pure-Go CI env —
  the package's CGO/native paths are at 0% by design (the pure helpers like
  `bridge`/`map`/`presets` are well covered). So a direct unit test of the new
  guard would require the native runtime and is out of scope.
- Verify instead that the change is behavior-preserving on the happy path: the
  existing `internal/kronk` tests (which cover the non-native helpers) stay green,
  and the build compiles. The guard itself is a 2-line fail-closed check.
- Do NOT add a flaky or engine-dependent test to chase coverage here.

## Done criteria

- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go vet ./...` exits 0
- [ ] `grep -n "len(resp.Data) == 0" internal/kronk/client.go` returns a match
- [ ] `grep -n "len(resp.Data) > 0" internal/kronk/client.go` returns nothing
- [ ] `go test ./internal/kronk/` passes; `make lint` exits 0
- [ ] Only `internal/kronk/client.go` and `plans/readme.md` modified
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report if:
- A production caller of `Embed` turns out to depend on the nil-hole behavior
  (build/test fails) — `grep -rn "\.Embed(" internal/ | grep -v _test` should show
  no live recall caller; if one exists, report before changing the contract.

## Scope

**In scope**: `internal/kronk/client.go`, `plans/readme.md` (status row).
**Out of scope**: the OpenAI `Embed` (already validates); the recall path
(`internal/knowledge` — FTS5, doesn't use this); any batching redesign (the
"one input per call" KISS comment stays).

## Git workflow

- Branch off `origin/main`: `improve/137-kronk-embed-nil-vector-contract`.
- One commit; subject e.g. `fix(kronk): error on empty embedding response instead of nil vector`.
- Do NOT push or open a PR.

## Maintenance notes

- When/if vector recall is wired (the `SearchActive` second-stage rerank), this
  guarantee — every returned vector is non-nil or the call errors — is what makes
  the consumer safe. Keep it.
