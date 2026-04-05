---
node_id: err-tf-rb03
title: "ロール未定義エラー"
error_code: TF-RB03
category: rbac
exit_code: 3
message_template: "role not found: %s"
user_action: |
  - 指定ロール名が rbac.roles に存在するか確認してください
  - タイポや大文字小文字の差異を修正してください
related_commands: ["rbac list", "rbac apply"]
depends_on: []
status: confirmed
---

## TF-RB03: ロール未定義エラー
### 発生条件
参照されたロールが設定ファイルに存在しない場合に発生します。

### エラーメッセージ例
`role not found: reviewer`

### 対処方法
ロール定義を追加するか、参照箇所のロール名を正しい値へ修正します。
