#!/usr/bin/env bash
#MISE description="Build WireGuard for MacOS client"

set -o errexit
set -o pipefail
set -o nounset

build_dir="$(mktemp -d)"
trap 'echo "Removing tmp dir $build_dir" && rm -rf "$build_dir"' EXIT

curl --location https://git.zx2c4.com/wireguard-tools/snapshot/wireguard-tools-1.0.20250521.tar.xz | \
  tar --extract --xz --directory "$build_dir" --strip-components=1
make --directory "$build_dir/src"
mkdir --parents bin/macos-client
cp "$build_dir/src/wg" ./bin/macos-client/
