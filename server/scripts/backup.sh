#!/usr/bin/env bash
set -euo pipefail

if (($# != 2)); then
  printf 'Usage: %s <pb_data> <off-host-destination>\n' "$0" >&2
  exit 64
fi

data_dir=$(realpath "$1")
destination=$2
if [[ ! -d $data_dir ]]; then
  printf 'The PocketBase data directory does not exist: %s\n' "$data_dir" >&2
  exit 66
fi
mkdir -p "$destination"
destination=$(realpath "$destination")
if [[ $data_dir == "$destination" || $destination == "$data_dir"/* ]]; then
  printf 'The backup destination must stay outside pb_data.\n' >&2
  exit 64
fi
if [[ ${BALAUR_ALLOW_SAME_DEVICE_BACKUP:-0} != 1 ]] &&
  [[ $(stat -c %d "$data_dir") == "$(stat -c %d "$destination")" ]]; then
  printf 'The backup destination must use an off-host mounted filesystem.\n' >&2
  exit 64
fi

backup_date=${BALAUR_BACKUP_NOW:-$(date -u +%F)}
if [[ ! $backup_date =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]]; then
  printf 'BALAUR_BACKUP_NOW must use YYYY-MM-DD.\n' >&2
  exit 64
fi

daily="$destination/daily-$backup_date.tar.gz"
temporary=$(mktemp "$destination/.backup-XXXXXX.tar.gz")
trap 'rm -f "$temporary"' EXIT HUP INT TERM

tar --create --gzip --file "$temporary" --directory "$data_dir" .
mv -f "$temporary" "$daily"
sha256sum "$daily" >"$daily.sha256"
trap - EXIT HUP INT TERM

weekday=$(date -u -d "$backup_date" +%u)
if [[ $weekday == 7 || ${BALAUR_FORCE_WEEKLY_BACKUP:-0} == 1 ]]; then
  week=$(date -u -d "$backup_date" +%G-W%V)
  weekly="$destination/weekly-$week.tar.gz"
  cp -f "$daily" "$weekly"
  sha256sum "$weekly" >"$weekly.sha256"
fi

prune_backups() {
  local pattern=$1
  local keep=$2
  local files=()
  while IFS= read -r file; do
    files+=("$file")
  done < <(compgen -G "$destination/$pattern" | sort -r)
  for ((index = keep; index < ${#files[@]}; index += 1)); do
    rm -f "${files[index]}" "${files[index]}.sha256"
  done
}

prune_backups 'daily-*.tar.gz' 7
prune_backups 'weekly-*.tar.gz' 4
printf '%s\n' "$daily"
