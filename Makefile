.PHONY: help test-server run deploy-worker install

help:
	@echo "Usage:"
	@echo "  make test-server       - Start a local HTTP server for testing on port 8000"
	@echo "  make run               - Start the l2c-cli client using config.json"
	@echo "  make deploy-worker     - Deploy the Cloudflare Worker"

test-server:
	@go run l2c-cli/test-server/main.go

run:
	@if [ ! -f "config.json" ]; then \
		echo "Error: config.json not found. Copy config.json.example to config.json first."; \
		exit 1; \
	fi
	@cd l2c-cli && go run main.go -config ../config.json

deploy-worker:
	@cd worker && pnpm run deploy
