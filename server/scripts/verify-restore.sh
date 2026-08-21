#!/usr/bin/env bash
set -euo pipefail

if (($# != 2)); then
  printf 'Usage: %s <backup.tar.gz> <pocketbase-binary>\n' "$0" >&2
  exit 64
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
server_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
backup=$(realpath "$1")
pocketbase=$(realpath "$2")
work_dir=$(mktemp -d)
port=${BALAUR_RESTORE_CHECK_PORT:-18090}
health_attempts=${BALAUR_RESTORE_HEALTH_ATTEMPTS:-50}
health_interval=${BALAUR_RESTORE_HEALTH_INTERVAL_SECONDS:-0.1}
log_file=$work_dir/pocketbase.log
process_id=
cleanup() {
  if [[ -n $process_id ]]; then
    kill "$process_id" 2>/dev/null || true
    wait "$process_id" 2>/dev/null || true
  fi
  rm -rf "$work_dir"
}
trap cleanup EXIT HUP INT TERM

"$script_dir/restore.sh" "$backup" "$work_dir/pb_data" >/dev/null
"$pocketbase" serve \
  --http="127.0.0.1:$port" \
  --dir="$work_dir/pb_data" \
  --hooksDir="$server_dir/pb_hooks" \
  --migrationsDir="$server_dir/pb_migrations" \
  >"$log_file" 2>&1 &
process_id=$!
health_url=${BALAUR_RESTORE_HEALTH_URL:-"http://127.0.0.1:$port/api/health"}
for _ in $(seq 1 "$health_attempts"); do
  if curl --fail --silent --show-error "$health_url" >/dev/null 2>&1; then
    printf 'Restore health check passed.\n'
    exit 0
  fi
  if ! kill -0 "$process_id" 2>/dev/null; then
    break
  fi
  sleep "$health_interval"
done
printf 'The restored Household Server failed its health check.\n' >&2
cat "$log_file" >&2
exit 1
