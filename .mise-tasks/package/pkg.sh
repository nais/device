#!/usr/bin/env bash
#MISE description="Package deb"
#MISE depends=["thirdparty/wireguard","thirdparty/wireguard-go","build:macos","icon:macos"]
./packaging/macos/build-pkg "$VERSION" "$RELEASE"
