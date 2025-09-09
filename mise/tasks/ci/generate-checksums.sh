#!/usr/bin/env bash
#MISE description="Generate checksums"

release_artifacts="$1"

if [[ ! -d "$release_artifacts" ]]; then
	echo "Usage: $0 <file>"
	echo "Example: $0 ./release-artifacts"
	exit 1
fi

shasum --algorithm 256 "$release_artifacts"/*
