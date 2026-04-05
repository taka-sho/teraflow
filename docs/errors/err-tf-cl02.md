---
node_id: err-tf-cl02
title: "引数不正エラー"
error_code: TF-CL02
category: cli
exit_code: 2
message_template: "invalid argument %q for %s"
user_action: |
  - 引数値の形式と許容値を `--help` で確認してください
  - サブコマンドごとの使用例を参照してください
related_commands: ["--help", "doctor"]
depends_on: []
status: confirmed
---

## TF-CL02: 引数不正エラー
### 発生条件
引数値が不正、または許容範囲外の場合に発生します。

### エラーメッセージ例
`invalid argument "foo" for --mode`

### 対処方法
正しい引数形式へ修正して再実行してください。
