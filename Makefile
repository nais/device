.PHONY: test
DATE = $(shell date "+%Y-%m-%d")
LAST_COMMIT = $(shell git --no-pager log -1 --pretty=%h)
VERSION ?= $(DATE)-$(LAST_COMMIT)
LDFLAGS := -X github.com/nais/device/pkg/version.Revision=$(shell git rev-parse --short HEAD) -X github.com/nais/device/pkg/version.Version=$(VERSION)
PKGID = io.nais.device
GOPATH ?= ~/go

all: test
local-postgres: stop-postgres run-postgres
dev-apiserver: local-postgres local-apiserver stop-postgres
integration-test: stop-postgres-test run-postgres-test run-integration-test stop-postgres-test
clients: linux-client macos-client windows-client

# Before building linux-client and debian package, these are needed
linux-init:
	sudo apt update
	sudo apt install build-essential libgtk-3-dev libappindicator3-dev ruby ruby-dev rubygems
	sudo gem install --no-document fpm

# Run by GitHub actions
controlplane:
	mkdir -p ./bin/controlplane
	GOOS=linux GOARCH=amd64 go build -o bin/controlplane/apiserver ./cmd/apiserver
	GOOS=linux GOARCH=amd64 go build -o bin/controlplane/bootstrap-api -ldflags "-s $(LDFLAGS)" ./cmd/bootstrap-api
	GOOS=linux GOARCH=amd64 go build -o bin/controlplane/gateway-agent -ldflags "-s $(LDFLAGS)" ./cmd/gateway-agent
	GOOS=linux GOARCH=amd64 go build -o bin/controlplane/prometheus-agent ./cmd/prometheus-agent

# Run by GitHub actions on linux
linux-client: bin/linux-client/device-agent bin/linux-client/device-agent-helper
bin/linux-client/device-agent-helper:
	mkdir -p ./bin/linux-client
	GOOS=linux GOARCH=amd64 go build -o bin/linux-client/device-agent-helper ./cmd/device-agent-helper
bin/linux-client/device-agent: cmd/device-agent/icons.go
	mkdir -p ./bin/linux-client
	GOOS=linux GOARCH=amd64 go build -o bin/linux-client/device-agent ./cmd/device-agent

# Run by GitHub actions on macos
macos-client: cmd/device-agent/icons.go
	mkdir -p ./bin/macos-client
	GOOS=darwin GOARCH=amd64 go build -o bin/macos-client/device-agent -ldflags "-s $(LDFLAGS)" ./cmd/device-agent
	GOOS=darwin GOARCH=amd64 go build -o bin/macos-client/device-agent-helper ./cmd/device-agent-helper

# Run by GitHub actions on linux
windows-client: cmd/device-agent/icons.go
	mkdir -p ./bin/windows-client
	go get github.com/akavel/rsrc
	${GOPATH}/bin/rsrc -arch amd64 -manifest ./packaging/windows/admin_manifest.xml -ico assets/nais-logo-blue.ico -o ./cmd/device-agent-helper/main_windows.syso
	${GOPATH}/bin/rsrc -ico assets/nais-logo-blue.ico -o ./cmd/device-agent/main_windows.syso
	GOOS=windows GOARCH=amd64 go build -o bin/windows-client/device-agent.exe -ldflags "-s $(LDFLAGS) -H=windowsgui" ./cmd/device-agent
	GOOS=windows GOARCH=amd64 go build -o bin/windows-client/device-agent-helper.exe ./cmd/device-agent-helper

local:
	mkdir -p ./bin/local
	go build -o bin/local/apiserver ./cmd/apiserver
	go build -o bin/local/gateway-agent -ldflags "-s $(LDFLAGS)" ./cmd/gateway-agent
	go build -o bin/local/prometheus-agent ./cmd/prometheus-agent
	go build -o bin/local/bootstrap-api ./cmd/bootstrap-api

run-postgres:
	docker run -e POSTGRES_PASSWORD=postgres --rm --name postgres -p 5432:5432 -d \
	  -v ${PWD}/apiserver/database/schema/schema.sql:/docker-entrypoint-initdb.d/schema.sql \
	  -v ${PWD}/testdata.sql:/docker-entrypoint-initdb.d/testdata.sql \
		postgres

run-postgres-test:
	docker run -e POSTGRES_PASSWORD=postgres --rm --name postgres-test -p 127.0.0.1:5433:5432 -d postgres

stop-postgres:
	docker stop postgres || echo "okidoki"

stop-postgres-test:
	docker stop postgres-test || echo "okidoki"

local-gateway-agent:
	$(eval config_dir := $(shell mktemp -d))
	wg genkey > $(config_dir)/private.key
	go run ./cmd/gateway-agent/main.go --api-server-url=http://localhost:8080 --name=gateway-1 --prometheus-address=127.0.0.1:3000 --development-mode=true --config-dir $(config_dir) --log-level debug

