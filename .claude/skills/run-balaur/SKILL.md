---
name: run-balaur
description: Use to launch the Balaur app locally and drive it in a browser — when asked to run/start/screenshot the app, or to /run or /verify a change in the real UI (especially after a UI change). The recipe the built-in /run and /verify skills discover for this repo. Covers how to start the server, wait for readiness, reach the UI at 127.0.0.1:8090, drive it with Playwright MCP, and make turns deterministic with the fake model.
---

# Run & verify Balaur in the browser

This is the launch+drive recipe for `/run` and `/verify`. Use it after a UI
change (or any change you can see in the running app) to confirm it actually
works — not just `go test`.

## 1. Launch (background)

The server is `go run . serve`. Start it with the Bash tool's
`run_in_background` so it stays up across turns:

```bash
go run . serve            # = make run; uses repo-local pb_data/ (already has the owner)
```

- `:8090` already taken? The always-on user service or a `make dev` is holding
  it. **Reuse it** — don't start a second. To take it over: `make stop-user-service`.
- Want hot reload while iterating: `make dev` (air) instead. Same port.

## 2. Wait for readiness (first run compiles)

```bash
curl -sf --retry 60 --retry-delay 1 --retry-all-errors http://127.0.0.1:8090/ >/dev/null && echo UP
```

## 3. Drive it — Playwright MCP

Load the tools in one ToolSearch call, then navigate → snapshot → act → screenshot:

```
ToolSearch "select:mcp__plugin_playwright_playwright__browser_navigate,mcp__plugin_playwright_playwright__browser_snapshot,mcp__plugin_playwright_playwright__browser_click,mcp__plugin_playwright_playwright__browser_type,mcp__plugin_playwright_playwright__browser_take_screenshot,mcp__plugin_playwright_playwright__browser_console_messages"
```

Key URLs:
- `http://127.0.0.1:8090/` — the product UI (what you're verifying)
- `http://127.0.0.1:8090/storybook` — components in isolation (verify a single component fast, no app state)
- `http://127.0.0.1:8090/_/` — PocketBase superuser (inspect data; not the product surface)

Use `browser_snapshot` (accessibility tree) to find elements before clicking;
`browser_console_messages` to catch JS/Datastar errors.

Fresh box only: if `/` redirects to onboarding/setup, complete it once. The
repo-local `pb_data/` is normally already set up.

## 4. Turns that need the model → make them deterministic

Don't depend on a real GGUF for a UI check. Script the fake model so a chat
turn says/does exactly what you expect (offline, deterministic):

```bash
python3 scripts/fake-model.py script.json &   # see README "CLI for agents & test harnesses"
```

## 5. Clean up

Kill **only the server you started** (TaskStop the background Bash, or its PID).
Never kill the owner's `make dev` / user service.

## Full PocketBase API access (as `claude`)

There is a dedicated **superuser** for agent use beside the owner's admin:
identity `claude@balaur.local`. Credential lives gitignored at
`.claude/balaur-claude-superuser.json` (never commit it; it's already ignored).

Authenticate, then use the token for any PocketBase API call (superuser tokens
bypass all collection rules — entire API surface):

```bash
CRED=.claude/balaur-claude-superuser.json
ID=$(python3 -c "import json;print(json.load(open('$CRED'))['identity'])")
PW=$(python3 -c "import json;print(json.load(open('$CRED'))['password'])")
TOKEN=$(curl -sf -X POST http://127.0.0.1:8090/api/collections/_superusers/auth-with-password \
  -H 'Content-Type: application/json' -d "{\"identity\":\"$ID\",\"password\":\"$PW\"}" \
  | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])")
curl -sf http://127.0.0.1:8090/api/collections -H "Authorization: $TOKEN"   # any endpoint
```

The web UI / `/_/` dashboard accept the same identity + password. If `pb_data/`
is ever reset, recreate the record from the stored password (idempotent):
`./balaur superuser upsert "$ID" "$PW"`.

## Report

State what you did and what you observed — clicked X, saw Y, screenshot shows Z,
console clean/errored. If it didn't behave, say so with the evidence.
