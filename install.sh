#!/usr/bin/env bash

set -o pipefail

latest_tag=$(curl --show-error --silent --fail -L "https://api.github.com/repos/nais/device/releases/latest" | grep 'tag_name' | sed -E 's/.*"([^"]+)".*/\1/' || exit 1)
pkg_url="https://github.com/nais/device/releases/download/${latest_tag}/naisdevice-${latest_tag}.pkg"

echo "downloading latest pkg from: $pkg_url"

temp_pkg=$(mktemp).pkg
curl --show-error --silent --fail -L "$pkg_url"  > "$temp_pkg" || exit 1

echo "installing new version: $latest_tag"
sudo installer -target / -pkg "$temp_pkg"
echo "done"
