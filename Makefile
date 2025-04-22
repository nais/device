.PHONY: test macos-client

LAST_COMMIT = $(shell git --no-pager log -1 --pretty=%h)
VERSION := $(shell date "+%Y-%m-%d-%H%M%S")
OTEL_COLLECTOR_ENDPOINT := "https://collector-internet.nav.cloud.nais.io"
LDFLAGS := -X github.com/nais/device/internal/version.Revision=$(shell git rev-parse --short HEAD) -X github.com/nais/device/internal/version.Version=$(VERSION) -X github.com/nais/device/internal/otel.endpointURL=$(OTEL_COLLECTOR_ENDPOINT)
PKGID = io.nais.device
RELEASE ?= false
GOPATH ?= ~/go
GOTAGS ?=

PROTOC = $(shell which protoc)

all: test
clients: linux-client macos-client windows-client

proto: install-protobuf-go
	PATH="${PATH}:$(shell go env GOPATH)/bin" ${PROTOC} --go-grpc_opt=paths=source_relative --go_opt=paths=source_relative --go_out=. --go-grpc_out=. internal/pb/protobuf-api.proto

install-protobuf-go:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1

# Before building linux-client and debian package, these are needed
linux-init:
	sudo apt update
	sudo apt install build-essential ruby ruby-dev rubygems imagemagick
	sudo gem install --no-document fpm

# Run by GitHub actions
controlplane:
	mkdir -p ./bin/controlplane
	GOOS=linux GOARCH=amd64 go build -o bin/controlplane/apiserver --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/apiserver
	GOOS=linux GOARCH=amd64 go build -o bin/controlplane/gateway-agent --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/gateway-agent
	GOOS=linux GOARCH=amd64 go build -o bin/controlplane/prometheus-agent --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/prometheus-agent

# Run by GitHub actions on linux
linux-client:
	mkdir -p ./bin/linux-client
	GOOS=linux GOARCH=amd64 go build -o bin/linux-client/naisdevice-systray --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/naisdevice-systray
	GOOS=linux GOARCH=amd64 go build -o bin/linux-client/naisdevice-agent --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/naisdevice-agent
	GOOS=linux GOARCH=amd64 go build -o bin/linux-client/naisdevice-helper --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/naisdevice-helper

# Run by GitHub actions on macos
macos-client:
	mkdir -p ./bin/macos-client
	GOOS=darwin GOARCH=amd64 go build -o bin/macos-client/naisdevice-agent --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/naisdevice-agent
	GOOS=darwin GOARCH=amd64 go build -o bin/macos-client/naisdevice-systray --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/naisdevice-systray
	GOOS=darwin GOARCH=amd64 go build -o bin/macos-client/naisdevice-helper --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/naisdevice-helper

# Run by GitHub actions on linux
windows-client:
	mkdir -p ./bin/windows-client

	go tool github.com/akavel/rsrc -arch amd64 -manifest ./packaging/windows/admin_manifest.xml -ico assets/nais-logo-blue.ico -o ./cmd/naisdevice-helper/main_windows.syso
	go tool github.com/akavel/rsrc -ico assets/nais-logo-blue.ico -o ./cmd/naisdevice-agent/main_windows.syso
	GOOS=windows GOARCH=amd64 go build -o bin/windows-client/naisdevice-systray.exe --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS) -H=windowsgui" ./cmd/naisdevice-systray
	./packaging/windows/sign-exe bin/windows-client/naisdevice-systray.exe ./packaging/windows/naisdevice.crt ./packaging/windows/naisdevice.key
	GOOS=windows GOARCH=amd64 go build -o bin/windows-client/naisdevice-agent.exe --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS) -H=windowsgui" ./cmd/naisdevice-agent
	./packaging/windows/sign-exe bin/windows-client/naisdevice-agent.exe ./packaging/windows/naisdevice.crt ./packaging/windows/naisdevice.key
	GOOS=windows GOARCH=amd64 go build -o bin/windows-client/naisdevice-helper.exe --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/naisdevice-helper
	./packaging/windows/sign-exe bin/windows-client/naisdevice-helper.exe ./packaging/windows/naisdevice.crt ./packaging/windows/naisdevice.key

local:
	mkdir -p ./bin/local
	go build -o bin/local/apiserver --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/apiserver
	go build -o bin/local/gateway-agent --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/gateway-agent
	go build -o bin/local/prometheus-agent --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/prometheus-agent
	go build -o bin/local/controlplane-cli --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/controlplane-cli
	go build -o bin/local/naisdevice-agent --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/naisdevice-agent
	go build -o bin/local/naisdevice-systray --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/naisdevice-systray
	go build -o bin/local/naisdevice-helper --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/naisdevice-helper

