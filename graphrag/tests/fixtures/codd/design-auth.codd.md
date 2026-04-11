---
codd:
  node_id: "design:auth"
  title: "認証設計"
  status: "confirmed"
  depends_on:
    - id: "policy:security"
      relation: references
  tags:
    - design
    - auth
---

# 認証設計

認証設計では OAuth2 + JWT を採用し、アクセストークン失効処理を実装する。
