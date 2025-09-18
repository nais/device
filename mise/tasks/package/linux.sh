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
target=$(basename "$outfile")
target="${target%.deb}"

arch="$GOARCH"
NAME="$target" ARCH="$arch" GOARCH="" go tool github.com/goreleaser/nfpm/v2/cmd/nfpm package \
	--packager deb \
	--config "./assets/linux/nfpm.yaml" \
	--target "$outfile"

# --deb-systemd assets/linux/naisdevice-helper.service \
# --deb-systemd-enable \
# --deb-systemd-auto-start \
# --deb-systemd-restart-after-upgrade \
