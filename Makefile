BINARY     := k8sdash
CMD        := ./cmd/k8sdash
OUT_DIR    := bin
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS    := -s -w -X main.version=$(VERSION)

.PHONY: build run clean release-check release-snapshot

## Build for the current OS/arch, into bin/k8sdash
build:
	go build -ldflags "$(LDFLAGS)" -o $(OUT_DIR)/$(BINARY) $(CMD)

## Build and run the current-platform binary
run: build
	./$(OUT_DIR)/$(BINARY)

clean:
	rm -rf $(OUT_DIR) dist

## Validate .goreleaser.yaml (`brew install goreleaser` if you don't have it)
release-check:
	goreleaser check

## Build every release target + archive locally into dist/, exactly as the
## release workflow will for a real tag, without needing one or touching
## GitHub — use this to sanity-check a change to .goreleaser.yaml.
release-snapshot:
	goreleaser release --snapshot --clean --skip=publish
