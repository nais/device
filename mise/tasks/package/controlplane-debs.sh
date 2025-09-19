#!/usr/bin/env bash
#MISE description="Package controlplane-debs"
#MISE depends=["build:controlplane"]

set -o errexit
set -o pipefail
set -o nounset

go tool github.com/goreleaser/nfpm/v2/cmd/nfpm package --packager deb --config ./packaging/controlplane/apiserver/nfpm.yaml --target apiserver.deb
go tool github.com/goreleaser/nfpm/v2/cmd/nfpm package --packager deb --config ./packaging/controlplane/gateway-agent/nfpm.yaml --target gateway-agent.deb
go tool github.com/goreleaser/nfpm/v2/cmd/nfpm package --packager deb --config ./packaging/controlplane/prometheus-agent/nfpm.yaml --target prometheus-agent.deb
