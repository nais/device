variable "apiserver_tunnel_ip" {
  type = string
}
variable "db_connection_uri" {
  type = string
}

data "google_compute_network" "default" {
  name = "default"
}

resource "google_compute_address" "apiserver" {
  name = "apiserver"
}

resource "google_compute_instance" "apiserver" {
  name         = "apiserver"
  machine_type = "f1-micro"
  zone         = "europe-north1-a"

  tags = ["wireguard"]

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-1804-lts"
    }
  }

  metadata = {
    startup-script = <<EOS
add-apt-repository --yes ppa:wireguard/wireguard
apt-get update --yes
apt-get install --yes wireguard

# Setup wgctrl
wgctrl_private_key=$(wg genkey)

mkdir -p /etc/wireguard
cat << EOF > /etc/wireguard/wgctrl.conf
[Interface]
ListenPort = 51820
PrivateKey = $wgctrl_private_key
EOF

ip link add dev wgctrl type wireguard
ip link set wgctrl mtu 1380
ip address add dev wgctrl ${var.apiserver_tunnel_ip}/21
wg setconf wgctrl /etc/wireguard/wgctrl.conf
ip link set wgctrl up

# Setup systemd service
cat << EOF > /etc/systemd/system/apiserver.service
[Unit]
Description=Gateway Agent

[Service]
ExecStartPre=/bin/bash -c '/usr/bin/curl -LO https://github.com/nais/device/releases/download/\$(curl --silent "https://api.github.com/repos/nais/device/releases/latest" | grep \'"tag_name":\' | sed -E \'s/.*"([^"]+)".*/\1/\')/gateway-agent'
ExecStartPre=/bin/chmod 700 gateway-agent
ExecStartPre=/bin/mkdir -p /opt/nais-device/bin/
ExecStartPre=/bin/mv gateway-agent /opt/nais-device/bin/
ExecStart=/opt/nais-device/bin/apiserver \
      --db-connection-uri "${var.db_connection_uri}"

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable apiserver
EOS
  }

  network_interface {
    network = "default"

    access_config {
      nat_ip = google_compute_address.apiserver.address
    }
  }
}
