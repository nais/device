data "google_compute_network" "default" {
  name = "default"
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