.PHONY: test macos-client

LAST_COMMIT = $(shell git --no-pager log -1 --pretty=%h)
VERSION := $(shell date "+%Y-%m-%d-%H%M%S")
LDFLAGS := -X github.com/nais/device/pkg/version.Revision=$(shell git rev-parse --short HEAD) -X github.com/nais/device/pkg/version.Version=$(VERSION)
PKGID = io.nais.device
GOPATH ?= ~/go
GOTAGS ?=

PROTOC = $(shell which protoc)

all: test
clients: linux-client macos-client windows-client

proto: install-protobuf-go
	${PROTOC} --go-grpc_opt=paths=source_relative --go_opt=paths=source_relative --go_out=. --go-grpc_out=. pkg/pb/protobuf-api.proto

install-protobuf-go:
	go install google.golang.org/protobuf/cmd/protoc-gen-go
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc

# Before building linux-client and debian package, these are needed
linux-init:
	sudo apt update
	sudo apt install build-essential libgtk-3-dev ruby ruby-dev rubygems imagemagick libayatana-appindicator3-dev
	sudo gem install --no-document fpm

# Run by GitHub actions
controlplane:
	mkdir -p ./bin/controlplane
	GOOS=linux GOARCH=amd64 go build -o bin/controlplane/apiserver --race --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/apiserver
	GOOS=linux GOARCH=amd64 go build -o bin/controlplane/bootstrap-api --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/bootstrap-api
	GOOS=linux GOARCH=amd64 go build -o bin/controlplane/gateway-agent --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/gateway-agent
	GOOS=linux GOARCH=amd64 go build -o bin/controlplane/prometheus-agent --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/prometheus-agent

# Run by GitHub actions on linux
linux-client:
	mkdir -p ./bin/linux-client
	GOOS=linux GOARCH=amd64 go build -o bin/linux-client/naisdevice-systray --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/systray
	GOOS=linux GOARCH=amd64 go build -o bin/linux-client/naisdevice-agent --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/device-agent
	GOOS=linux GOARCH=amd64 go build -o bin/linux-client/naisdevice-helper --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/helper

# Run by GitHub actions on macos
macos-client:
	mkdir -p ./bin/macos-client
	GOOS=darwin GOARCH=amd64 go build -o bin/macos-client/naisdevice-agent --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/device-agent
	GOOS=darwin GOARCH=amd64 go build -o bin/macos-client/naisdevice-systray --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/systray
	GOOS=darwin GOARCH=amd64 go build -o bin/macos-client/naisdevice-helper --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/helper

# Run by GitHub actions on linux
windows-client:
	mkdir -p ./bin/windows-client
	
	go run github.com/akavel/rsrc -arch amd64 -manifest ./packaging/windows/admin_manifest.xml -ico assets/nais-logo-blue.ico -o ./cmd/helper/main_windows.syso
	go run github.com/akavel/rsrc -ico assets/nais-logo-blue.ico -o ./cmd/device-agent/main_windows.syso
	GOOS=windows GOARCH=amd64 go build -o bin/windows-client/naisdevice-systray.exe --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS) -H=windowsgui" ./cmd/systray
	./packaging/windows/sign-exe bin/windows-client/naisdevice-systray.exe ./packaging/windows/naisdevice.crt ./packaging/windows/naisdevice.key
	GOOS=windows GOARCH=amd64 go build -o bin/windows-client/naisdevice-agent.exe --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS) -H=windowsgui" ./cmd/device-agent
	./packaging/windows/sign-exe bin/windows-client/naisdevice-agent.exe ./packaging/windows/naisdevice.crt ./packaging/windows/naisdevice.key
	GOOS=windows GOARCH=amd64 go build -o bin/windows-client/naisdevice-helper.exe --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/helper
	./packaging/windows/sign-exe bin/windows-client/naisdevice-helper.exe ./packaging/windows/naisdevice.crt ./packaging/windows/naisdevice.key

local:
	mkdir -p ./bin/local
	go build -o bin/local/apiserver --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/apiserver
	go build -o bin/local/gateway-agent --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/gateway-agent
	go build -o bin/local/prometheus-agent --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/prometheus-agent
	go build -o bin/local/bootstrap-api --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/bootstrap-api
	go build -o bin/local/controlplane-cli --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/controlplane-cli
	go build -o bin/local/naisdevice-agent --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/device-agent
	go build -o bin/local/naisdevice-systray --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/systray
	go build -o bin/local/naisdevice-helper --tags "$(GOTAGS)" -ldflags "-s $(LDFLAGS)" ./cmd/helper

