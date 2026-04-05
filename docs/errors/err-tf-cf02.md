---
node_id: err-tf-cf02
title: "設定ファイル不正エラー"
error_code: TF-CF02
category: config
exit_code: 1
message_template: "configuration file is invalid: %s"
user_action: |
  - `teraflow.yml` の YAML 構文を確認してください
  - 必須項目（project, slcp_jcf, agents 等）が欠落していないか確認してください
related_commands: ["config", "doctor"]
depends_on: []
status: confirmed
---

## TF-CF02: 設定ファイル不正エラー
### 発生条件
`teraflow.yml` が破損している、または必須キーが欠落している場合に発生します。

### エラーメッセージ例
`configuration file is invalid: yaml: line 12: did not find expected key`

### 対処方法
YAML 構文と必須設定を修正したうえで、再実行してください。
