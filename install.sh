#!/usr/bin/env bash
# shellcheck disable=SC2015

set -o pipefail

err='no error'
latest_tag=''

message="Would you like to install naisdevice? 😎"

ok() {
  echo -e "..[\033[32mok\033[0m]"
}

fail() {
  echo -e "[\033[31mfail\033[0m]"
  echo "$err"
  exit 1
}

case "$(uname -s)" in
   Darwin)
     installer_ext=pkg
     install() {
       /usr/bin/osascript -e "do shell script \"pkill device-agent; pkill device-agent-helper; installer -target / -pkg '$temp_installer'\" with prompt \"${message}\" with administrator privileges"
     }
     ;;

   Linux)
     installer_ext=deb

     askpass=$(mktemp)
     cat <<-EOF > "$askpass"
				#!/bin/bash
				set -o pipefail
				echo -e "SETPROMPT ${message}\nGETPIN" | pinentry | grep "^D " | sed "s/^D //"
				EOF
		 chmod a+x "$askpass"

     guisudo() {
       err=$(SUDO_ASKPASS="$askpass" sudo -A "$@")
     }

     install() {
       pkill naisdevice
       guisudo chown _apt:root "${temp_installer}" || fail
       guisudo chmod 400 "${temp_installer}" || fail
       guisudo apt-get install --assume-yes "${temp_installer}" || fail
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
latest_tag=$(curl --show-error --silent --fail --location "https://api.github.com/repos/nais/device/releases/latest" | grep 'tag_name' | sed -E 's/.*"([^"]+)".*/\1/') && ok || fail

echo -n "downloading latest installer (${latest_tag})......."
installer_url="https://github.com/nais/device/releases/download/${latest_tag}/naisdevice.${installer_ext}"
temp_installer="$(mktemp).${installer_ext}"
err=$(curl --show-error --silent --fail --location "$installer_url" --output "$temp_installer") && ok || fail

echo -n "installing package..........."
err=$(install) && ok || fail
