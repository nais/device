.PHONY: test alpine

all: test alpine

alpine:
	go build -a -installsuffix cgo -o bin/apiserver cmd/apiserver/main.go

linux:
	GOOS=linux GOARCH=amd64 go build -o bin/apiserver cmd/apiserver/main.go
	GOOS=linux GOARCH=amd64 go build -o bin/gateway-agent cmd/gateway-agent/main.go
	GOOS=linux GOARCH=amd64 go build -o bin/client-agent cmd/client-agent/main.go

local:
	go build -o bin/apiserver cmd/apiserver/main.go
	go build -o bin/gateway-agent cmd/gateway-agent/main.go
	go build -o bin/client-agent cmd/client-agent/main.go

local-apiserver:
	go run ./cmd/apiserver/main.go --bind-address=127.0.0.1:8080 --db-connection-uri=${DB_CONNECTION_URI}

test:
	go test ./... -count=1
