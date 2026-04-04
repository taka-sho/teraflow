---
codd:
  node_id: "docs:cmd-incident"
  title: "incident コマンドリファレンス"
  depends_on:
    - id: "design:cli-interface"
      relation: implements
---
# teraflow incident

## 概要

運用ステージで障害Issueを起票し、対応状況と統計を確認するコマンド群。

## 使用方法

```bash
teraflow incident <subcommand> [flags]
```

## サブコマンド

- `create`: 障害報告Issueを起票
- `status`: 障害対応状況を表示
- `stats`: MTTRや件数を集計表示

## フラグ・オプション

- `teraflow incident create`
  - `--title <string>`: 障害タイトル（必須）
  - `--severity <string>`: `critical | major | minor`（デフォルト: `major`）
  - `--description <string>`: 障害概要
- `teraflow incident status`
  - `--open`: 未解決の障害のみ表示
- `teraflow incident stats`
  - `--since <date>`: 集計開始日
  - `--metric <string>`: `mttr | count | severity`（デフォルト: `all`）

## 出力例

```text
$ teraflow incident create --title "DBが応答しない" --severity critical
✓ Issue #123 created
  labels: stage:運用, incident, severity:critical
```

## 使用例

```bash
teraflow incident create --title "API timeout" --severity major --description "Order API response > 30s"
teraflow incident status --open
teraflow incident stats --since 2026-04-01 --metric mttr
```
