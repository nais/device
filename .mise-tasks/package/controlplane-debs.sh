#!/usr/bin/env bash
#MISE description="Package controlplane-debs"
for packager in ./packaging/controlplane/*/build-deb; do
	$packager "$VERSION"
done
