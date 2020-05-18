# Nais device

![Kolide checks criticality](https://github.com/nais/device/workflows/Kolide%20checks%20criticality/badge.svg)

## Wireguard setup (for all vms)
  1. `# apt-get install --yes wireguard`
  2. mkdir -p /usr/local/etc/nais-device
  3. wg genkey > /usr/local/etc/nais-device/private.key

## Gateway
  1. `# sed -i -e 's/#net.ipv4.ip_forward=1/net.ipv4.ip_forward=1/' /etc/sysctl.conf`
  2. `# sysctl -p`

## Apiserver

## Postgres
  1. set up managed postgres (TODO)

## Prometheus
  0. wireguard setup
  1. create prometheus vm
  2. `# apt get install prometheus`
  3. add apiserver/gateways tunnel ips as targets to: `/etc/prometheus/prometheus.yml`
  4. sudo systemctl restart prometheus
  5. on apiserver/gateways, `# apt install prometheus-node-exporter` (default setup is fine)
  6. $$$ profit $$$

