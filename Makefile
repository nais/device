.PHONY: test alpine

all: test alpine
dev-apiserver: teardown-postgres run-postgres insert-testdata local-apiserver
integration-test: run-postgres-test run-integration-test teardown-postgres-test

alpine:
	go build -a -installsuffix cgo -o bin/apiserver cmd/apiserver/main.go

linux:
	GOOS=linux GOARCH=amd64 go build -o bin/apiserver ./cmd/apiserver
	GOOS=linux GOARCH=amd64 go build -o bin/gateway-agent ./cmd/gateway-agent
	GOOS=linux GOARCH=amd64 go build -o bin/prometheus-agent ./cmd/prometheus-agent
	php -d phar.readonly=off device-health-checker/create-phar.php device-health-checker/get-checks.php device-health-checker/bin
	php -d phar.readonly=off device-health-checker/create-phar.php device-health-checker/device-health-checker.php device-health-checker/bin

local:
	go build -o bin/apiserver ./cmd/apiserver
	go build -o bin/gateway-agent ./cmd/gateway-agent
	go build -o bin/device-agent ./cmd/device-agent
	go build -o bin/device-agent-helper ./cmd/device-agent-helper
	go build -o bin/prometheus-agent cmd/prometheus-agent/main.go

run-postgres:
	docker run -e POSTGRES_PASSWORD=postgres --rm --name postgres -p 5432:5432 postgres &
	for attempt in {0..5}; do \
 		sleep 2;\
		PGPASSWORD=postgres psql -h localhost -U postgres -f apiserver/database/schema/schema.sql && break;\
    done

insert-testdata:
	PGPASSWORD=postgres psql -h localhost -U postgres -f testdata.sql

run-postgres-test:
	docker run -e POSTGRES_PASSWORD=postgres --rm --name postgres-test -p 5433:5432 postgres &
	for attempt in {0..5}; do \
 		sleep 2;\
		PGPASSWORD=postgres psql -h localhost -p 5433 -U postgres -l && break;\
    done

teardown-postgres:
	docker rm -f postgres || echo "okidoki"

teardown-postgres-test:
	docker rm -f postgres-test || echo "okidoki"

local-gateway-agent:
	go run ./cmd/gateway-agent/main.go --api-server-url=http://localhost:8080 --name=gw0 --prometheus-address=127.0.0.1:3000 --development-mode=true

local-apiserver:
	$(eval confdir := $(shell mktemp -d))
	wg genkey > ${confdir}/private.key
	go run ./cmd/apiserver/main.go --db-connection-uri=postgresql://postgres:postgres@localhost/postgres --bind-address=127.0.0.1:8080 --slack-token=${APISERVER_SLACK_TOKEN} --config-dir=${confdir} --development-mode=true --prometheus-address=127.0.0.1:3000 --credential-entries=nais:device
	echo ${confdir}

test:
	go test ./... -count=1

run-integration-test:
	RUN_INTEGRATION_TESTS="true" go test ./... -count=1

