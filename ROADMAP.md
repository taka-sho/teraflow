---
codd:
  node_id: "docs:roadmap"
  title: "teraflow Development Roadmap"
  depends_on:
    - id: "req:teraflow-overview"
      relation: implements
---

# teraflow Development Roadmap

## Released

### v0.1.x — Foundation

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

### v0.2.x — Governance

- Per-agent-type provider assignments
- Audit logging (audit list)
- RBAC + Gate + Constraint engine
- SLCP-JCF process tracking
- CI cost optimization

### v0.3.x — Provider Extensibility

- Claude Code provider (local AI development)
- Custom provider support (BYOM - Bring Your Own Model)
- Provider fallback chains

### v0.4.x — CoDD + Agent Improvements

- CoDD (Code-Driven Documentation) framework: frontmatter, scan, validate, impact
- YAML multi-layer validation
- req-agent improvements: error comments, UX, thread depth, footer
- doc generate bug fixes
- setup verify command
- Template tag pinning
- Release pipeline with warmup + smoke-test

### v0.5.0 — GraphRAG + V-Model Pipeline

- **GraphRAG integration**
  - graph status/check/export (Go)
  - Python entity extraction + knowledge graph construction
  - Query engine integration (search/impact)
- **V-Model full-phase pipeline**
  - CoDD consistency check enhancements
  - Change propagation (Green/Amber/Gray)
  - Wave-based design doc auto-generation
  - Implementation auto-generation
- **Requirements Discovery (grill-me)**
  - Design + Phase 1/2 implementation
  - Discussion-based interactive requirements gathering

### v0.5.1–v0.5.15 — Discovery Hardening (Current: v0.5.15)

- Release pipeline fixes (warmup proxy, EOF delimiters, API key OR conditions)
- req-agent discovery auto-trigger on Discussion creation
- Discovery init response improvements (body/title, skill prompt, fallback)
- E2E CI test infrastructure + actionlint integration
- Test coverage 72% → 85%
- Version pinning in teraflow.yml + `teraflow update` command
- Workflow consolidation (hooks → req-agent unified)
- Discovery response format: numbered short-list with a/b/c/d choices
- i18n design doc (go-i18n v2 + YAML)
- CoDD enforcement design doc (validate/impact/hook extensions)
- `teraflow doc index` — document index for discovery context
- Discovery conversation history compaction (structured summary)
- GraphQL replies query fix (comments + replies timeline merge)
- Confirm flow fix: CoDD PR creation + body preservation
- Skills go:embed — external repos load skills without skills/ directory
- discussion_comment list-type handling fix
- Discovery response format hardening + post-processing formatter
- **Multi-round conversation answer accumulation** (v0.5.15)
  - User answer parser (choice/free-text patterns)
  - Parsed answers applied to decision tree branches
  - Confirmed list auto-update from parsed answers
  - Prompt prohibition on re-asking confirmed items
  - Dialogue node meta extension with parsed_answers

## In Progress

### v0.6.0 — i18n + Verification + DX

- **i18n implementation** — multi-language support based on go-i18n v2 design
- **CoDD enforcement (Phase 1)** — CI gate with warn mode
- **Test result → requirements feedback loop** — auto-propagate test results to requirements/design
- **Code/document drift detection** (`teraflow drift`) — detect post-generation code changes
- **Requirements traceability matrix** — req → design → impl → test linkage visualization
- **Review support** (`teraflow review`) — PR cross-reference with design/requirements
- **Project progress visualization** (`teraflow status`) — V-Model phase completion rates
- **Release notes auto-generation** — from CoDD + commits + Discussions
- **Discovery E2E automated testing** — full-cycle validation (setup → conversation → confirm → PR check)

## Future

### v1.0+

- GitHub Enterprise support
- Multi-repository project management
- Advanced AI: autonomous mode with human-in-the-loop
- Plugin system for custom workflows
- Dashboard web UI
- GraphRAG-powered discovery (context-aware question generation)

## AI Provider Support

### Currently Supported

| Provider | Status | Models |
|----------|--------|--------|
| Anthropic (Claude) | Stable | claude-haiku-4-5-20251001, claude-sonnet-4-6, claude-opus-4-6 |
| OpenAI (GPT) | Stable | gpt-4o-mini, gpt-4o |

### Planned

| Provider | Status | Timeline |
|----------|--------|----------|
| Google Gemini | Planned | v1.0+ |
| Local LLM (Ollama) | Planned | v1.0+, offline/air-gapped support |
