#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
server_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
repository_dir=$(CDPATH= cd -- "$server_dir/.." && pwd)

if git -C "$repository_dir" ls-files --error-unmatch server/.env >/dev/null 2>&1; then
  printf 'server/.env must not be committed.\n' >&2
  exit 1
fi
for key in \
  BALAUR_SETUP_SECRET \
  BALAUR_GOOGLE_OAUTH_CLIENT_ID \
  BALAUR_GOOGLE_OAUTH_CLIENT_SECRET \
  BALAUR_CALENDAR_ENCRYPTION_KEY; do
  value=$(sed -n "s/^${key}=//p" "$server_dir/.env.example" | tail -n 1)
  if [[ -n $value ]]; then
    printf 'The example contains a value for %s.\n' "$key" >&2
    exit 1
  fi
done
printf 'No committed Household credential value was found.\n'
