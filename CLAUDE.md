# teraflow - AI Agent Guide

## What is teraflow?
`teraflow` is a CLI for lifecycle-aware software project management. It standardizes project state transitions (stage/phase), change logging, and operational records so agents can execute and audit workflows consistently.

## Quick Reference

### Project State Check
```bash
teraflow status --format json
teraflow phase list --format json
teraflow stage list --format json
```

### Workflow Operations
```bash
# Complete current phase
teraflow phase complete --format json

# Record a rework
teraflow rework create --group <id> --target-phase <phase> --reason "<reason>" --format json

# Append changelog
teraflow changelog add feat "<message>" --format json
```

## Rules for AI Agents
1. Always use `--format json` for structured output parsing.
2. Check `teraflow status --format json` before any state-modifying operation.
3. Verify `teraflow scan --format json` before workflow operations to confirm project initialization.
4. Record phase/stage transitions with `teraflow changelog add`.
5. Treat non-zero exit code as failure and parse JSON error payload when available.

## Key Files
- `.github/teraflow.yml` - project configuration
- `.github/project-state.yml` - current stage/phase state
- `.teraflow/rework-log.yml` - rework history
- `.teraflow/incident-log.yml` - incident history

## Documentation
- [Concepts](docs/guide/concepts.md)
- [Agent Integration](docs/agent/integration.md)
- [Agent Workflows](docs/agent/workflows.md)
