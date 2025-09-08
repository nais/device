#!/usr/bin/env bash
#MISE description="Build wireguard-go for MacOS client"

set -o errexit
set -o pipefail
set -o nounset

build_dir="$(mktemp -d)"
trap 'echo "Removing tmp dir $build_dir" && rm -rf "$build_dir"' EXIT

curl --location https://git.zx2c4.com/wireguard-go/snapshot/wireguard-go-0.0.20250522.tar.xz | \
  tar --extract --xz --directory "$build_dir" --strip-components=1
make --directory "$build_dir"
mkdir --parents bin/macos-client
cp "$build_dir/wireguard-go" ./bin/macos-client/