linux-icon: packaging/linux/icons/*/apps/naisdevice.png
packaging/linux/icons/*/apps/naisdevice.png: assets/svg/blue.svg
	for size in 16x16 32x32 64x64 128x128 256x256 512x512 ; do \
		mkdir -p packaging/linux/icons/$$size/apps/ ; \
		convert -background transparent -resize $$size -gravity center -extent $$size assets/svg/blue.svg packaging/linux/icons/$$size/apps/naisdevice.png ; \
  	done

windows-icon: packaging/windows/assets/naisdevice.ico
packaging/windows/assets/naisdevice.ico: assets/svg/blue.svg
	mkdir -p packaging/windows/assets/
	convert -background transparent -resize 256x256 -gravity center -extent 256x256 assets/svg/blue.svg -define icon:auto-resize=48,64,96,128,256 packaging/windows/assets/naisdevice.ico


macos-icon: packaging/macos/icons/naisdevice.icns
packaging/macos/icons/naisdevice.icns:
	mkdir -p packaging/macos/icons/
	magick assets/svg/blue.svg -background transparent -resize 1024x1024 -gravity center -extent 1024x1024 packaging/macos/icons/naisdevice.png
	go tool github.com/jackmordaunt/icns/v2/cmd/icnsify -i packaging/macos/icons/naisdevice.png -o packaging/macos/icons/naisdevice.icns

wg: bin/macos-client/wg
bin/macos-client/wg:
	mkdir -p bin/macos-client
	curl -L https://git.zx2c4.com/wireguard-tools/snapshot/wireguard-tools-1.0.20210914.tar.xz | tar xJ
	cd wireguard-tools-*/src && make && cp wg ../../bin/macos-client/
	rm -rf ./wireguard-tools-*

wireguard-go: bin/macos-client/wireguard-go
bin/macos-client/wireguard-go:
	mkdir -p bin/macos-client
	curl -L https://git.zx2c4.com/wireguard-go/snapshot/wireguard-go-0.0.20230223.tar.xz | tar xJ
	cd wireguard-go-*/ && make && cp wireguard-go ../bin/macos-client/
	rm -rf ./wireguard-go-*

test:
	@go test $(shell go list ./... | grep -v systray) -count=1

test-race:
	@go test $(shell go list ./... | grep -v systray) -count=1 -race

# Run by GitHub actions on macos
pkg: wg wireguard-go macos-icon macos-client
	./packaging/macos/build-pkg $(VERSION) $(RELEASE)

# Run by GitHub actions on linux
deb: linux-client linux-icon
	./packaging/linux/build-deb $(VERSION)

# Run by GitHub actions on linux(!)
nsis: windows-client windows-icon
	./packaging/windows/build-nsis $(VERSION)

controlplane_paths = $(wildcard packaging/controlplane/*)
controlplane_components = $(controlplane_paths:packaging/controlplane/%=%)

controlplane-debs: $(controlplane_components)
$(controlplane_components): controlplane
	@echo packaging $@
	./packaging/controlplane/$@/build-deb $(VERSION)


clean:
	rm -rf *.deb
	rm -rf wireguard-go-*
	rm -rf wireguard-tools-*
	rm -rf naisdevice.app
	rm -f naisdevice-*.pkg
	rm -f naisdevice-*.deb
	rm -f ./packaging/windows/naisdevice*.exe
	rm -rf ./bin
	rm -rf ./packaging/*/icons
	rm -rf ./packaging/*/assets

mocks:
	go tool github.com/vektra/mockery/v2
	find internal -type f -name "mock_*.go" -exec go tool mvdan.cc/gofumpt -w {} \;

# controlplane is autoreleased for every push
release-frontend:
	git tag ${VERSION}
	git push --tags

buildreleaseenroller:
	docker build -t europe-north1-docker.pkg.dev/nais-io/nais/images/naisdevice-enroller:${VERSION} -f cmd/enroller/Dockerfile .
	docker push europe-north1-docker.pkg.dev/nais-io/nais/images/naisdevice-enroller:${VERSION}

buildreleaseauthserver:
	cd cmd/auth-server && docker build -t europe-north1-docker.pkg.dev/nais-io/nais/images/naisdevice-auth-server:${VERSION} .
	docker push europe-north1-docker.pkg.dev/nais-io/nais/images/naisdevice-auth-server:${VERSION}

generate-sqlc:
	go tool github.com/sqlc-dev/sqlc/cmd/sqlc generate
	go tool mvdan.cc/gofumpt -w ./internal/apiserver/sqlc/

fmt:
	go tool mvdan.cc/gofumpt -w ./

lint:
	go tool github.com/golangci/golangci-lint/cmd/golangci-lint run

staticcheck:
	go tool honnef.co/go/tools/cmd/staticcheck ./...

generate-guievent-strings:
	go tool golang.org/x/tools/cmd/stringer -type=GuiEvent ./internal/systray

generate: generate-guievent-strings mocks generate-sqlc proto

govulncheck:
	go tool golang.org/x/vuln/cmd/govulncheck ./...

check: staticcheck govulncheck
