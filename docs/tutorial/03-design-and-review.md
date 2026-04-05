---
codd:
  node_id: "docs:tutorial-03-design-and-review"
  title: "チュートリアル03: 設計とレビュー"
  depends_on:
    - id: "docs:tutorial-02-requirements-with-ai"
      relation: implements
---

# チュートリアル 03: 設計とレビュー

この手順では設計 Discussion、PRレビュー、Skill 利用をつなげます。

## 1. 設計 Discussion を作成

```bash
gh discussion create --repo <owner>/<repo> --category "Design" --title "認証方式の設計" --body "候補案と比較観点を記述"
```

## 2. 設計案を合意

- 代替案比較
- リスク
- 採用理由

を Discussion に残し、決定事項を `docs/design/` に反映します。

## 3. PRでAIコードレビュー

```bash
git checkout -b feat/auth-design
# 実装変更...
git commit -am "feat: add auth flow"
git push -u origin feat/auth-design
gh pr create --fill
```

レビューでは AI provider を `assignments.review` で指定し、コメント方針を統一します。

## 4. Skill を使う

```bash
teraflow skill list
teraflow skill show --name review
teraflow skill validate
```

Skill でレビュー観点（設計整合性、テスト観点、安全性）を再利用できます。

## 5. 次へ

- [04: ライフサイクル管理](./04-lifecycle-management.md)
