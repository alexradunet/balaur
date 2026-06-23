#!/usr/bin/env bash
# pb — authenticated PocketBase REST calls for Balaur, as the `claude` superuser.
#
# Usage:  pb <METHOD> <PATH> [extra curl args...]
#   pb GET  /api/collections
#   pb GET  "/api/collections/tasks/records?perPage=5&sort=-created"
#   pb POST /api/collections/tasks/records -d '{"title":"x","status":"todo"}'
#   pb PATCH /api/collections/tasks/records/ID -d '{"status":"done"}'
#   pb DELETE /api/collections/tasks/records/ID
#
# Reads creds from the gitignored .claude/balaur-claude-superuser.json (NEVER
# commit that file). Token is cached and auto-refreshed on a 401.
# ponytail: cache + 401-retry instead of parsing token expiry. Ceiling: if the
# token is revoked mid-call you get one retry then a real error — fine for a CLI.
set -euo pipefail

BASE="${BALAUR_URL:-http://127.0.0.1:8090}"
ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
CRED="${BALAUR_PB_CRED:-$ROOT/.claude/balaur-claude-superuser.json}"
CACHE="${TMPDIR:-/tmp}/balaur-pb-token"

[ -f "$CRED" ] || { echo "pb: no creds at $CRED (create the claude superuser first)" >&2; exit 1; }

auth() {
  python3 - "$CRED" "$BASE" <<'PY'
import json, sys, urllib.request
cred, base = sys.argv[1], sys.argv[2]
try:
    c = json.load(open(cred))
    req = urllib.request.Request(
        base + "/api/collections/_superusers/auth-with-password",
        data=json.dumps({"identity": c["identity"], "password": c["password"]}).encode(),
        headers={"Content-Type": "application/json"})
    print(json.load(urllib.request.urlopen(req))["token"])
except Exception as e:
    print(f"pb: auth failed: {e}", file=sys.stderr); sys.exit(1)
PY
}

token() {
  [ -s "$CACHE" ] && { cat "$CACHE"; return; }
  local t
  t=$(auth) || { echo "pb: cannot authenticate — is Balaur running at $BASE? (start it: see the run-balaur skill)" >&2; exit 1; }
  printf '%s' "$t" >"$CACHE"; printf '%s' "$t"
}

method="${1:?usage: pb METHOD PATH [curl args]}"; shift
path="${1:?usage: pb METHOD PATH [curl args]}"; shift

call() {
  curl -s -w '\n%{http_code}' -X "$method" "$BASE$path" \
    -H "Authorization: $(token)" -H 'Content-Type: application/json' "$@"
}

resp=$(call "$@") || true
code=$(printf '%s' "$resp" | tail -n1)
if [ "$code" = "401" ]; then auth >"$CACHE"; resp=$(call "$@") || true; code=$(printf '%s' "$resp" | tail -n1); fi

body=$(printf '%s' "$resp" | sed '$d')
printf '%s' "$body" | python3 -m json.tool 2>/dev/null || printf '%s\n' "$body"

case "$code" in 2*) : ;; *) echo "pb: HTTP $code" >&2; exit 22 ;; esac
