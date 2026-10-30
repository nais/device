#!/usr/bin/env bash
#MISE description="Run go vet"
set -o errexit -o nounset -o pipefail

# Check for the host OS and linux, so local runs match CI (linux-only packages
# like cmd/gateway-agent are skipped on other platforms).
for goos in $(printf '%s\n' "$(go env GOHOSTOS)" linux | sort -u); do
	echo "GOOS=$goos"
	GOOS=$goos go vet ./...
done
