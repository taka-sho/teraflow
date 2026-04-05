---
node_id: err-tf-rb02
title: "ユーザー解決失敗エラー"
error_code: TF-RB02
category: rbac
exit_code: 3
message_template: "rbac enabled but current user could not be determined"
user_action: |
  - `--user` フラグでユーザー名を明示してください
  - CI 環境では実行ユーザー情報が取得できるよう設定してください
related_commands: ["gate approve", "rbac check"]
depends_on: []
status: confirmed
---

## TF-RB02: ユーザー解決失敗エラー
### 発生条件
RBAC が有効で、かつ実行ユーザー名を解決できない場合に発生します。

### エラーメッセージ例
`rbac enabled but current user could not be determined`

### 対処方法
`--user <name>` を指定するか、実行環境のユーザー情報取得条件を整備してください。
