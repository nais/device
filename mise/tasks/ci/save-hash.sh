#!/usr/bin/env bash

file="$1"

if [[ -z "$file" ]]; then
	echo "Usage: $0 <file>"
	echo "Example: $0 archive_linux_amd64_1.2.3-patch.tar.gz"
	exit 1
fi

hashes_file="release_file_hashes.json"
if [[ ! -f "$hashes_file" ]]; then
	echo "{}" >"$hashes_file"
fi

sha256_base16="$(shasum --algorithm 256 "$file" | cut -d' ' -f 1)"
sha256_base32="$(base32 --wrap=0 <<<"${sha256_base16^^}")"

# for arch in amd64 arm64; do
# 	# Generate hashes for debs
# 	file="nais-cli_${version}_${arch}.deb"
# 	hash="$(nix hash --type sha256 --flat "./deb-${arch}/${file}")"
# 	echo "$hash  $file" >>checksums.txt
#
# 	# Generate hashes for archives
# 	for os in linux darwin windows; do
# 		file="nais-cli_${version}_${os}_${arch}.tar.gz"
# 		hash16="$(nix hash --type sha256 --flat "./archive-${os}-${arch}/${file}")"
# 		hash32="$(nix hash --type sha256 --flat --base32 "./archive-${os}-${arch}/${file}")"
# 		echo "$hash16  $file" >>checksums.txt
#
# 		# This is used by the external release jobs (nur, homebrew, scoop)
# 		jq --arg os "$os" --arg arch "$arch" --arg encoding "base16" --arg hash "$hash16" '.[$os][$arch][filename] = $file' hashes.json >new_hashes.json
# 		jq --arg os "$os" --arg arch "$arch" --arg encoding "base16" --arg hash "$hash16" '.[$os][$arch][$encoding] = $hash' hashes.json >new_hashes.json
# 		mv {new_,}hashes.json
# 		jq --arg os "$os" --arg arch "$arch" --arg encoding "base32" --arg hash "$hash32" '.[$os][$arch][$encoding] = $hash' hashes.json >new_hashes.json
# 		mv {new_,}hashes.json
# 	done
# done
