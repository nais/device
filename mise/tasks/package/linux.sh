#!/usr/bin/env bash
#MISE description="Package deb"
#MISE depends=["build:linux"]

set -o errexit
set -o pipefail
set -o nounset

if [[ -z "$VERSION" ]]; then
	echo "Missing VERSION"
	exit 1
fi

# shellcheck disable=SC2153
outfile="$OUTFILE"
name=$(basename "$outfile" | cut -d '_' -f 1)

arch="$GOARCH"
VERSION="1:${VERSION#v}" NAME="$name" ARCH="$arch" GOARCH="" go tool github.com/goreleaser/nfpm/v2/cmd/nfpm package \
	--packager deb \
	--config "./assets/linux/nfpm.yaml" \
	--target "$outfile"
