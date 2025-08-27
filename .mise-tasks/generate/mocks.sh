#!/usr/bin/env bash
#MISE description="Generate mocks using mockery"
go tool github.com/vektra/mockery/v3
find internal -type f -name "mock_*.go" -exec go tool mvdan.cc/gofumpt -w {} \;
