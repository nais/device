#!/usr/bin/env bash
#MISE description="Build and release enroller"
docker build -t europe-north1-docker.pkg.dev/nais-io/nais/images/naisdevice-enroller:${VERSION} -f cmd/enroller/Dockerfile .
docker push europe-north1-docker.pkg.dev/nais-io/nais/images/naisdevice-enroller:${VERSION}
