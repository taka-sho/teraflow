---
codd:
  node_id: "docs:tutorial-02-requirements-with-ai"
  title: "チュートリアル02: AIと要件定義"
  depends_on:
    - id: "docs:tutorial-01-first-project"
      relation: implements
---

# チュートリアル 02: AIと要件定義

この手順では GitHub Discussion を使って要求を固め、`要求確定` トリガーで次工程に進めます。

## 1. Requirements Discussion を作成

```bash
gh discussion create --repo <owner>/<repo> --category "Requirements" --title "ユーザー登録機能の要求" --body "課題と期待値を記述"
```

## 2. AIとの壁打ち

Discussion 上で要件候補を会話し、次を明確化します。

- スコープ（何を作るか）
- 非スコープ（何を作らないか）
- 受け入れ条件（完了定義）

## 3. 「要求確定」トリガー

最終コメントで `要求確定` を明示します。運用ルールで自動タスク化する場合は、このキーワードをワークフロー条件に使います。

## 4. AIプロバイダ設定

`.github/teraflow.yml` の `ai.default_provider` と `assignments.requirements` を設定します。

```yaml
ai:
  default_provider: anthropic

assignments:
  requirements:
    provider: anthropic
    model: claude-haiku-4-5-20251001
```

## 5. 次へ

- [03: 設計とレビュー](./03-design-and-review.md)
