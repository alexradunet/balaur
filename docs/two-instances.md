# Two instances: prod (8080) + dev/staging (8090)

This box runs Balaur twice, production/staging style. Both are reachable over
the NetBird mesh; nothing changes in the binary — only ports, data dirs, and
the host firewall differ. Neither is a daemon: each runs in the foreground via a
`make` target, kept alive in a [zellij](https://zellij.dev) session.

| | **Prod** (live personal) | **Dev / staging** (hot reload) |
|---|---|---|
| Port | `8080` | `8090` |
| How it runs | `make run` (foreground, in zellij) | `make dev` (air, on demand) |
| Bind | `0.0.0.0:8080`; mesh port always open on `wt0` | `0.0.0.0:8090`; mesh port opened on demand by `make dev` (ufw on `wt0`), not always-on |
| Data dir | `~/.local/share/balaur/pb_data` (real data) | `<repo>/pb_data` (throwaway) |
| Extensions | `~/.local/share/balaur/pb_extensions` | `<repo>/pb_extensions` |

The two never share data: `pb_extensions` and `search.db` follow each
instance's `--dir`, so they split automatically. The read-only model and
runtime dirs (`~/.local/share/balaur/{models,kronk/lib}`) are shared safely.

## Running them

Both run in the foreground, so park each in its own zellij tab/session:

```bash
# Prod — your real data, always-on:
zellij attach -c balaur     # create/attach a persistent session
make run                    # serves :8080 from ~/.local/share/balaur/pb_data
# Ctrl-o d to detach; Balaur keeps running. `loginctl enable-linger "$USER"`
# once keeps the session alive across reboots.

# Staging — start when you want to develop:
cd <repo> && make dev       # air rebuilds on save, serves :8090 from repo pb_data
```

Both can run at the same time — that is the whole point of the 8080/8090 split:
different ports *and* different data dirs, so they never collide.

## Upgrading prod from staging

Both instances run your live working tree (prod via `go run`, staging via air),
so prod picks up code changes the next time you restart `make run` — there is no
separate built-binary snapshot to promote.

1. Validate on staging (`make dev`, `:8090`).
2. `go test ./...` green, then commit + push to `main`. The checkout is shared,
   so prod runs whatever is in the tree — land it first.
3. **Snapshot prod data before restarting** if the change includes new
   `migrations/` (they apply on the next start):

   ```bash
   cp -a ~/.local/share/balaur/pb_data ~/.local/share/balaur/pb_data.bak-$(date +%Y%m%d-%H%M%S)
   ```
4. In the prod zellij tab: stop `make run` (Ctrl-c) and start it again. The
   migration applies on boot.
5. Verify: `curl http://127.0.0.1:8080/` → 200 (or the mesh URL below).

Only code + migrations cross over — never staging's `./pb_data`; prod keeps its
own data. Prune old `pb_data.bak-*` yourself. Rollback: `git checkout
<good-commit>`, restart `make run`, restoring a snapshot if a migration touched
data.

## Network

Reachable over NetBird only (ufw is default-deny inbound). Only prod is always
allowed (`8080/wt0`); the staging port (`8090/wt0`) is opened on demand by
`make dev` (or `make dev-port-open`) and closed again when `make dev` exits (or
`make dev-port-close`). So `:8090` is only reachable while you are developing.
From a mesh device:

```
http://100.124.113.87:8080/    # or http://balaur-113-87.netbird.cloud:8080/   (prod, while `make run` is up)
http://100.124.113.87:8090/    # or http://balaur-113-87.netbird.cloud:8090/   (staging, only while `make dev` runs)
```

> NetBird ACLs are a *separate* gate from ufw (dashboard-managed; see
> `docs/netbird.md`). ufw is the on-box switch this repo controls; if your ACL
> policy is per-port, also allow `8090` there when you need mesh access to staging.

Both `make run` and `make dev` export the `BALAUR_ALLOWED_HOSTS` default (in the
`Makefile`), which already lists the mesh FQDN + IP so the host guard does not
403 mesh requests.

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
`balaur_mesh_ports: [8080]` (the prod ports ufw allows on `wt0`, always-on) and
`balaur_dev_port: 8090` (opened on demand, never provisioned open — the firewall
role also deletes any lingering rule so a re-run converges to "dev port closed").
The prod bind (`0.0.0.0:8080`) and `BALAUR_ALLOWED_HOSTS` live in the repo
`Makefile`, not an env file. Re-run the playbook to reconcile the box.
