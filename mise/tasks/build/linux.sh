#!/usr/bin/env bash
#MISE description="Build linux client"
#MISE env={ GOOS = "linux" }

set -o errexit
set -o pipefail
set -o nounset

gotags="${GOTAGS:-""}"

ldflags="-X github.com/nais/device/internal/version.Version=${VERSION:-local} -X github.com/nais/device/internal/otel.endpointURL=https://collector-internet.nav.cloud.nais.io"
mkdir -p ./bin/linux-client

go build -o bin/linux-client/naisdevice-systray --tags "$gotags" -ldflags "-s $ldflags" ./cmd/naisdevice-systray
go build -o bin/linux-client/naisdevice-agent --tags "$gotags" -ldflags "-s $ldflags" ./cmd/naisdevice-agent
go build -o bin/linux-client/naisdevice-helper --tags "$gotags" -ldflags "-s $ldflags" ./cmd/naisdevice-helper
