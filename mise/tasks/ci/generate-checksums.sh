#!/usr/bin/env bash
#MISE description="Generate checksums"

set -o errexit
set -o nounset
set -o pipefail

release_artifacts="$1"

if [[ ! -d "$release_artifacts" ]]; then
	echo "Usage: $0 <directory>"
	exit 1
fi

shopt -s extglob
release_artifacts="${release_artifacts%%+(/)}"

shasum --algorithm 256 "$release_artifacts"/*
