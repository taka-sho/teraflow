---
codd:
  node_id: "docs:ai-providers"
  title: "AI Provider Configuration Guide"
  depends_on:
    - id: "docs:agent-integration"
      relation: extends
---

# AI Provider Configuration Guide

## 1. Overview

teraflow supports multiple AI providers for different agent tasks. You can choose one global default and optionally override provider/model per agent type.

Supported providers:

- `anthropic` (stable)
- `openai` (stable)
- `claude-code` (experimental)
- `custom` (experimental)

Fallback chain:

1. `assignments.<type>`
2. `ai.default_provider`
3. `anthropic` (system fallback)

## 2. Quick Setup

### 2.1 Anthropic (Claude)

- Get API key: <https://console.anthropic.com/>
- Set GitHub Secret: `ANTHROPIC_API_KEY`
- Recommended default model: `claude-haiku-4-5-20251001`

### 2.2 OpenAI (GPT)

- Get API key: <https://platform.openai.com/api-keys>
- Set GitHub Secret: `OPENAI_API_KEY`
- Recommended default model: `gpt-4o-mini`

### 2.3 Claude Code (Experimental)

- Prerequisite: Claude CLI installed and authenticated
- No API key required in GitHub Actions for local CLI execution

### 2.4 Custom Provider (Experimental)

- Use an external command path that accepts task input and returns model output
- Ensure command exit code and output format are stable for automation

## 3. Configuration

### 3.1 Global default (`ai.default_provider`)

Set one default provider in `.github/teraflow.yml`:

```yaml
ai:
  default_provider: anthropic
```

### 3.2 Per-type assignments

Override provider/model by agent type:

```yaml
assignments:
  requirements:
    provider: anthropic
    model: claude-haiku-4-5-20251001
  implement:
    provider: openai
    model: gpt-4o
  review:
    provider: anthropic
    model: claude-sonnet-4-6
```

### 3.3 Fallback chain

If a type assignment is missing, teraflow uses `ai.default_provider`. If that is also missing, it falls back to `anthropic`.

### 3.4 Model options

- Anthropic examples: `claude-haiku-4-5-20251001`, `claude-sonnet-4-6`
- OpenAI examples: `gpt-4o`, `gpt-4o-mini`
- Choose lower-cost models for routine tasks, higher-quality models for review and incident analysis

## 4. GitHub Actions Integration

Set required secrets in your repository:

1. Open GitHub repository settings
2. Go to `Secrets and variables` > `Actions`
3. Add:
   - `ANTHROPIC_API_KEY` (when using `anthropic`)
   - `OPENAI_API_KEY` (when using `openai`)

During workflow execution, teraflow reads `.github/teraflow.yml` and selects provider/model according to assignment and fallback rules.

## 5. Troubleshooting

Run:

```bash
teraflow doctor --check-ai
```

Common issues:

- Missing secret: add provider-specific API key in GitHub Actions secrets
- Provider mismatch: verify `assignments.<type>.provider` values
- Unsupported model: switch to a known model name for the selected provider
- CLI-based provider error (`claude-code`): verify local CLI auth/session status

## 6. Provider Comparison

| Provider | Cost | Speed | Quality | Best For |
|----------|------|-------|---------|----------|
| Anthropic | Medium | Fast | High | Requirements, review, incident analysis |
| OpenAI | Medium | Fast | High | Implementation, CI fix, general tasks |
| Claude Code | Low (API-less local) | Medium | Medium-High | Local development workflows |
| Custom | Depends | Depends | Depends | Specialized environments and internal models |
