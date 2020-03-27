.PHONY: test alpine

all: test alpine

alpine:
	go build -a -installsuffix cgo -o bin/apiserver cmd/apiserver/main.go

test:
	go test ./... -count=1
