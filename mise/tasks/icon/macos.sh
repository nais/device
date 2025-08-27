#!/usr/bin/env bash
#MISE description="Generate MacOS icons"
mkdir -p packaging/macos/icons/
magick -background none \
	assets/svg/blue.svg \
	-resize 1024x1024 \
	-gravity center \
	-extent 1024x1024 \
	packaging/macos/icons/naisdevice.png
