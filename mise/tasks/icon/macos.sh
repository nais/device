#!/usr/bin/env bash
#MISE description="Generate MacOS icons"

tmp="$(mktemp --suffix .png)"
trap 'echo "Removing tmp file $tmp" && rm -f "$tmp"' EXIT
out="./assets/macos/icon/naisdevice.icns"
mkdir -p "$(dirname $out)"

convert -background none \
	assets/icon/src/blue.svg \
	-resize 1024x1024 \
	-gravity center \
	-extent 1024x1024 \
	"$tmp"

go tool github.com/jackmordaunt/icns/v2/cmd/icnsify \
	-i "$tmp" \
	-o "$out"
