#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
server_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
# shellcheck source=../pocketbase-version.env
. "$server_dir/pocketbase-version.env"
work_dir=$(mktemp -d)
trap 'rm -rf "$work_dir"' EXIT HUP INT TERM

"$script_dir/validate-no-secrets.sh"
archive=$(
  "$script_dir/package-server.sh" "$work_dir" amd64
)
sha256sum --check --status "$archive.sha256"
tar --extract --gzip --file "$archive" --directory "$work_dir"
package_dir=$work_dir/balaur-household-server
for path in \
  pocketbase \
  run.sh \
  .env.example \
  Caddyfile.example \
  pb_hooks/calendar_connection.pb.js \
  pb_hooks/household_archive.pb.js \
  pb_migrations/1787280200_create_calendar_connection.js; do
  if [[ ! -e $package_dir/$path ]]; then
    printf 'The release archive does not contain %s.\n' "$path" >&2
    exit 1
  fi
done
bash -n "$package_dir/run.sh"
"$package_dir/pocketbase" --version >/dev/null
bash -n "$script_dir/setup-wizard.sh"
printf 'Household Server packaging checks passed.\n'
