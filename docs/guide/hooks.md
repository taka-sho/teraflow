---
codd:
  node_id: "docs:guide-hooks"
  title: "Hooks フレームワーク利用ガイド"
  depends_on:
    - id: "docs:guide-index-summary"
      relation: related
---

# Hooks フレームワーク利用ガイド

## 1. 概要

Hooks は GitHub イベントを起点に `teraflow` のアクションを自動実行する仕組みである。  
`discussion`/`push`/`pull_request` などのイベントに応じて、応答生成や index 更新をイベント駆動で回せる。

## 2. `teraflow.yml` の `hooks:` 記法

`.github/teraflow.yml` に `hooks:` セクションを追加し、イベントごとにアクション配列を定義する。

```yaml
hooks:
  on_discussion_created:
    - action: respond
      skill: requirements
      conditions:
        categories: ["要件定義", "Requirements"]
        not_author: ["github-actions[bot]"]

  on_discussion_comment:
    - action: respond
      skill: requirements
      conditions:
        categories: ["要件定義", "Requirements"]
        not_author: ["github-actions[bot]"]

  on_push:
    - action: index_update
```

## 3. イベント一覧（5種）

- `on_discussion_created`
- `on_discussion_comment`
- `on_confirmation`
- `on_push`
- `on_pr_opened`

## 4. アクション一覧

- `respond`: `agent assign` を呼び出して応答生成
- `summarize`: 要約モードで応答生成
- `index_update`: `.teraflow/index.yml` を更新
- `summary_update`: 要約キャッシュを更新
- `generate`: 将来拡張用（現状は stub）

## 5. 条件フィルタ

各 action には `conditions` を設定できる。

- `not_author`: 除外する投稿者
- `categories`: Discussion カテゴリ一致
- `labels`: ラベル一致
- `paths`: 変更ファイルパス一致

例:

```yaml
conditions:
  not_author: ["github-actions[bot]"]
  categories: ["要件定義", "Requirements"]
  labels: ["urgent"]
  paths: ["docs/**", "cmd/**"]
```

## 6. `teraflow hook list`

定義済み hooks の確認:

```bash
teraflow hook list --config .github/teraflow.yml
```

出力にはイベントごとの action 数と conditions 要約が含まれる。

## 7. `teraflow hook run <event> --dry-run`

イベント手動実行（コマンド実行前の確認）:

```bash
teraflow hook run on_discussion_created \
  --author "alice" \
  --category "Requirements" \
  --discussion-id "D_kwDOXXXXXX" \
  --input "要件を整理したい" \
  --dry-run \
  --config .github/teraflow.yml
```

`--dry-run` では実処理せず、実行予定コマンドとマッチした action を確認できる。

## 8. `teraflow setup actions --hooks`

hooks 定義から GitHub Actions を生成:

```bash
teraflow setup actions --hooks --config .github/teraflow.yml
```

イベント構成に応じて以下の workflow が生成される。

- `teraflow-hooks-pr.yml`

## 9. サンプル: Discussion 作成時に req-agent 起動

```yaml
hooks:
  on_discussion_created:
    - action: respond
      skill: requirements
      conditions:
        categories: ["要件定義", "Requirements"]
        not_author: ["github-actions[bot]"]
```

この設定で、要件カテゴリの Discussion 作成時に `requirements` スキルの応答処理が起動する。
