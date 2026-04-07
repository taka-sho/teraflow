.PHONY: setup-hooks test lint validate-yaml build

setup-hooks: ## Install git hooks from .github/hooks/
	git config core.hooksPath .github/hooks
	@echo "Git hooks installed. pre-commit lint+actionlint check enabled."

test: ## Run all tests
	GOTOOLCHAIN=auto go test ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

validate-yaml: ## Validate GitHub Actions workflow YAML with actionlint
	actionlint .github/workflows/*.yml

build: ## Build the binary
	GOTOOLCHAIN=auto go build -o bin/teraflow .
