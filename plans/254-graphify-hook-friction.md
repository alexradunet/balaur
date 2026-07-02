# Plan 254: Allowlist the mandated graphify command and make the graphify hooks fire once per session, on code files only

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, record this plan's status in
> `plans/README.md`: **no row for 254 exists yet — you create it**, exactly as
> specified in Step 4 — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 077318a..HEAD -- .claude/settings.json .claude/hooks/ plans/README.md`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: dx
- **Planned at**: commit `077318a`, 2026-07-01

## Why this matters

The repo's own Claude Code hooks declare `graphify` **MANDATORY** ("You MUST
run `graphify query ...` before grepping/reading raw files"), yet the committed
permissions allowlist in `.claude/settings.json` covers `go`/`gofmt`/`make`/`git`
but **not** `graphify`. Every fresh session, machine, or executor worktree pays
a permission prompt per graphify invocation — the tool the config itself
mandates. Only the untracked, personal `.claude/settings.local.json` papers
over this locally, and dispatched executors do not inherit it.

Separately, the two `PreToolUse` hooks are stateless substring matchers: they
re-inject the full ~60-word MANDATORY block on *every* matching tool call for
the whole session, and they match too broadly — the Read/Glob matcher fires on
`.md`/`.txt`/`.rst` prose (where the code graph is weakest) and even on `.json`
files, because the check `'.js' in path` substring-matches `.json`. This wastes
context tokens and dilutes the mandate through repetition.

After this plan: `graphify` is pre-approved in the committed settings; the hint
fires only for code files (extension-anchored, so `.js` no longer matches
`.json`); and each session gets the full MANDATORY block exactly once, with a
one-line reminder on later hits. The mandate itself is unchanged — first
contact per session still injects the full rule.

## Current state

Relevant files:

- `.claude/settings.json` — the committed Claude Code project settings:
  permissions allowlist + all hooks. Everything this plan changes lives here.
- `.claude/hooks/` — **does not exist yet** (verified:
  `ls .claude/` shows only `settings.json`, `settings.local.json`, `skills/`,
  `worktrees/`). This plan creates it with one script.
- `.claude/settings.local.json` — untracked and git-ignored (verified with
  `git check-ignore`). It already contains personal graphify allow entries,
  which is why the owner's own sessions don't see the prompt. **Do not touch
  this file** — the point is to fix the *committed* config that fresh
  sessions and executor worktrees actually get.
- `CLAUDE.md` — states the graphify policy in prose ("For codebase questions,
  first run `graphify query ...`"). The policy stands; this plan changes only
  the hook *mechanics*, not the rule.
- No `.tours/*.tour` file references `.claude/settings.json` (verified:
  `grep -rn "settings.json" .tours/` → no matches), so no tour maintenance is
  needed.
- `plans/README.md` — the plan index. It has **no row or batch section for
  plan 254** (verified: `grep -n '254' plans/README.md` → no matches at all);
  its most recent status table (the "Phase-1 & follow-up builds" section at the
  end of the file, plans 230–234) uses the three-column format
  `| Plan | Builds | Status |`. Step 4 has you append a new section in that
  same format — you create the 254 row; there is nothing pre-existing to
  update.

The committed allowlist as it exists today — note the absence of any
`graphify` entry (`.claude/settings.json:3-22`):

```json
  "permissions": {
    "allow": [
      "Bash(go build:*)",
      "Bash(go test:*)",
      "Bash(go vet:*)",
      "Bash(go run:*)",
      "Bash(gofmt:*)",
      "Bash(make dev:*)",
      "Bash(make run:*)",
      "Bash(make build:*)",
      "Bash(make test:*)",
      "Bash(make vet:*)",
      "Bash(make fmt:*)",
      "Bash(make lint:*)",
      "Bash(git status:*)",
      "Bash(git diff:*)",
      "Bash(git log:*)",
      "Bash(git show:*)"
    ]
  },
```

The `PostToolUse` gofmt hook (`.claude/settings.json:24-35`) is **out of scope
and must remain byte-identical**:

```json
    "PostToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "jq -r '.tool_input.file_path // empty' | { read -r f; case \"$f\" in *.go) command -v gofmt >/dev/null 2>&1 && gofmt -w \"$f\";; esac; } 2>/dev/null || true",
            "statusMessage": "gofmt"
          }
        ]
      }
    ],
```

The two `PreToolUse` hooks this plan replaces. First, the Bash matcher
(`.claude/settings.json:36-45`):

```json
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "CMD=$(python3 -c \"import json,sys; d=json.load(sys.stdin); print(d.get('tool_input',d).get('command',''))\" 2>/dev/null || true); case \"$CMD\" in *grep*|*rg\\ *|*ripgrep*|*find\\ *|*fd\\ *|*ack\\ *|*ag\\ *)   [ -f graphify-out/graph.json ] &&   echo '{\"hookSpecificOutput\":{\"hookEventName\":\"PreToolUse\",\"additionalContext\":\"MANDATORY: graphify-out/graph.json exists. You MUST run `graphify query \\\"<question>\\\"` before grepping raw files. Only grep after graphify has oriented you, or to modify/debug specific lines.\"}}'   || true ;; esac"
          }
        ]
      }
    ],
```

Second, the Read/Glob matcher (`.claude/settings.json:47-53`). Note the two
bugs this plan fixes: the extension tuple includes prose (`'.md','.rst','.txt','.mdx'`),
and the match is an unanchored substring test (`any(e in s for e in exts)`),
so `.js` matches `.json`:

```json
      {
        "matcher": "Read|Glob",
        "hooks": [
          {
            "type": "command",
            "command": "HIT=$(python3 -c \"import json,sys;d=json.load(sys.stdin);t=d.get('tool_input',d);s=(str(t.get('file_path') or '')+' '+str(t.get('pattern') or '')+' '+str(t.get('path') or '')).lower().replace(chr(92),'/');exts=('.py','.js','.ts','.tsx','.jsx','.go','.rs','.java','.rb','.c','.h','.cpp','.hpp','.cc','.cs','.kt','.swift','.php','.scala','.lua','.sh','.md','.rst','.txt','.mdx');sys.stdout.write('1' if 'graphify-out/' not in s and any(e in s for e in exts) else '')\" 2>/dev/null || true); if [ \"$HIT\" = 1 ] && [ -f graphify-out/graph.json ]; then echo '{\"hookSpecificOutput\":{\"hookEventName\":\"PreToolUse\",\"additionalContext\":\"MANDATORY: graphify-out/graph.json exists. You MUST run graphify before reading source files. Use: `graphify query \\\"<question>\\\"` (scoped subgraph), `graphify explain \\\"<concept>\\\"`, or `graphify path \\\"<A>\\\" \\\"<B>\\\"`. Only read raw files after graphify has oriented you, or to modify/debug specific lines. This rule applies to subagents too — include it in every subagent prompt involving code exploration.\"}}'; fi || true"
          }
        ]
      }
```

The file ends with an `enabledPlugins` block (`.claude/settings.json:57-59`)
that must also remain unchanged:

```json
  "enabledPlugins": {
    "ponytail@ponytail": true
  }
```

### Design decision (already made — do not re-litigate)

Adding once-per-session marker logic to those inline one-liners would push
each embedded program well past ~10 lines of JSON-escaped python/shell —
unreadable and untestable config. So the logic moves to **one small script**,
`.claude/hooks/graphify-hint.py`, handling both matchers (dispatch on the
`tool_name` field of the hook's stdin JSON), and `settings.json` shrinks to a
single one-line `PreToolUse` entry. This also makes the "run it twice by hand"
verification possible.

Facts the script design relies on (all verifiable in this repo):

- Claude Code hooks receive a JSON object on **stdin** containing (among
  others) `session_id`, `tool_name`, and `tool_input`; the existing inline
  hooks already parse `tool_input` from stdin exactly this way.
- Hook commands run with the project root as working directory — the existing
  hooks depend on this via the relative check `[ -f graphify-out/graph.json ]`.
  Keep the relative check for parity. `$CLAUDE_PROJECT_DIR` is set by Claude
  Code when running hooks; the command below uses it with a `.` fallback.
- A hook that prints nothing and exits 0 is a no-op; stdout JSON with
  `hookSpecificOutput.additionalContext` injects context. Exit code 2 would
  *block* the tool call — this script must therefore **always** exit 0.
- `python3` (3.13) and `jq` (1.7) are on this host (verified).
- The host `/tmp` is a small tmpfs; the marker files are zero-byte and keyed
  by session id, so this is fine.
- Permission-rule syntax in the committed file is the `Bash(<prefix>:*)` form
  (e.g. `"Bash(go build:*)"`). The new entry must match that form:
  `"Bash(graphify:*)"` (covers `graphify query/explain/path/update ...`).

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| JSON validity | `jq . .claude/settings.json` | pretty-printed JSON, exit 0 |
| Script syntax | `python3 -m py_compile .claude/hooks/graphify-hint.py` | exit 0, no output |
| Hook dry-run | `printf '%s' '<json>' \| python3 .claude/hooks/graphify-hint.py` (run from repo root) | see per-step expectations |
| Full test gate (merge gate) | `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` | exit 0 (host `/tmp` is a small tmpfs; the Go linker OOMs there) |
| Vet | `go vet ./...` | exit 0 |
| Format | `gofmt -l .` | empty output |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |

No Go source changes in this plan — the Go gates exist only to prove the tree
is untouched/green before handoff.

## Scope

**In scope** (the only files you may modify/create):
- `.claude/settings.json` — add one allow entry; replace the two `PreToolUse`
  entries with one.
- `.claude/hooks/graphify-hint.py` — create (new file, new directory).
- `plans/README.md` — append the new 254 section + row exactly as given in
  Step 4; touch nothing else in the file.

**Out of scope** (do NOT touch, even though they look related):
- `.claude/settings.local.json` — personal, untracked, git-ignored. It already
  carries the owner's local graphify allows; leave it alone.
- `CLAUDE.md` / `AGENTS.md` — the graphify *policy* prose stands unchanged.
- The `PostToolUse` gofmt hook inside `.claude/settings.json` — keep it
  byte-identical.
- Anything under `internal/`, `migrations/`, `.tours/`, `graphify-out/` — no
  product code changes; never run `graphify update`.

## Git workflow

- Executor runs in an isolated git worktree branched from `origin/main`;
  branch name `advisor/254-graphify-hook-friction`.
- Conventional-commit subjects (`feat`/`fix`/`docs`/`refactor`/`style`/`test`/`chore`);
  suggested single commit for this plan:
  `chore(claude): allowlist graphify; session-scoped, code-only hint hooks`
- Commit with explicit pathspecs only — the main checkout is shared by
  parallel agents, so stage only your own files:
  `git add .claude/settings.json .claude/hooks/graphify-hint.py plans/README.md`
- **NEVER push.** The reviewer merges.

## Steps

### Step 1: Create `.claude/hooks/graphify-hint.py`

Create the directory and the file with exactly this content (the message
strings must stay word-for-word identical to the two `additionalContext`
strings quoted in "Current state" — the mandate text is load-bearing):

```python
#!/usr/bin/env python3
"""PreToolUse hook: nudge agents to orient with graphify before raw code access.

Invoked from .claude/settings.json for Bash, Read, and Glob tool calls, with
the hook input JSON on stdin. Prints additionalContext JSON on stdout when the
call targets code (Bash search commands, or Read/Glob on code-file paths).

Design:
- Extension matching is anchored (str.endswith), so ".js" does not match
  ".json"; prose extensions (.md/.txt/.rst/.mdx) are deliberately excluded --
  the code graph has little to say about prose.
- The full MANDATORY block is injected once per session (zero-byte marker in
  the temp dir keyed by session_id); later hits get a one-line reminder.
- Never blocks the tool: always exits 0 and prints nothing on any error.
"""
import json
import os
import sys
import tempfile

FULL_READ = (
    "MANDATORY: graphify-out/graph.json exists. You MUST run graphify before "
    'reading source files. Use: `graphify query "<question>"` (scoped '
    'subgraph), `graphify explain "<concept>"`, or `graphify path "<A>" '
    '"<B>"`. Only read raw files after graphify has oriented you, or to '
    "modify/debug specific lines. This rule applies to subagents too — "
    "include it in every subagent prompt involving code exploration."
)
FULL_BASH = (
    "MANDATORY: graphify-out/graph.json exists. You MUST run `graphify query "
    '"<question>"` before grepping raw files. Only grep after graphify has '
    "oriented you, or to modify/debug specific lines."
)
SHORT = "Reminder: orient with `graphify query` before raw greps/reads of unfamiliar code."

CODE_EXTS = (
    ".py", ".js", ".ts", ".tsx", ".jsx", ".go", ".rs", ".java", ".rb",
    ".c", ".h", ".cpp", ".hpp", ".cc", ".cs", ".kt", ".swift", ".php",
    ".scala", ".lua", ".sh",
)
# Same search-command tokens the previous inline shell `case` matched.
SEARCH_TOKENS = ("grep", "rg ", "ripgrep", "find ", "fd ", "ack ", "ag ")


def wants_hint(data):
    tool = data.get("tool_name") or ""
    tool_input = data.get("tool_input") or {}
    if tool == "Bash":
        cmd = str(tool_input.get("command") or "")
        return any(tok in cmd for tok in SEARCH_TOKENS)
    if tool in ("Read", "Glob"):
        fields = [
            str(tool_input.get(k) or "").lower().replace("\\", "/")
            for k in ("file_path", "pattern", "path")
        ]
        if any("graphify-out/" in f for f in fields):
            return False
        return any(f.endswith(CODE_EXTS) for f in fields if f)
    return False


def main():
    data = json.load(sys.stdin)
    if not os.path.isfile("graphify-out/graph.json"):
        return
    if not wants_hint(data):
        return
    sid = "".join(
        c for c in str(data.get("session_id") or "") if c.isalnum() or c in "-_"
    ) or "unknown"
    marker = os.path.join(tempfile.gettempdir(), "balaur-graphify-hint-" + sid)
    if os.path.exists(marker):
        msg = SHORT
    else:
        try:
            open(marker, "w").close()
        except OSError:
            pass
        msg = FULL_BASH if data.get("tool_name") == "Bash" else FULL_READ
    print(json.dumps({"hookSpecificOutput": {
        "hookEventName": "PreToolUse",
        "additionalContext": msg,
    }}))


if __name__ == "__main__":
    try:
        main()
    except Exception:
        pass
    sys.exit(0)
```

**Verify**: `python3 -m py_compile .claude/hooks/graphify-hint.py && echo OK` → `OK`

### Step 2: Dry-run the script by hand — full block once, short form after

All commands run from the repo root (the relative `graphify-out/graph.json`
check requires it — that mirrors how Claude Code invokes hooks).

First clear any stale marker, then run the same Read call twice:

```sh
rm -f "${TMPDIR:-/tmp}/balaur-graphify-hint-plan254dry"
printf '%s' '{"session_id":"plan254dry","tool_name":"Read","tool_input":{"file_path":"internal/turn/turn.go"}}' | python3 .claude/hooks/graphify-hint.py
printf '%s' '{"session_id":"plan254dry","tool_name":"Read","tool_input":{"file_path":"internal/turn/turn.go"}}' | python3 .claude/hooks/graphify-hint.py
```

**Verify**: first run prints one JSON line whose `additionalContext` starts
with `MANDATORY: graphify-out/graph.json exists. You MUST run graphify before
reading source files.`; second run prints one JSON line whose
`additionalContext` is exactly the `SHORT` string
(`Reminder: orient with ...`).

Now the negative and Bash cases (each `printf ... | python3 ...` as above):

| stdin JSON | Expected stdout |
|---|---|
| `{"session_id":"plan254dry","tool_name":"Read","tool_input":{"file_path":".claude/settings.json"}}` | *(nothing — `.json` no longer matches `.js`; this is the substring-bug fix)* |
| `{"session_id":"plan254dry","tool_name":"Read","tool_input":{"file_path":"AGENTS.md"}}` | *(nothing — prose extensions dropped)* |
| `{"session_id":"plan254dry","tool_name":"Glob","tool_input":{"pattern":"graphify-out/**/*.go"}}` | *(nothing — graphify-out skip)* |
| `{"session_id":"plan254dry","tool_name":"Glob","tool_input":{"pattern":"internal/**/*.go"}}` | one JSON line, `SHORT` reminder (marker already set) |
| `{"session_id":"plan254bash","tool_name":"Bash","tool_input":{"command":"grep -rn foo internal/"}}` | one JSON line containing `MANDATORY: ... before grepping raw files.` (fresh session id → full Bash block) |
| `{"session_id":"plan254bash","tool_name":"Bash","tool_input":{"command":"go test ./..."}}` | *(nothing — not a search command)* |

