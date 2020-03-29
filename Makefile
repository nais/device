.PHONY: test alpine

all: test alpine

alpine:
	go build -a -installsuffix cgo -o bin/apiserver cmd/apiserver/main.go

local-apiserver:
	go run ./cmd/apiserver/main.go --db-connection-uri=${DB_CONNECTION_URI} || echo "forget to export DB_CONNECTION_URL=<DSN> ?"

test:
	go test ./... -count=1
