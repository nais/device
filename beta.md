# How to beta
  1. Slack: `/msg @Kolide installers`
  2. Select platform and wait for Kolide to create your installer
  3. Install package (xkxp-\*-kolide-launcher.pkg)
  4. Check your devices status: `/msg @Kolide status` and fix errors if there are any
  5. `curl -OL https://github.com/nais/device/releases/download/beta/naisdevice-beta.pkg`
  6. `sudo installer -target / -pkg ./naisdevice-beta.pkg`
  7. `/opt/naisdevice/start`

# Kubeconfig
  1. open /etc/hosts as admin and comment out or remove the lines containing `apiserver.*.nais.io`
  2. in kubeconfigs repo: `git pull && git checkout naisdevice`
