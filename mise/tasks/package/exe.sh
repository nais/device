#!/usr/bin/env bash
#MISE description="Package msi"
#MISE depends=["build:windows","icon:windows"]
./packaging/windows/sign-exe bin/windows-client/naisdevice-systray.exe ./packaging/windows/naisdevice.crt ./packaging/windows/naisdevice.key
./packaging/windows/sign-exe bin/windows-client/naisdevice-agent.exe ./packaging/windows/naisdevice.crt ./packaging/windows/naisdevice.key
./packaging/windows/sign-exe bin/windows-client/naisdevice-helper.exe ./packaging/windows/naisdevice.crt ./packaging/windows/naisdevice.key

./packaging/windows/build-nsis "$VERSION"
