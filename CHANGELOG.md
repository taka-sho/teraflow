# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added
- Error code system (TF-CCNN format) for structured error handling
- `internal/errors` package with AppError, catalog, and code constants
- Error catalog in `docs/errors/` (18 error definitions)
- `doctor error` subcommand to look up error code details
- CI lint job to verify error catalog sync with generated code
- AppError migration for gate/rbac/agent commands with proper exit codes
- Integration tests for TF-RB01/TF-AI01 AppError handling and JSON error output
- Guide: `docs/guide/error-codes.md`

## [0.3.0] - 2026-04-06

### Added
- `teraflow doc generate --discussion <N>`: Generate CoDD documents from GitHub Discussions
- `teraflow doc list`: List all CoDD documents with filtering
- `teraflow trace <node_id>`: Trace document dependency graph (upstream/downstream)
- `on_confirmation` hook `generate` action: Auto-generate CoDD on discussion confirmation
- New packages: `internal/doc/`, `internal/trace/`
- `cmd/doc.go`, `cmd/trace.go` CLI commands

### Improved
- Index builder integration with doc generation pipeline
- Discussion-to-document traceability via `source` frontmatter field
