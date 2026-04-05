---
codd:
  node_id: "docs:tutorial-05-ci-cd-integration"
  title: "チュートリアル05: CI/CDとの統合"
  depends_on:
    - id: "docs:tutorial-04-lifecycle-management"
      relation: implements
---

# チュートリアル 05: CI/CDとの統合

この手順では GitHub Actions と Codecov をセットアップします。

## 1. Actions ワークフローを生成

```bash
teraflow setup actions
```

## 2. ワークフロー実行確認

```bash
git add .github/workflows
git commit -m "ci: add teraflow workflows"
git push
```

## 3. Codecov 連携

GitHub の `Settings > Secrets and variables > Actions` で必要な値を設定し、カバレッジワークフローを有効化します。

- `CODECOV_TOKEN`（必要なリポジトリの場合）

## 4. 最終チェック

```bash
teraflow doctor --check-ai
teraflow status
```

これで要件定義から運用までを CI/CD 上で継続運用できます。
