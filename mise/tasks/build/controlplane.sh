#!/usr/bin/env bash
#MISE description="Build controlplane components"
#MISE env={ GOOS = "linux", GOARCH = "amd64" }
set -e
ldflags="-X github.com/nais/device/internal/version.Version=${VERSION:-local} -X github.com/nais/device/internal/otel.endpointURL=https://collector-internet.nav.cloud.nais.io"
mkdir -p ./bin/controlplane

go build -o bin/controlplane/apiserver -ldflags "-s $ldflags" ./cmd/apiserver
go build -o bin/controlplane/gateway-agent -ldflags "-s $ldflags" ./cmd/gateway-agent
go build -o bin/controlplane/prometheus-agent -ldflags "-s $ldflags" ./cmd/prometheus-agent
CGO_ENABLED=0 go build -o bin/controlplane/auth-server -ldflags "-s $ldflags" ./cmd/auth-server
CGO_ENABLED=0 go build -o bin/controlplane/enroller -ldflags "-s $ldflags" ./cmd/enroller
