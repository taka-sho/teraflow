# Skills Guide

## Overview

Skills are scenario-specific prompt definitions for AI agents.
They are loaded from `skills/*.yml` and used by `teraflow agent assign` to:

- Standardize agent behavior per task type
- Reuse curated system prompts and output style
- Automatically select suitable prompts by agent type and trigger context

## Skill File Format

Each skill file is YAML and typically includes:

- `name`: unique skill name (for `--skill` and lookup)
- `version`: skill schema/content version
- `description`: concise description of purpose
- `trigger`: selection hints
- `prompts`: system/confirm prompts used for generation
- `context`: additional context include rules
- `output`: output header/footer and mode-specific formatting
- `options`: generation parameters such as `max_tokens` and `temperature`

Example:

```yaml
name: requirements
version: "1"
description: "要件議論の壁打ち・構造化・確定"
trigger:
  labels: ["requirements"]
  agent_types: ["requirements"]
prompts:
  system: |
    You are a requirements consultant.
output:
  dialogue:
    header: "## 💬 AI壁打ち"
```

## Default Skills

The repository provides these default skills:

- `req` / `requirements`: requirements dialogue, structuring, confirmation
- `design`: architecture and design proposal support
- `review`: code review support
- `bugfix`: bug analysis and fix guidance
- `change-request`: change impact analysis and request handling

Use `teraflow skill list` to check the currently available set in your workspace.

## Create a Custom Skill

1. Create `skills/<your-skill>.yml`
2. Define required metadata (`name`, `version`, `description`)
3. Add prompt blocks under `prompts` for your use case
4. Add `trigger` conditions for automatic selection
5. Run `teraflow skill validate` to verify format
6. Execute with explicit selection:
   `teraflow agent assign --type <agent-type> --skill <your-skill> ...`

## Skill CLI Commands

### List skills

```bash
teraflow skill list
```

### Show one skill

```bash
teraflow skill show --name req
```

### Validate skills

```bash
teraflow skill validate
```

For machine-readable output, add `--format json` where supported.
