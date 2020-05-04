variable "gateways" {
  type = map
}

resource "google_compute_address" "gateway" {
  for_each = var.gateways
  name     = each.key
  project  = each.value.project
}

resource "google_compute_instance" "gateway" {
  for_each = var.gateways
  project  = each.value.project
  name     = each.key
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
    subnetwork = each.value.subnetwork

    access_config {
      nat_ip = google_compute_address.gateway[each.key].address
    }
  }

  allow_stopping_for_update = true

  service_account {
    email  = google_service_account.gateway[each.key].email
    scopes = ["cloud-platform"]
  }
}

resource "google_secret_manager_secret" "api-server-password" {
  for_each = var.gateways
  provider = google-beta

  project   = "nais-device"
  secret_id = "${each.value.project}_${each.key}_api-server-password"

  labels = {
    type    = "api-server-password"
    gateway = "${each.value.project}_${each.key}"
  }

  replication {
    automatic = true
  }
}

resource "google_secret_manager_secret_iam_member" "access" {
  for_each = var.gateways
  provider = google-beta

  project   = "nais-device"
  secret_id = google_secret_manager_secret.api-server-password[each.key].secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.gateway[each.key].email}"
}

data "google_service_account" "apiserver" {
  account_id = "apiserver"
  project    = "nais-device"
}

resource "google_secret_manager_secret_iam_member" "apiserver_access" {
  for_each = var.gateways
  provider = google-beta

  project   = "nais-device"
  secret_id = google_secret_manager_secret.api-server-password[each.key].secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${data.google_service_account.apiserver.email}"
}

resource "google_service_account" "gateway" {
  for_each = var.gateways
  project  = each.value.project

  account_id   = each.key
  display_name = "${each.key} service account"
}

resource "random_password" "password" {
  for_each = var.gateways

  length           = 16
  special          = true
  override_special = "_%@"
}

resource "google_secret_manager_secret_version" "gateway" {
  for_each = var.gateways
  provider = google-beta

  secret      = google_secret_manager_secret.api-server-password[each.key].id
  secret_data = random_password.password[each.key].result
}
