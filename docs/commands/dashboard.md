---
codd:
  node_id: "docs:cmd-dashboard"
  title: "dashboard コマンドリファレンス"
  depends_on:
    - id: "design:cli-interface"
      relation: implements
---
# teraflow dashboard

## 概要

プロジェクト全体の進捗・手戻り・障害・予測情報をダッシュボードとして生成/表示するコマンド群。

## 使用方法

```bash
teraflow dashboard <subcommand> [flags]
```

## サブコマンド

- `show`: 現在のダッシュボード要約を表示
- `generate`: ダッシュボードHTMLを生成

## フラグ・オプション

- `teraflow dashboard show`
  - `--group, -g <string>`: グループ単位で表示
- `teraflow dashboard generate`
  - `--output, -o <path>`: 出力先（デフォルト: `.teraflow/dashboard/index.html`）
  - `--no-ai`: AIサマリを無効化
  - `--open`: 生成後にブラウザで開く

## 出力例

```text
$ teraflow dashboard generate --output .teraflow/dashboard/index.html
✓ dashboard generated
  path: .teraflow/dashboard/index.html
```

## 使用例

```bash
teraflow dashboard show
teraflow dashboard show --group group-auth
teraflow dashboard generate --open
```
