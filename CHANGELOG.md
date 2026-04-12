# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

## [0.5.10] - 2026-04-12

### Bug Fixes
- fix: release automation update

## [0.5.9] - 2026-04-12

### Bug Fixes
- fix(discovery): persist conversation history to state file (tree/draft/title)
- fix(discovery): enforce numbered choice format in AI responses

## [0.5.8] - 2026-04-12

### Bug Fixes
- fix: add scripts/release.sh to automate main.go version updates

## [0.5.7] - 2026-04-12

### Features
- feat(discovery): improve response format - numbered list with choices (a/b/c/d)
- feat(discovery): add fallback comment formatting with choices

### Bug Fixes
- fix: remove teraflow-hooks-discussion/push workflow generation (consolidate to req-agent)

### Documentation
- docs(design): add i18n design document for v0.6.0

## [0.5.6] - 2026-04-12

### Features
- feat(discovery): include Discussion body/title in init input for better analysis
- feat(discovery): improved init prompt with category-based questions and recommendations

### Bug Fixes
- fix(e2e): add setup templates before doctor in E2E workflow
- fix(doctor): downgrade template checks to warning in --ci mode

### Tests
- test: improve coverage from 72% to 85%

## [0.5.5] - 2026-04-12

### Features
- feat(update): add `teraflow update` command for version management
- feat(init): pin teraflow_version in teraflow.yml on init
- feat(deploy-workflows): embed pinned version from teraflow.yml

### Bug Fixes
- fix(templates): add newline after cat to fix GITHUB_OUTPUT delimiter

## [0.5.4] - 2026-04-12

### Bug Fixes
- fix(templates): accept OPENAI_API_KEY as alternative to ANTHROPIC_API_KEY
- fix(templates): use unique delimiter for GITHUB_OUTPUT to avoid EOF collision
- fix(templates): replace hardcoded @v0.5.0 with dynamic LATEST_TAG in all templates

## [0.5.3] - 2026-04-12

### Features
- feat(discovery): auto-enable discovery mode for 01-Requirements category discussions

### Bug Fixes
- fix(release): add retry + GONOSUMCHECK to Go proxy warmup
- fix(templates): add GONOSUMCHECK to all teraflow install steps
- fix(templates): add git config for discovery commit step
- fix(templates): handle GITHUB_OUTPUT EOF when agent fails


## [0.5.0] - 2026-04-12

### Features
- GraphRAG × CoDD integration (PR #128, #129, #130, #131, #132, #133, #134)
- V字モデル SLCP-JCF pipeline: validate/impact/generate/plan/implement/discovery (PR #141, #142, #143, #144, #145, #146)

### Bug Fixes
- req-agent rework and workflow robustness improvements from v0.4.x stream (PR #135, #136, #137, #138, #139, #140)

### Documentation
- V-model usage guide and release docs updates (PR #147)

### CI/CD
- Workflow validation and release pipeline improvements (PR #148, #149)

### Included PRs (#128-#149)
- #128 #129 #130 #131 #132 #133 #134 #135 #136 #137 #138 #139 #140 #141 #142 #143 #144 #145 #146 #147 #148 #149

## [0.4.1] - 2026-04-07

### Fixed
- Remove excess blank lines from 16 action templates (fixes GitHub Actions YAML parse errors)
- Prevent `github-actions[bot]` loop in req-agent workflow
- Fix `updateDiscussionComment` mutation in ACK update step

## [0.4.0] - 2026-04-06

### Added
- Error code system (TF-CCNN format) for structured error handling
- `internal/errors` package with AppError, catalog, and code constants
- Error catalog in `docs/errors/` (18 error definitions)
- `doctor error` subcommand to look up error code details
- CI lint job to verify error catalog sync with generated code
- AppError migration for gate/rbac/agent commands with proper exit codes
- Integration tests for TF-RB01/TF-AI01 AppError handling and JSON error output
- Guide: `docs/guide/error-codes.md`
- Auto CoDD doc generate + PR on requirement confirmation (req-agent)
- hooks section in teraflow.yml template + push workflow

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
