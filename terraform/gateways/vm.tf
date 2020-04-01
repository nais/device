variable "apiserver_tunnel_ip" {
  type = string
}

variable "apiserver_public_key" {
  type = string
}

variable "apiserver_endpoint" {
  type = string
}

variable "gateways" {
  type = list
}

data "google_compute_network" "default" {
  name = "default"
}

resource "google_compute_address" "gateway" {
  count = length(var.gateways)
  name  = "gateway-${count.index}"
}

resource "google_compute_instance" "gateway" {
  count        = length(var.gateways)
  name         = var.gateways[count.index].name
  machine_type = "f1-micro"
  zone         = "europe-north1-a"

  tags = ["wireguard"]

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-1804-lts"
    }
  }

  metadata_startup_script = <<EOS
add-apt-repository --yes ppa:wireguard/wireguard
apt-get update --yes
apt-get install --yes wireguard

# Setup wgctrl
wgctrl_private_key=$(wg genkey)
wg pubkey <<< "$wgctrl_private_key" > /root/wgctrl-public.key

mkdir -p /etc/wireguard
cat << EOF > /etc/wireguard/wgctrl.conf
[Interface]
PrivateKey = $wgctrl_private_key

[Peer]
Endpoint = ${var.apiserver_endpoint}
PublicKey = ${var.apiserver_public_key}
AllowedIPs = ${var.apiserver_tunnel_ip}
EOF

ip link add dev wgctrl type wireguard
ip link set wgctrl mtu 1380
ip address add dev wgctrl ${var.gateways[count.index].ctrl_tunnel_ip}/21
wg setconf wgctrl /etc/wireguard/wgctrl.conf
ip link set wgctrl up

# Enable ip forward
sed -i -e 's/#net.ipv4.ip_forward=1/net.ipv4.ip_forward=1/' /etc/sysctl.conf
sysctl -p

# Setup wgdata (interface only)
wg genkey | tee /etc/wireguard/private.key | wg pubkey > /etc/wireguard/public.key

ip link add dev wgdata type wireguard
ip link set wgdata mtu 1380
ip address add dev wgdata ${var.gateways[count.index].data_tunnel_ip}/21
ip link set wgdata up

# Setup systemd service
cat << EOF > /etc/systemd/system/gateway-agent.service
[Unit]
Description=Gateway Agent

[Service]
ExecStartPre=/bin/bash -c '/usr/bin/curl -LO https://github.com/nais/device/releases/download/\$(curl --silent "https://api.github.com/repos/nais/device/releases/latest" | grep \'"tag_name":\' | sed -E \'s/.*"([^"]+)".*/\1/\')/gateway-agent'
ExecStartPre=/bin/chmod 700 gateway-agent
ExecStartPre=/bin/mkdir -p /opt/nais-device/bin/
ExecStartPre=/bin/mv gateway-agent /opt/nais-device/bin/
ExecStart=/opt/nais-device/bin/gateway-agent \
    --name ${var.gateways[count.index].name}

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable gateway-agent
EOS

  network_interface {
    network = "default"

    access_config {
      nat_ip = google_compute_address.gateway[count.index].address
    }
  }
}
