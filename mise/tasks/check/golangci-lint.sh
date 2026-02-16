#!/usr/bin/env bash
#MISE description="Run golangci-lint"
go tool golangci-lint run --timeout=2m --tests=false
