#!/usr/bin/env bash
#MISE description="Build and release auth server"
docker build -t "europe-north1-docker.pkg.dev/nais-io/nais/images/naisdevice-auth-server:${VERSION}" -f cmd/auth-server/Dockerfile .
docker push "europe-north1-docker.pkg.dev/nais-io/nais/images/naisdevice-auth-server:${VERSION}"
