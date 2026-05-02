.PHONY: help setup test-server run deploy-worker install

help:
	@echo "Usage:"
	@echo "  make setup             - Interactive setup for Worker and CLI"
	@echo "  make test-server       - Start a local HTTP server for testing on port 8000"
	@echo "  make run               - Start the l2c-cli client using config.json"
	@echo "  make deploy-worker     - Deploy the Cloudflare Worker"

test-server:
	@go run l2c-cli/test-server/main.go

run:
	@go run l2c-cli/main.go run

deploy-worker:
	@cd l2c-cli/cmd/worker_src && pnpm run deploy

setup:
	@go run l2c-cli/main.go setup
