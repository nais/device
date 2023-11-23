//go:build tools
// +build tools

package tools

import (
	_ "github.com/akavel/rsrc"
	_ "github.com/jackmordaunt/icns/v2/cmd/icnsify"
	_ "github.com/sqlc-dev/sqlc/cmd/sqlc"
	_ "github.com/vektra/mockery/v2"
	_ "golang.org/x/vuln/cmd/govulncheck"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	_ "honnef.co/go/tools/cmd/staticcheck"
	_ "mvdan.cc/gofumpt"
)
