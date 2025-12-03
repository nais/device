#!/usr/bin/env bash
#MISE description="Package deb"
#MISE depends=["build:macos"]

set -o errexit
set -o pipefail
set -o nounset

# shellcheck disable=SC2153
version="$VERSION"
# shellcheck disable=SC2153
release="$RELEASE"
# shellcheck disable=SC2153
outfile="$OUTFILE"

if [[ "$release" == "true" ]]; then
	notarize_d="$APPLE_NOTARIZE_D"
	notarize_i="$APPLE_NOTARIZE_I"
	base64 -d >AuthKey.p8 <<<"$APPLE_NOTARIZE_AUTH_KEY_P8_BASE64"
fi

build_dir="$(mktemp -d)"
app_dir="$build_dir/naisdevice.app"
build_plist="$build_dir/naisdevice.plist"

app_cert='Developer ID Application: Arbeids- og velferdsetaten (GC9RAU27PY)'
install_cert='Developer ID Installer: Arbeids- og velferdsetaten (GC9RAU27PY)'
pkgid="io.nais.device"

# make Application
mkdir -p "$app_dir/Contents/"{MacOS,Resources}
cp ./bin/macos-client/* "$build_dir/naisdevice.app/Contents/MacOS/"
cp ./assets/macos/icon/naisdevice.icns "$build_dir/naisdevice.app/Contents/Resources/"

sed "s/VERSIONSTRING/$version/" ./assets/macos/Info.plist.tpl >"$build_dir/naisdevice.app/Contents/Info.plist"

if [ "$release" == "true" ]; then
	codesign -s "$app_cert" -f -v --timestamp --deep --options runtime "$build_dir/naisdevice.app/Contents/MacOS/"*
	codesign -s "$app_cert" -f -v --timestamp --deep --options runtime "$build_dir/naisdevice.app/Contents/Info.plist"
fi

# mage Package
mkdir -p "$build_dir/"{scripts/,pkgroot/}
cp ./assets/macos/{preinstall,postinstall} "$build_dir/scripts/"
cp -r "$app_dir" "$build_dir/pkgroot/"
xattr -rc "$build_dir/pkgroot/$(basename "$app_dir")"

sign_flag=()
if [ "$release" == "true" ]; then
	sign_flag=(--sign "$install_cert")
fi

pkgbuild --analyze --root "$build_dir/pkgroot/" "$build_plist"
pkgbuild \
	--root "$build_dir/pkgroot/" \
	--component-plist "$build_plist" \
	--identifier "$pkgid" \
	--install-location "/Applications" \
	--scripts "$build_dir/scripts" \
	--version "$version" \
	"${sign_flag[@]}" \
	--ownership recommended \
	"$outfile"

if [ "$release" == "true" ]; then
	xcrun notarytool submit "$outfile" \
		--key AuthKey.p8 \
		--key-id "$notarize_d" \
		--issuer "$notarize_i" \
		--wait
	xcrun stapler staple "$outfile"
fi
