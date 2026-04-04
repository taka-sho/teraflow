## 15. 実装フェーズ: 人間/Agentハイブリッド

前版セクション9と同一の仕様を適用する。共通フロー、実装Issue自動生成、実装エージェント、PR統一フォーマット、バトンリレー、上限回数とエスカレーション、Agent信頼度の段階的緩和。

追加仕様: 継続的改善ステージの実装フェーズでは、以下が追加される。
- リグレッションテストの自動実行
- 本番環境への影響範囲チェック（CoDD連携）
- リリースフェーズへの接続（実装→テスト→リリース）

---

## 16. ハーネスエンジニアリング3層

前版セクション10と同一の3層構造を適用する。

**エージェント構成（10体に拡張）**:

| # | エージェント | ステージ | 機能 |
|---|---|---|---|
| 1 | 要求整理 | 初期開発, 継続的改善 | Discussion上でのAI対話 |
| 2 | 要件定義 | 初期開発, 継続的改善 | 要件の分解・明確化 |
| 3 | 実装 | 初期開発, 継続的改善 | Issue→ブランチ→実装→PR |
| 4 | CI失敗自動修正 | 全ステージ | CIログ解析→修正 |
| 5 | 自動レビュー | 全ステージ | 設計適合性+品質チェック |
| 6 | レビュー指摘修正 | 全ステージ | 指摘に基づく自動修正 |
| 7 | 停滞監視 | 全ステージ | 停止検知→復旧→通知 |
| 8 | コンフリクト解消 | 全ステージ | マージ競合の解決 |
| 9 | 障害調査 | 運用 | 障害原因の特定・修正方針提示 |
| 10 | 保守作業 | 保守 | ライブラリ更新・リファクタ |

---

## 17. ダッシュボード

### 17.1 表示セクション（ステージ対応に拡張）

1. **ライフサイクル状態**: 全ステージの状態（active/inactive/frozen/completed）を表示
2. **現在フェーズ**: アクティブステージごとのフェーズ進行状況
3. **フェーズ進捗**: 確定済み成果物ファイル数 / 想定ファイル数をプログレスバーで表示
4. **サイクル進捗**（継続的改善）: 各サイクルの進捗状況
5. **スケジュール状況**: 予定 vs 楽観/標準/悲観の見込み
6. **確定済み成果物一覧**: docs/ 配下の確定済みファイルをフェーズ別にリスト表示
7. **手戻り状況**: 進行中の手戻り件数・影響範囲・累計統計・フェーズ履歴タイムライン
8. **障害状況**（運用）: 未解決障害一覧、MTTR推移
9. **保守スコア**（保守）: 定期スコアリング結果の推移
10. **ブロッカー**: 7日以上停滞しているIssue一覧
11. **Agent実績**: 人間/Agent完了比率・成功率

---

## 18. 設定ファイル統合

```yaml
# .github/teraflow.yml

project:
  name: ""
  description: ""
  repository: ""

lifecycle:
  stages:
    # 各ステージの定義（セクション5.4参照）

phases:
  # 各フェーズのゲート条件（セクション6.4参照）

confirmation:
  trigger: "確定"                  # Issueコメントのトリガーワード（厳密一致）
  req_trigger: "要求確定"          # Discussion用（要求フェーズのみ）
  file_patterns:                   # フェーズごとの成果物ファイル命名
    要求整理:
      prefix: "REQ"
      path: "docs/01_requirements"
    要件定義:
      prefix: "REQDEF"
      path: "docs/02_requirement-definition"
    基本設計:
      prefix: "BD"
      path: "docs/03_basic-design"
    詳細設計:
      prefix: "DD"
      path: "docs/04_detailed-design"
    テスト:
      prefix: "TEST"
      path: "docs/05_test"

rework:
  re_execution_policy: "impact_based"
  # full:         戻し先から全フェーズを再通過
  # impact_based: CoDD影響分析で影響ありの成果物のみ再確認
  # manual:       PM/POが都度判断
  approve_trigger: "手戻り承認"       # Issue コメントのトリガーワード（厳密一致）

agent:
  enabled: true
  default_trust_level: "supervised"
  trust_levels:
    supervised:
      required_reviewers: 2
      auto_merge: false
    semi-autonomous:
      required_reviewers: 1
      auto_merge: false
    autonomous:
      required_reviewers: 0
      auto_merge: true
      conditions:
        max_changed_lines: 50
        ci_all_pass: true
  retry_limits:
    ci_fix: 3
    review_fix: 5
    conflict_resolve: 2
    stuck_recovery: 2
  req_agent:
    enabled: true
    model: "claude-sonnet-4-20250514"
    max_tokens: 2048
  implement_agent:
    enabled: true
    timeout_minutes: 60
  incident_agent:
    enabled: false          # 運用ステージで有効化
  maintenance_agent:
    enabled: false          # 保守ステージで有効化

ci:
  checks:
    - id: build
      required: true
    - id: unit_test
      required: true
    - id: lint
      required: true
    - id: static_analysis
      required: true
    - id: layer_dependency
      required: true
    - id: coverage
      required: true
      threshold: 80
    - id: regression_test
      required: false       # 継続的改善のリリースフェーズで有効化
      stages: [継続的改善]

scoring:
  enabled: false
  schedule: "weekly"
  auto_issue_threshold: 60  # スコアがこの値未満で自動Issue起票

dashboard:
  enabled: true
  deploy_target: "github-pages"

notifications:
  slack_webhook: ""
```

---

## 19. トレーサビリティ（ファイルベース）
