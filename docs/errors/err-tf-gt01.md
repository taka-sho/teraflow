---
node_id: err-tf-gt01
title: "ゲート条件未達エラー"
error_code: TF-GT01
category: gate
exit_code: 5
message_template: "gate conditions not met: %s"
user_action: |
  - エラーメッセージに表示された未達条件を順に解消してください
  - `teraflow doctor` や対象フェーズの出力を確認してください
related_commands: ["gate approve", "doctor"]
depends_on: []
status: confirmed
---

## TF-GT01: ゲート条件未達エラー
### 発生条件
Gate 評価で必要条件を満たしていない場合に発生します。

### エラーメッセージ例
`gate conditions not met: tests failed; coverage below threshold`

### 対処方法
未達条件を解消後、再度 `gate approve` を実行してください。
