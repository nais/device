#!/usr/bin/env bash
#MISE description="Package controlplane-debs"
#MISE depends=["build:controlplane"]

set -o errexit
set -o pipefail
set -o nounset

version="$(date "+%Y-%m-%d-%H%M%S")"
VERSION="$version" go tool github.com/goreleaser/nfpm/v2/cmd/nfpm package --packager deb --config ./packaging/controlplane/apiserver/nfpm.yaml --target apiserver.deb
VERSION="$version" go tool github.com/goreleaser/nfpm/v2/cmd/nfpm package --packager deb --config ./packaging/controlplane/gateway-agent/nfpm.yaml --target gateway-agent.deb
VERSION="$version" go tool github.com/goreleaser/nfpm/v2/cmd/nfpm package --packager deb --config ./packaging/controlplane/prometheus-agent/nfpm.yaml --target prometheus-agent.deb
