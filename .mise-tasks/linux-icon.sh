#!/usr/bin/env bash
#MISE description="Generate Linux icons"
for size in 16x16 32x32 64x64 128x128 256x256 512x512; do
	mkdir -p "packaging/linux/icons/${size}/apps/"
	magick \
		-background none \
		assets/svg/blue.svg \
		-gravity center \
		-resize "$size" \
		"packaging/linux/icons/${size}/apps/naisdevice.png"
done
