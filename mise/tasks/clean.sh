#!/usr/bin/env bash
#MISE description="Clean up build artifacts"
rm -f ./*.deb ./*.pkg
rm -rf ./*.app
rm -f ./packaging/windows/naisdevice*.exe
rm -rf ./bin
