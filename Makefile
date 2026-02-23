.PHONY: build test lint clean install release-dry release deps fmt build-all mocks clean-mocks

# Variables
BINARY_NAME=claudectl
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Build for current platform
build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME) .

# Build for all platforms
build-all:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)_darwin_amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)_darwin_arm64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)_linux_amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)_linux_arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)_windows_amd64.exe .
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)_windows_arm64.exe .

# Run tests
test:
	go test -v -race -cover ./...

# Run linter
lint:
	go vet ./...
	@which golangci-lint > /dev/null 2>&1 || echo "golangci-lint not installed, skipping"
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run || true

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf dist/
	go clean

# Install to GOPATH/bin
install: build
	mkdir -p $(GOPATH)/bin
	cp bin/$(BINARY_NAME) $(GOPATH)/bin/

# Release using goreleaser (dry-run)
release-dry:
	goreleaser release --snapshot --clean

# Release using goreleaser
release:
	goreleaser release --clean

# Download dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Generate mocks using mockery
mocks:
	@which mockery > /dev/null 2>&1 || (echo "mockery not installed. Run: go install github.com/vektra/mockery/v2@latest" && exit 1)
	mockery

# Clean generated mocks
clean-mocks:
	find internal -type d -name "mocks" -exec rm -rf {} + 2>/dev/null || true
