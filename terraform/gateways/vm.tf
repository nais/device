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

mkdir -p /usr/local/etc/nais-device
wg genkey > /usr/local/etc/nais-device/private.key

# Enable ip forward
sed -i -e 's/#net.ipv4.ip_forward=1/net.ipv4.ip_forward=1/' /etc/sysctl.conf
sysctl -p

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
