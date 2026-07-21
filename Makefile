VERSION ?= 0.2.0
GO ?= go
PLUGIN_ID = opencode-go-pool
OUT = dist/opencode-go-pool-v$(VERSION).so
OUT_WINDOWS = dist/opencode-go-pool-v$(VERSION).dll
ARCHIVE = dist/$(PLUGIN_ID)_$(VERSION)_linux_amd64.zip
ARCHIVE_WINDOWS = dist/$(PLUGIN_ID)_$(VERSION)_windows_amd64.zip
ARCHIVE_CHECKSUM = $(ARCHIVE).sha256
ARCHIVE_CHECKSUM_WINDOWS = $(ARCHIVE_WINDOWS).sha256
DEPLOY_DIR ?= ../../plugins/linux/amd64

.PHONY: build build-windows package package-windows test deploy clean

build:
	test "$$($(GO) env GOOS)" = "linux"
	test "$$($(GO) env GOARCH)" = "amd64"
	mkdir -p "$(dir $(OUT))"
	CGO_ENABLED=1 $(GO) build -buildvcs=false -trimpath -buildmode=c-shared \
		-ldflags "-s -w -X main.pluginVersion=$(VERSION)" -o "$(OUT)" ./src
	rm -f dist/*.h

package: build
	$(GO) run -buildvcs=false ./.github/scripts/package-release.go \
			-library "$(OUT)" -entry "$(PLUGIN_ID).so" \
			-archive "$(ARCHIVE)" -checksum "$(ARCHIVE_CHECKSUM)"
	cp "$(ARCHIVE_CHECKSUM)" dist/checksums.txt

build-windows:
	mkdir -p "$(dir $(OUT_WINDOWS))"
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 $(GO) build -buildvcs=false -trimpath -buildmode=c-shared \
		-ldflags "-s -w -X main.pluginVersion=$(VERSION)" -o "$(OUT_WINDOWS)" ./src
	-rm -f dist/*.h dist/*.def

package-windows: build-windows
	$(GO) run -buildvcs=false ./.github/scripts/package-release.go \
			-library "$(OUT_WINDOWS)" -entry "$(PLUGIN_ID).dll" \
			-archive "$(ARCHIVE_WINDOWS)" -checksum "$(ARCHIVE_CHECKSUM_WINDOWS)"
	cp "$(ARCHIVE_CHECKSUM_WINDOWS)" dist/checksums-windows.txt

test:
	$(GO) vet ./...
	$(GO) test ./...

deploy: build
	mkdir -p "$(DEPLOY_DIR)"
	cp "$(OUT)" "$(DEPLOY_DIR)/"
	docker restart cli-proxy-api

clean:
	rm -rf dist
