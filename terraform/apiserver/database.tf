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
  }
}
*/