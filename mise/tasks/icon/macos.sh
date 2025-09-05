#!/usr/bin/env bash
#MISE description="Generate MacOS icons"

tmp="$(mktemp --suffix .png)"
out="assets/macos/icon/naisdevice.icns"
mkdir -p "$(dirname $out)"

magick -background none \
	assets/icon/src/blue.svg \
	-resize 1024x1024 \
	-gravity center \
	-extent 1024x1024 \
	"$tmp"

go tool github.com/jackmordaunt/icns/v2/cmd/icnsify \
	-i "$tmp" \
	-o "$out"
rm "$tmp"
