# Installation

## Install Kolide-binary
1. Slack: `/msg @Kolide installers`
2. Select platform and wait for Kolide to create your installer
3. Install package (xkxp-\*-kolide-launcher.{pkg,msi,deb,rpm})
4. Wait a couple of minutes to let Kolide initialize device state
5. Check your devices status: `/msg @Kolide status` and fix errors if there are any

## Install naisdevice agent
#### MacOS 
1. `curl -OL https://github.com/nais/device/releases/download/beta/naisdevice-beta.pkg`
2. `sudo installer -target / -pkg ./naisdevice-beta.pkg`
3. `/opt/naisdevice/start` ([allow ~20 seconds before filing issues](https://github.com/nais/device/issues/38))

#### Windows
1. Download and install [WireGuard](https://www.wireguard.com/install/)
2. ...


# Connecting to NAIS clusters
  1. open /etc/hosts as admin and comment out or remove the lines containing `apiserver.*.nais.io`
  2. in kubeconfigs repo: `git pull && git checkout naisdevice`

# FAQ

> I seem unable to reach any Microsoft services after running naisdevice agent. Is this an intentional crusade against Microsoft?

Actually, no. See https://github.com/nais/device/issues/17 to track issue.
