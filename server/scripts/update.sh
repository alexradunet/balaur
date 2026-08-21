#!/usr/bin/env bash
set -euo pipefail

if (($# != 1)); then
  printf 'Usage: %s <new-versioned-image>\n' "$0" >&2
  exit 64
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
server_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
new_image=$1
data_dir=${BALAUR_DATA_DIR:-"$server_dir/pb_data"}
backup_destination=${BALAUR_BACKUP_DESTINATION:-}
health_url=${BALAUR_HEALTH_URL:-http://127.0.0.1:8090/api/health}
health_attempts=${BALAUR_UPDATE_HEALTH_ATTEMPTS:-60}
health_interval=${BALAUR_UPDATE_HEALTH_INTERVAL_SECONDS:-1}
if [[ -z $backup_destination ]]; then
  printf 'Set BALAUR_BACKUP_DESTINATION to an off-host mounted directory.\n' >&2
  exit 64
fi
if [[ ! $new_image =~ :[^:]+$ ]]; then
  printf 'Use a versioned image reference with an explicit tag.\n' >&2
  exit 64
fi

cd "$server_dir"
previous_image=$(docker compose config --images | sed -n '1p')
if [[ -z $previous_image ]]; then
  printf 'The current Household Server image was not found.\n' >&2
  exit 69
fi

persist_image() {
  local image=$1
  local env_file=${BALAUR_ENV_FILE:-"$server_dir/.env"}
  local temporary
  temporary=$(mktemp "$server_dir/.env-XXXXXX")
  if [[ -f $env_file ]]; then
    grep -v '^BALAUR_HOUSEHOLD_IMAGE=' "$env_file" >"$temporary" || true
  fi
  printf 'BALAUR_HOUSEHOLD_IMAGE=%s\n' "$image" >>"$temporary"
  chmod 600 "$temporary"
  mv -f "$temporary" "$env_file"
}

rollback() {
  printf 'The update failed. Restoring image %s.\n' "$previous_image" >&2
  export BALAUR_HOUSEHOLD_IMAGE=$previous_image
  docker compose up --detach
  persist_image "$previous_image"
}

export BALAUR_HOUSEHOLD_IMAGE=$previous_image
docker compose stop
if ! "$script_dir/backup.sh" "$data_dir" "$backup_destination" >/dev/null; then
  docker compose up --detach
  printf 'The pre-update backup failed. The update stopped.\n' >&2
  exit 1
fi

export BALAUR_HOUSEHOLD_IMAGE=$new_image
if ! docker compose pull || ! docker compose up --detach; then
  rollback
  exit 1
fi
for _ in $(seq 1 "$health_attempts"); do
  if curl --fail --silent --show-error "$health_url" >/dev/null 2>&1; then
    persist_image "$new_image"
    printf 'Household Server update completed: %s\n' "$new_image"
    exit 0
  fi
  sleep "$health_interval"
done
rollback
printf 'The new Household Server failed its health check.\n' >&2
exit 1
