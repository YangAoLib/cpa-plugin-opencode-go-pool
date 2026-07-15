VERSION ?= 0.1.0
GO_IMAGE ?= golang:1.26-bookworm
CACHE ?= /tmp/gocache-ocgp
OUT = dist/opencode-go-pool-v$(VERSION).so
DEPLOY_DIR ?= ../../plugins/linux/amd64

.PHONY: build test deploy clean

build:
	mkdir -p "$(dir $(OUT))"
	docker run --rm -v "$(CURDIR)":/src -v $(CACHE):/gocache -w /src \
		-e GOMODCACHE=/gocache/mod -e GOCACHE=/gocache/build $(GO_IMAGE) \
		sh -c 'CGO_ENABLED=1 go build -buildvcs=false -trimpath -buildmode=c-shared -ldflags "-s -w -X main.pluginVersion=$(VERSION)" -o $(OUT) . && rm -f dist/*.h'

test:
	docker run --rm -v "$(CURDIR)":/src -v $(CACHE):/gocache -w /src \
		-e GOMODCACHE=/gocache/mod -e GOCACHE=/gocache/build $(GO_IMAGE) \
		sh -c 'go vet ./... && go test ./...'

deploy: build
	mkdir -p "$(DEPLOY_DIR)"
	cp "$(OUT)" "$(DEPLOY_DIR)/"
	docker restart cli-proxy-api

clean:
	rm -rf dist
