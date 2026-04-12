.PHONY: setup-hooks test lint actionlint validate-yaml build

setup-hooks: ## Install git hooks from .github/hooks/
	git config core.hooksPath .github/hooks
	@echo "Git hooks installed. pre-commit lint+actionlint check enabled."

test: ## Run all tests
	GOTOOLCHAIN=auto go test ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

actionlint: ## Validate GitHub Actions workflows (repo + rendered templates)
	@command -v actionlint >/dev/null 2>&1 || (echo "actionlint not found; installing..." && GOBIN="$$(go env GOPATH)/bin" go install github.com/rhysd/actionlint/cmd/actionlint@latest)
	TMP_REPO="$$(mktemp -d)"; \
	mkdir -p "$$TMP_REPO/.github"; \
	cp .github/teraflow.yml "$$TMP_REPO/.github/teraflow.yml"; \
	GOTOOLCHAIN=auto go run . setup actions --config "$$TMP_REPO/.github/teraflow.yml" --force; \
	actionlint -ignore 'is potentially untrusted' -ignore 'shellcheck reported issue' .github/workflows/*.yml "$$TMP_REPO"/.github/workflows/*.yml

validate-yaml: ## Backward-compatible alias for actionlint
	$(MAKE) actionlint

build: ## Build the binary
	GOTOOLCHAIN=auto go build -o bin/teraflow .
