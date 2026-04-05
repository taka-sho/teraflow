---
node_id: err-tf-ix01
title: "インデックス未作成エラー"
error_code: TF-IX01
category: index
exit_code: 1
message_template: "index not found (run 'teraflow index build' first)"
user_action: |
  - `teraflow index build` を実行してインデックスを生成してください
  - index ディレクトリが削除されていないか確認してください
related_commands: ["index build", "index search"]
depends_on: []
status: confirmed
---

## TF-IX01: インデックス未作成エラー
### 発生条件
検索や要約でインデックスが必要だが、未作成の状態で実行した場合に発生します。

### エラーメッセージ例
`index not found (run 'teraflow index build' first)`

### 対処方法
`teraflow index build` を実行し、生成後に再試行してください。
