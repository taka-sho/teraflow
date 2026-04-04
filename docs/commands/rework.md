---
codd:
  node_id: "docs:cmd-rework"
  title: "rework コマンドリファレンス"
  depends_on:
    - id: "design:cli-interface"
      relation: implements
---

# teraflow rework

## 概要

手戻りタスクの起票・一覧確認を行います。

## 使用方法

```bash
teraflow rework create --group <group-id> --target-phase <phase> --reason <reason>
teraflow rework list [--group <group-id>] [--open]
```

## フラグ・オプション

- `rework create --group`, `-g <string>`: 対象グループ
- `rework create --target-phase <string>`: 手戻り先フェーズ
- `rework create --reason <string>`: 手戻り理由
- `rework list --open`: 未解決手戻りのみ表示

## 出力例

```text
Created rework issue #42
group: group-auth
target phase: basic_design
reason: API仕様変更
```

## エラーコード

- `E4001`: 対象手戻りIssue未検出
- `E4002`: 既に承認済み
- `E5001`: gh未インストール
- `E5002`: gh未認証

## 使用例

```bash
teraflow rework create --group group-auth --target-phase basic_design --reason "API仕様変更"
teraflow rework list --open
```
