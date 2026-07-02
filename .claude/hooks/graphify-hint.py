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
