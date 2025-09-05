#!/usr/bin/env bash
#MISE description="Build windows client"
#MISE env={ GOOS = "windows" }
set -e
ldflags="-X github.com/nais/device/internal/version.Version=${VERSION:-local} -X github.com/nais/device/internal/otel.endpointURL=https://collector-internet.nav.cloud.nais.io"
mkdir -p ./bin/windows-client

REALGOARCH="${GOARCH:-amd64}"
GOOS="" GOARCH="" go tool github.com/akavel/rsrc -arch "$REALGOARCH" -manifest ./assets/windows/admin_manifest.xml -ico ./assets/windows/icon/naisdevice.ico -o ./cmd/naisdevice-helper/main_windows.syso
GOOS="" GOARCH="" go tool github.com/akavel/rsrc -arch "$REALGOARCH" -ico assets/windows/icon/naisdevice.ico -o ./cmd/naisdevice-agent/main_windows.syso
go build -o bin/windows-client/naisdevice-systray.exe --tags "$GOTAGS" -ldflags "-s $ldflags -H=windowsgui" ./cmd/naisdevice-systray
go build -o bin/windows-client/naisdevice-agent.exe --tags "$GOTAGS" -ldflags "-s $ldflags -H=windowsgui" ./cmd/naisdevice-agent
go build -o bin/windows-client/naisdevice-helper.exe --tags "$GOTAGS" -ldflags "-s $ldflags" ./cmd/naisdevice-helper
