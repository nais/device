#!/usr/bin/env bash
#MISE description="Run smoke test on macOS using the .pkg installer"

set -o errexit
set -o pipefail
set -o nounset

installer_pkg="${1:?usage: mise run smoke-test:macos <path-to-pkg>}"

cleanup() {
	echo "Cleaning up..."
	sudo launchctl unload /Library/LaunchDaemons/io.nais.device.helper.plist 2>/dev/null || true
	sudo rm -rf /Applications/naisdevice.app
	sudo rm -f /Library/LaunchDaemons/io.nais.device.helper.plist
}
trap cleanup EXIT

echo "==> Building smoke-test binary"
go build -o ./smoke-test ./cmd/smoke-test

echo "==> Installing $installer_pkg"
sudo installer -pkg "$installer_pkg" -target /

echo "==> Waiting for helper to start"
for i in $(seq 1 10); do
	if sudo test -S /var/run/naisdevice/helper.sock; then
		echo "helper is running"
		break
	fi
	if [ "$i" -eq 10 ]; then
		echo "helper socket not found after install"
		cat /Library/Logs/device-agent-helper-*.log 2>/dev/null || true
		exit 1
	fi
	sleep 3
done

echo "==> Running smoke test"
sudo ./smoke-test
echo "==> Smoke test passed"
