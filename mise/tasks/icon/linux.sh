#!/usr/bin/env bash
#MISE description="Generate Linux icons"
out="assets/linux/icon/"
for size in 16x16 32x32 64x64 128x128 256x256 512x512; do
	mkdir -p "$out/$size/apps/"
	convert -background none \
		assets/icon/src/blue.svg \
		-resize "$size" \
		-gravity center \
		-extent "$size" \
		"$out/$size/apps/naisdevice.png"
done
