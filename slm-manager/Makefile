.PHONY: all build build-windows build-darwin build-linux frontend clean install-deps dev

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)

all: frontend build

install-deps:
	@echo "Installing Go dependencies..."
	go mod tidy
	@echo "Installing frontend dependencies..."
	cd frontend && npm install

frontend:
	@echo "Building frontend..."
	cd frontend && npm run build

build: frontend
	@echo "Building for current platform..."
	go build -tags desktop -ldflags "$(LDFLAGS)" -o slm-manager .

build-windows: frontend
	@echo "Building for Windows (amd64)..."
	GOOS=windows GOARCH=amd64 go build -tags desktop -ldflags "$(LDFLAGS) -H=windowsgui" -o slm-manager.exe .
	@echo "Copying slm-server.exe..."
	cp ../slm-server/slm-server.exe ./

build-darwin-arm64: frontend
	@echo "Building for macOS (arm64)..."
	GOOS=darwin GOARCH=arm64 go build -tags desktop -ldflags "$(LDFLAGS)" -o slm-manager-darwin-arm64 .

build-darwin-amd64: frontend
	@echo "Building for macOS (amd64)..."
	GOOS=darwin GOARCH=amd64 go build -tags desktop -ldflags "$(LDFLAGS)" -o slm-manager-darwin-amd64 .

build-linux: frontend
	@echo "Building for Linux (amd64)..."
	GOOS=linux GOARCH=amd64 go build -tags desktop -ldflags "$(LDFLAGS)" -o slm-manager-linux-amd64 .

build-all: build-windows build-darwin-arm64 build-darwin-amd64 build-linux

dev:
	@echo "Starting development mode..."
	cd frontend && npm run dev &
	go run -tags desktop .

clean:
	rm -f slm-manager slm-manager.exe slm-manager-*
	rm -rf frontend/dist

# Package for Windows distribution
package-windows: build-windows
	@echo "Packaging for Windows..."
	mkdir -p dist/windows
	cp slm-manager.exe dist/windows/
	cp slm-server.exe dist/windows/
	cp scripts/install-windows.ps1 dist/windows/ 2>/dev/null || true
	@echo "Package ready at dist/windows/"

# Build slm-server first, then manager
full-build: 
	@echo "Building slm-server..."
	cd ../slm-server && go build -ldflags "-s -w" -o slm-server .
	cd ../slm-server && GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o slm-server.exe .
	@echo "Building slm-manager..."
	$(MAKE) build-windows

