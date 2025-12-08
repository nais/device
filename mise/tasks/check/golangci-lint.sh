#!/usr/bin/env bash
#MISE description="Run golangci-lint"
golangci-lint run --timeout=2m --tests=false
