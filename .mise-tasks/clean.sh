#!/usr/bin/env bash
#MISE description="Clean up build artifacts"
rm -rf *.deb
rm -rf wireguard-go-*
rm -rf wireguard-tools-*
rm -rf naisdevice.app
rm -f naisdevice-*.pkg
rm -f naisdevice-*.deb
rm -f ./packaging/windows/naisdevice*.exe
rm -rf ./bin
rm -rf ./packaging/*/icons
rm -rf ./packaging/*/assets
