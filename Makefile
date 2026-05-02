.PHONY: help setup test-server run deploy-worker install

help:
	@echo "Usage:"
	@echo "  make setup             - Interactive setup for Worker and CLI"
	@echo "  make test-server       - Start a local HTTP server for testing on port 8000"
	@echo "  make run               - Start the l2c-cli client using config.json"
	@echo "  make deploy-worker     - Deploy the Cloudflare Worker"
	@echo "  make build-all         - Build binaries for all supported platforms"

test-server:
	@go run l2c-cli/test-server/main.go

run:
	@go run l2c-cli/main.go run

deploy-worker:
	@cd l2c-cli/cmd/worker_src && pnpm run deploy

setup:
	@go run l2c-cli/main.go setup

build-all:
	@echo "Building binaries for all platforms..."
	@mkdir -p bin
	@cd l2c-cli && GOOS=linux GOARCH=amd64 go build -o ../bin/l2c-linux-amd64 main.go
	@cd l2c-cli && GOOS=darwin GOARCH=amd64 go build -o ../bin/l2c-darwin-amd64 main.go
	@cd l2c-cli && GOOS=darwin GOARCH=arm64 go build -o ../bin/l2c-darwin-arm64 main.go
	@cd l2c-cli && GOOS=windows GOARCH=amd64 go build -o ../bin/l2c-windows-amd64.exe main.go
	@echo "Binaries available in bin/ directory"
