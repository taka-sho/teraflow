---
node_id: err-tf-ai03
title: "プロバイダー設定不正エラー"
error_code: TF-AI03
category: ai
exit_code: 4
message_template: "provider configuration error: %s"
user_action: |
  - provider 名と model 設定が有効値か確認してください
  - `teraflow.yml` の ai.providers セクションを見直してください
related_commands: ["config", "doctor"]
depends_on: []
status: confirmed
---

## TF-AI03: プロバイダー設定不正エラー
### 発生条件
AI プロバイダー設定が不完全または不正な場合に発生します。

### エラーメッセージ例
`provider configuration error: unknown provider "foo"`

### 対処方法
サポート対象の provider/model を設定してください。
