#!/usr/bin/env bash
set -euo pipefail

if (($# < 1 || $# > 2)); then
  printf 'Usage: %s <output-directory> [amd64|arm64]\n' "$0" >&2
  exit 64
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
server_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
# shellcheck source=../pocketbase-version.env
. "$server_dir/pocketbase-version.env"
output_dir=$1
requested_arch=${2:-all}
mkdir -p "$output_dir"
output_dir=$(realpath "$output_dir")

package_architecture() {
  local archive_arch=$1
  local release_arch expected checksum url work root archive
  case $archive_arch in
    amd64)
      release_arch=x64
      expected=$POCKETBASE_SHA256_LINUX_AMD64
      ;;
    arm64)
      release_arch=arm64
      expected=$POCKETBASE_SHA256_LINUX_ARM64
      ;;
    *)
      printf 'Unsupported release architecture: %s\n' "$archive_arch" >&2
      exit 64
      ;;
  esac
  work=$(mktemp -d)
  root=$work/balaur-household-server
  mkdir -p "$root"
  archive=$work/pocketbase.zip
  url="https://github.com/pocketbase/pocketbase/releases/download/v$POCKETBASE_VERSION/pocketbase_${POCKETBASE_VERSION}_linux_${archive_arch}.zip"
  curl --fail --location --silent --show-error "$url" --output "$archive"
  printf '%s  %s\n' "$expected" "$archive" | sha256sum --check --status
  unzip -q "$archive" pocketbase -d "$root"
  cp -R "$server_dir/pb_hooks" "$root/pb_hooks"
  cp -R "$server_dir/pb_migrations" "$root/pb_migrations"
  cp "$server_dir/.env.example" "$root/.env.example"
  cp "$server_dir/README.md" "$root/README.md"
  cp "$server_dir/Caddyfile.example" "$root/Caddyfile.example"
  cat >"$root/run.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
set -a
if [[ -f $root/.env ]]; then
  # shellcheck disable=SC1091
  . "$root/.env"
fi
set +a
exec "$root/pocketbase" serve \
  --http="0.0.0.0:${BALAUR_PORT:-8090}" \
  --dir="${BALAUR_DATA_DIR:-$root/pb_data}" \
  --hooksDir="$root/pb_hooks" \
  --migrationsDir="$root/pb_migrations"
EOF
  chmod +x "$root/pocketbase" "$root/run.sh"
  output="$output_dir/balaur-household-server-${POCKETBASE_VERSION}-linux-${release_arch}.tar.gz"
  tar --create --gzip --file "$output" --directory "$work" balaur-household-server
  sha256sum "$output" >"$output.sha256"
  rm -rf "$work"
  printf '%s\n' "$output"
}

case $requested_arch in
  all)
    package_architecture amd64
    package_architecture arm64
    ;;
  amd64|arm64)
    package_architecture "$requested_arch"
    ;;
  *)
    printf 'Select amd64 or arm64.\n' >&2
    exit 64
    ;;
esac
