# teraflow

A lifecycle-aware CLI for managing software projects with stage/phase flow, traceability, and operational governance.

ステージ・フェーズ管理、トレーサビリティ、運用統制を一体化して扱うプロジェクト管理CLIです。

## Concept / コンセプト

- **Structured delivery**: Manage project progress with explicit stage and phase commands.
- **Traceability-first**: Connect requirements, design, implementation, and operations through documented flows.
- **Automation-ready**: Use machine-friendly command patterns for CI/CD and team operations.

- **構造化された開発進行**: ステージ/フェーズ単位で進捗を明示的に管理。
- **トレーサビリティ重視**: 要求から実装・運用までのつながりを追跡可能に。
- **自動化しやすいCLI**: CI/CDやチーム運用に組み込みやすい設計。

## Installation / インストール

```bash
go install github.com/taka-sho/teraflow@latest
```

## Quick Start / クイックスタート

```bash
cd your-project
teraflow init --name "my-project" --non-interactive
teraflow status
```

## Commands / コマンド一覧

Implemented/planned command groups in cmd_088 scope:

- `init`
- `status`
- `stage` (`list`, `status`, `advance`)
- `phase` (`list`, `start`, `complete`)
- `scan`
- `teraflow doctor` — プロジェクト健全性チェック（環境・設定・整合性）
- `rework` (`create`, `list`)
- `config` (`show`, `set`)
- `incident`
- `changelog`
- `label`
- `discussion`
- `schedule`
- `dashboard`

## Documentation / ドキュメント

- `docs/getting-started.md` — first-time setup and basic flow
- `docs/commands/*.md` — per-command reference
- `docs/design/cli-interface.md` — command interface design baseline

## Documentation

- [Getting Started](docs/getting-started.md)
- [AI Agent Guide](CLAUDE.md) — AI Agentからの利用ガイド
- [Agent Integration](docs/agent/integration.md)
- [Agent Workflows](docs/agent/workflows.md)
- [Concepts](docs/guide/concepts.md) — コア概念・用語集
- [Lifecycle Guide](docs/guide/lifecycle.md) — ライフサイクル詳細
- [Use Cases](docs/guide/use-cases.md) — ユースケース集
- [Command Reference](docs/commands/) — 全コマンドリファレンス
- [Command Reference Index](docs/commands/index.md) — 全コマンド一覧

## License / ライセンス

MIT
