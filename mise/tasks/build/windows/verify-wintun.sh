#!/usr/bin/env bash
#MISE description="Verify wintun checksums"
#MISE env={ GOOS = "windows" }

set -o errexit
set -o pipefail
set -o nounset

readonly WINTUN_MANIFEST="${MISE_PROJECT_ROOT}/assets/windows/wintun.sha256"

verify_wintun() {
	local arch dll expected actual
	arch="$1"
	dll="${MISE_PROJECT_ROOT}/assets/windows/wintun-${arch}.dll"

	if [[ ! -f "$dll" ]]; then
		echo "wintun DLL not found: $dll" >&2
		exit 1
	fi

	expected="$(awk '$2 == "wintun-'"${arch}"'.dll" {print $1}' "$WINTUN_MANIFEST")"
	if [[ -z "$expected" ]]; then
		echo "No checksum entry for wintun-${arch}.dll in $WINTUN_MANIFEST" >&2
		exit 1
	fi

	actual="$(shasum -a 256 "$dll" | awk '{print $1}')"
	if [[ "$actual" != "$expected" ]]; then
		echo "wintun checksum mismatch for $arch: got $actual, want $expected" >&2
		exit 1
	fi
}

REALGOARCH="${GOARCH:-amd64}"
verify_wintun "$REALGOARCH"
