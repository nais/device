#!/usr/bin/env bash
#MISE description="Build controlplane components"
#MISE env={ GOOS = "linux", GOARCH = "amd64" , OTEL_COLLECTOR_ENDPOINT = "https://collector-internet.nav.cloud.nais.io", LDFLAGS = "-X github.com/nais/device/internal/version.Version=${VERSION:-local} -X github.com/nais/device/internal/otel.endpointURL=${OTEL_COLLECTOR_ENDPOINT}" }
mkdir -p ./bin/controlplane
go build -o bin/controlplane/apiserver -ldflags "-s $LDFLAGS" ./cmd/apiserver
go build -o bin/controlplane/gateway-agent -ldflags "-s $LDFLAGS" ./cmd/gateway-agent
go build -o bin/controlplane/prometheus-agent -ldflags "-s $LDFLAGS" ./cmd/prometheus-agent
CGO_ENABLED=0 go build -o bin/controlplane/auth-server -ldflags "-s $LDFLAGS" ./cmd/auth-server
CGO_ENABLED=0 go build -o bin/controlplane/enroller -ldflags "-s $LDFLAGS" ./cmd/enroller
