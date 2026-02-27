#!/usr/bin/env bash
#MISE description="Run smoke test on Windows using the NSIS installer"

set -o errexit
set -o pipefail
set -o nounset

installer_exe="${1:?usage: mise run smoke-test:windows <path-to-exe>}"
install_dir="${PROGRAMFILES:-C:\\Program Files}\\NAV\\naisdevice"

cleanup() {
	echo "Cleaning up..."
	uninstaller="$install_dir\\uninstaller.exe"
	if [ -f "$uninstaller" ]; then
		powershell.exe -Command "Start-Process -FilePath '$uninstaller' -ArgumentList '/S' -Wait"
		echo "Uninstaller completed"
	fi
}
trap cleanup EXIT

echo "==> Building smoke-test binary"
go build -o ./smoke-test.exe ./cmd/smoke-test

echo "==> Running installer"
powershell.exe -Command "Start-Process -FilePath '$(cygpath -w "$installer_exe")' -ArgumentList '/S' -Wait"

echo "==> Verifying files installed"
if [ ! -f "$install_dir/naisdevice-helper.exe" ]; then
	echo "naisdevice-helper.exe not found in $install_dir"
	exit 1
fi
echo "Binary installed at $install_dir"

echo "==> Waiting for service to start"
for i in $(seq 1 10); do
	status=$(powershell.exe -Command "(Get-Service -Name NaisDeviceHelper -ErrorAction SilentlyContinue).Status" | tr -d '\r')
	if [ "$status" = "Running" ]; then
		echo "NaisDeviceHelper is running"
		break
	fi
	if [ "$i" -eq 10 ]; then
		echo "Service is not running after waiting: $status"
		log_dir="${PROGRAMDATA:-C:\\ProgramData}\\NAV\\naisdevice\\logs"
		if [ -d "$log_dir" ]; then
			for f in "$log_dir"/*; do
				echo "--- $(basename "$f") ---"
				tail -50 "$f" 2>/dev/null || true
			done
		fi
		exit 1
	fi
	sleep 3
done

echo "==> Running smoke test"
./smoke-test.exe
echo "==> Smoke test passed"
