#!/usr/bin/env bash
#MISE description="Generate hashes using git-cliff"

release_artifacts="$1"

if [[ ! -d "$release_artifacts" ]]; then
	echo "Usage: $0 <file>"
	echo "Example: $0 ./release-artifacts"
	exit 1
fi

sums="$(shasum --algorithm 256 "$release_artifacts"/* | tee checksums.txt)"

# assoc array
while read -r hash file; do
	# strip directory + suffix
	base=${file##*/} # basename without calling external `basename`
	base=${base%.*}  # remove extension

	# normalize key: uppercase + replace non-alnum with _
	key=${base^^}
	key=${key//[^A-Z0-9]/_}

	# normalize hash: uppercase
	hash16=${hash^^}
	hash32=$(basenc --base16 -d <<<"${hash16}" | basenc --base32)

	echo "${key}_BASE16=${hash16}"
	echo "${key}_BASE32=${hash32}"
done <<<"$sums"
