---
node_id: err-tf-gh03
title: "GitHub APIエラー"
error_code: TF-GH03
category: github
exit_code: 4
message_template: "GitHub API error: %s"
user_action: |
  - APIレスポンスの詳細を確認してください
  - レート制限や権限不足の可能性を確認してください
related_commands: ["scan", "discussion", "label"]
depends_on: []
status: confirmed
---

## TF-GH03: GitHub APIエラー
### 発生条件
GitHub API 呼び出しが失敗した場合に発生します。

### エラーメッセージ例
`GitHub API error: 403 Forbidden`

### 対処方法
認証権限、対象リポジトリ権限、レート制限の状態を確認してください。
