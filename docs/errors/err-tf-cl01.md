---
node_id: err-tf-cl01
title: "必須フラグ未指定エラー"
error_code: TF-CL01
category: cli
exit_code: 2
message_template: "required flag %q not set"
user_action: |
  - エラーメッセージで指定された必須フラグを追加してください
  - `--help` で必須引数・フラグを確認してください
related_commands: ["--help"]
depends_on: []
status: confirmed
---

## TF-CL01: 必須フラグ未指定エラー
### 発生条件
必須フラグが渡されないままコマンド実行した場合に発生します。

### エラーメッセージ例
`required flag "project" not set`

### 対処方法
不足しているフラグを指定し、再実行してください。
