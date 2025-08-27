#!/usr/bin/env bash
#MISE description="Build WireGuard for MacOS client"
mkdir -p bin/macos-client
curl -L https://git.zx2c4.com/wireguard-tools/snapshot/wireguard-tools-1.0.20250521.tar.xz | tar xJ
cd wireguard-tools-*/src && make && cp wg ../../bin/macos-client/
rm -rf ./wireguard-tools-*
