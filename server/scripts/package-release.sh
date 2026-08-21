#!/usr/bin/env bash
set -euo pipefail

if (($# != 1)); then
  printf 'Usage: %s <output-directory>\n' "$0" >&2
  exit 64
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
server_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
# shellcheck source=../pocketbase-version.env
. "$server_dir/pocketbase-version.env"
output_dir=$1
mkdir -p "$output_dir"
output_dir=$(realpath "$output_dir")
image_archive="$output_dir/balaur-household-server-${POCKETBASE_VERSION}-linux-multiarch.oci.tar"

docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag "balaur-household-server:$POCKETBASE_VERSION" \
  --output "type=oci,dest=$image_archive" \
  "$server_dir"
sha256sum "$image_archive" >"$image_archive.sha256"
"$script_dir/package-server.sh" "$output_dir" all
printf 'Release artifacts are in %s\n' "$output_dir"
