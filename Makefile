.PHONY: setup-hooks test lint build

setup-hooks: ## Install git hooks from .github/hooks/
	git config core.hooksPath .github/hooks
	@echo "Git hooks installed. pre-commit lint check enabled."

test: ## Run all tests
	GOTOOLCHAIN=auto go test ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

build: ## Build the binary
	GOTOOLCHAIN=auto go build -o bin/teraflow .
