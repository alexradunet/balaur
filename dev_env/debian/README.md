# Balaur dev environment — Debian 13 (trixie)

A re-runnable Ansible playbook that turns a fresh Debian 13 VPS into a
ready-to-develop **Balaur** box: Go toolchain + dev tools, the repo's own `make`
targets wired up, the systemd service, browser UI verification, the `graphify`
knowledge-graph CLI, `zellij`, NetBird mesh access, and the security hardening
applied to this box (SSH key-only, ufw, fail2ban, auto-updates, sysctl).

It mirrors the structure of the owner's `github.com/alexradunet/dev-environment`
example, trimmed to what Balaur needs. The playbook lives **inside** the balaur
repo, so the repo is already checked out — no clone step.

## Quick start

```bash
# On the VPS, from a clone of the balaur repo:
cd dev_env/debian
./bootstrap.sh                 # installs Ansible if missing, then provisions
                               # (-K asks for sudo; press Enter if sudo is passwordless)
```

Then open a new shell (PATH changes) and verify:

```bash
go version && air -v && zellij --version && timedatectl
```

## Common runs

| Command | What it does |
|---|---|
| `make setup` / `./bootstrap.sh` | Full provision |
| `make check` | Dry run (`--check --diff`) — change nothing |
| `make system` | Privileged tasks only (apt, NetBird, hardening), under sudo |
| `make user` | User-local toolchains only, no sudo |
| `make vulkan` | Opt-in: install Vulkan GPU inference packages |
| `./bootstrap.sh --skip-tags hardening` | Dev toolchain only (skip SSH/ufw/etc.) |
| `./bootstrap.sh --tags gotools` | Re-install just the Go tools |
| `./bootstrap.sh -e netbird_setup_key=XXXX` | Also join the NetBird mesh |

## Configure

Edit `group_vars/all.yml`. Key knobs: `go_target` (pin to go.mod's `go` line),
`timezone`, `balaur_install_service`, `zellij_version`, and the hardening vars.

**Secrets are never committed.** Pass them at run time:

- **NetBird**: `-e netbird_setup_key=...` (or run `sudo netbird up` for SSO).
- **SSH key**: set `ssh_authorized_keys` in `group_vars/all.yml` (or `-e`). The
  play **refuses to disable password auth unless a key is present** (anti-lockout).

## What it installs

| Role | Installs | Where |
|------|----------|-------|
| `common` | git, make, build-essential, curl/wget, python3, sqlite3, jq + **timezone** | apt (root) |
| `shellenv` | Go PATH block | `~/.bashrc` |
| `golang` | Go (pinned, checksum-verified tarball) | `~/.local/go` |
| `gotools` | gopls, dlv, air, staticcheck, govulncheck | `~/go/bin` |
| `github` | `gh` + git identity | apt + global |
| `claude` | Claude Code (native installer) | `~/.local/bin` |
| `balaur` | data dirs, git hooks, env file, `make build`, systemd `--user` service | `~/.local/share/balaur`, `~/.config` |
| `node` + `playwright` | Node + Chromium + system libs (for `/verify`) | apt + `~/.cache/ms-playwright` |
| `graphify` | `graphifyy` via pipx (CLI `graphify`) | `~/.local/bin` |
| `zellij` | static musl binary | `~/.local/bin` |
| `netbird` | mesh client + optional join | apt + systemd |
| `hardening` | SSH key-only, ufw, fail2ban, unattended-upgrades, sysctl | system |
| `vulkan` *(opt-in)* | libvulkan1, mesa-vulkan-drivers, vulkan-tools | apt (root) |

## Notes

- **Single phase, not two.** Debian 13 ships classic `sudo` (not Ubuntu 26.04's
  `sudo-rs`), so Ansible `become` works in one pass. Install the full `ansible`
  package (not `ansible-core`) — it bundles `community.general` + `ansible.posix`.
- **Go is pinned** to the go.mod version with `GOTOOLCHAIN=local`, so a stale Go
  fails loudly instead of silently fetching a toolchain (which breaks behind a
  TLS proxy — see `docs/hyperagent-sandbox.md` + `scripts/goproxy-shim.py`).
- **Runtime & models are not installed here.** The llama.cpp `.so` and GGUF
  models are owner-installed via Balaur's Models page; this only creates their
  dirs under `~/.local/share/balaur`.
- **NetBird + Balaur**: after `netbird up`, read the overlay IP/FQDN from
  `netbird status`, set `balaur_allowed_hosts`, and re-run `-t balaur`. Per
  `docs/netbird.md`, **NetBird ACLs are the only gate** — Balaur's UI has no
  login, so any peer that can reach the prod port (`:8080`) or the dev/staging
  port (`:8090`) gets full owner access. ufw stays default-deny inbound (NetBird
  is outbound WireGuard, so it still works). See `docs/two-instances.md`.
- **wsh**: there is no standalone installer; Wave Terminal auto-deploys `wsh`
  into `~/.waveterm/bin` the first time you SSH to this box from Wave on your
  laptop. Nothing to provision.
- **Hardening is idempotent** but changes SSH/firewall state — test with a second
  SSH session open, or scope it out with `--skip-tags hardening` until ready. The
  templates should match this box's existing `/etc/ssh/sshd_config.d/10-hardening.conf`,
  `/etc/fail2ban/jail.local`, `/etc/apt/apt.conf.d/{20auto-upgrades,52-autoreboot.conf}`,
  and `/etc/sysctl.d/99-hardening.conf`; run `make check` first to diff.
- **Passwordless sudo is left as-is** (owner's choice).
