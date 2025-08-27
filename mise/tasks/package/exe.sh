#!/usr/bin/env bash
#MISE description="Package msi"
#MISE depends=["build:windows","icon:windows"]
./packaging/windows/build-nsis "$VERSION"
