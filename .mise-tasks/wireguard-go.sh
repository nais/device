#!/usr/bin/env bash
#MISE description="Build wireguard-go for MacOS client"
mkdir -p bin/macos-client
curl -L https://git.zx2c4.com/wireguard-go/snapshot/wireguard-go-0.0.20250522.tar.xz | tar xJ
cd wireguard-go-*/ && make && cp wireguard-go ../bin/macos-client/
rm -rf ./wireguard-go-*
