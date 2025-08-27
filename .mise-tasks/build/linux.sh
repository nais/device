#!/usr/bin/env bash
#MISE description="Build linux client"
#MISE env={ GOOS = "linux", GOARCH = "amd64" , OTEL_COLLECTOR_ENDPOINT = "https://collector-internet.nav.cloud.nais.io", LDFLAGS = "-X github.com/nais/device/internal/version.Version=${VERSION:-local} -X github.com/nais/device/internal/otel.endpointURL=${OTEL_COLLECTOR_ENDPOINT}" }
mkdir -p ./bin/linux-client

go build -o bin/linux-client/naisdevice-systray --tags "$GOTAGS" -ldflags "-s $LDFLAGS" ./cmd/naisdevice-systray
go build -o bin/linux-client/naisdevice-agent --tags "$GOTAGS" -ldflags "-s $LDFLAGS" ./cmd/naisdevice-agent
go build -o bin/linux-client/naisdevice-helper --tags "$GOTAGS" -ldflags "-s $LDFLAGS" ./cmd/naisdevice-helper
