variable "projects" {
  type = list
}

resource "google_compute_firewall" "wireguard-rule" {
  count   = length(var.projects)
  name    = "allow-wireguard"
  project = var.projects[count.index].name
  network = var.projects[count.index].network

  allow {
    protocol = "udp"
    ports    = ["51820"]
  }

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  target_tags = ["wireguard"]
}