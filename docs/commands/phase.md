---
codd:
  node_id: "docs:cmd-phase"
  title: "phase コマンドリファレンス"
  depends_on:
    - id: "design:cli-interface"
      relation: implements
---

# teraflow phase

## 概要

フェーズの一覧表示、開始、完了を管理します。グループ単位の進行管理に利用します。

## 使用方法

```bash
teraflow phase list [--group <group-id>]
teraflow phase start --group <group-id> --phase <phase-name>
teraflow phase complete --group <group-id>
```

## フラグ・オプション

- `--group`, `-g <string>`: 対象グループID
- `--phase <string>`: 対象フェーズ名（start時）
- `--yes`, `-y`: 確認プロンプト省略（complete時）

## 出力例

```text
Group       Phase              Status
group-auth  implementation     in_progress
group-api   basic_design       in_progress
```

## エラーコード

- `E2002`: 成果物未確定で遷移不可
- `E2003`: 同期ポイント未達
- `E3001`: グループ未定義

## 使用例

```bash
teraflow phase list
teraflow phase start --group group-auth --phase implementation
teraflow phase complete --group group-auth
```
