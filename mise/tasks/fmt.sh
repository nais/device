#!/usr/bin/env bash
#MISE description="Format go files using gofumpt"
go tool mvdan.cc/gofumpt -w ./
buf format -w pkg/pb/protobuf-api.proto
