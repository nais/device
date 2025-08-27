#!/usr/bin/env bash
#MISE description="Build windows client"
#MISE env={ GOOS = "windows", GOARCH = "amd64" , OTEL_COLLECTOR_ENDPOINT = "https://collector-internet.nav.cloud.nais.io", LDFLAGS = "-X github.com/nais/device/internal/version.Version=${VERSION:-local} -X github.com/nais/device/internal/otel.endpointURL=${OTEL_COLLECTOR_ENDPOINT}" }
mkdir -p ./bin/windows-client

GOOS="" GOARCH="" go tool github.com/akavel/rsrc -arch amd64 -manifest ./packaging/windows/admin_manifest.xml -ico assets/nais-logo-blue.ico -o ./cmd/naisdevice-helper/main_windows.syso
GOOS="" GOARCH="" go tool github.com/akavel/rsrc -ico assets/nais-logo-blue.ico -o ./cmd/naisdevice-agent/main_windows.syso
go build -o bin/windows-client/naisdevice-systray.exe --tags "$GOTAGS" -ldflags "-s $LDFLAGS -H=windowsgui" ./cmd/naisdevice-systray
go build -o bin/windows-client/naisdevice-agent.exe --tags "$GOTAGS" -ldflags "-s $LDFLAGS -H=windowsgui" ./cmd/naisdevice-agent
go build -o bin/windows-client/naisdevice-helper.exe --tags "$GOTAGS" -ldflags "-s $LDFLAGS" ./cmd/naisdevice-helper

# TODO: move somewhere else?
./packaging/windows/sign-exe bin/windows-client/naisdevice-systray.exe ./packaging/windows/naisdevice.crt ./packaging/windows/naisdevice.key
./packaging/windows/sign-exe bin/windows-client/naisdevice-agent.exe ./packaging/windows/naisdevice.crt ./packaging/windows/naisdevice.key
./packaging/windows/sign-exe bin/windows-client/naisdevice-helper.exe ./packaging/windows/naisdevice.crt ./packaging/windows/naisdevice.key
