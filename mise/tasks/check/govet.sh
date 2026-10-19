#!/usr/bin/env bash
#MISE description="Run go vet"
set -o errexit -o nounset -o pipefail

# Defaults to the host OS. Set CHECK_GOOS to check other OSes (space separated),
# or use `mise run check-cross`.
for goos in ${CHECK_GOOS:-$(go env GOHOSTOS)}; do
	echo "GOOS=$goos"
	GOOS=$goos go vet ./...
done
