VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X github.com/grace/tarnished/internal/version.Version=$(VERSION) \
           -X github.com/grace/tarnished/internal/version.Commit=$(COMMIT) \
           -X github.com/grace/tarnished/internal/version.BuildDate=$(BUILD_DATE)

.PHONY: build build-all clean test tidy

build:
	go build -ldflags "$(LDFLAGS)" -o bin/tarnished ./cmd/tarnished

build-all:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/tarnished-linux-amd64 ./cmd/tarnished
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/tarnished-linux-arm64 ./cmd/tarnished
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/tarnished-windows-amd64.exe ./cmd/tarnished
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/tarnished-darwin-arm64 ./cmd/tarnished

test:
	go test -v ./...

tidy:
	go mod tidy

clean:
	rm -rf bin/
