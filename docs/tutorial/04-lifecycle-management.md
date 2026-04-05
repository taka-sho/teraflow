---
codd:
  node_id: "docs:tutorial-04-lifecycle-management"
  title: "チュートリアル04: ライフサイクル管理"
  depends_on:
    - id: "docs:tutorial-03-design-and-review"
      relation: implements
---

# チュートリアル 04: ライフサイクル管理

この手順では `stage advance` / `phase complete` と整合性チェックを実行します。

## 1. 現在の状態確認

```bash
teraflow status
teraflow stage status
teraflow phase list
```

## 2. ステージ遷移

```bash
teraflow stage advance --to release
```

## 3. フェーズ完了

```bash
teraflow phase complete --name implementation
```

## 4. 整合性チェック

```bash
teraflow scan
teraflow doctor
```

- `scan`: 成果物・リンク・依存の整合性確認
- `doctor`: 環境、設定、必須要素の健全性確認

## 5. 次へ

- [05: CI/CDとの統合](./05-ci-cd-integration.md)
