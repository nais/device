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

case "$(uname -s)" in
   Darwin)
     installer_ext=pkg
     install() {
       /usr/bin/osascript -e "do shell script \"pkill device-agent; pkill device-agent-helper; installer -target / -pkg '$temp_installer'\" with prompt \"naisdevice wonders if anyone ever reads this message? also it would like to install itself 😎\" with administrator privileges"
     }
     ;;

   Linux)
     installer_ext=deb
     install() {
       pkill naisdevice
       pkexec dpkg --install "${temp_installer}"
     }
     ;;

   *)
     err="This install script does not support your OS :("
     fail
     ;;
esac

echo "##################################"
echo "# Installing naisdevice           "
echo "##################################"
echo

echo -n "determining latest version..."
latest_tag=$(curl --show-error --silent --fail -L "https://api.github.com/repos/nais/device/releases/latest" | grep 'tag_name' | sed -E 's/.*"([^"]+)".*/\1/') && ok || fail

echo -n "downloading latest installer......."
installer_url="https://github.com/nais/device/releases/download/${latest_tag}/naisdevice.${installer_ext}"
temp_installer="$(mktemp).${installer_ext}"
err=$(curl --show-error --silent --fail -L "$installer_url" -o "$temp_installer") && ok || fail

echo -n "installing package..........."
err=$(install) && ok || fail
