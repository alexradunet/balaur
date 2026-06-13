# Plan 049: Cleanup & docs truth — delete SyntheticClient, fix env docs, mark the head-tools doc as design

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat dd9e60b..HEAD -- internal/llm/env.go README.md contrib/systemd/balaur.env.example docs/head-tools-design.md`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: run AFTER plan 048 lands if both are in flight
  (048 must not see `env.go` churn mid-run; no file overlap otherwise)
- **Category**: tech-debt + docs
- **Planned at**: commit `dd9e60b`, 2026-06-12

## Why this matters

Three small truths drifted. (1) `SyntheticClient` and its constants/env
vars are dead code — zero callers anywhere (its own comment admits "no
production callers"); the owner approved removal this cycle (deferred
since cycle 1). (2) The env documentation is wrong in both directions:
`BALAUR_ALLOWED_HOSTS` — a production-relevant security knob — is missing
from the README table and the systemd env example, while the env example
still documents `BALAUR_REMOTE_URL/MODEL/API_KEY`, which **no code reads**
(verified: zero grep hits), under a heading naming "Kronk", an engine that
was replaced by the llamafile supervisor. (3) `docs/head-tools-design.md`
reads like documentation of shipped behavior; it is a design for future
work (head chat is tool-free today — `internal/web/headsmgmt.go` runs
`turn.RunFor` with `Tools: nil`).

## Current state

### A. Dead code — `internal/llm/env.go:21-53`

```go
	SyntheticBaseURL    = "https://api.synthetic.new/v1"
	SyntheticSmallModel = "syn:small:text"
	SyntheticLargeModel = "syn:large:text"
```

…and (same file, lines 38-53):

```go
// SyntheticAPIKey reads the internal/experimental synthetic API credentials.
// BALAUR_SYNTHETIC_API_KEY and SYNTHETIC_API_KEY are undocumented, reserved
// for testing and internal development (SyntheticClient has no production callers).
func SyntheticAPIKey() string { ... }

func SyntheticClient(model string) *OpenAIClient { ... }
```

`grep -rn 'Synthetic' --include='*.go' internal main.go | grep -v env.go`
→ empty at `dd9e60b`. The constants that MUST stay in this file:
`DefaultChatModelName/File/URL`, `DefaultChatModelPath`,
`DefaultChatModelDownloadCommand`, `Collect` (everything non-Synthetic).

### B. README env table — `README.md:204-220`

The table lists 13 `BALAUR_*` vars (CHAT_MODEL, AUTO_MODEL, LLAMAFILE,
EMBED_MODEL, OS_ACCESS, SOURCE, MAX_STEPS, EXT_DIR, RECAP, NUDGE,
BRIEFING, BRIEFING_HOUR, DEV_SEED). `BALAUR_ALLOWED_HOSTS` is absent. It
is read at `internal/web/web.go:132` and documented only in
`docs/netbird.md:52-57`. The guard comment (`web.go:97-98`) describes it:
comma-separated `host[:port]` values allowed as Host headers beyond
loopback.

### C. systemd env example — `contrib/systemd/balaur.env.example`

Current content (relevant parts):

```
# Local GGUF model through Kronk:
# BALAUR_CHAT_MODEL=/path/to/model.gguf
# BALAUR_EMBED_MODEL=/path/to/embedding.gguf

# OpenAI-compatible endpoint:
# BALAUR_REMOTE_URL=http://127.0.0.1:11434/v1
# BALAUR_REMOTE_MODEL=qwen
# BALAUR_REMOTE_API_KEY=
```

Facts: "Kronk" is gone (local inference is the llamafile supervisor —
see `internal/llama/supervisor.go:1-10`); `BALAUR_REMOTE_*` has zero
readers in Go (remote providers are configured in the UI at
/settings/models and stored in PocketBase, not env). `BALAUR_HTTP` and
`BALAUR_DATA_DIR` at the top of the example are **legitimate** — they are
consumed by `contrib/systemd/balaur.service` (`ExecStart=… --http
${BALAUR_HTTP} --dir ${BALAUR_DATA_DIR}`), not by Go: leave them.

### D. Design doc framing — `docs/head-tools-design.md:1-7`

```markdown
# Head tools design: scoped tool access through the grants boundary

## Goal

Allow a sub-head's chat turns to call a limited set of tools, where that
set is derived from the head's grants rows — without breaking the existing
rule boundary or adding new schema.
```

No status line. The shipped reality: heads chat tool-free
(`internal/turn/turn.go` `RunFor` comment "scoped tools are a future
slice"; `internal/self/knowledge.md:23` "tool-free today").

Repo conventions: docs stay truthful to shipped behavior (AGENTS.md:
"Do not claim it in user-facing copy until real"); README env rows are
one-liners matching the existing table voice.

## Commands you will need

| Purpose   | Command                        | Expected on success |
|-----------|--------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...` | exit 0              |
| All tests | `go test ./...`                | ok                  |
| Vet/fmt   | `go vet ./...` / `gofmt -l .`  | silent / empty      |

Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need
the GOPROXY shim — see `docs/hyperagent-sandbox.md`.

