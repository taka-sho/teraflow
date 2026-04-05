---
node_id: err-tf-gh02
title: "GitHub認証未完了エラー"
error_code: TF-GH02
category: github
exit_code: 4
message_template: "GitHub is not authenticated"
user_action: |
  - `gh auth login` を実行して認証してください
  - `gh auth status` で認証状態を確認してください
related_commands: ["doctor", "scan"]
depends_on: []
status: confirmed
---

## TF-GH02: GitHub認証未完了エラー
### 発生条件
GitHub API 連携が必要な処理で認証情報が無い場合に発生します。

### エラーメッセージ例
`GitHub is not authenticated`

### 対処方法
`gh auth login` 実行後に再試行してください。
