#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
work_dir=$(mktemp -d)
cleanup() {
  rm -rf "$work_dir"
}
trap cleanup EXIT HUP INT TERM

source_data=$work_dir/source-pb_data
backup_destination=$work_dir/off-host
mkdir -p "$source_data" "$backup_destination"
printf 'household recovery probe\n' >"$source_data/probe.txt"

latest_backup=
for index in $(seq 0 7); do
  backup_date=$(date -u -d "2026-06-07 +$index week" +%F)
  latest_backup=$(BALAUR_ALLOW_SAME_DEVICE_BACKUP=1 \
    BALAUR_BACKUP_NOW=$backup_date \
    "$script_dir/backup.sh" "$source_data" "$backup_destination")
done
shopt -s nullglob
daily_backups=("$backup_destination"/daily-*.tar.gz)
weekly_backups=("$backup_destination"/weekly-*.tar.gz)
shopt -u nullglob
if ((${#daily_backups[@]} != 7)); then
  printf 'Expected seven daily backups.\n' >&2
  exit 1
fi
if ((${#weekly_backups[@]} != 4)); then
  printf 'Expected four weekly backups.\n' >&2
  exit 1
fi

restore_target=$work_dir/restored-pb_data
"$script_dir/restore.sh" "$latest_backup" "$restore_target" >/dev/null
cmp "$source_data/probe.txt" "$restore_target/probe.txt"

pocketbase=$("$script_dir/download-pocketbase.sh")
"$script_dir/verify-restore.sh" "$latest_backup" "$pocketbase"
if BALAUR_RESTORE_HEALTH_URL=http://127.0.0.1:1/unavailable \
  BALAUR_RESTORE_HEALTH_ATTEMPTS=2 \
  BALAUR_RESTORE_HEALTH_INTERVAL_SECONDS=0.01 \
  "$script_dir/verify-restore.sh" "$latest_backup" "$pocketbase" \
  >/dev/null 2>&1; then
  printf 'The failed restore health check unexpectedly passed.\n' >&2
  exit 1
fi

fake_bin=$work_dir/fake-bin
mkdir -p "$fake_bin"
cat >"$fake_bin/docker" <<'EOF'
#!/usr/bin/env bash
if [[ $* == *'config --images'* ]]; then
  printf 'balaur-household-server:old\n'
fi
printf 'docker %s\n' "$*" >>"$BALAUR_FAKE_COMMAND_LOG"
EOF
cat >"$fake_bin/curl" <<'EOF'
#!/usr/bin/env bash
exit 1
EOF
chmod +x "$fake_bin/docker" "$fake_bin/curl"
command_log=$work_dir/update-commands.log
release_env=$work_dir/release.env
if PATH="$fake_bin:$PATH" \
  BALAUR_FAKE_COMMAND_LOG=$command_log \
  BALAUR_ALLOW_SAME_DEVICE_BACKUP=1 \
  BALAUR_DATA_DIR=$source_data \
  BALAUR_BACKUP_DESTINATION=$backup_destination \
  BALAUR_ENV_FILE=$release_env \
  BALAUR_UPDATE_HEALTH_ATTEMPTS=1 \
  BALAUR_UPDATE_HEALTH_INTERVAL_SECONDS=0 \
  "$script_dir/update.sh" balaur-household-server:new >/dev/null 2>&1; then
  printf 'The failed update health check unexpectedly passed.\n' >&2
  exit 1
fi
if [[ $(<"$release_env") != 'BALAUR_HOUSEHOLD_IMAGE=balaur-household-server:old' ]]; then
  printf 'The failed update did not persist the rollback image.\n' >&2
  exit 1
fi
up_count=$(awk '$1 == "docker" && $2 == "compose" && $3 == "up" {count += 1} END {print count + 0}' "$command_log")
if ((up_count < 2)); then
  printf 'The failed update did not start the rollback image.\n' >&2
  exit 1
fi

printf 'Household Server operational checks passed.\n'