local-apiserver:
	$(eval confdir := $(shell mktemp -d))
	wg genkey > ${confdir}/private.key
	go run ./cmd/apiserver/main.go --db-connection-uri=postgresql://postgres:postgres@localhost/postgres --bind-address=127.0.0.1:8080 --config-dir=${confdir} --development-mode=true --prometheus-address=127.0.0.1:3000 --credential-entries="nais:device,gateway-1:password"
	echo ${confdir}

cmd/device-agent/icons.go: assets/*.ico assets/icon.go
	cd assets && go run icon.go | gofmt -s > ../cmd/device-agent/icons.go

linux-icon: packaging/linux/icons/*/apps/naisdevice.png
packaging/linux/icons/*/apps/naisdevice.png: assets/nais-logo-blue.png
	for size in 16x16 32x32 64x64 128x128 256x256 512x512 ; do \
		mkdir -p packaging/linux/icons/$$size/apps/ ; \
		convert assets/nais-logo-blue.png -scale $$size^ -background none -gravity center -extent $$size packaging/linux/icons/$$size/apps/naisdevice.png ; \
  	done

macos-icon: assets/naisdevice.icns
assets/naisdevice.icns:
	rm -rf MyIcon.iconset
	mkdir -p MyIcon.iconset
	sips -z 16 16     assets/nais-logo-blue.png --out MyIcon.iconset/icon_16x16.png
	sips -z 32 32     assets/nais-logo-blue.png --out MyIcon.iconset/icon_16x16@2x.png
	sips -z 32 32     assets/nais-logo-blue.png --out MyIcon.iconset/icon_32x32.png
	sips -z 64 64     assets/nais-logo-blue.png --out MyIcon.iconset/icon_32x32@2x.png
	sips -z 128 128   assets/nais-logo-blue.png --out MyIcon.iconset/icon_128x128.png
	sips -z 256 256   assets/nais-logo-blue.png --out MyIcon.iconset/icon_128x128@2x.png
	sips -z 256 256   assets/nais-logo-blue.png --out MyIcon.iconset/icon_256x256.png
	sips -z 512 512   assets/nais-logo-blue.png --out MyIcon.iconset/icon_256x256@2x.png
	sips -z 512 512   assets/nais-logo-blue.png --out MyIcon.iconset/icon_512x512.png
	cp assets/nais-logo-blue.png MyIcon.iconset/icon_512x512@2x.png
	iconutil -c icns MyIcon.iconset
	mv MyIcon.icns assets/naisdevice.icns
	rm -R MyIcon.iconset

wg: bin/macos-client/wg
bin/macos-client/wg:
	mkdir -p bin/macos-client
	curl -L https://git.zx2c4.com/wireguard-tools/snapshot/wireguard-tools-1.0.20200513.tar.xz | tar x
	cd wireguard-tools-*/src && make && cp wg ../../bin/macos-client/
	rm -rf ./wireguard-tools-*

wireguard-go: bin/macos-client/wireguard-go
bin/macos-client/wireguard-go:
	mkdir -p bin/macos-client
	curl -L https://git.zx2c4.com/wireguard-go/snapshot/wireguard-go-0.0.20200320.tar.xz | tar x
	cd wireguard-go-*/ && make && cp wireguard-go ../bin/macos-client/
	rm -rf ./wireguard-go-*

app: wg wireguard-go macos-icon macos-client
	rm -rf naisdevice.app
	mkdir -p naisdevice.app/Contents/{MacOS,Resources}
	cp bin/macos-client/* naisdevice.app/Contents/MacOS
	cp assets/naisdevice.icns naisdevice.app/Contents/Resources
	sed 's/VERSIONSTRING/${VERSION}/' packaging/macos/Info.plist.tpl > naisdevice.app/Contents/Info.plist
	gon --log-level=debug packaging/macos/gon-app.json

test:
	go test ./... -count=1

run-integration-test:
	RUN_INTEGRATION_TESTS="true" go test ./... -count=1

# Run by GitHub actions on macos
pkg: app
	rm -f ./naisdevice*.pkg
	rm -rf ./pkgtemp
	mkdir -p ./pkgtemp/{scripts,pkgroot/Applications}
	cp -r ./naisdevice.app ./pkgtemp/pkgroot/Applications/
	cp ./packaging/macos/postinstall ./pkgtemp/scripts/postinstall
	pkgbuild --root ./pkgtemp/pkgroot --identifier ${PKGID} --scripts ./pkgtemp/scripts --version ${VERSION} --ownership recommended ./component.pkg
	productbuild --identifier ${PKGID}.${VERSION} --package ./component.pkg ./unsigned.pkg
	productsign --sign "Developer ID Installer: Torbjorn Hallenberg" unsigned.pkg naisdevice.pkg
	rm -f ./component.pkg ./unsigned.pkg
	rm -rf ./pkgtemp ./naisdevice.app
	# gon --log-level=debug packaging/macos/gon-pkg.json

# Run by GitHub actions on linux
deb: linux-client linux-icon
	./packaging/linux/build-deb $(VERSION)

clean:
	rm -rf wireguard-go-*
	rm -rf wireguard-tools-*
	rm -rf naisdevice.app
	rm -f naisdevice-*.pkg
	rm -rf ./bin
	rm -rf ./packaging/linux/icons

