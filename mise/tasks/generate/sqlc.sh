#!/usr/bin/env bash
#MISE description="Generate sqlc code"
go tool github.com/sqlc-dev/sqlc/cmd/sqlc generate
go tool mvdan.cc/gofumpt -w ./internal/apiserver/sqlc/
