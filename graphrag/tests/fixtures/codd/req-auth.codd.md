---
codd:
  node_id: "req:auth"
  title: "認証要件"
  status: "confirmed"
  depends_on:
    - id: "design:auth"
      relation: requires
  tags:
    - requirements
    - auth
---

# 認証要件

ユーザー認証は JWT に対応し、期限切れトークンは再認証を必須とする。
