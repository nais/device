#!/usr/bin/env bash
#MISE description="Prepare template vars"

set -o errexit
set -o nounset
set -o pipefail

checksums_txt="$1"
assets_json="$2"

while read -r hash file; do
	# strip directory + suffix
	base=${file##*/} # basename
	base=${base%.*}  # remove extension

	# normalize key: uppercase + replace non-alnum with _
	key=${base^^}
	key=${key//[^A-Z0-9]/_}

	# normalize hash: uppercase
	hash16=${hash^^}
	hash32=$(basenc --base16 -d <<<"${hash16}" | basenc --base32)

	# TODO
	url=$(jq -r --arg file "$file" '.[] | select(.name == $file) | .browser_download_url' "$assets_json")

	echo "${key}_HASH_BASE16=${hash16}"
	echo "${key}_HASH_BASE32=${hash32}"
	echo "${key}_URL=$url"
done <"$checksums_txt"
