#!/usr/bin/env bash
#MISE description="Generate protobuf"
protoc \
  --go-grpc_opt=paths=source_relative \
  --go_opt=paths=source_relative \
  --go_out=. \
  --go-grpc_out=. \
  pkg/pb/protobuf-api.proto
