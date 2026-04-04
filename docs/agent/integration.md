---
codd:
  node_id: "docs:agent-integration"
  title: "AI Agent Integration Guide"
  depends_on:
    - id: "design:cli-interface"
      relation: implements
    - id: "docs:concepts"
      relation: extends
---

# AI Agent Integration Guide

## 1. Use `--format json` as the default interface

For agent automation, prefer machine-readable output for every command.

### JSON output examples

#### Project status
```bash
teraflow status --format json
```

```json
{"stage":"initial_development","phase":"requirements"}
```

#### Phase list
```bash
teraflow phase list --format json
```

```json
{"current":"requirements","phases":["requirements","basic_design","detailed_design","implementation","testing","integration_test"]}
```

#### Stage list
```bash
teraflow stage list --format json
```

```json
{"current":"initial_development","stages":["initial_development","release","operation","maintenance","continuous_improvement","retirement"]}
```

#### Scan
```bash
teraflow scan --format json
```

```json
{"status":"ok","checks":[{"file":".github/teraflow.yml","exists":true},{"file":".github/project-state.yml","exists":true},{"file":"docs/","exists":true},{"file":".teraflow/rework-log.yml","exists":false,"optional":true}]}
```

#### Rework list
```bash
teraflow rework list --format json
```

```json
{"reworks":[{"id":"rw-001","group":"group-auth","target_phase":"basic_design","status":"open"}]}
```

#### Incident list (when incident command is enabled)
```bash
teraflow incident list --format json
```

```json
{"incidents":[{"id":"inc-001","title":"DB timeout","severity":"major","status":"open"}]}
```

### `jq` extraction examples

```bash
teraflow status --format json | jq '.stage'
teraflow rework list --format json | jq '[.reworks[] | select(.status=="open")]'
```

## 2. Error handling

### Exit code policy
- `0`: success
- `1`: general error (configuration, validation, initialization)
- `10-19`: business logic error (gate/flow violations)
- `20-29`: external integration error (GitHub/API)

### Detecting `E0001` (not initialized)
`E0001` means the project has not been initialized (or required files are missing).

```bash
if ! teraflow scan --format json >/tmp/scan.json 2>/tmp/scan.err; then
  if grep -q "E0001" /tmp/scan.err; then
    echo "project not initialized"
    exit 1
  fi
fi
```

### JSON error payload format

```json
{"status":"error","errors":[{"code":"E0001","message":"Not a teraflow project. Run `teraflow init` first."}]}
```

## 3. Required state check flow

Run this sequence before any state-changing command:

1. `teraflow scan --format json`
2. `teraflow status --format json`
3. execute operation command

When a command supports dry-run in your environment, run dry-run before applying writes.

## 4. Best practices

- Atomic operations: one command for one state transition.
- Changelog discipline: append changelog after each major phase/stage transition.
- Rework-first rollback: when regression occurs, record rework with `teraflow rework create` before corrective action.
- Deterministic parsing: branch only on JSON fields and exit codes, not on text output.
