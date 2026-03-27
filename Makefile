.PHONY: build-linux clean format

BINARY_NAME=nptx_core

build-linux:
	@echo "Building binary for Linux amd64..."
	@mkdir -p build
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o build/$(BINARY_NAME)_linux_amd64 ./cmd/nptx/main.go
	@echo "Done! Binary is in build/$(BINARY_NAME)_linux_amd64"

format:
	@go fmt ./...

clean:
	@echo "Cleaning up..."
	@rm -rf build/
	@echo "Cleaned."
