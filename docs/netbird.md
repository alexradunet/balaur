# Reaching Balaur over NetBird

Balaur is loopback-first by design. To reach it from your NetBird network,
run NetBird as a **host daemon** (its normal deployment) and point Balaur at
the NetBird interface. This needs no changes to the Balaur binary — only two
settings it already understands. Embedding NetBird's `client/embed` package
into the binary was considered and deliberately rejected: it pulls a very
large dependency tree into an otherwise minimal, CGO-free build, and host
networking belongs outside this repository.

> This box runs two instances (see `docs/two-instances.md`): the prod `make run`
> instance on **8080** and the `make dev` hot-reload staging instance on
> **8090**. The steps below configure the prod instance; the example port in
> some commands is illustrative — substitute the port for the instance you mean.

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

`make run` serves prod on `0.0.0.0:8080`, already reachable by NetBird peers.
To bind a different address (e.g. only the overlay IP), pass PocketBase's
`--http` flag — `make run PROD_HTTP=...`, or run the binary directly:

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

## Troubleshooting: "Connected" peer but the page won't load

If `netbird status` shows this box as a **Connected** peer but a browser/curl to
`http://<overlay-ip>:8080/` just hangs, the **NetBird ACL policy** is almost
certainly dropping the port. NetBird enforces ACLs on the box in nftables
(`table ip netbird`, chain `netbird-acl-input-filter`): it accepts only the
source+port pairs your policies created, then `iifname "wt0" drop`s the rest.
Binding the port and opening ufw on `wt0` is **not** enough — that gets the
decrypted packet onto `wt0`, but the ACL drops it before the socket.

Signature, on the box:

- `sudo tcpdump -ni wt0 'tcp port 8080'` — the client's `Flags [S]` (SYN) keeps
  arriving and retransmitting, but no `Flags [S.]` (SYN-ACK) goes back.
- `ss -tan | grep :8080` shows only `LISTEN`, never `SYN-RECV`.
- `ping <peer-overlay-ip>` from the box is 100% loss.
- `sudo nft list table ip netbird` lists only the allowed dports (often just SSH
  22 / 22022) — your app port is absent.

Fix in the NetBird dashboard (**Access Control → Policies**): allow your devices
→ this peer on TCP **8080** (prod) and **8090** (the `make dev` staging
instance). It syncs in seconds; no service restart. Do **not** hand-edit
`table ip netbird` — NetBird reconciles it and wipes manual `nft` rules.
