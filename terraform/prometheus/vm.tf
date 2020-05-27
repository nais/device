data "google_compute_network" "naisdevice" {
  name = "naisdevice"
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

  network_interface {
    network = data.google_compute_network.naisdevice.self_link
    subnetwork = "naisdevice"

    access_config {
      nat_ip = google_compute_address.prometheus.address
    }
  }

  allow_stopping_for_update = true

  service_account {
    email  = google_service_account.prometheus.email
    scopes = ["cloud-platform"]
  }
}

resource "google_secret_manager_secret" "api-server-password" {
  provider = google-beta

  project   = "nais-device"
  secret_id = "nais-device_prometheus_api-server-password"

  labels = {
    type      = "api-server-password"
    component = "prometheus"
  }

  replication {
    automatic = true
  }
}

resource "google_secret_manager_secret_iam_member" "access" {
  provider = google-beta

  project   = "nais-device"
  secret_id = google_secret_manager_secret.api-server-password.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.prometheus.email}"
}

data "google_service_account" "apiserver" {
  account_id = "apiserver"
  project    = "nais-device"
}

resource "google_secret_manager_secret_iam_member" "apiserver_access" {
  provider = google-beta

  project   = "nais-device"
  secret_id = google_secret_manager_secret.api-server-password.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${data.google_service_account.apiserver.email}"
}

resource "google_service_account" "prometheus" {
  project = "nais-device"

  account_id   = "prometheus"
  display_name = "prometheus service account"
}

resource "random_password" "password" {
  length           = 16
  special          = true
  override_special = "_%@"
}

resource "google_secret_manager_secret_version" "access" {
  provider = google-beta

  secret      = google_secret_manager_secret.api-server-password.id
  secret_data = random_password.password.result
}
