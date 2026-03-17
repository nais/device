#!/usr/bin/env bash
#MISE description="Run smoke test on Linux using the .deb installer"

set -o errexit
set -o pipefail
set -o nounset

installer_deb="${1:?usage: mise run smoke-test:linux <path-to-deb>}"

cleanup() {
	echo "Cleaning up..."
	sudo dpkg -r naisdevice 2>/dev/null || true
}
trap cleanup EXIT

echo "==> Building smoke-test binary"
go build -o ./smoke-test ./cmd/smoke-test

echo "==> Installing dependencies"
sudo apt-get update && sudo apt-get install --yes jq

echo "==> Installing $installer_deb"
sudo apt-get install --yes "$installer_deb"

echo "==> Waiting for helper to start"
for i in $(seq 1 10); do
	if sudo test -S /run/naisdevice/helper.sock; then
		echo "helper is running"
		break
	fi
	if [ "$i" -eq 10 ]; then
		echo "helper socket not found after install"
		sudo journalctl -u naisdevice-helper.service --no-pager -n 50 || true
		exit 1
	fi
	sleep 3
done

echo "==> Running smoke test"
sudo ./smoke-test
echo "==> Smoke test passed"
