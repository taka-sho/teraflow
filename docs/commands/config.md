---
codd:
  node_id: "docs:cmd-config"
  title: "config コマンドリファレンス"
  depends_on:
    - id: "design:cli-interface"
      relation: implements
---

# teraflow config

## 概要

teraflow の設定値を表示・更新します。

## 使用方法

```bash
teraflow config show
teraflow config set --key <path> --value <value>
```

## フラグ・オプション

- `config set --key <string>`: 更新対象キー（例: `ai.default_provider`）
- `config set --value <string>`: 設定値
- `--config <path>`: 設定ファイルパス（既定値: `.github/teraflow.yml`）

## 出力例

```text
version: "1"
project:
  name: "my-project"
ai:
  default_provider: anthropic
```

## エラーコード

- `E0002`: 設定ファイル未検出
- `E0003`: 設定ファイル不正

## 使用例

```bash
teraflow config show
teraflow config set --key ai.default_provider --value openai
```
