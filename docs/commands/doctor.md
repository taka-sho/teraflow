---
codd:
  node_id: "docs:cmd-doctor"
  title: "doctor コマンドリファレンス"
  depends_on:
    - id: "design:doctor-command"
      relation: implements
---

# teraflow doctor

## 概要

プロジェクト健全性を 4 カテゴリ（Environment / Configuration / Project Integrity / AI Integration）で診断します。

## 使用方法

```bash
teraflow doctor [--check-ai] [--format text|json]
```

## フラグ・オプション

- `--check-ai`: AI 連携（`ANTHROPIC_API_KEY`）を追加チェック
- `--format <type>`: 出力形式（`text` / `json`）
- `--config <path>`: 設定ファイルパス（既定: `.github/teraflow.yml`）

## 出力例（text）

```text
teraflow doctor

Environment
  ✓ go — go1.22.0
  ✓ gh — gh 2.50.0 (authenticated)
  ✓ git — git found

Configuration
  ✓ teraflow.yml — teraflow.yml (valid)
  ✗ project-state.yml — not initialized (run teraflow init)

Project Integrity
  ✓ rework-log.yml — 3 entries
  ✓ incident-log.yml — 0 entries

AI Integration (skipped — use --check-ai to test)

Result: 1 issue found
```

## 出力例（json）

```json
{
  "status": "issues_found",
  "checks": [
    {"category": "environment", "name": "go", "ok": true, "message": "go1.22.0"},
    {"category": "configuration", "name": "project-state.yml", "ok": false, "message": "not initialized (run teraflow init)"}
  ],
  "issues": 1
}
```

## 使用例

```bash
teraflow doctor
teraflow doctor --check-ai
teraflow doctor --format json
```
