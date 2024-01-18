.PHONY: all build image test help
all: help

MODULE = "$(shell go list -m)"
PROGRAM = "goit"
VERSION ?= "dev"

build: ## Build the project
	@go build -ldflags "-X $(MODULE)/res.Version=$(VERSION)" -o ./bin/$(PROGRAM) ./src

image: ## Build the project image
	@docker build -t $(PROGRAM):$(VERSION) --build-arg version=$(VERSION) .

test: ## Run unit tests
	@go test ./...

help: ## Display help information
	@grep -E '^[a-zA-Z_-]+:.*?##.*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## *"}; {printf "\033[36m%-6s\033[0m %s\n", $$1, $$2}'
