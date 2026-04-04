---
codd:
  node_id: "docs:agent-workflows"
  title: "AI Agent Workflow Patterns"
  depends_on:
    - id: "docs:agent-integration"
      relation: extends
---

# AI Agent Workflow Patterns

## Workflow 1: Start a phase

```bash
# 1. Check current state
teraflow status --format json
# 2. Scan project initialization and required files
teraflow scan --format json
# 3. Record phase start
teraflow changelog add feat "Start {phase_name} phase" --format json
```

## Workflow 2: Verify deliverables and complete phase

```bash
# 1. Check current phase
teraflow phase list --format json
# 2. Validate project state
teraflow scan --format json
# 3. Complete phase
teraflow phase complete --format json
# 4. Record changelog
teraflow changelog add feat "Complete {phase_name}: {summary}" --format json
# 5. Re-check state
teraflow status --format json
```

## Workflow 3: Handle rework

```bash
# 1. Record rework
teraflow rework create \
  --group <group_id> \
  --target-phase <target> \
  --reason "<reason>" \
  --format json
# 2. Analyze impact (when CoDD tooling is enabled)
# codd impact
# 3. Confirm rework list
teraflow rework list --format json
```

## Workflow 4: Incident response in operation stage

```bash
# 1. Create incident
teraflow incident create --title "<title>" --severity <level> --format json
# 2. Append changelog
teraflow changelog add fix "Incident <id>: <title>" --format json
# 3. Close incident after resolution
teraflow incident close --id <id> --format json
```

## Programmatic state branching example

```bash
STAGE=$(teraflow status --format json | jq -r '.stage')
if [ "$STAGE" = "initial_development" ]; then
  echo "In development stage"
fi
```
