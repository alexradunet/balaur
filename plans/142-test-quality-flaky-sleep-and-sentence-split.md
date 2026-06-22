# Plan 142: De-flake the autodate test sleep + handle `?` in `splitSentences`

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If a
> STOP condition occurs, stop and report. When done, update the status row for
> this plan in `plans/readme.md`.
>
> **Drift check (run first)**: `git diff --stat 0c06da8..HEAD -- internal/store/llm_settings_test.go internal/verify/verify.go`

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tests
- **Planned at**: commit `0c06da8`, 2026-06-22

## Why this matters

Two small, long-deferred quality items: (1) a test relies on a real
`time.Sleep(2ms)` so the autodate `updated` timestamp advances — a flaky,
wall-clock-dependent assertion the project's own rule ("no `time.Sleep` for
synchronization") forbids; and (2) `verify.splitSentences` (used by the
words-vs-deeds honesty check to chunk a reply into claim sentences) splits on
`.`/`!`/`\n` but not `?`, so a question-form claim isn't isolated.

## Current state

`internal/store/llm_settings_test.go` (~244-256) — the flaky sleep:
```go
	beforeUpdated := rec.GetString("updated")
	// Sleep long enough that the autodate timestamp can advance …
	time.Sleep(2 * time.Millisecond)
	// Same chat tag, different embed tag => the found record changes and MUST
	// still be persisted (the change path is not skipped).
	id2, err := SaveLocalModel(app, "gemma4:e4b", "embed-new")
	…
```
(Read the whole test function to see how `beforeUpdated` is later asserted — it
exists to prove the "changed → persisted" path ran.)

`internal/verify/verify.go` (100-104):
```go
func splitSentences(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return r == '.' || r == '!' || r == '\n'
	})
}
```

## Commands you will need

| Purpose | Command                              | Expected |
|---------|--------------------------------------|----------|
| Build   | `CGO_ENABLED=0 go build ./...`       | exit 0   |
| Tests   | `go test ./internal/store/ ./internal/verify/` | all pass |
| Lint    | `make lint`                          | exit 0   |

## Steps

### Step 1: Remove the flaky sleep; assert the changed content instead

Read the full test function around the sleep. The point of the test is that
saving with a *changed* field (a different embed tag) actually persists (the
"skip unchanged write" optimization does not swallow it). Make the assertion
deterministic by checking the **changed content field**, not the wall-clock
`updated` timestamp:
- Delete the `time.Sleep(2 * time.Millisecond)` line and the `beforeUpdated`
  capture if it was only used for the timestamp comparison.
- After `SaveLocalModel(app, "gemma4:e4b", "embed-new")`, re-find the record and
  assert its `embed_model` (or the relevant changed field) equals `"embed-new"` —
  proving the change persisted, with no dependency on time advancing.
- If `beforeUpdated`/`updated` is used elsewhere in the test for a different
  assertion, keep that part; only remove the sleep-dependent timestamp check.

**Verify**: `go test ./internal/store/ -run <the test's name>` → pass;
`grep -n "time.Sleep" internal/store/llm_settings_test.go` → returns nothing.

### Step 2: Split on `?` too

In `splitSentences`, add `?` to the separator set:
```go
		return r == '.' || r == '!' || r == '?' || r == '\n'
```

**Verify**: `go test ./internal/verify/` → pass.

### Step 3: Test `splitSentences`

Add a small table test `TestSplitSentences` to `internal/verify/verify_test.go`
(create if absent, else append) covering: a multi-sentence string with `.`/`!`/`?`
delimiters splits into the expected sentences; a single sentence returns one
element; empty input returns empty. Model the style on the existing
`internal/verify` tests.

**Verify**: `go test ./internal/verify/` → pass, including `TestSplitSentences`.

### Step 4: Full gate

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0; `make lint` → exit 0.

## Test plan

- Step 1 makes the existing autodate test deterministic (content assertion, no
  sleep) — it must still prove the "changed write persists" property.
- Step 3 adds `TestSplitSentences` covering `.`/`!`/`?`/`\n` splitting, single
  sentence, empty.
- Run the full `go test ./internal/store/ ./internal/verify/` to confirm both
  packages stay green.

## Done criteria

- [ ] `grep -n "time.Sleep" internal/store/llm_settings_test.go` returns nothing
- [ ] `grep -n "r == '?'" internal/verify/verify.go` returns a match
- [ ] `go test ./internal/store/ ./internal/verify/` passes, including `TestSplitSentences`
- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `make lint` exits 0
- [ ] Only `internal/store/llm_settings_test.go`, `internal/verify/verify.go`,
      `internal/verify/verify_test.go`, and `plans/readme.md` modified
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report if:
- The autodate test uses `beforeUpdated`/`updated` for an assertion that genuinely
  needs the timestamp to advance (not just the changed-content check) — if a
  deterministic content assertion can't replace it, report rather than weakening
  the test.
- Adding `?` to `splitSentences` breaks an existing `internal/verify` test (the
  honesty check's behavior shifted) — report; the `?` split should only *add*
  boundaries, not remove any.

## Scope

**In scope**: `internal/store/llm_settings_test.go`, `internal/verify/verify.go`,
`internal/verify/verify_test.go`, `plans/readme.md` (status row).
**Out of scope**: the `SaveLocalModel` production logic; the rest of the
words-vs-deeds verify logic beyond the sentence splitter.

## Git workflow

- Branch off `origin/main`: `improve/142-test-quality-sleep-and-sentence-split`.
- One commit; subject e.g. `test: de-flake autodate sleep; split sentences on ?`.
- Do NOT push or open a PR.

## Maintenance notes

- The "no `time.Sleep` for synchronization" rule is in the `go-standards` skill —
  assert on observable content, not on time advancing.
