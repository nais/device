#!/usr/bin/env bash
#MISE description="Build and release auth server"
cd cmd/auth-server && docker build -t europe-north1-docker.pkg.dev/nais-io/nais/images/naisdevice-auth-server:${VERSION} .
docker push europe-north1-docker.pkg.dev/nais-io/nais/images/naisdevice-auth-server:${VERSION}
