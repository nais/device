.PHONY: test alpine
DATE = $(shell date "+%Y-%m-%d")
LAST_COMMIT = $(shell git --no-pager log -1 --pretty=%h)
VERSION ?= $(DATE)-$(LAST_COMMIT)
LDFLAGS := -X github.com/nais/device/pkg/version.Revision=$(shell git rev-parse --short HEAD) -X github.com/nais/device/pkg/version.Version=$(VERSION)
PKGTITLE = naisdevice
PKGID = io.nais.device
GOPATH ?= ~/go

all: test alpine
local-postgres: stop-postgres run-postgres
dev-apiserver: local-postgres local-apiserver stop-postgres
integration-test: run-postgres-test run-integration-test stop-postgres-test
clients: linux-client macos-client windows-client

linux:
	mkdir -p ./bin/linux
	GOOS=linux GOARCH=amd64 go build -o bin/apiserver ./cmd/apiserver
	GOOS=linux GOARCH=amd64 go build -o bin/bootstrap-api ./cmd/bootstrap-api
	GOOS=linux GOARCH=amd64 go build -o bin/gateway-agent -ldflags "-s $(LDFLAGS)" ./cmd/gateway-agent
	GOOS=linux GOARCH=amd64 go build -o bin/prometheus-agent ./cmd/prometheus-agent
	php -d phar.readonly=off device-health-checker/create-phar.php device-health-checker/device-health-checker.php device-health-checker/bin

linux-client:
	mkdir -p ./bin/linux
	GOOS=linux GOARCH=amd64 go build -o bin/linux/device-agent ./cmd/device-agent
	GOOS=linux GOARCH=amd64 go build -o bin/linux/device-agent-helper ./cmd/device-agent-helper

macos-client:
	mkdir -p ./bin/macos
	GOOS=darwin GOARCH=amd64 go build -o bin/macos/device-agent -ldflags "-s $(LDFLAGS)" ./cmd/device-agent
	GOOS=darwin GOARCH=amd64 go build -o bin/macos/device-agent-helper ./cmd/device-agent-helper

windows-client:
	mkdir -p ./bin/windows
	go get github.com/akavel/rsrc
	${GOPATH}/bin/rsrc -arch amd64 -manifest ./windows/admin_manifest.xml -o ./cmd/device-agent-helper/main_windows.syso
	GOOS=windows GOARCH=amd64 go build -o bin/windows/device-agent.exe -ldflags "-s $(LDFLAGS) -H=windowsgui" ./cmd/device-agent
	GOOS=windows GOARCH=amd64 go build -o bin/windows/device-agent-helper.exe ./cmd/device-agent-helper

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
	docker run -e POSTGRES_PASSWORD=postgres --rm --name postgres-test -p 5433:5432 -d postgres

stop-postgres:
	docker stop postgres || echo "okidoki"

stop-postgres-test:
	docker stop postgres-test || echo "okidoki"

local-gateway-agent:
	$(eval config_dir := $(shell mktemp -d))
	wg genkey > $(config_dir)/private.key
	go run ./cmd/gateway-agent/main.go --api-server-url=http://localhost:8080 --name=gateway-1 --api-server-password=password --prometheus-address=127.0.0.1:3000 --development-mode=true --config-dir $(config_dir) --log-level debug

local-apiserver:
	$(eval confdir := $(shell mktemp -d))
	wg genkey > ${confdir}/private.key
	go run ./cmd/apiserver/main.go --db-connection-uri=postgresql://postgres:postgres@localhost/postgres --bind-address=127.0.0.1:8080 --config-dir=${confdir} --development-mode=true --prometheus-address=127.0.0.1:3000 --credential-entries="nais:device,gateway-1:password"
	echo ${confdir}

icon:
	cd assets && go run icon.go | gofmt -s > ../cmd/device-agent/icons.go
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

wireguard-go:
	curl -L https://git.zx2c4.com/wireguard-tools/snapshot/wireguard-tools-1.0.20200513.tar.xz | tar x
	cd wireguard-tools-*/src && make

wg:
	curl -L https://git.zx2c4.com/wireguard-go/snapshot/wireguard-go-0.0.20200320.tar.xz | tar x
	cd wireguard-go-*/ && make

app: macos-client wg wireguard-go icon
	rm -rf naisdevice.app
	mkdir -p naisdevice.app/Contents/{MacOS,Resources}
	cp ./wireguard-go-*/wireguard-go ./bin/macos/
	cp ./wireguard-tools-*/src/wg ./bin/macos/
	cp bin/macos/* naisdevice.app/Contents/MacOS
	cp assets/naisdevice.icns naisdevice.app/Contents/Resources
	sed 's/VERSIONSTRING/${VERSION}/' Info.plist.tpl > naisdevice.app/Contents/Info.plist

test:
	go test ./... -count=1

run-integration-test:
	RUN_INTEGRATION_TESTS="true" go test ./... -count=1

pkg: app
	rm -f ./naisdevice*.pkg
	rm -rf ./pkgtemp
	mkdir -p ./pkgtemp/{scripts,pkgroot/Applications}
	cp -r ./naisdevice.app ./pkgtemp/pkgroot/Applications/
	cp ./scripts/postinstall ./pkgtemp/scripts/postinstall
	pkgbuild --root ./pkgtemp/pkgroot --identifier ${PKGID} --scripts ./pkgtemp/scripts --version ${VERSION} --ownership recommended ./${PKGTITLE}-${VERSION}.component.pkg
	productbuild --identifier ${PKGID}.${VERSION} --package ./${PKGTITLE}-${VERSION}.component.pkg ./${PKGTITLE}-${VERSION}.pkg
	rm -f ./${PKGTITLE}-${VERSION}.component.pkg 
	rm -rf ./pkgtemp ./naisdevice.app

clean:
	rm -rf wireguard-go-*
	rm -rf wireguard-tools-*
	rm -rf naisdevice.app
	rm -f naisdevice-*.pkg
	rm -rf ./bin
