# How to beta
  1. Slack: `/msg @Kolide installers`
  2. `brew install wireguard-tools`
  3. `ln -s /usr/local/bin/{wg,naisdevice-wg} && ln -s /usr/local/bin/{wireguard-go,naisdevice-wireguard-go}`
  4. `make local`
  5. `sudo ./bin/device-agent`
  6. follow instructions given by agent

# Kubeconfig
In kubeconfigs repo: `git checkout naisdevice`
