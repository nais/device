#!/usr/bin/env bash
#MISE description="Build and release auth server"
GO_VERSION=$(awk '/^go /{print $2}' go.mod)
docker build --build-arg GO_VERSION="${GO_VERSION}" -t "europe-north1-docker.pkg.dev/nais-io/nais/images/naisdevice-auth-server:${VERSION}" -f cmd/auth-server/Dockerfile .
docker push "europe-north1-docker.pkg.dev/nais-io/nais/images/naisdevice-auth-server:${VERSION}"
