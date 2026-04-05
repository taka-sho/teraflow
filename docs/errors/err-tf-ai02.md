---
node_id: err-tf-ai02
title: "AI API呼び出し失敗エラー"
error_code: TF-AI02
category: ai
exit_code: 4
message_template: "AI API error: %s"
user_action: |
  - APIレスポンスの詳細を確認してください
  - 入力トークン量、モデル名、ネットワーク状態を確認してください
related_commands: ["requirements", "design", "review"]
depends_on: []
status: confirmed
---

## TF-AI02: AI API呼び出し失敗エラー
### 発生条件
AI プロバイダー API からエラー応答が返った場合に発生します。

### エラーメッセージ例
`AI API error: context_length_exceeded`

### 対処方法
入力サイズやモデル設定を見直し、再試行してください。
