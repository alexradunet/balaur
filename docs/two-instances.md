# Two instances: prod (8080) + dev/staging (8090)

This box runs Balaur twice, production/staging style. Both are reachable over
the NetBird mesh; nothing changes in the binary — only ports, data dirs, and
the host firewall differ.

| | **Prod** (live personal) | **Dev / staging** (hot reload) |
|---|---|---|
| Port | `8080` | `8090` |
| How it runs | systemd `--user` service (`balaur.service`) | `make dev` (air, on demand) |
| Bind | overlay IP `100.124.113.87:8080` (Option B) | `0.0.0.0:8090`, gated by ufw on `wt0` (Option A) |
| Data dir | `~/.local/share/balaur/pb_data` (real data) | `<repo>/pb_data` (throwaway) |
| Extensions | `~/.local/share/balaur/pb_extensions` | `<repo>/pb_extensions` |

The two never share data: `pb_extensions` and `search.db` follow each
instance's `--dir`, so they split automatically. The read-only model and
runtime dirs (`~/.local/share/balaur/{models,kronk/lib}`) are shared safely.

## Running them

```bash
# Prod — managed by systemd, always on:
systemctl --user status balaur
make logs-user-service            # journalctl -f

# Staging — start in a terminal/zellij session when you want to develop:
cd <repo> && make dev             # air rebuilds on save, serves :8090
```

Both can run at the same time — that is the whole point of the 8080/8090 split.

## Network

Reachable over NetBird only (ufw is default-deny inbound; allows `8080/wt0` and
`8090/wt0`). From a mesh device:

```
http://100.124.113.87:8080/    # or http://balaur-113-87.netbird.cloud:8080/   (prod)
http://100.124.113.87:8090/    # or http://balaur-113-87.netbird.cloud:8090/   (staging)
```

`make dev`'s `BALAUR_ALLOWED_HOSTS` default (in the `Makefile`) already lists the
mesh FQDN + IP so the host guard does not 403 staging requests. Prod gets the
same identity from Ansible (`dev_env/debian/group_vars/all.yml` →
`~/.config/balaur/env`).

> **No login.** Reachability equals full owner access on *both* ports — the
> NetBird ACL is the only gate (see `docs/netbird.md`). The host guard strips
> ports, so 8080 and 8090 are same-origin to it and to browser cookies; it does
> **not** isolate the two instances from each other. PocketBase session tokens
> are per-instance signed, so a cookie sent cross-instance just fails to auth.

## Inference

On a VPS, point each instance at a **remote** OpenAI-compatible provider from its
own Models page (`/settings/models`) — config lives in each instance's own DB, so
set it once per instance. Embeddings stay local; each instance loads its (small)
embed model from the shared `models/` dir. No local chat model is loaded, so two
concurrent instances do not double a multi-GB model in RAM.

## Provisioning

Ports/firewall/bind are Ansible-managed in `dev_env/debian`:
`balaur_http` (prod bind), `balaur_mesh_ports: [8080, 8090]` (the ufw allows on
`wt0`), and `balaur_allowed_hosts`. Re-run the playbook to reconcile the box.
