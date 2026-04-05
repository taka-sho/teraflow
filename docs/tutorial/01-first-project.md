---
codd:
  node_id: "docs:tutorial-01-first-project"
  title: "チュートリアル01: 最初のプロジェクト"
  depends_on:
    - id: "docs:getting-started"
      relation: implements
---

# チュートリアル 01: 最初のプロジェクト

この手順では `init` から `status`、`stage`、`phase` までの最小フローを体験します。

## 1. 初期化

```bash
mkdir -p ~/tmp/teraflow-demo && cd ~/tmp/teraflow-demo
git init
teraflow init --name "teraflow-demo" --non-interactive
```

## 2. 状態確認

```bash
teraflow status
```

## 3. ステージ・フェーズ確認

```bash
teraflow stage list
teraflow phase list
```

## 4. `.github/teraflow.yml` の確認

`teraflow init` 実行後に `.github/teraflow.yml` が生成されます。主に以下を管理します。

- `project`: プロジェクト名と基本設定
- `workflow`: ステージ/フェーズ進行ルール
- `ai`: デフォルトAIプロバイダ
- `assignments`: タスク種別ごとのモデル割り当て

## 5. 次へ

- [02: AIと要件定義](./02-requirements-with-ai.md)
