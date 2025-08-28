#!/usr/bin/env bash
#MISE description="Build macos client"
#MISE env={ GOOS = "darwin", GOARCH = "amd64" }
set -e
ldflags="-X github.com/nais/device/internal/version.Version=${VERSION:-local} -X github.com/nais/device/internal/otel.endpointURL=https://collector-internet.nav.cloud.nais.io"
mkdir -p ./bin/macos-client

go build -o bin/macos-client/naisdevice-agent --tags "$GOTAGS" -ldflags "-s $ldflags" ./cmd/naisdevice-agent
CGO_ENABLED=1 go build -o bin/macos-client/naisdevice-systray --tags "$GOTAGS" -ldflags "-s $ldflags" ./cmd/naisdevice-systray
go build -o bin/macos-client/naisdevice-helper --tags "$GOTAGS" -ldflags "-s $ldflags" ./cmd/naisdevice-helper
