---
node_id: err-tf-cf01
title: "プロジェクト未初期化エラー"
error_code: TF-CF01
category: config
exit_code: 1
message_template: "not a teraflow project: %s"
user_action: |
  - プロジェクトルートで `teraflow init` を実行してください
  - `teraflow.yml` の配置場所が正しいか確認してください
related_commands: ["init", "doctor"]
depends_on: []
status: confirmed
---

## TF-CF01: プロジェクト未初期化エラー
### 発生条件
teraflow 管理対象外のディレクトリでコマンドを実行した場合に発生します。

### エラーメッセージ例
`not a teraflow project: /path/to/dir`

### 対処方法
プロジェクトルートへ移動するか、未初期化の場合は `teraflow init` を実行します。
