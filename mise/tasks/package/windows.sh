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

wireguard="${MISE_PROJECT_ROOT}/assets/windows/wireguard-${GOARCH}-0.5.3.msi"
makensis -NOCD "-DWIREGUARD=$wireguard" "-DOUTFILE=$outfile" "-DVERSION=$version" "-DCERT_FILE=$cert_file" "-DKEY_FILE=$key_file" ./assets/windows/naisdevice.nsi
