variable "prometheus_tunnel_ip" {
  type = string
}

data "google_compute_network" "default" {
  name = "default"
}

resource "google_compute_address" "prometheus" {
  name = "prometheus"
}

resource "google_compute_instance" "prometheus" {
  name         = "prometheus"
  machine_type = "f1-micro"
  zone         = "europe-north1-a"

  tags = ["wireguard", "prometheus"]

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-2004-lts"
    }
  }

  metadata = {
    startup-script = <<EOS
add-apt-repository --yes ppa:wireguard/wireguard
apt-get update --yes
apt-get install --yes wireguard

# Generate wg private key
mkdir -p "/usr/local/etc/nais-device"
wg genkey > "/usr/local/etc/nais-device/private.key"

EOS
  }

  network_interface {
    network = data.google_compute_network.default.self_link

    access_config {
      nat_ip = google_compute_address.prometheus.address
    }
  }
}
