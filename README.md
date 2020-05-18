# device-agent
device-agent is responsible for enabling the end-user to connect to it's permitted gateways.
To be able to connect, a series of prerequisites must be in place. These will be helped/ensured by device-agent.

A information exchange between end-user and NAIS device administrator/slackbot:
If BootstrapTokenPath is not present, user will be prompted to enroll using a generated token, and the agent will exit.
When device-agent detects a valid bootstrap token, it will generate a WireGuard config file called wg0.conf placed in `cfg.ConfigDir`
This file will initially only contain the Interface definition and the APIServer peer.

It will run the device-agent-helper with params....TODO

loop:
Fetch device config from APIServer and configure generate and write WireGuard config to disk
(loop:
Monitor all connections, if one starts failing, re-fetch config and reset timer) // TODO



# Wireguard setup (for all vms)
  1. `# apt-get install --yes wireguard`
  2. mkdir -p /usr/local/etc/nais-device
  3. wg genkey > /usr/local/etc/nais-device/private.key

# Gateway
  1. `# sed -i -e 's/#net.ipv4.ip_forward=1/net.ipv4.ip_forward=1/' /etc/sysctl.conf`
  2. `# sysctl -p`

# Apiserver

# Postgres
  1. set up managed postgres (TODO)

# Prometheus
  0. wireguard setup
  1. create prometheus vm
  2. `# apt get install prometheus`
  3. add apiserver/gateways tunnel ips as targets to: `/etc/prometheus/prometheus.yml`
  4. sudo systemctl restart prometheus
  5. on apiserver/gateways, `# apt install prometheus-node-exporter` (default setup is fine)
  6. $$$ profit $$$