## Scope

**In scope** (the only files you should modify):
- `internal/llm/env.go`
- `README.md` (env table only)
- `contrib/systemd/balaur.env.example`
- `docs/head-tools-design.md` (preamble only)

**Out of scope** (do NOT touch):
- `internal/self/knowledge.md` — already accurate (verified this cycle).
- `docs/netbird.md` — already accurate; the README row will complement it.
- Any Go file other than `env.go`; any test file (nothing tests Synthetic).
- The design doc's body/decisions — only the status preamble is added.

## Git workflow

- Branch: `advisor/049-cleanup-docs`
- Conventional commit, e.g. `chore: drop dead SyntheticClient; true up env docs; mark head-tools doc as design`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Delete the Synthetic block

In `internal/llm/env.go` remove: the three `Synthetic*` constants from the
const block, `SyntheticAPIKey`, and `SyntheticClient`. Keep everything
else byte-identical. Remove `"os"` from imports ONLY if nothing else in
the file uses it (check first).

**Verify**: `grep -rn 'Synthetic' --include='*.go' .` → no matches;
`CGO_ENABLED=0 go build ./...` → exit 0; `go test ./...` → ok.

### Step 2: README row

Add to the env table (keep alphabetical-ish position near the top with the
other serving knobs; match the table voice):

```markdown
| `BALAUR_ALLOWED_HOSTS` | (unset) | Comma-separated `host[:port]` values allowed as the Host header beyond loopback (LAN names, NetBird — see [docs/netbird.md](docs/netbird.md)) |
```

**Verify**: `grep -n 'BALAUR_ALLOWED_HOSTS' README.md` → 1 match.

### Step 3: Fix the env example

In `contrib/systemd/balaur.env.example`:

- `# Local GGUF model through Kronk:` → `# Local model (llamafile engine; see README "Models"):`
  and add `# BALAUR_LLAMAFILE=/path/to/llamafile` under it (the engine
  override, README-documented).
- Delete the three `BALAUR_REMOTE_*` lines and their
  `# OpenAI-compatible endpoint:` heading; replace with the comment
  `# OpenAI-compatible providers are configured in the UI under /settings/models.`
- Add, in its own block:
  ```
  # Allow non-loopback Host headers (LAN / NetBird names), comma-separated:
  # BALAUR_ALLOWED_HOSTS=balaur.netbird.cloud
  ```
- Leave `BALAUR_HTTP`/`BALAUR_DATA_DIR` untouched (consumed by the
  systemd unit, not Go).

**Verify**: `grep -n 'BALAUR_REMOTE\|Kronk' contrib/systemd/balaur.env.example` → no matches;
`grep -n 'BALAUR_ALLOWED_HOSTS' contrib/systemd/balaur.env.example` → 1 match.

### Step 4: Design-doc preamble

In `docs/head-tools-design.md`, insert between the title and `## Goal`:

```markdown
> **Status: design for future work — not shipped.** Sub-head chat is
> tool-free today (`turn.RunFor` passes no tools); this document specifies
> how scoped tools would be granted when that slice is built.
```

**Verify**: `head -5 docs/head-tools-design.md` shows the status line.

### Step 5: Full gate

**Verify**: `gofmt -l .` → empty; `go vet ./...` → silent;
`go test ./...` → ok; `CGO_ENABLED=0 go build ./...` → exit 0;
`git diff --check` → empty.

## Test plan

No new tests — Step 1 is deletion (nothing referenced it; the build and
existing suites are the net), Steps 2–4 are docs. `go test ./...` green is
the gate.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -rn 'Synthetic' --include='*.go' .` → no matches
- [ ] `grep -c 'BALAUR_ALLOWED_HOSTS' README.md contrib/systemd/balaur.env.example` → 1 each
- [ ] `grep -n 'BALAUR_REMOTE\|Kronk' contrib/systemd/balaur.env.example` → no matches
- [ ] `grep -n 'not shipped' docs/head-tools-design.md` → 1 match
- [ ] `go test ./...` exits 0; `gofmt -l .` empty; `go vet ./...` silent; `CGO_ENABLED=0 go build ./...` exits 0
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- `grep -rn 'Synthetic' --include='*.go' .` finds a caller outside
  `env.go` (drift since `dd9e60b` — the deletion premise is gone).
- A test references `BALAUR_SYNTHETIC_API_KEY` or `SYNTHETIC_API_KEY`.
- You find a Go reader of `BALAUR_REMOTE_*` (would contradict this plan's
  verified premise — report, don't delete the docs lines).

## Maintenance notes

- If a synthetic/dev provider is ever wanted again, it returns as a real
  provider kind through the settings UI with tests — not as env-var
  constants.
- The README env table and `balaur.env.example` now both carry
  `BALAUR_ALLOWED_HOSTS`; future env vars should land in both in the same
  commit (plus README) — drift between them is what this plan cleaned up.
- Reviewer should scrutinize: `env.go` diff contains only deletions (plus
  possibly the `os` import), and the design-doc body is untouched.
