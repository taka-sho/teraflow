---
codd:
  node_id: "docs:roadmap"
  title: "teraflow Development Roadmap"
  depends_on:
    - id: "req:teraflow-overview"
      relation: implements
---

# teraflow Development Roadmap

## Current Features (v0.1.x)
- Project lifecycle management (stage/phase)
- Requirements traceability (scan)
- AI Agent framework (7 agent types)
- AI Providers: Anthropic (stable), OpenAI (stable)
- GitHub templates (issue/discussion)
- Health check (doctor)
- Configuration management (config show/set)
- Rework tracking
- Incident management
- Changelog generation

## In Progress (v0.2.x)
- Per-agent-type provider assignments
- Audit logging (audit list)
- RBAC + Gate + Constraint engine
- SLCP-JCF process tracking
- CI cost optimization

## Near-term (v0.3.x)
- Claude Code provider (local AI development)
- Custom provider support (BYOM - Bring Your Own Model)
- Provider fallback chains
- GitHub Projects board automation

## Future (v1.0+)
- GitHub Enterprise support
- Multi-repository project management
- Advanced AI: autonomous mode with human-in-the-loop
- Plugin system for custom workflows
- Dashboard web UI

## AI Provider Roadmap

### Currently Supported

| Provider | Status | Models |
|----------|--------|--------|
| Anthropic (Claude) | Stable | claude-haiku-4-5-20251001, claude-sonnet-4-6, claude-opus-4-6 |
| OpenAI (GPT) | Stable | gpt-4o-mini, gpt-4o |

### Planned

| Provider | Status | Timeline |
|----------|--------|----------|
| Claude Code | Experimental | v0.3.x |
| Custom | Experimental | v0.3.x |

### Future Consideration

| Provider | Status | Notes |
|----------|--------|-------|
| Google Gemini | Planned | v1.0+ |
| Local LLM (Ollama) | Planned | v1.0+, offline/air-gapped support |
