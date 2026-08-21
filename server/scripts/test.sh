#!/bin/sh
set -eu

server_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
project_root=$(CDPATH= cd -- "$server_dir/.." && pwd)
pocketbase_binary=$("$server_dir/scripts/download-pocketbase.sh")

cd "$project_root"
POCKETBASE_BINARY=$pocketbase_binary \
  flutter test test/household_server/bootstrap_route_integration_test.dart "$@"