linux-icon: packaging/linux/icons/*/apps/naisdevice.png
packaging/linux/icons/*/apps/naisdevice.png: assets/nais-logo-blue.png
	for size in 16x16 32x32 64x64 128x128 256x256 512x512 ; do \
		mkdir -p packaging/linux/icons/$$size/apps/ ; \
		convert assets/nais-logo-blue.png -scale $$size^ -background none -gravity center -extent $$size packaging/linux/icons/$$size/apps/naisdevice.png ; \
  	done

windows-icon: packaging/windows/naisdevice.ico
packaging/windows/naisdevice.ico: assets/nais-logo-blue.png
	mkdir -p packaging/windows/assets/
	convert assets/nais-logo-blue.png -background transparent -define icon:auto-resize=48,64,96,128,256 -gravity center packaging/windows/assets/naisdevice.ico


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
	curl -L https://git.zx2c4.com/wireguard-tools/snapshot/wireguard-tools-1.0.20210914.tar.xz | tar xJ
	cd wireguard-tools-*/src && make && cp wg ../../bin/macos-client/
	rm -rf ./wireguard-tools-*

wireguard-go: bin/macos-client/wireguard-go
bin/macos-client/wireguard-go:
	mkdir -p bin/macos-client
	curl -L https://git.zx2c4.com/wireguard-go/snapshot/wireguard-go-0.0.20230223.tar.xz | tar xJ
	cd wireguard-go-*/ && make && cp wireguard-go ../bin/macos-client/
	rm -rf ./wireguard-go-*

gon:
	curl -LO https://github.com/mitchellh/gon/releases/download/v0.2.5/gon_macos.zip
	unzip gon_macos.zip
	chmod +x ./gon

app: wg wireguard-go macos-icon macos-client gon
	rm -rf naisdevice.app
	mkdir -p naisdevice.app/Contents/{MacOS,Resources}
	cp bin/macos-client/* naisdevice.app/Contents/MacOS
	cp packaging/macos/jq-osx-amd64 naisdevice.app/Contents/MacOS/jq
	cp assets/naisdevice.icns naisdevice.app/Contents/Resources
	sed 's/VERSIONSTRING/${VERSION}/' packaging/macos/Info.plist.tpl > naisdevice.app/Contents/Info.plist
	#	./gon --log-level=debug packaging/macos/gon-app.json
	codesign -s "Developer ID Application: Torbjorn Hallenberg (T7D7Y5484F)" -f -v --timestamp --deep --options runtime naisdevice.app/Contents/MacOS/*

test:
	@go test $(shell go list ./... | grep -v systray) -count=1

# Run by GitHub actions on macos
pkg: app gon
	rm -f ./naisdevice*.pkg
	rm -rf ./pkgtemp
	mkdir -p ./pkgtemp/{scripts,pkgroot/Applications}
	cp -r ./naisdevice.app ./pkgtemp/pkgroot/Applications/
	cp ./packaging/macos/postinstall ./pkgtemp/scripts/postinstall
	cp ./packaging/macos/preinstall ./pkgtemp/scripts/preinstall
	pkgbuild --root ./pkgtemp/pkgroot --identifier ${PKGID} --scripts ./pkgtemp/scripts --version ${VERSION} --ownership recommended ./component.pkg
	productbuild --identifier ${PKGID}.${VERSION} --package ./component.pkg ./unsigned.pkg
	productsign --sign "Developer ID Installer: Torbjorn Hallenberg" unsigned.pkg naisdevice.pkg
	rm -f ./component.pkg ./unsigned.pkg
	rm -rf ./pkgtemp ./naisdevice.app
	#./gon --log-level=debug packaging/macos/gon-pkg.json

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
	rm -rf ./packaging/linux/icons
	rm -rf ./packaging/windows/assets

mocks:
	go run github.com/vektra/mockery/v2 --case underscore --all --dir pkg/ --inpackage --recursive
	rm ./pkg/apiserver/auth/mock_authenticator.go
	rm ./pkg/auth/mock_token_validator.go

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
	go run github.com/kyleconroy/sqlc/cmd/sqlc generate
	go run mvdan.cc/gofumpt -w ./pkg/apiserver/sqlc/

fmt:
	go run mvdan.cc/gofumpt -w ./

check:
	go run honnef.co/go/tools/cmd/staticcheck ./...
	go run golang.org/x/vuln/cmd/govulncheck ./...
