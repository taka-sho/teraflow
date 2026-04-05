---
node_id: err-tf-gt02
title: "ゲートルール未定義エラー"
error_code: TF-GT02
category: gate
exit_code: 5
message_template: "gate rule not found for process: %s"
user_action: |
  - teraflow.yml の gate_rules に対象プロセス定義を追加してください
  - プロセス名の不一致（typing）を確認してください
related_commands: ["gate approve", "config"]
depends_on: []
status: confirmed
---

## TF-GT02: ゲートルール未定義エラー
### 発生条件
対象プロセスに対応する gate_rules 設定が存在しない場合に発生します。

### エラーメッセージ例
`gate rule not found for process: testing`

### 対処方法
`teraflow.yml` の gate_rules に該当プロセスを定義してください。
