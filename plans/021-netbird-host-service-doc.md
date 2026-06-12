# Plan 021 — Access Balaur over NetBird via the host daemon (docs only)

- **Written against commit:** `b6b7f34`
- **Category:** docs / DX
- **Priority:** P2 (enabling doc; no runtime change)
- **Effort:** S
- **Risk of the change:** LOW (adds one Markdown file + one README link; no source code, no deps, no schema)
- **Depends on:** nothing
- **Status:** TODO

## Why this exists

The owner wants to reach Balaur from their NetBird network. The obvious
route — embedding NetBird's `client/embed` package (PR #3239) into the
binary — was rejected on purpose: it drags a very large dependency tree
(gRPC, pion/ICE, wireguard-go, gVisor netstack) into a repo whose stated
ethos is suckless / minimal-deps / strictly CGO-free, and `CLAUDE.md`
explicitly says *"Keep host operating-system setup outside this
repository."*

Balaur already has everything needed to be reached over NetBird **with zero
code changes**, because NetBird's standard deployment is a host daemon:

1. Run the `netbird` daemon on the box → the box becomes a NetBird peer
   with its own overlay IP (in `100.64.0.0/10`) and a `*.netbird.*` FQDN.
2. Bind Balaur to a reachable address with PocketBase's existing
   `serve --http` flag (instead of the loopback default).
3. Add the NetBird host to `BALAUR_ALLOWED_HOSTS`, which the existing host
   guard already honours.

This plan adds **one documentation file** (`docs/netbird.md`) recording
that path, plus a one-line pointer from the README. It changes no Go code.

## Hard scope boundaries

- **In scope:** create `docs/netbird.md`; add one link line to `README.md`.
- **Out of scope — do NOT do any of these:**
  - Do not add `github.com/netbirdio/netbird` (or any dependency) to
    `go.mod`. If you find yourself running `go get`, STOP — wrong plan.
  - Do not edit `internal/web/web.go`, `main.go`, migrations, or any `.go`
    file.
  - Do not write distro-specific NetBird install steps (apt/yum/systemd
    unit bodies) into the repo — `CLAUDE.md` keeps host OS setup out of the
    tree. Link to NetBird's own install docs instead; the doc below already
    does this.
  - Do not invent new `BALAUR_*` environment variables. The doc uses only
    the two knobs that already exist (`--http` is a PocketBase flag;
    `BALAUR_ALLOWED_HOSTS` is already read at `internal/web/web.go:119`).

## Facts the doc relies on (already true at `b6b7f34` — verify before writing)

1. **The host guard reads `BALAUR_ALLOWED_HOSTS`.** Confirm:
   ```bash
   grep -n "BALAUR_ALLOWED_HOSTS" internal/web/web.go
   ```
   Expected: a hit around line 119 inside `isAllowedHost`, splitting the
   value on commas and matching `host` (no port). Loopback IPs and
   `localhost` are allowed unconditionally; everything else must be listed.
2. **The guard compares the bare host** (port stripped via
   `net.SplitHostPort`) — so list `BALAUR_ALLOWED_HOSTS` values WITHOUT a
   port. Confirm by reading `isAllowedHost` and `guardLocalUI`
   (`internal/web/web.go:89`–`129`).
3. **`serve --http` is a PocketBase flag, not a Balaur one.** Confirm the
   binary exposes it:
   ```bash
   go run . serve --help 2>&1 | grep -- "--http"
   ```
   Expected: a `--http` line whose default is `127.0.0.1:8090`. (If the
   sandbox blocks `go run`, this is also documented in PocketBase's serve
   command; the default loopback bind is stated in `README.md`'s Quick
   start, which already shows `http://127.0.0.1:8090/`.)
4. **The UI has no app-level login.** Confirm there is no auth/login
   middleware on the Balaur routes — `guardLocalUI` is the only gate, and
   it checks host/origin, not identity:
   ```bash
   grep -n "BindFunc\|Login\|auth" internal/web/web.go | head
   ```
   This is why the doc's security note is not optional.

If any of these four facts no longer holds at the executor's commit, STOP
and report which one — the doc would otherwise be wrong.

## Step 1 — Create `docs/netbird.md`

Create the file with EXACTLY this content:

```markdown
# Reaching Balaur over NetBird

Balaur is loopback-first by design. To reach it from your NetBird network,
run NetBird as a **host daemon** (its normal deployment) and point Balaur at
the NetBird interface. This needs no changes to the Balaur binary — only two
settings it already understands. Embedding NetBird's `client/embed` package
into the binary was considered and deliberately rejected: it pulls a very
large dependency tree into an otherwise minimal, CGO-free build, and host
networking belongs outside this repository.

## 1. Make the box a NetBird peer

Install and start the NetBird client on the host following NetBird's own
documentation: <https://docs.netbird.io/how-to/getting-started>. After
`netbird up` (with a setup key or SSO login) the box joins your network and
is assigned:

- an overlay IP in the `100.64.0.0/10` range (run `netbird status` to see
  it), and
- a peer FQDN such as `my-box.netbird.cloud` (depends on your NetBird DNS
  configuration).

Keep this host setup — service units, setup keys, OS packages — outside the
Balaur repository.

## 2. Bind Balaur to a reachable address

By default Balaur serves on `127.0.0.1:8090`, reachable only from the box
itself. Bind it so NetBird peers can reach it, using PocketBase's `--http`
flag:

```bash
# Option A — bind every interface; let NetBird's ACLs be the firewall.
balaur serve --http 0.0.0.0:8090

