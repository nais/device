#!/usr/bin/env bash
#MISE description="Build macos client"
#MISE env={ GOOS = "darwin", GOARCH = "amd64" , OTEL_COLLECTOR_ENDPOINT = "https://collector-internet.nav.cloud.nais.io", LDFLAGS = "-X github.com/nais/device/internal/version.Version=${VERSION:-local} -X github.com/nais/device/internal/otel.endpointURL=${OTEL_COLLECTOR_ENDPOINT}" }
mkdir -p ./bin/macos-client
go build -o bin/macos-client/naisdevice-agent --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/naisdevice-agent
CGO_ENABLED=1 go build -o bin/macos-client/naisdevice-systray --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/naisdevice-systray
go build -o bin/macos-client/naisdevice-helper --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/naisdevice-helper
