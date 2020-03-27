terraform {
  backend "gcs" {

  }
}

variable "apiserver_tunnel_ip" {
  type = string
}

provider "google" {
  project = "nais-device"
  region  = "europe-north1"
  version = "3.14"
}

provider "google-beta" {
  project = "nais-device"
  region  = "europe-north1"
  version = "3.14"
}

data "google_compute_network" "default" {
  name = "default"
}

resource "google_compute_address" "apiserver" {
  name = "apiserver"
}

resource "google_compute_firewall" "wireguard-rule" {
  name    = "allow-wireguard"
  network = data.google_compute_network.default.name

  allow {
    protocol = "udp"
    ports    = ["51820"]
  }

  target_tags = ["wireguard"]
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

  metadata_startup_script = <<EOS
add-apt-repository ppa:wireguard/wireguard --yes
apt-get update --yes
apt-get install wireguard --yes

private=$(wg genkey)
wg pubkey <<< "$private" > /root/pubkey

ip link add dev ctrl type wireguard
ip link set ctrl mtu 1380
ip address add dev ctrl ${var.apiserver_tunnel_ip}/21

mkdir -p /etc/wireguard
cat << EOF > /etc/wireguard/ctrl.conf
[Interface]
ListenPort = 51820
PrivateKey = $private
EOF

wg setconf ctrl /etc/wireguard/ctrl.conf
ip link set ctrl up
EOS

  network_interface {
    network = "default"

    access_config {
      nat_ip = google_compute_address.apiserver.address
    }
  }
}

/*
resource "google_sql_database" "database" {
  name     = "nais-device"
  instance = google_sql_database_instance.instance.name
}

resource "google_sql_database_instance" "instance" {
  name             = "nais-device"
  database_version = "POSTGRES_11"
  settings {
    tier = "db-f1-micro"

    ip_configuration {
      ipv4_enabled    = false
      private_network = data.google_compute_network.default.self_link
    }
  }
}

resource "google_compute_global_address" "private_db_ip" {
  provider = google-beta

  name         = "private-db-ip"
  purpose      = "VPC_PEERING"
  address_type = "INTERNAL"
  prefix_length = 30
  network      = data.google_compute_network.default.self_link
}

resource "google_service_networking_connection" "private_vpc_connection" {
  provider = google-beta

  network                 = data.google_compute_network.default.self_link
  service                 = "servicenetworkpeering.googlepis.com"
  reserved_peering_ranges = [google_compute_global_address.private_db_ip.name]
}

resource "google_sql_user" "apiserver" {
  name     = "apiserver"
  instance = google_sql_database_instance.instance.name
  password = "secret"
}
*/
