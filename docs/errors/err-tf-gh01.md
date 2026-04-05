---
node_id: err-tf-gh01
title: "GitHub CLI未導入エラー"
error_code: TF-GH01
category: github
exit_code: 4
message_template: "GitHub CLI (gh) is not installed"
user_action: |
  - GitHub CLI をインストールしてください
  - `gh --version` で利用可能か確認してください
related_commands: ["setup", "doctor"]
depends_on: []
status: confirmed
---

## TF-GH01: GitHub CLI未導入エラー
### 発生条件
GitHub 操作に `gh` が必要なコマンドで、CLI が見つからない場合に発生します。

### エラーメッセージ例
`GitHub CLI (gh) is not installed`

### 対処方法
GitHub CLI を導入し、PATH に含まれることを確認してください。
