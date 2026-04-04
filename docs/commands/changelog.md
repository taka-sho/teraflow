---
codd:
  node_id: "docs:cmd-changelog"
  title: "changelog コマンドリファレンス"
  depends_on:
    - id: "design:cli-interface"
      relation: implements
---
# teraflow changelog

## 概要

プロジェクト変更履歴の追記とリリース向けサマリー生成を行うコマンド群。変更ログの保管先は `.teraflow/changelog/`。

## 使用方法

```bash
teraflow changelog <subcommand> [flags]
```

## サブコマンド

- `add`: 変更エントリを追加
- `generate`: 期間やタグを指定して変更サマリーを生成
- `list`: 変更履歴を一覧表示

## フラグ・オプション

- `teraflow changelog add`
  - `<type>`: 変更種別（例: `feat`, `fix`, `docs`, `chore`）
  - `<message>`: 変更内容
  - `--group, -g <string>`: 対象グループ
- `teraflow changelog generate`
  - `--from <date>`: 集計開始日
  - `--to <date>`: 集計終了日
  - `--format <string>`: `md | json`
- `teraflow changelog list`
  - `--last <int>`: 直近N件

## 出力例

```text
$ teraflow changelog add feat "要件定義完了"
✓ changelog entry appended
  file: .teraflow/changelog/2026-04.jsonl
```

## 使用例

```bash
teraflow changelog add feat "決済API追加"
teraflow changelog list --last 20
teraflow changelog generate --from 2026-04-01 --to 2026-04-30 --format md
```
