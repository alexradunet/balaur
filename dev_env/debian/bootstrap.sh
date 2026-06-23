#!/usr/bin/env bash
# Provision this Debian 13 (trixie) box for Balaur development.
#
# SINGLE PHASE (unlike the Ubuntu dev-environment example repo): Debian 13 ships
# classic `sudo`, not Ubuntu 26.04's `sudo-rs`, so Ansible's `become` works
# normally. Privileged tasks carry `become: true`; everything else runs as you.
#
# `-K` asks for your sudo (become) password once. If this box has passwordless
# sudo, just press Enter at the prompt.
#
# Usage:
#   ./bootstrap.sh                         # full provision
#   ./bootstrap.sh --check --diff          # dry run, change nothing
#   ./bootstrap.sh --skip-tags hardening   # dev toolchain only, skip SSH/ufw
#   ./bootstrap.sh -e netbird_setup_key=XXXX   # also join the NetBird mesh
set -euo pipefail
cd "$(dirname "$0")"

if ! command -v ansible-playbook >/dev/null 2>&1; then
  echo ">> Ansible not found — installing via apt (enter your password)…"
  sudo apt-get update
  # The full `ansible` package bundles community.general + ansible.posix, which
  # the timezone / ufw / authorized_key tasks need. `ansible-core` alone won't do.
  sudo apt-get install -y ansible
fi

echo ">> Provisioning the Balaur dev box…"
ansible-playbook playbook.yml -K "$@"

echo
echo ">> Done. Open a new shell (or: source ~/.bashrc) to pick up PATH changes,"
echo "   then verify:  go version && air -v && zellij --version && timedatectl"
