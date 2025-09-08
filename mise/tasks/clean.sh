#!/usr/bin/env bash
#MISE description="Clean up build artifacts"
rm -f ./*.deb ./*.pkg
rm -rf ./*.app
rm -rf wireguard-go-*
rm -rf wireguard-tools-*
rm -f ./packaging/windows/naisdevice*.exe
rm -rf ./bin
