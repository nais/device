#!/usr/bin/env bash
terraform init \
  -backend-config="bucket=nais-device-tfstate"

terraform apply "$@"
