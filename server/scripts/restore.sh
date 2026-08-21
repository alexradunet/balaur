#!/usr/bin/env bash
set -euo pipefail

if (($# != 2)); then
  printf 'Usage: %s <backup.tar.gz> <empty-pb_data-target>\n' "$0" >&2
  exit 64
fi

backup=$(realpath "$1")
target=$2
if [[ ! -f $backup ]]; then
  printf 'The backup archive does not exist: %s\n' "$backup" >&2
  exit 66
fi
if [[ -f $backup.sha256 ]]; then
  sha256sum --check --status "$backup.sha256"
else
  printf 'The backup checksum does not exist: %s.sha256\n' "$backup" >&2
  exit 65
fi
if [[ -e $target ]]; then
  shopt -s nullglob dotglob
  target_contents=("$target"/*)
  shopt -u nullglob dotglob
  if ((${#target_contents[@]} > 0)); then
    printf 'The restore target must be empty: %s\n' "$target" >&2
    exit 73
  fi
fi
while IFS= read -r member; do
  if [[ $member == /* || $member == ../* || $member == *'/../'* ]]; then
    printf 'The backup contains an unsafe path: %s\n' "$member" >&2
    exit 65
  fi
done < <(tar --list --gzip --file "$backup")

mkdir -p "$target"
target=$(realpath "$target")
tar --extract --gzip --file "$backup" --directory "$target" --no-same-owner
printf '%s\n' "$target"