Error path never blocks:

```sh
echo 'not json' | python3 .claude/hooks/graphify-hint.py; echo "exit=$?"
```

**Verify**: prints only `exit=0` (no output, no traceback).

Clean up the test markers:

```sh
rm -f "${TMPDIR:-/tmp}"/balaur-graphify-hint-plan254*
```

### Step 3: Edit `.claude/settings.json` — allow graphify, point hooks at the script

Two edits, nothing else changes (`$schema`, `PostToolUse`, `enabledPlugins`
stay byte-identical):

1. In `permissions.allow`, insert `"Bash(graphify:*)",` on its own line
   directly after `"Bash(gofmt:*)",`.
2. Replace the entire `PreToolUse` array (both entries, lines 36–54 in the
   excerpts above) with a single entry whose matcher covers all three tools:

```json
    "PreToolUse": [
      {
        "matcher": "Bash|Read|Glob",
        "hooks": [
          {
            "type": "command",
            "command": "python3 \"${CLAUDE_PROJECT_DIR:-.}/.claude/hooks/graphify-hint.py\""
          }
        ]
      }
    ]
```

**Verify** (all four):

- `jq . .claude/settings.json >/dev/null && echo VALID` → `VALID`
- `jq -r '.permissions.allow[]' .claude/settings.json | grep -F 'Bash(graphify:*)'` → prints `Bash(graphify:*)`
- `jq -c '[.hooks.PreToolUse[].matcher]' .claude/settings.json` → `["Bash|Read|Glob"]`
- `grep -F -e '.md' -e '.rst' -e "'.txt'" .claude/settings.json; echo "exit=$?"` → prints only `exit=1` (no prose extensions left anywhere in the file)

