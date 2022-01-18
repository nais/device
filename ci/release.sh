#!/usr/bin/env bash
version="v1" # bump manually
tag="ghcr.io/nais/naisdevice-ci:${version}"
docker build -t "$tag" . && docker push "$tag"
