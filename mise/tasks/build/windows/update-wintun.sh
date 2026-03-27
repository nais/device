#!/usr/bin/env bash
#MISE description="Update bundled wintun DLLs"

set -o errexit
set -o pipefail
set -o nounset

MISE_PROJECT_ROOT="${MISE_PROJECT_ROOT:-$(pwd)}"

version="${1:?usage: mise run build:windows:update-wintun <version>}"
if [[ "$version" =~ ^v ]]; then
	version="${version#v}"
fi

base_url="https://www.wintun.net"
zip_rel_path="builds/wintun-${version}.zip"
zip_url="${base_url}/${zip_rel_path}"
manifest_url="${base_url}/"

workdir="$(mktemp -d)"
trap 'rm -rf "$workdir"' EXIT

dst_dir="${MISE_PROJECT_ROOT}/assets/windows"
manifest="${dst_dir}/wintun.sha256"

curl -fsSL "$manifest_url" -o "$workdir/index.html"

zip_sha="$(sed -nE 's/.*SHA2-256: <code>([0-9a-f]{64})<\/code>.*/\1/p' "$workdir/index.html" | head -n 1)"
if [[ -z "$zip_sha" ]]; then
	echo "failed to read wintun zip SHA256 from ${manifest_url}" >&2
	exit 1
fi

declared_zip_path="$(grep -Eo 'builds/wintun-[0-9]+\.[0-9]+\.[0-9]+\.zip' "$workdir/index.html" | head -n 1)"
if [[ -n "$declared_zip_path" && "$declared_zip_path" != "$zip_rel_path" ]]; then
	echo "requested wintun version ${version}, but website currently publishes ${declared_zip_path}" >&2
	exit 1
fi

curl -fsSL "$zip_url" -o "$workdir/wintun.zip"

actual_zip_sha="$(shasum -a 256 "$workdir/wintun.zip" | awk '{print $1}')"
if [[ "$actual_zip_sha" != "$zip_sha" ]]; then
	echo "wintun zip checksum mismatch: got $actual_zip_sha, want $zip_sha" >&2
	exit 1
fi

unzip -q "$workdir/wintun.zip" -d "$workdir"

install -m 0644 "$workdir/wintun/bin/amd64/wintun.dll" "${dst_dir}/wintun-amd64.dll"
install -m 0644 "$workdir/wintun/bin/arm64/wintun.dll" "${dst_dir}/wintun-arm64.dll"

amd64_sha="$(shasum -a 256 "${dst_dir}/wintun-amd64.dll" | awk '{print $1}')"
arm64_sha="$(shasum -a 256 "${dst_dir}/wintun-arm64.dll" | awk '{print $1}')"

{
	echo "${amd64_sha}  wintun-amd64.dll"
	echo "${arm64_sha}  wintun-arm64.dll"
} >"${manifest}"

cat "${manifest}"
