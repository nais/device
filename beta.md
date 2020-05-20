# How to beta
  1. Slack: `/msg @Kolide installers`
  2. Select platform and wait for Kolide to create your installer
  3. Install package (xkxp-\*-kolide-launcher.pkg)
  4. Check your devices status: `/msg @Kolide status`
  5. `curl -OL https://github.com/nais/device/releases/download/beta/naisdevice-beta.pkg`
  6. `sudo installer -target / -pkg ./naisdevice-beta.pkg`
  7. `/opt/naisdevice/start`

# Kubeconfig
In kubeconfigs repo: `git pull && git checkout naisdevice`
