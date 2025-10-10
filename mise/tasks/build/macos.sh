#!/usr/bin/env bash
#MISE description="Build macos client"
#MISE env={ GOOS = "darwin" }

set -o errexit
set -o pipefail
set -o nounset

gotags="${GOTAGS:-""}"

ldflags="-X github.com/nais/device/internal/version.Version=${VERSION:-local} -X github.com/nais/device/internal/otel.endpointURL=https://collector-internet.nav.cloud.nais.io"
mkdir -p ./bin/macos-client

go build -o bin/macos-client/naisdevice-agent --tags "$gotags" -ldflags "-s $ldflags" ./cmd/naisdevice-agent
CGO_ENABLED=1 \
	CGO_CFLAGS="-mmacosx-version-min=10.14" \
	CGO_LDFLAGS="-mmacosx-version-min=10.14" \
	go build -o bin/macos-client/naisdevice-systray --tags "$gotags" -ldflags "-s $ldflags" ./cmd/naisdevice-systray
go build -o bin/macos-client/naisdevice-helper --tags "$gotags" -ldflags "-s $ldflags" ./cmd/naisdevice-helper
