#!/usr/bin/env bash
#MISE description="Package msi"
#MISE depends=["build:windows"]

set -o errexit
set -o pipefail
set -o nounset

# Windows MSI requires X.X.X.X version format
# shellcheck disable=SC2153
version="${VERSION#"v"}.0"
# shellcheck disable=SC2153
outfile=$OUTFILE
# shellcheck disable=SC2153
release="$RELEASE"

if [[ "$release" == "true" ]]; then
	cert_file=$(mktemp --suffix .crt)
	key_file=$(mktemp --suffix .key)
	trap 'echo "Removing temporary files" && rm -f "$key_file" "$cert_file"' EXIT

	echo "$MSI_SIGN_CERT" >"$cert_file"
	echo "$MSI_SIGN_KEY" >"$key_file"

	for bin in bin/windows-client/*; do
		if [[ "$bin" != *.exe ]]; then
			echo "Skipping non-exe file $bin"
			continue
		fi

		mise run package:windows:sign-exe "$bin" "$cert_file" "$key_file"
	done

	sign_flags="-DCERT_FILE=$cert_file -DKEY_FILE=$key_file"
else
	sign_flags=""
fi

wireguard="${MISE_PROJECT_ROOT}/assets/windows/wireguard-${GOARCH}-0.5.3.msi"
wireguard_filename=$(basename "$wireguard")
# shellcheck disable=SC2086
makensis -NOCD "-DWIREGUARD=$wireguard" "-DWIREGUARD_FILENAME=$wireguard_filename" "-DOUTFILE=$outfile" "-DVERSION=$version" $sign_flags ./assets/windows/naisdevice.nsi
