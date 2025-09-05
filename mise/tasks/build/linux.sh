#!/usr/bin/env bash
#MISE description="Build linux client"
#MISE env={ GOOS = "linux" }
set -e
ldflags="-X github.com/nais/device/internal/version.Version=${VERSION:-local} -X github.com/nais/device/internal/otel.endpointURL=https://collector-internet.nav.cloud.nais.io"
mkdir -p ./bin/linux-client

go build -o bin/linux-client/naisdevice-systray --tags "$GOTAGS" -ldflags "-s $ldflags" ./cmd/naisdevice-systray
go build -o bin/linux-client/naisdevice-agent --tags "$GOTAGS" -ldflags "-s $ldflags" ./cmd/naisdevice-agent
go build -o bin/linux-client/naisdevice-helper --tags "$GOTAGS" -ldflags "-s $ldflags" ./cmd/naisdevice-helper
