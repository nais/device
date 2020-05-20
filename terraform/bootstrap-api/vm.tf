resource "google_service_account" "bootstrap-api" {
  project = "nais-device"

  account_id   = "bootstrap-api"
  display_name = "bootstrap-api service account"
}

// network
resource "google_compute_network" "bootstrap-api" {
  name = "bootstrap-api"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "bootstrap-api" {
  name          = "bootstrap-api"
  ip_cidr_range = "10.7.10.128/28"
  region        = "europe-north1"
  network       = google_compute_network.bootstrap-api.id
}

data "google_compute_lb_ip_ranges" "ranges" {}

resource "google_compute_firewall" "lb" {
  name    = "bootstrap-api-lb-firewall"
  network = google_compute_network.bootstrap-api.name
  allow {
    protocol = "tcp"
    ports    = ["8080"]
  }
  source_ranges = data.google_compute_lb_ip_ranges.ranges.network
  target_tags = [
    "bootstrap-api",
  ]
}

resource "google_compute_firewall" "allow-ssh" {
  name    = "allow-ssh"
  network = google_compute_network.bootstrap-api.name

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  target_tags = ["allow-ssh"]
}

resource "google_compute_address" "bootstrap-api" {
  name = "bootstrap-api"
}

// loadbalancing
resource "google_compute_managed_ssl_certificate" "device-nais-io" {
  provider = google-beta
  name = "device-nais-io-cert"
  managed {
    domains = [
      "bootstrap.device.nais.io"
    ]
  }
}

resource "google_compute_target_https_proxy" "bootstrap-api" {
  name             = "bootstrap-api"
  url_map          = google_compute_url_map.bootstrap-api.self_link
  ssl_certificates = [google_compute_managed_ssl_certificate.device-nais-io.self_link]
}

resource "google_compute_url_map" "bootstrap-api" {
  name        = "bootstrap-api-lb"
  description = "Bootstrap API loadbalancer"
  default_service = google_compute_backend_service.bootstrap-api.self_link
  host_rule {
    hosts        = ["bootstrap.device.nais.io"]
    path_matcher = "allpaths"
  }
  path_matcher {
    name            = "allpaths"
    default_service = google_compute_backend_service.bootstrap-api.self_link
    path_rule {
      paths   = ["/*"]
      service = google_compute_backend_service.bootstrap-api.self_link
    }
  }
}

resource "google_compute_backend_service" "bootstrap-api" {
  name        = "bootstrap-api-backend"
  port_name   = "http"
  protocol    = "HTTP"
  timeout_sec = 10
  health_checks = [google_compute_health_check.default.self_link]
  backend {
    group = google_compute_instance_group.bootstrap-api.self_link
  }
}

resource "google_compute_health_check" "default" {
  name               = "bootstrap-api-health-check"
  timeout_sec        = 1
  check_interval_sec = 1
  http_health_check {
    port         = "8080"
    request_path = "/isalive"
  }
}

resource "google_compute_global_address" "bootstrap-api" {
  name = "bootstrap-api"
}

resource "google_compute_global_forwarding_rule" "default" {
  provider = google-beta
  ip_address            = google_compute_global_address.bootstrap-api.address
  load_balancing_scheme = "EXTERNAL"
  name                  = "bootstrap-api-forwarding-rule"
  target                = google_compute_target_https_proxy.bootstrap-api.self_link
  port_range            = 443
}

// compute
resource "google_compute_instance" "bootstrap-api" {
  name         = "bootstrap-api"
  machine_type = "f1-micro"
  zone         = "europe-north1-a"

  tags = ["bootstrap-api", "allow-ssh"]

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-2004-lts"
    }
  }

  network_interface {
    network = google_compute_network.bootstrap-api.self_link
    subnetwork = google_compute_subnetwork.bootstrap-api.self_link

      access_config {
      nat_ip = google_compute_address.bootstrap-api.address
    }
  }

  allow_stopping_for_update = true

  service_account {
    email  = google_service_account.bootstrap-api.email
    scopes = ["cloud-platform"]
  }
}


resource "google_compute_instance_group" "bootstrap-api" {
  name = "bootstrap-api"

  network = google_compute_network.bootstrap-api.self_link

  instances = [
    google_compute_instance.bootstrap-api.self_link,
  ]

  named_port {
    name = "http"
    port = "8080"
  }

  zone = "europe-north1-a"
}

// secrets
resource "google_secret_manager_secret" "bootstrap-api-password" {
  provider = google-beta

  project   = "nais-device"
  secret_id = "nais-device_api-server_bootstrap-api-password"

  labels = {
    type      = "bootstrap-api-password"
    component = "bootstrap-api"
  }

  replication {
    automatic = true
  }
}

resource "google_secret_manager_secret_iam_member" "access" {
  provider = google-beta

  project   = "nais-device"
  secret_id = google_secret_manager_secret.bootstrap-api-password.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.bootstrap-api.email}"
}

data "google_service_account" "apiserver" {
  account_id = "apiserver"
  project    = "nais-device"
}

resource "google_secret_manager_secret_iam_member" "apiserver_access" {
  provider = google-beta

  project   = "nais-device"
  secret_id = google_secret_manager_secret.bootstrap-api-password.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${data.google_service_account.apiserver.email}"
}

resource "random_password" "password" {
  length           = 16
  special          = true
  override_special = "_%@"
}

resource "google_secret_manager_secret_version" "access" {
  provider = google-beta

  secret      = google_secret_manager_secret.bootstrap-api-password.id
  secret_data = random_password.password.result
}
