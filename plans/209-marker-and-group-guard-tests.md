# Plan 209: Two guard tests — tool-result marker round-trip/isolation (`internal/tools`) and `heads.Groups`↔`ToolsForHead` wiring (`internal/turn`)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat d4f38d0..HEAD -- internal/tools internal/turn/tools.go internal/heads/heads.go` (expect EMPTY — re-baselined after 206/208)
> If any in-scope source changed, re-confirm the marker/Parse function names and
> `heads.Groups` against the live code; on a mismatch, treat it as a STOP
> condition.

## Status

- **Priority**: P3
- **Effort**: S (tests only — NO production code changes)
- **Risk**: LOW
- **Depends on**: none. (If plan #210's `turn.Tools` refactor lands first, the
  group test still holds — it asserts behavior, not structure.)
- **Category**: tests
- **Planned at**: commit `07fb4d6`, 2026-06-26

## Why this matters

Two compile-unchecked contracts can drift silently:

1. **Tool-result markers** (finding #2). `internal/tools` defines five
   NUL-prefixed markers, each with a `Mark*` encoder and a `Parse*` decoder:
   `UICardMarker`/`MarkUICard`/`ParseUICard`,
   `ChoicesMarker`/`MarkChoices`/`ParseChoices`,
   `ProposalMarker`/`MarkProposal`/`ParseProposal`,
   `RefreshMarker`/`MarkRefresh`/`ParseRefresh`,
   `ArtifactMarker`/`MarkArtifact`/`ParseArtifact`. The encoding contract lives
   nowhere and is enforced nowhere; each gateway re-decodes the set in its own
   ladder (`web/chatstream.go` does all five, `cli/chat.go` only two). A new
   sixth marker could collide with, or be mis-parsed by, an existing `Parse*`.
   A colocated round-trip + cross-isolation test pins the contract: every marker
   round-trips through its own `Parse*`, and every `Parse*` rejects every OTHER
   marker's output and plain text.

   > This does NOT change the per-gateway decoding (that divergence is correct
   > medium-adaptation — a `uicard` has no terminal rendering, so the CLI rightly
   > skips it). It only guards the encode/decode contract. Do **NOT** add an
   > `Event.Card` field or a `tools.ClassifyResult` — out of scope.

2. **Head capability groups** (finding #12). `heads.Groups`
   (`internal/heads/heads.go:38`) and the `switch`/`if sel[...]` ladder in
   `turn.ToolsForHead` (`internal/turn/tools.go:69`) must stay in sync by hand. A
   typo (a group key in `heads.Groups` that no branch in `ToolsForHead` handles)
   yields a head whose group maps to no tools — compile-unchecked. A table-driven
   guard test in package `turn` (which already imports `heads`) asserts every
   group in `heads.Groups` adds at least one tool beyond the always-on core. Do
   **NOT** add a `turn.ToolGroups()` for `heads` to reference — that inverts a
   domain→pipeline dependency and risks a cycle.

## Current state

### Markers (`internal/tools`)

Constants and functions (file:line):
- `UICardMarker` / `MarkUICard(typ string, params map[string]string, modelText string) string` / `ParseUICard(s string) (typ, query, rest string, ok bool)` — `ui.go:25/29/43`. `ParseUICard` returns `ok=false` unless `typ` is a registered card (`cards.Get`).
- `ChoicesMarker` / `MarkChoices(prompt string, choices []Choice, modelText string) string` / `ParseChoices(s string) (prompt string, choices []Choice, modelText string, ok bool)` — `choices.go:17/33/40`. `Choice` is a struct at `choices.go:20`.
- `ProposalMarker` / `MarkProposal(kind, id, modelText string) string` / `ParseProposal(s string) (kind, id, rest string, ok bool)` — `knowledge.go:43/48/53`.
- `RefreshMarker` / `MarkRefresh(types []string, modelText string) string` / `ParseRefresh(s string) (types []string, rest string, ok bool)` — `refresh.go:15/19/27`.
- `ArtifactMarker` / `MarkArtifact(cs []cards.Card, title, modelText string) string` / `ParseArtifact(s string) (title string, cs []cards.Card, rest string, ok bool)` — `artifact.go:20/30/41`. `ParseArtifact` calls `cards.ValidateCards`, so its sample cards must validate.

The card registry is populated by `cards` package `init()` (so `cards.All()` is
non-empty in any test importing `cards`; `internal/tools` does). `cards.Spec` has
`Type` + `Params []ParamSpec`; the `"today"` card has **no params** (a safe,
always-valid sample). `cards.Card{Type: "today"}` validates cleanly.

### Group wiring (`internal/turn/tools.go` + `internal/heads/heads.go`)

`heads.Groups = []string{"memory", "tasks", "life", "journal", "os", "extensions"}`.

`turn.ToolsForHead(app, groups)`:
- empty `groups` → returns the full `Tools(app)`.
- non-empty `groups` matching nothing → returns the always-on core
  (`ChoiceTools` + `UITools` + `HeadsTools` + `ProfileTools` + `self`).
- each handled group appends ≥1 tool: `memory`→`KnowledgeTools`, `tasks`→
  `TaskTools`, `life`→`LifeTools`, `journal`→`JournalTools`, `os`→`OSAccess`
  (only when `BALAUR_OS_ACCESS=1`), `extensions`→`ext.ProposeTool` (+ approved
  `ext.Tools`).

## Commands you will need

| Purpose   | Command                                            | Expected         |
|-----------|----------------------------------------------------|------------------|
| Test pkg  | `go test ./internal/tools/... ./internal/turn/...` | PASS             |
| Full test | `go test ./...`                                     | all pass         |
| Vet       | `go vet ./...`                                       | exit 0           |
| gofmt     | `gofmt -l internal/tools internal/turn`            | prints nothing   |

> If `go test ./...` fails the link step with "No space left on device", set
> `TMPDIR=/home/alex/.cache/go-tmp` and retry.

## Scope

**In scope** (test files only):
- `internal/tools/markers_test.go` (create)
- `internal/turn/toolsforhead_test.go` (create, or add to an existing `internal/turn/*_test.go`)

**Out of scope** (do NOT touch):
- Any production file. This plan adds tests only.
- The per-gateway decode ladders in `web`/`cli` — their divergence is
  intentional.
- No `Event.Card`, no `tools.ClassifyResult`, no `turn.ToolGroups()`.

## Git workflow

- Branch: `advisor/209-marker-and-group-guard-tests`
- Conventional-commit subject, e.g. `test(tools,turn): guard marker contract + head-group wiring`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Marker round-trip + isolation test

Create `internal/tools/markers_test.go`. Build a table of all five markers, each
with a sample produced by its `Mark*` and a predicate wrapping its `Parse*`'s
`ok`. Use the no-params `"today"` card for the uicard and artifact samples:

```go
package tools

import "testing"

// markerCases is EXHAUSTIVE: every NUL-prefixed marker constant in this package
// must appear here. Adding a Mark*/Parse* pair without adding a case is exactly
// the silent-drift this test guards against.
func markerCases() []struct {
	name   string
	sample string
	parse  func(string) bool // its own Parse*'s ok
} {
	return []struct {
		name   string
		sample string
		parse  func(string) bool
	}{
		{"uicard", MarkUICard("today", map[string]string{}, "showing today"),
			func(s string) bool { _, _, _, ok := ParseUICard(s); return ok }},
		{"choices", MarkChoices("Pick one", []Choice{{ /* fill required fields per choices.go */ }}, "choose"),
			func(s string) bool { _, _, _, ok := ParseChoices(s); return ok }},
		{"proposal", MarkProposal("memory", "abc123", "proposed a memory"),
			func(s string) bool { _, _, _, ok := ParseProposal(s); return ok }},
		{"refresh", MarkRefresh([]string{"quests"}, "refreshed"),
			func(s string) bool { _, _, ok := ParseRefresh(s); return ok }},
		{"artifact", MarkArtifact([]cards.Card{{Type: "today"}}, "Cluster", "showing a cluster"),
			func(s string) bool { _, _, _, ok := ParseArtifact(s); return ok }},
	}
}

func TestMarkersRoundTripAndIsolate(t *testing.T) {
	cases := markerCases()

	// Round-trip: each marker's own Parse accepts its own sample.
	for _, c := range cases {
		if !c.parse(c.sample) {
			t.Errorf("%s: own Parse rejected its own Mark output", c.name)
		}
	}

	// Isolation: every Parse rejects every OTHER marker's output.
	for _, owner := range cases {
		for _, other := range cases {
			if owner.name == other.name {
				continue
			}
			if owner.parse(other.sample) {
				t.Errorf("%s.Parse accepted %s's output (markers must not cross-match)", owner.name, other.name)
			}
		}
	}

	// Plain text: every Parse rejects unmarked text.
	for _, c := range cases {
		if c.parse("just some assistant text, no marker") {
			t.Errorf("%s.Parse accepted plain text", c.name)
		}
	}
}
```

Notes for the executor:
- Add the `cards` import (`"github.com/alexradunet/balaur/internal/cards"`) — the
  artifact sample needs `cards.Card`.
- For the `choices` case, open `internal/tools/choices.go` and fill the `Choice`
  struct literal with whatever fields it requires so `MarkChoices` produces a
  parseable sample (`ParseChoices` round-trips JSON — any non-empty valid
  `Choice` works). If `Choice` has no required-non-zero fields, `[]Choice{{}}` is
  fine.
- If `"today"` is no longer a registered no-params card (drift), pick another
  registered card with `len(spec.Params) == 0` by scanning `cards.All()` — or
  for the uicard sample any registered type works (`ParseUICard` doesn't validate
  params, only `cards.Get(typ)`); only the artifact sample needs a no-required-
  params type so `cards.ValidateCards` passes.

**Verify**: `go test ./internal/tools/ -run TestMarkersRoundTripAndIsolate -v` → PASS

### Step 2: Head-group wiring guard test

Create `internal/turn/toolsforhead_test.go` (or append to an existing
`internal/turn/*_test.go`). Use the same temp-app constructor the other
`internal/turn` tests use (it must run the migration chain so the `extensions`
collection exists — check a sibling test in `internal/turn` for the exact
helper, e.g. a `tests.NewTestApp`/`store` temp-app pattern):

```go
package turn

import (
	"testing"

	"github.com/alexradunet/balaur/internal/heads"
)

func TestToolsForHeadGroupsAllWired(t *testing.T) {
	t.Setenv("BALAUR_OS_ACCESS", "1") // so the "os" group contributes tools
	app := newTestApp(t)              // <-- match the existing internal/turn test helper
	// defer app.Cleanup() if the helper requires it

	// A non-empty group slice that matches nothing returns the always-on core.
	baseline := len(ToolsForHead(app, []string{"__definitely_not_a_real_group__"}))

	for _, g := range heads.Groups {
		got := len(ToolsForHead(app, []string{g}))
		if got <= baseline {
			t.Errorf("capability group %q is in heads.Groups but ToolsForHead wires no extra tools for it (got %d, core-only baseline %d) — the switch in turn/tools.go and heads.Groups have drifted", g, got, baseline)
		}
	}
}
```

Why it works: `ToolsForHead` with a bogus non-empty group returns only the
always-on core; each real group appends ≥1 tool, so `got > baseline`. A group
key in `heads.Groups` with no matching branch (a typo) would equal `baseline` and
fail the test.

**Verify**: `go test ./internal/turn/ -run TestToolsForHeadGroupsAllWired -v` → PASS

### Step 3: Full verification

**Verify**:
- `gofmt -l internal/tools internal/turn` → prints nothing
- `go vet ./...` → exit 0
- `go test ./internal/tools/... ./internal/turn/...` → PASS
- `go test ./...` → all pass

## Test plan

This plan IS tests. Self-checks:
- Sanity-prove the marker test bites: temporarily make `ParseRefresh` also accept
  the `ProposalMarker` prefix (or comment out a `HasPrefix` guard) and confirm
  `TestMarkersRoundTripAndIsolate` fails; revert.
- Sanity-prove the group test bites: temporarily add a bogus `"typo"` entry to a
  local copy of the groups list passed in (do NOT edit `heads.Groups`) and
  confirm the loop would flag it — or temporarily comment out the `if sel["life"]`
  branch in `ToolsForHead` and confirm the `"life"` case fails; revert.
- Verification: `go test ./internal/tools/... ./internal/turn/...` → PASS after revert.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `internal/tools/markers_test.go` exists; `TestMarkersRoundTripAndIsolate` passes
- [ ] `internal/turn/toolsforhead_test.go` (or the chosen file) defines `TestToolsForHeadGroupsAllWired`; it passes
- [ ] `go test ./internal/tools/... ./internal/turn/...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `gofmt -l internal/tools internal/turn` prints nothing
- [ ] No PRODUCTION (`*.go` non-test) files modified (`git status` shows only the two test files)
- [ ] `go test ./...` exits 0
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- A `Mark*`/`Parse*` name or signature differs from the list above (drift in
  `internal/tools`) — update the test and report.
- `cards.All()` is empty in the `tools` test (the `cards` init didn't run / no
  registered card has zero params) — report; the artifact sample needs a
  validating card.
- You cannot find the `internal/turn` temp-app test helper, or `ToolsForHead`
  panics without a migrated app — report which helper the sibling tests use.
- The group test's baseline (bogus group) does NOT return only the core (e.g.
  `ToolsForHead` treats an unknown non-empty group as "all") — the assertion
  shape is wrong; report.

## Maintenance notes

- The marker test's `markerCases` table must be extended whenever a new
  `Mark*`/`Parse*` marker is added — that is the point. (Go can't reflect over
  package consts, so the table is the source of truth; the prominent comment says
  so.) When you add a marker, also wire its decode into BOTH gateway ladders
  (`web/chatstream.go` and, if it has terminal meaning, `cli/chat.go`).
- The group test fails the instant `heads.Groups` gains a key with no matching
  branch in `turn.ToolsForHead` — fix the wiring, not the test.
- Reviewer: confirm these are tests only (no production diff) and that both tests
  actually fail when their guarded invariant is broken (the self-checks above).
