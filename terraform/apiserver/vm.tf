data "google_compute_network" "default" {
  name = "default"
}

resource "google_compute_address" "apiserver" {
  name = "apiserver"
}

locals {
  secrets = toset(["slack-token", "database-uri", "device-health-checker"])
}

resource "google_secret_manager_secret" "secret" {
  for_each  = local.secrets
  provider  = google-beta
  secret_id = each.key

  labels = {
    component = "apiserver"
  }

  replication {
    automatic = true
  }
}

resource "google_secret_manager_secret_iam_member" "membership" {
  for_each = local.secrets
  provider = google-beta

  project   = google_secret_manager_secret.secret[each.key].project
  secret_id = google_secret_manager_secret.secret[each.key].secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.apiserver.email}"
}

resource "google_project_iam_member" "apiserver-view" {
  project = "nais-device"
  role    = "roles/secretmanager.viewer"
  member  = "serviceAccount:${google_service_account.apiserver.email}"
}

resource "google_service_account" "apiserver" {
  account_id   = "apiserver"
  display_name = "apiserver service account"
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

  network_interface {
    network = "default"

    access_config {
      nat_ip = google_compute_address.apiserver.address
    }
  }

  allow_stopping_for_update = true

  service_account {
    email  = google_service_account.apiserver.email
    scopes = ["cloud-platform"]
  }
}
