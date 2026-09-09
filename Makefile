BINARY     := k8sdash
CMD        := ./cmd/k8sdash
OUT_DIR    := bin
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS    := -s -w -X main.version=$(VERSION)

.PHONY: build build-all \
	build-windows-amd64 build-darwin-arm64 build-linux-arm64 \
	clean run

## Build for the current OS/arch, into bin/k8sdash
build:
	go build -ldflags "$(LDFLAGS)" -o $(OUT_DIR)/$(BINARY) $(CMD)

## Build all release targets: windows/amd64, darwin/arm64, linux/arm64
build-all: build-windows-amd64 build-darwin-arm64 build-linux-arm64

## Windows, x86-64
build-windows-amd64:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" \
		-o $(OUT_DIR)/$(BINARY)-windows-amd64.exe $(CMD)

## macOS, Apple Silicon (arm64)
build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" \
		-o $(OUT_DIR)/$(BINARY)-darwin-arm64 $(CMD)

## Linux, arm64
build-linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" \
		-o $(OUT_DIR)/$(BINARY)-linux-arm64 $(CMD)

## Build and run the current-platform binary
run: build
	./$(OUT_DIR)/$(BINARY)

clean:
	rm -rf $(OUT_DIR)
