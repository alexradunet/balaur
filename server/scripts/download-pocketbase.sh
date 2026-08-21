#!/bin/sh
set -eu

server_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
# shellcheck source=../pocketbase-version.env
. "$server_dir/pocketbase-version.env"

case "$(uname -m)" in
  x86_64|amd64)
    archive_arch=amd64
    expected_sha256=$POCKETBASE_SHA256_LINUX_AMD64
    ;;
  aarch64|arm64)
    archive_arch=arm64
    expected_sha256=$POCKETBASE_SHA256_LINUX_ARM64
    ;;
  *)
    printf 'PocketBase does not support this test architecture: %s\n' "$(uname -m)" >&2
    exit 64
    ;;
esac

cache_dir=${POCKETBASE_CACHE_DIR:-"$server_dir/.cache/pocketbase/$POCKETBASE_VERSION/$archive_arch"}
binary="$cache_dir/pocketbase"
if [ -x "$binary" ]; then
  printf '%s\n' "$binary"
  exit 0
fi

temporary_dir=$(mktemp -d)
trap 'rm -rf "$temporary_dir"' EXIT HUP INT TERM
archive="$temporary_dir/pocketbase.zip"
url="https://github.com/pocketbase/pocketbase/releases/download/v$POCKETBASE_VERSION/pocketbase_${POCKETBASE_VERSION}_linux_${archive_arch}.zip"

curl --fail --location --silent --show-error "$url" --output "$archive"
printf '%s  %s\n' "$expected_sha256" "$archive" | sha256sum --check --status
unzip -q "$archive" pocketbase -d "$temporary_dir"
mkdir -p "$cache_dir"
install -m 0755 "$temporary_dir/pocketbase" "$binary"
printf '%s\n' "$binary"
