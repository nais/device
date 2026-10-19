#!/usr/bin/env bash
#MISE description="Run vulncheck"
set -o errexit -o nounset -o pipefail

bin=$(go tool -n golang.org/x/vuln/cmd/govulncheck)
# Defaults to the host OS. Set CHECK_GOOS to check other OSes (space separated),
# or use `mise run check-cross`. `go tool -n` gives the host-built tool binary,
# so GOOS only affects what is analyzed.
for goos in ${CHECK_GOOS:-$(go env GOHOSTOS)}; do
	echo "GOOS=$goos"
	GOOS=$goos "$bin" ./...
done
