#!/usr/bin/env bash
#MISE description="Prepare template vars"

set -o errexit
set -o nounset
set -o pipefail

checksums_txt="$1"
verbose="${2:-""}"

if [[ $verbose == "-v" ]]; then
	>&2 echo "checksums:"
	>&2 cat "$checksums_txt"
fi

echo "VERSION=$VERSION"
while read -r hash file; do
	if [[ "$file" = */checksums.txt ]]; then
		continue
	fi

	# strip directory + suffix
	basename=${file##*/} # basename
	base=${basename%.*}  # remove extension

	# normalize key: uppercase + replace non-alnum with _
	key=${base^^}
	key=${key//[^A-Z0-9]/_}

	# normalize hash: uppercase
	hash16=${hash^^}
	hash32=$(basenc --base16 -d <<<"${hash16}" | basenc --base32)

	url="https://github.com/nais/device/releases/download/$VERSION/$basename"

	echo "${key}_HASH_BASE16=${hash16}"
	echo "${key}_HASH_BASE32=${hash32}"
	echo "${key}_URL=$url"
done <"$checksums_txt"
