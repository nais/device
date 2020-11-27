#!/usr/bin/env bash
set -o pipefail

err='no error'
latest_tag=''

ok() {
  echo -e "..[\033[32mok\033[0m]"
}

fail() {
  echo -e "[\033[31mfail\033[0m]"
  echo $err
  exit 1
}

echo "##################################"
echo "# Installing naisdevice           "
echo "##################################"
echo

echo -n "determining latest version..."
latest_tag=$(curl --show-error --silent --fail -L "https://api.github.com/repos/nais/device/releases/latest" | grep 'tag_name' | sed -E 's/.*"([^"]+)".*/\1/') && ok || fail

echo -n "downloading latest pkg......."
pkg_url="https://github.com/nais/device/releases/download/${latest_tag}/naisdevice.pkg"
temp_pkg=$(mktemp).pkg
err=$(curl --show-error --silent --fail -L "$pkg_url" -o "$temp_pkg") && ok || fail

echo -n "installing package..........."
err=$(/usr/bin/osascript -e "do shell script \"pkill device-agent; pkill device-agent-helper; installer -target / -pkg '$temp_pkg'\" with prompt \"naisdevice wonders if anyone ever reads this message\" with administrator privileges") && ok || fail
