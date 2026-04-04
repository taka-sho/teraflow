---
codd:
  node_id: "docs:cmd-label"
  title: "label コマンドリファレンス"
  depends_on:
    - id: "design:cli-interface"
      relation: implements
---
# teraflow label

## 概要

GitHubラベル体系をプロジェクトへ同期し、ステージ/フェーズ/分類ラベルを初期化するコマンド群。

## 使用方法

```bash
teraflow label <subcommand> [flags]
```

## サブコマンド

- `sync`: 定義済みラベルをGitHubへ反映
- `list`: 管理対象ラベルを表示
- `validate`: 現在のラベル状態を検証

## フラグ・オプション

- `teraflow label sync`
  - `--dry-run`: 変更内容のみ表示
  - `--repo <string>`: 対象リポジトリ
- `teraflow label list`
  - `--category <string>`: `stage | phase | type | priority | severity`
- `teraflow label validate`
  - `--strict`: 不足ラベルがあればエラー終了

## 出力例

```text
$ teraflow label sync
✓ stage labels synced
✓ phase labels synced
✓ classification labels synced
```

## 使用例

```bash
teraflow label sync --dry-run
teraflow label sync --repo org/order-system
teraflow label validate --strict
```
