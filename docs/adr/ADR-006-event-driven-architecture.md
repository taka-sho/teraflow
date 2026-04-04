---
codd:
  node_id: "adr:006-event-driven"
  title: "ADR-006: イベント駆動アーキテクチャ（Webhook/Actions連携）"
  depends_on:
    - id: "req:phase2-github-integration"
      relation: implements
    - id: "adr:005-github-actions-design"
      relation: refines
---

# ADR-006: イベント駆動アーキテクチャ

## ステータス

提案（Proposed）

## コンテキスト

Phase2ではIssue/PR/Discussionイベントをトリガーにproject-state.ymlを自動更新する。イベント処理の設計方針を決定する必要がある。

## 決定

**GitHub Actionsのon:トリガーによるイベント駆動を採用する。**

### イベントフロー

```
[ユーザー操作]
    │ Issue作成 / ラベル付与 / PRマージ / Discussionコメント
    ▼
[GitHub Webhook]
    │
    ▼
[GitHub Actions trigger]
    │ on: issues / pull_request / discussion / check_run / schedule
    ▼
[Actions Job]
    ├── git pull origin main --rebase
    ├── teraflow <command>  ← CLIバイナリ実行
    ├── git commit + push
    └── 結果通知（Issueコメント / ラベル更新）
```

### イベント分類

| 分類 | トリガー | 特性 |
|------|---------|------|
| ユーザー起点 | issues, pull_request, discussion | 即座にActions起動（~30秒遅延） |
| システム起点 | check_run (failure) | CI失敗時の自動修正 |
| 定期実行 | schedule (cron) | 停滞監視、スケジュール予測、保守スコア |

### 遅延の許容

GitHub Actionsの起動遅延（~30秒）はPhase2の用途で許容範囲:
- フェーズ遷移: 手動操作の延長。30秒は問題なし
- changelog記録: PR マージ後の非同期処理。遅延許容
- Agent起動: AI処理自体が数分かかるため30秒は無視可能

### 代替案（却下）

| 案 | 却下理由 |
|----|---------|
| GitHub App Webhook（リアルタイム） | Phase3の範囲。Phase2ではActions遅延で十分 |
| ポーリング方式 | 非効率。APIレートリミットの懸念 |

## 影響

- 全teraflowワークフローは `concurrency: teraflow-state-update` で排他制御
- Actions実行ログがGitHub上で確認可能（デバッグ容易）
- Phase3（GitHub App）移行時にWebhookサーバーへの置き換えが必要