### Step 4: Prove the tree is otherwise untouched and green

```sh
git status --porcelain
```

**Verify**: exactly two entries for this change —
`M  .claude/settings.json` and `?? .claude/hooks/` (plus `M plans/README.md`
once the index entry below is added). Nothing else.

Now record the plan in the index. `plans/README.md` has no row for 254 —
append this new section verbatim at the **end of the file** (after the
"Phase-1 & follow-up builds" section's closing paragraph), reusing that
section's `| Plan | Builds | Status |` table format:

```markdown

## Standalone DX build (254; 2026-07-01)

| Plan | Builds | Status |
|------|--------|--------|
| 254 | Graphify hook friction — `Bash(graphify:*)` added to the committed allowlist; the two inline `PreToolUse` hooks replaced by `.claude/hooks/graphify-hint.py` (full MANDATORY block once per session, extension-anchored so `.js` no longer matches `.json`, prose extensions dropped) | BUILT — awaiting review/merge |
```

**Verify**: `grep -c '^| 254 |' plans/README.md` → `1`

```sh
gofmt -l . ; go vet ./... && CGO_ENABLED=0 go build ./... && TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1
```

**Verify**: `gofmt -l .` prints nothing; the chained command exits 0 with all
tests `ok`.

Then commit:

```sh
git add .claude/settings.json .claude/hooks/graphify-hint.py plans/README.md
git commit -m "chore(claude): allowlist graphify; session-scoped, code-only hint hooks"
```

## Test plan

There is no Go code to test; the verification is the structural + dry-run
matrix above. Concretely, the cases covered (all in Step 2):

- Happy path: first code-file Read in a session → full MANDATORY block.
- The repetition fix: second matching call, same `session_id` → `SHORT` line.
- The substring bug this plan fixes: `.json` path → no output.
- Prose narrowing: `.md` path → no output.
- `graphify-out/` self-exclusion → no output.
- Bash dispatch: `grep` command → full Bash block (fresh session id);
  non-search command → no output.
- Fail-open: malformed stdin → no output, exit 0 (a hook exiting non-zero
  could block the tool call).

Final gate: `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0
(unchanged tree stays green). `.tours` lint is not needed — no tour anchors
any in-scope file (verified in "Current state").

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `jq . .claude/settings.json >/dev/null; echo $?` → `0`
- [ ] `jq -r '.permissions.allow[]' .claude/settings.json | grep -cF 'Bash(graphify:*)'` → `1`
- [ ] `jq -c '[.hooks.PreToolUse[].matcher]' .claude/settings.json` → `["Bash|Read|Glob"]`
- [ ] `grep -F -e '.md' -e '.rst' .claude/settings.json | wc -l` → `0`
- [ ] `jq -r '.hooks.PostToolUse[0].hooks[0].statusMessage' .claude/settings.json` → `gofmt` (PostToolUse untouched)
- [ ] `python3 -m py_compile .claude/hooks/graphify-hint.py; echo $?` → `0`
- [ ] The Step 2 double-run prints a `MANDATORY:`-containing JSON line first, the `Reminder:` line second; the `.json`, `.md`, and `graphify-out` rows print nothing
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0
- [ ] `git status --porcelain` shows only `.claude/settings.json`, `.claude/hooks/graphify-hint.py`, and `plans/README.md` — no files outside the in-scope list
- [ ] `grep -c '^| 254 |' plans/README.md` → `1` (the new index section + row from Step 4, created by this plan)

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows `.claude/settings.json` changed since `077318a`, or
  the live file no longer matches the "Current state" excerpts (another agent
  may have restructured the hooks — the shared checkout has parallel sessions).
- `.claude/hooks/` already exists with content, or `graphify-hint.py` already
  exists — someone landed a variant of this; do not overwrite.
- Any Step 2 dry-run row produces the wrong output twice after a reasonable
  fix attempt (e.g. the marker never suppresses the full block, or `.json`
  still triggers).
- The permission entry `Bash(graphify:*)` appears to need a different rule
  syntax than the existing `Bash(go build:*)`-style entries — do not invent a
  new syntax; report.
- You find yourself wanting to edit `.claude/settings.local.json`, `CLAUDE.md`,
  or the `PostToolUse` gofmt hook — all out of scope.
- `internal/self/knowledge.md` seems to need updating — it does NOT: this
  change is agent-tooling config for the development harness, not a change to
  the running binary's architecture or capabilities. If you believe otherwise,
  stop and report rather than editing it.

## Maintenance notes

- **Live-behavior verification is deferred to the reviewer**: hook and
  permission changes in `settings.json` take effect for *new* Claude Code
  sessions (and hook edits may need re-approval via `/hooks` in an existing
  session). The dry-run matrix proves the script's logic; the reviewer should
  observe one full MANDATORY block early in their next fresh session and only
  short reminders after, and no prompt on `graphify query`.
- The marker files (`$TMPDIR/balaur-graphify-hint-<session_id>`, zero bytes)
  accumulate until reboot clears the tmpfs; that is intentional — no cleanup
  machinery for empty files. Two parallel first-hits in one session can both
  print the full block (check-then-create race); harmless, deliberately not
  locked.
- If graphify's CLI ever grows subcommands that mutate beyond
  `graphify update .`, revisit whether the broad `Bash(graphify:*)` allow is
  still appropriate, or split it per subcommand like the untracked local
  settings do.
- Reviewer scrutiny points: the two FULL message strings must be word-for-word
  what the old inline hooks emitted (the mandate's wording is deliberate), and
  the `PostToolUse` gofmt hook must be byte-identical.
- Deferred: the Bash-side search-token matching keeps the old loose substring
  semantics (`"ag "` also matches e.g. `"flag "`) for parity; tightening it was
  judged not worth the risk of under-triggering.
