.PHONY: test alpine

all: test alpine
dev-apiserver: local-postgres local-apiserver

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

local-postgres:
	docker rm -f postgres || echo "okidoki"
	docker run -e POSTGRES_PASSWORD=postgres --rm --name postgres -p 5432:5432 postgres &
	sleep 5
	PGPASSWORD=postgres psql -h localhost -U postgres -f db-bootstrap.sql

local-apiserver:
	sudo go run ./cmd/apiserver/main.go --db-connection-uri=postgresql://postgres:postgres@localhost/postgres --bind-address=127.0.0.1:8080 --slack-token=${APISERVER_SLACK_TOKEN} --skip-setup-interface=true

test:
	go test ./... -count=1
