#!/usr/bin/env bash
#MISE description="Generate Windows icons"
mkdir -p packaging/windows/assets/
magick -background none \
	assets/svg/blue.svg \
	-resize 256x256 \
	-gravity center \
	-extent 256x256 \
	-define icon:auto-resize=48,64,96,128,256 \
	packaging/windows/assets/naisdevice.ico
