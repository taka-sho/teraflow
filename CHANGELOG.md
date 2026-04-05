# Changelog

All notable changes to this project will be documented in this file.

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
