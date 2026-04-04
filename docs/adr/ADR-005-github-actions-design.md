---
codd:
  node_id: "adr:005-github-actions-design"
  title: "ADR-005: GitHub Actionsワークフロー設計方針"
  depends_on:
    - id: "req:phase2-github-integration"
      relation: implements
    - id: "adr:004-data-github"
      relation: extends
---

# ADR-005: GitHub Actionsワークフロー設計方針

## ステータス

提案（Proposed）

## コンテキスト

Phase2ではGitHub.comからの操作が必要になる。非エンジニア（PM/PMO）がブラウザのみでフェーズ管理・手戻り・インシデント報告等を行えるようにする。Phase1で実装したCLIのロジックをGitHub Actions環境でも活用する設計が必要。

## 決定

**GitHub Actions + teraflow CLIバイナリ呼び出しパターンを採用する。**

### 設計方針

1. **CLIバイナリの再利用**: Actions内でteraflow CLIバイナリをダウンロードし、コマンドを実行する。ロジックの二重実装を防ぐ
2. **イベント駆動**: GitHub Webhook（issues, pull_request, discussion等）をトリガーにActions を起動
3. **State更新はCLI経由**: project-state.yml等の更新はteraflow CLIコマンドを介して行い、直接ファイル編集しない
4. **16ワークフロー体制**: Phase2で16本のワークフローを段階的に導入

### CLIバイナリのインストール

```yaml
- name: Install teraflow
  run: |
    VERSION=$(curl -s https://api.github.com/repos/taka-sho/teraflow/releases/latest | jq -r .tag_name)
    curl -sL "https://github.com/taka-sho/teraflow/releases/download/${VERSION}/teraflow_${VERSION#v}_linux_amd64.tar.gz" | tar xz
    chmod +x teraflow && sudo mv teraflow /usr/local/bin/
```

### 代替案（却下）

| 案 | 却下理由 |
|----|---------|
| Webhookサーバー（自前） | インフラ管理コスト。SSoT=GitHub原則に反する |
| GitHub App Webhook | Phase3の範囲。Phase2はActions で十分 |
| Actions内でGoビルド | 毎回ビルドは遅い。リリースバイナリの利用が効率的 |

## 影響

- CD-001（release.yml）でリリースされたバイナリがActionsから利用される
- Actions実行にはGITHUB_TOKENの権限設定が必要（contents: write, issues: write, pull-requests: write）
- バイナリバージョンとリポジトリのteraflow.yml互換性を維持する必要がある
