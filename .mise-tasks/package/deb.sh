#!/usr/bin/env bash
#MISE description="Package deb"
#MISE depends=["build:linux","icon:linux"]
./packaging/linux/build-deb "$VERSION"
