---
codd:
  node_id: "docs:repository-setup"
  title: "Repository Setup Guide"
  depends_on:
    - id: "docs:getting-started"
      relation: extends
---

# Repository Setup Guide

teraflow のエージェント機能（req-agent, review-agent 等）をリポジトリに組み込む際に
必要な GitHub 設定をまとめます。

## 1. 必要な機能の有効化

- **GitHub Discussions**: req-agent を使う場合に必要
  Settings → Features → Discussions ☑

## 2. Workflow Permissions

teraflow の自動 PR 作成エージェント（req-agent 等）は GITHUB_TOKEN で PR を作成します。
以下の設定が必要です。

### 2.1 GUI で設定する

1. リポジトリの Settings → Actions → General
2. "Workflow permissions" セクションで:
   - ☑ **Read and write permissions**
   - ☑ **Allow GitHub Actions to create and approve pull requests**
3. Save

### 2.2 CLI で設定する（推奨）

`<OWNER>` と `<REPO>` を実際の値に置き換えてください:

```bash
gh api -X PUT repos/<OWNER>/<REPO>/actions/permissions/workflow \
  -F default_workflow_permissions=write \
  -F can_approve_pull_request_reviews=true
```

確認:
```bash
gh api repos/<OWNER>/<REPO>/actions/permissions/workflow
```

期待値:
```json
{
  "default_workflow_permissions": "write",
  "can_approve_pull_request_reviews": true
}
```

### 2.3 なぜ必要か

ワークフロー YAML 側で `permissions: pull-requests: write` を宣言しても、
リポジトリ設定で `can_approve_pull_request_reviews: false` の場合は
`gh pr create` が以下のエラーで失敗します:

```
Error: gh pr create: pull request create failed:
GraphQL: GitHub Actions is not permitted to create or approve
pull requests (createPullRequest)
```

これは GitHub のセキュリティ機構で、yaml 側の宣言だけでは bypass できません。

## 3. Secrets

teraflow の AI エージェントは LLM プロバイダーを呼び出します。
以下のいずれかを Settings → Secrets and variables → Actions に登録してください:

| Provider | Secret 名 |
|---|---|
| Anthropic Claude | `ANTHROPIC_API_KEY` |
| OpenAI GPT | `OPENAI_API_KEY` |

`.github/teraflow.yml` の `assignments` で各エージェントが使うプロバイダーを指定します。
詳細は [AI Provider 設定ガイド](ai-providers.md) を参照してください。

## 4. teraflow ファイルの初期配置

```bash
teraflow init
```

これで `.github/workflows/teraflow-*.yml` および `.github/teraflow.yml` が配置されます。

## 5. 動作確認

設定後、以下のコマンドで確認できます:

```bash
teraflow setup verify
```

リポジトリの設定（permissions, secrets, discussions）を読み取り、
未設定項目があれば警告を表示します。

## 6. トラブルシューティング

### `gh pr create: ... not permitted to create or approve pull requests`

→ §2 の Workflow Permissions を設定してください。

### `Neither ANTHROPIC_API_KEY nor OPENAI_API_KEY is set`

→ §3 の Secrets を設定してください。

### req-agent が応答しない

→ Discussions が有効か確認してください。また `.github/teraflow.yml` の
`assignments` が正しく設定されているかチェックしてください。

### `go install` で旧バージョンが入る

既知の proxy.golang.org list キャッシュ遅延の問題です。
teraflow v0.4.11 以降のテンプレートは GitHub Releases API tag pin 方式で
回避済みです。手動 install する場合は `go install ...@v0.4.x` のように
バージョンを指定してください。
