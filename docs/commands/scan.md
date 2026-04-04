---
codd:
  node_id: "docs:cmd-scan"
  title: "scan コマンドリファレンス"
  depends_on:
    - id: "design:cli-interface"
      relation: implements
---

# teraflow scan

## 概要

プロジェクト状態をスキャンし、`required_artifacts` の存在確認と不整合検出を行います。

## 使用方法

```bash
teraflow scan [--format text|json]
```

## フラグ・オプション

- `--format <type>`: 出力形式 (`text` / `json`)
- `--verbose`, `-v`: 欠損成果物や検出詳細を表示

## 出力例

```text
Scan result: warning
Required artifacts:
  - docs/shared/01_requirements/index.md: found
  - docs/group-auth/03_basic-design/api-spec.md: missing
Inconsistencies: 1
```

## エラーコード

- `E0001`: teraflow未初期化プロジェクト
- `E0002`: 設定ファイル未検出
- `E0003`: 設定ファイル不正

## 使用例

```bash
teraflow scan
teraflow scan --format json
```
