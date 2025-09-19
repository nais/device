#!/usr/bin/env bash
#MISE description="Ensure all code is formatted"
#MISE depends=["fmt"]

set -o errexit
set -o nounset
set -o pipefail

if ! git diff --exit-code --name-only; then
	echo "The file(s) listed above are not formatted correctly. Please run \`mise run fmt\` and commit the changes."
	exit 1
fi
