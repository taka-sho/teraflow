---
node_id: err-tf-rb01
title: "権限不足エラー"
error_code: TF-RB01
category: rbac
exit_code: 3
message_template: "permission denied: user %q lacks %s"
user_action: |
  - teraflow.yml の rbac.roles で該当ユーザーに権限を付与してください
  - `teraflow rbac list` で現在のロール設定を確認してください
related_commands: ["gate approve", "rbac check", "rbac list"]
depends_on: []
status: confirmed
---

## TF-RB01: 権限不足エラー
### 発生条件
実行ユーザーが必要な権限を持たない状態で保護コマンドを実行した場合に発生します。

### エラーメッセージ例
`permission denied: user "alice" lacks gate.approve.testing`

### 対処方法
RBAC ロール定義に必要な permission を追加し、対象ユーザーへ割り当てます。
