BINARY  := ferry
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-X main.Version=$(VERSION) -s -w"
DIST    := dist

.PHONY: build test clean release release-local

build:
	go build $(LDFLAGS) -o $(BINARY) .

test:
	go test ./...

clean:
	rm -rf $(BINARY) $(DIST)

# Cross-compile all targets locally (requires Go toolchain)
release-local: clean
	mkdir -p $(DIST)
	GOOS=linux  GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-linux-amd64 .
	GOOS=linux  GOARCH=arm64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-darwin-arm64 .
	@echo "Built:"
	@ls -lh $(DIST)/

# Build all targets inside Docker (no local Go toolchain required)
release:
	docker build -f Dockerfile.release --build-arg VERSION=$(VERSION) -t ferry-builder .
	docker run --rm -v "$(PWD)/$(DIST):/out" ferry-builder
	@echo "Built:"
	@ls -lh $(DIST)/
