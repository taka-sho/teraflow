# Contributing

## Setup

```bash
# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/rhysd/actionlint/cmd/actionlint@latest

# Install git hooks (runs golangci-lint + actionlint on commit)
make setup-hooks
```

## Validation

```bash
make test          # Run tests
make lint          # Run golangci-lint
make validate-yaml # Validate GitHub Actions workflows with actionlint
```

## Workflow Templates

GitHub Actions workflow templates are in `internal/actions/templates/`.
Run `make validate-yaml` and `go test ./internal/actions/...` before modifying templates.
