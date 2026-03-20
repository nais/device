#!/usr/bin/env bash
#MISE description="Build and release enroller"
GO_VERSION=$(awk '/^go /{print $2}' go.mod)
docker build --build-arg GO_VERSION="${GO_VERSION}" -t "europe-north1-docker.pkg.dev/nais-io/nais/images/naisdevice-enroller:${VERSION}" -f cmd/enroller/Dockerfile .
docker push "europe-north1-docker.pkg.dev/nais-io/nais/images/naisdevice-enroller:${VERSION}"
