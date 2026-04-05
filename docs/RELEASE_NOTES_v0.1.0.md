# teraflow v0.1.0 リリースノート

## Phase2: GitHub.com連携の正式リリース

### 新機能

#### GitHub Actions 自動化 (16ワークフロー)
- **TF-001** `teraflow-phase-gate` - PRマージ前フェーズ整合性チェック
- **TF-002** `teraflow-permission-guard` - CODEOWNERSロール権限チェック
- **TF-003** `teraflow-phase-transition` - Issue操作によるフェーズ自動遷移
- **TF-004** `teraflow-req-agent` - AI要件整理エージェント
- **TF-005** `teraflow-artifact-finalize` - 成果物確定とchangelog自動記録
- **TF-006** `teraflow-changelog-update` - PRマージ時changelog自動追記
- **TF-007** `teraflow-implement-agent` - AI実装提案エージェント
- **TF-008** `teraflow-ci-fix-agent` - CI失敗時AI修正提案
- **TF-009** `teraflow-review-agent` - PRオープン時AIコードレビュー
- **TF-010** `teraflow-conflict-agent` - コンフリクト自動解消エージェント
- **TF-011** `teraflow-stuck-monitor` - 平日9時停滞プロジェクト検知
- **TF-012** `teraflow-rework-impact` - 手戻り影響分析自動実行
- **TF-013** `teraflow-dashboard-deploy` - GitHub Pages自動ダッシュボード
- **TF-014** `teraflow-incident-agent` - インシデント自動調査エージェント
- **TF-015** `teraflow-maintenance-agent` - 週次保守スコアリング
- **TF-016** `teraflow-schedule-predict` - 週次スケジュール予測更新

#### 新コマンド
- `teraflow setup actions` - 16ワークフローをリポジトリに一括展開
- `teraflow setup templates` - IssueテンプレートとDiscussionカテゴリを展開
- `teraflow agent assign --type <type>` - AI Agentの手動実行
- `teraflow agent status` - 利用可能なAgentタイプ一覧

#### Issueテンプレート (6種)
フェーズ開始/完了/スキップ、手戻り依頼、インシデント報告、AI実装依頼

#### Discussionカテゴリ (3種)
要件議論、設計議論、振り返り

### 品質
- テストカバレッジ: 90.2%
- E2E: Phase1 31チェック + Phase2 Wave1-4 全PASS
- golangci-lint v2 全クリア

### 必要環境
- Go 1.25.8+
- GitHub Actions (GITHUB_TOKEN)
- ANTHROPIC_API_KEY (AI機能 - 任意)

### アップグレード方法
```bash
# 最新バイナリをダウンロード
curl -sL "https://github.com/taka-sho/teraflow/releases/download/v0.1.0/teraflow_0.1.0_linux_amd64.tar.gz" | tar xz

# Phase2セットアップ（新規）
teraflow setup actions
teraflow setup templates
```
