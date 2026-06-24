# Two instances: prod (8080) + dev/staging (8090)

This box runs Balaur twice, production/staging style. Both are reachable over
the NetBird mesh; nothing changes in the binary — only ports, data dirs, and
the host firewall differ.

| | **Prod** (live personal) | **Dev / staging** (hot reload) |
|---|---|---|
| Port | `8080` | `8090` |
| How it runs | systemd `--user` service (`balaur.service`) | `make dev` (air, on demand) |
| Bind | overlay IP `100.124.113.87:8080` (Option B) | `0.0.0.0:8090`; mesh port opened on demand by `make dev` (ufw on `wt0`), not always-on |
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

## Upgrading prod from staging

Staging runs your live working tree (air); prod runs a built binary snapshot at
`~/.local/bin/balaur`, so prod only changes when you rebuild + reinstall it.
"Promoting" bakes the code you validated on staging into prod's binary:

1. Validate on staging (`make dev`, `:8090`).
2. `go test ./...` green, then commit + push to `main`. The checkout is shared,
   so prod builds whatever is in the tree — land it first.
3. `make promote` — one guarded command that:
   - refuses a dirty working tree,
   - runs the suite,
   - builds (a compile failure here leaves prod on the old binary),
   - stops prod and snapshots `pb_data` to `pb_data.bak-<timestamp>` — a clean
     pre-migration restore point, since pending `migrations/` apply on the next
     start,
   - reinstalls the binary + unit and restarts.
4. Verify: `make status-user-service`; `curl http://100.124.113.87:8080/` → 200.

Only code + migrations cross over — never staging's `./pb_data`; prod keeps its
own data. Snapshots accumulate under `~/.local/share/balaur/`; prune old
`pb_data.bak-*` yourself. Rollback: `git checkout <good-commit> && make promote`,
restoring a snapshot if a migration touched data.

## Network

Reachable over NetBird only (ufw is default-deny inbound). Only prod is always
allowed (`8080/wt0`); the staging port (`8090/wt0`) is opened on demand by
`make dev` (or `make dev-port-open`) and closed again when `make dev` exits (or
`make dev-port-close`). So `:8090` is only reachable while you are developing.
From a mesh device:

```
http://100.124.113.87:8080/    # or http://balaur-113-87.netbird.cloud:8080/   (prod, always on)
http://100.124.113.87:8090/    # or http://balaur-113-87.netbird.cloud:8090/   (staging, only while `make dev` runs)
```

> NetBird ACLs are a *separate* gate from ufw (dashboard-managed; see
> `docs/netbird.md`). ufw is the on-box switch this repo controls; if your ACL
> policy is per-port, also allow `8090` there when you need mesh access to staging.

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
`balaur_http` (prod bind), `balaur_mesh_ports: [8080]` (the prod ports ufw allows
on `wt0`, always-on), `balaur_dev_port: 8090` (opened on demand, never
provisioned open — the firewall role also deletes any lingering rule so a re-run
converges to "dev port closed"), and `balaur_allowed_hosts`. Re-run the playbook
to reconcile the box.
