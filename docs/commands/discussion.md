---
codd:
  node_id: "docs:cmd-discussion"
  title: "discussion コマンドリファレンス"
  depends_on:
    - id: "design:cli-interface"
      relation: implements
---
# teraflow discussion

## 概要

要求整理フェーズ向けに、GitHub Discussions を起点とした対話フローを管理するコマンド群。

## 使用方法

```bash
teraflow discussion <subcommand> [flags]
```

## サブコマンド

- `create`: 要求相談Discussionを作成
- `list`: Discussion一覧を表示
- `finalize`: 要求確定をトリガーしIssue/成果物生成フローへ接続

## フラグ・オプション

- `teraflow discussion create`
  - `--title <string>`: 件名（必須）
  - `--category <string>`: Discussionカテゴリ
  - `--body <string>`: 本文
- `teraflow discussion list`
  - `--open`: オープンのみ表示
  - `--limit <int>`: 最大件数
- `teraflow discussion finalize`
  - `--id <string>`: 対象Discussion ID（必須）

## 出力例

```text
$ teraflow discussion create --title "注文検索条件を相談したい"
✓ Discussion #58 created
  category: requirement-discussion
```

## 使用例

```bash
teraflow discussion create --title "顧客検索要件" --category requirement-discussion
teraflow discussion list --open --limit 20
teraflow discussion finalize --id 58
```
