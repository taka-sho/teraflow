---
codd:
  node_id: "docs:cmd-stage"
  title: "stage コマンドリファレンス"
  depends_on:
    - id: "design:cli-interface"
      relation: implements
---

# teraflow stage

## 概要

ステージの一覧表示・状態確認・遷移を行います。

## 使用方法

```bash
teraflow stage list
teraflow stage status
teraflow stage advance [--yes] [--force]
```

## フラグ・オプション

- `stage advance --yes`, `-y`: 確認プロンプト省略
- `stage advance --force`: ゲート条件未達でも強制遷移
- グローバル: `--format`, `--verbose`, `--dry-run`

## 出力例

```text
Stage            Status
initial_development  active
release              pending
operation            inactive
```

## エラーコード

- `E2001`: ステージ遷移ゲート未達
- `E0001`: teraflow未初期化プロジェクト

## 使用例

```bash
teraflow stage list
teraflow stage status
teraflow stage advance --yes
```
