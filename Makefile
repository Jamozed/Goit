.PHONY: all build test help
all: help

MODULE = "github.com/Jamozed/Goit"
PROGRAM = "goit"
VERSION = "0.0.0"

build: ## Build the project
	@go build -ldflags "-X $(MODULE)/res.Version=$(VERSION)" -o ./bin/$(PROGRAM) .

test: ## Run unit tests
	@go test ./...

help: ## Display help information
	@grep -E '^[a-zA-Z_-]+:.*?##.*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## *"}; {printf "\033[36m%-6s\033[0m %s\n", $$1, $$2}'
