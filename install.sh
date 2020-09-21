#!/usr/bin/env bash
set -o pipefail

err='no error'
latest_tag=''

ok() {
  echo -e "..[\e[38;5;82mok\e[0;5;82m]"
}

fail() {
  echo -e "[\e[31;5;82mfail\e[0;5;82m]"
  echo $err
  exit 1
}

admin() {
  echo -e "[\e[33;5;82msudo\e[0;5;82m]"
  sudo whoami > /dev/null
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
err=$(/usr/bin/osascript -e "do shell script \"pkill device-agent; pkill device-agent-helper; installer -target / -pkg '$temp_pkg'\" with administrator privileges") && ok || fail
