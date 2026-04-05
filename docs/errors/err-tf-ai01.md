---
node_id: err-tf-ai01
title: "AI APIキー未設定エラー"
error_code: TF-AI01
category: ai
exit_code: 4
message_template: "API key not set for provider: %s"
user_action: |
  - 対象プロバイダーの API キーを環境変数へ設定してください
  - `teraflow doctor` で設定状態を確認してください
related_commands: ["doctor", "requirements"]
depends_on: []
status: confirmed
---

## TF-AI01: AI APIキー未設定エラー
### 発生条件
選択中プロバイダーに対応する API キーが未設定の場合に発生します。

### エラーメッセージ例
`API key not set for provider: openai`

### 対処方法
`.env` または CI secrets に API キーを設定して再実行してください。
