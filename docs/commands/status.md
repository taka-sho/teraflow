---
codd:
  node_id: "docs:cmd-status"
  title: "status コマンドリファレンス"
  depends_on:
    - id: "design:cli-interface"
      relation: implements
---

# teraflow status

## 概要

現在のステージ・フェーズ・進捗率を表示します。`project-state.yml` を読み取り、プロジェクト全体の状態を要約します。

## 使用方法

```bash
teraflow status [--format text|json]
```

## フラグ・オプション

- `--format <type>`: 出力形式 (`text` / `json`)
- `--verbose`, `-v`: 詳細表示
- `--quiet`, `-q`: エラー以外を抑制

## 出力例

```text
Project: my-project
Current stage: initial_development
Current phase: requirements
Progress: 20%
```

## エラーコード

- `E0001`: teraflow未初期化プロジェクト
- `E0002`: 設定ファイル未検出
- `E0003`: 設定ファイル不正

## 使用例

```bash
teraflow status
teraflow status --format json
```
