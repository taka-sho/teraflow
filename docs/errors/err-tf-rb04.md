---
node_id: err-tf-rb04
title: "最終管理者削除禁止エラー"
error_code: TF-RB04
category: rbac
exit_code: 3
message_template: "cannot remove the last admin: %s"
user_action: |
  - 先に別ユーザーへ admin 権限を付与してください
  - 最低1名の admin を維持した状態で再実行してください
related_commands: ["rbac apply", "rbac list"]
depends_on: []
status: confirmed
---

## TF-RB04: 最終管理者削除禁止エラー
### 発生条件
唯一の admin を削除しようとした場合に発生します。

### エラーメッセージ例
`cannot remove the last admin: taka-sho`

### 対処方法
admin ユーザーを追加してから、対象変更を再実行してください。
