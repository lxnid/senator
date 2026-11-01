# Senator CLI

.PHONY: build run test clean install

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT_SHA ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS = -ldflags "-X github.com/lxnid/senator/cmd.Version=$(VERSION) -X github.com/lxnid/senator/cmd.CommitSHA=$(COMMIT_SHA) -X github.com/lxnid/senator/cmd.BuildDate=$(BUILD_DATE)"

# Default target
all: build

# Build the binary
build:
	@echo "🔨 Building senator..."
	go build $(LDFLAGS) -o senator .
	@echo "✅ Build complete: ./senator"

# Run the application
run: build
	./senator

# Run tests
test:
	@echo "🧪 Running tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	rm -f senator
	rm -f senator-*
	@echo "✅ Clean complete"

# Install to GOPATH/bin
install:
	@echo "📦 Installing senator..."
	go install $(LDFLAGS) .
	@echo "✅ Installed to $(shell go env GOPATH)/bin/senator"

# Tidy dependencies
tidy:
	@echo "📦 Tidying dependencies..."
	go mod tidy
	@echo "✅ Dependencies tidied"

# Format code
fmt:
	@echo "🎨 Formatting code..."
	go fmt ./...
	@echo "✅ Code formatted"

# Lint code
lint:
	@echo "🔍 Linting code..."
	golangci-lint run
	@echo "✅ Lint complete"

# Cross-platform builds
build-all:
	@echo "🔨 Building for all platforms..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o senator-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o senator-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o senator-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o senator-windows-amd64.exe .
	@echo "✅ All builds complete"

# Help
help:
	@echo "Senator CLI Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build      Build the senator binary"
	@echo "  make run        Build and run senator"
	@echo "  make test       Run tests"
	@echo "  make clean      Remove build artifacts"
	@echo "  make install    Install to GOPATH/bin"
	@echo "  make tidy       Tidy go.mod dependencies"
	@echo "  make fmt        Format code"
	@echo "  make lint       Run linter"
	@echo "  make build-all  Build for all platforms"
	@echo "  make help       Show this help"