# Option B — bind only the NetBird overlay IP (replace with yours from
# `netbird status`). Nothing off the NetBird network can connect at all.
balaur serve --http 100.x.y.z:8090
```

Option B is the tighter choice: the listener never exists off the overlay.
Use Option A only if you also restrict inbound access with NetBird ACLs
and/or a host firewall.

## 3. Allow the NetBird host in Balaur's guard

Balaur rejects requests whose `Host` header is not a loopback address — a
DNS-rebinding defence. Tell it your NetBird name and/or IP are legitimate
via `BALAUR_ALLOWED_HOSTS` (comma-separated, **no port**):

```bash
BALAUR_ALLOWED_HOSTS="my-box.netbird.cloud,100.x.y.z" \
  balaur serve --http 100.x.y.z:8090
```

List every host you'll actually type in the browser bar — the FQDN if you
use NetBird DNS, the raw `100.x.y.z` IP otherwise, or both.

## 4. Open it

From any other device on your NetBird network:

```
http://my-box.netbird.cloud:8090/      # or http://100.x.y.z:8090/
```

## Security: NetBird is your only gate

Balaur's web UI has **no login**. It trusts whoever can reach it — that is
the whole point of the loopback-first model. Once it is reachable over
NetBird, **every peer your NetBird ACLs allow to reach this box gets full
owner access to your personal AI**: chat, memory, tasks, life log,
everything.

That means:

- Lock down the NetBird **access-control policy** so only the devices you
  intend can reach this peer on port 8090. Treat that policy as the
  password.
- Prefer **Option B** (bind the overlay IP) so the port is invisible off the
  NetBird network.
- Do **not** combine this with a public bind or a public reverse proxy
  unless you have a real threat model — see the warning in `AGENTS.md`.
- The PocketBase admin dashboard at `/_/` keeps its own superuser login and
  is unaffected by the host guard, but it is now reachable over NetBird too;
  use a strong superuser password.

If you later want defence in depth — requiring a logged-in session for
non-loopback requests, so reachability no longer equals full trust — that is
a separate piece of work (app-level auth on the `internal/web` gateway), not
covered here.
```

### Verify Step 1

```bash
test -f docs/netbird.md && echo "exists"
# Sanity: the doc must NOT introduce a new env var or a go-get line.
grep -nE "go get|BALAUR_(?!ALLOWED_HOSTS)" docs/netbird.md || echo "no stray env/go-get: ok"
```
Expected: `exists`, then `no stray env/go-get: ok`. (`BALAUR_ALLOWED_HOSTS`
is the only `BALAUR_*` token allowed to appear.)

## Step 2 — Link the doc from the README

Add one line to `README.md` so the doc is discoverable. Find the existing
"Optional environment variables" table or the deployment/quick-start prose
and add a short pointer near it. Match the README's existing link style
(it uses inline Markdown links to `docs/` files, e.g. the hyperagent-sandbox
reference).

First locate a good anchor:
```bash
grep -n "BALAUR_ALLOWED_HOSTS\|loopback\|docs/" README.md | head
```

Then add a single line such as (place it right after the optional-env-vars
table, adjusting surrounding blank lines to match the file):

```markdown
To reach Balaur from a NetBird network without embedding any VPN code into
the binary, see [docs/netbird.md](docs/netbird.md).
```

If `README.md` already documents `BALAUR_ALLOWED_HOSTS` in its env-var
table, leave that row as-is — just add the pointer line; do not rewrite the
table.

### Verify Step 2

```bash
grep -n "docs/netbird.md" README.md
```
Expected: one hit.

## Done criteria (all must pass)

```bash
test -f docs/netbird.md && echo ok1            # doc exists
grep -q "docs/netbird.md" README.md && echo ok2 # README links it
grep -q "BALAUR_ALLOWED_HOSTS" docs/netbird.md && echo ok3 # uses the real knob
grep -q "no login" docs/netbird.md && echo ok4  # security note present
git diff --stat -- '*.go' go.mod go.sum | grep -q . && echo "FAIL: code changed" || echo ok5
git diff --check && echo ok6                     # no whitespace errors
```
Expected output: `ok1 ok2 ok3 ok4 ok5 ok6` (note `ok5` prints only when NO
Go/dep files changed — if you see `FAIL: code changed`, you edited something
out of scope; revert it).

No `go build` / `go test` is required — this plan changes no compiled code.

## Test plan

None. This is documentation only; there is nothing to unit-test. The "done
criteria" greps are the verification.

## Maintenance note

- If Balaur ever gains app-level authentication on the web gateway, update
  the security section of `docs/netbird.md` (the "NetBird is your only gate"
  framing becomes "defence in depth") and drop the closing "separate piece
  of work" paragraph.
- If `BALAUR_ALLOWED_HOSTS` is ever renamed or the guard's loopback rule
  changes (`internal/web/web.go`), this doc's Step 3 goes stale — grep for
  `BALAUR_ALLOWED_HOSTS` across `docs/` when touching that guard.
- The companion, larger piece of work the owner may want next is the
  non-loopback auth gate; if it gets planned, cross-link the two docs.

## Escape hatches

- If `grep -n "BALAUR_ALLOWED_HOSTS" internal/web/web.go` returns nothing,
  the guard has been refactored since this plan was written — STOP and
  report; the doc's Step 3 instructions may be wrong.
- If `go run . serve --help` shows no `--http` flag, the bind mechanism has
  changed — STOP and report rather than inventing a substitute flag.
