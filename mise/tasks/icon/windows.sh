#!/usr/bin/env bash
#MISE description="Generate Windows icons"
out="assets/windows/icon/naisdevice.ico"
mkdir -p "$(dirname $out)"
convert -background none \
	assets/icon/src/blue.svg \
	-resize 256x256 \
	-gravity center \
	-extent 256x256 \
	-define icon:auto-resize=48,64,96,128,256 \
	"$out"
