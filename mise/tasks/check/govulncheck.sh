#!/usr/bin/env bash
#MISE description="Run vulncheck"
set -o errexit -o nounset -o pipefail

# Check for the host OS and linux, so local runs match CI (linux-only packages
# like cmd/gateway-agent are skipped on other platforms).
# `go tool -n` gives the host-built tool binary, so GOOS only affects analysis.
bin=$(go tool -n golang.org/x/vuln/cmd/govulncheck)
for goos in $(printf '%s\n' "$(go env GOHOSTOS)" linux | sort -u); do
	echo "GOOS=$goos"
	GOOS=$goos "$bin" ./...
done
