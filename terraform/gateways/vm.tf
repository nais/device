variable "gateways" {
  type = list
}

resource "google_compute_address" "gateway" {
  count   = length(var.gateways)
  name    = var.gateways[count.index].name
  project = var.gateways[count.index].project
}

resource "google_compute_instance" "gateway" {
  count   = length(var.gateways)
  project = var.gateways[count.index].project
  name    = var.gateways[count.index].name
  labels = {
    "usage" : "nais-device"
  }
  machine_type = "f1-micro"
  zone         = "europe-north1-a"

  tags = ["wireguard", "local-internet-gateway", "allow-ssh"]

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-1804-lts"
    }
  }

  network_interface {
    subnetwork = var.gateways[count.index].subnetwork

    access_config {
      nat_ip = google_compute_address.gateway[count.index].address
    }
  }
}
