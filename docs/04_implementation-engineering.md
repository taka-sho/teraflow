## 18. 実装フェーズ: 人間/Agentハイブリッド

前版セクション9と同一の仕様を適用する。共通フロー、実装Issue自動生成、実装エージェント、PR統一フォーマット、バトンリレー、上限回数とエスカレーション、Agent信頼度の段階的緩和。

追加仕様: 継続的改善ステージの実装フェーズでは、以下が追加される。
- リグレッションテストの自動実行
- 本番環境への影響範囲チェック（CoDD連携）
- リリースフェーズへの接続（実装→テスト→リリース）

---

## 19. ハーネスエンジニアリング3層

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

## 20. ダッシュボード・定期レポート

### 20.1 2種類の出力

| | 常時ダッシュボード | 定期レポート |
|---|---|---|
| 形式 | GitHub Pages（HTML） | Markdown or PDF |
| 更新 | 日々自動更新 | `teraflow report generate` で手動 or スケジュール生成 |
| 用途 | いつ見ても最新の状態がわかる | 進捗会議用スナップショット。前回との差分を含む |

### 20.2 表示セクション（全10セクション）

#### セクションA: プロジェクト全体サマリ

全体の一目把握。経営層が最初に見るセクション。

表示内容: 現在のステージ / 全体進捗（プログレスバー） / 健全度（🟢正常・🟡注意・🔴危険） / 予定完了日 vs 見込完了日（楽観・標準・悲観） / アラート一覧

健全度の判定基準:

```
🟢 正常: 全グループが予定±3日以内
🟡 注意: いずれかのグループが予定+4〜7日
🔴 危険: いずれかのグループが予定+8日以上、
         または未解決の手戻りが2件以上、
         または同期ポイントが7日以上ブロック中
```

#### セクションB: グループ別フェーズ進捗

各グループの現在フェーズ・成果物確定数・進捗率・見込完了日を一覧表示。同期ポイントの状態（完了・ブロック中）も併せて表示。

表示例:

```
  Group        Phase     成果物     進捗          見込完了
  backend      詳細設計   8/10件   ████████░░ 80%  06/18 (予定06/20 ✅)
  frontend     基本設計   5/8件    ██████░░░░ 63%  06/10 (予定05/30 🔴+7日)
  infra        実装       3/5件    ██████░░░░ 60%  07/01 (予定06/25 🟡+4日)
```

#### セクションC: フェーズ進行タイムライン

グループごとのフェーズ履歴をガントチャート形式で表示。手戻りによる中断・再開も視覚的に表現。

表示例:

```
  backend:
    要求整理  04/01━━━04/15 ✅
    要件定義  04/16━━━05/05 ✅
    基本設計  05/06━━05/18 🔄 手戻り→要件定義
    要件定義  05/18━━━05/25 ✅ (rework #42)
    基本設計  05/26━━━06/08 ✅ (rework #42)
    詳細設計  06/09━━━━━━━ ◀ 進行中
```

#### セクションD: 成果物確定状況

docs/ 配下の確定済みファイルをフェーズ別・グループ別にリスト表示。直近の確定一覧（日付・ファイル名・グループ・確定者）を含む。

#### セクションE: スケジュール予測

グループごとの各フェーズの予定日 vs 楽観/標準/悲観の見込みをテーブル表示。プロジェクト全体の完了見込み（最遅グループ基準）と主な遅延要因を記載。

表示例:

```
  グループ: backend
  Phase          予定      楽観     標準     悲観
  詳細設計       06/20    06/18   06/22   06/28   ◀ 進行中
  実装           07/25    07/28   08/05   08/18
  テスト         08/15    08/18   08/28   09/10

  プロジェクト全体（最遅グループ基準）:
    予定完了: 08/15  標準見込: 09/01 (+13営業日)
```

#### セクションF: 手戻り状況

未解決・解決済みの手戻り一覧（ID・発覚フェーズ・原因フェーズ・グループ・影響スコア・解決日数）。手戻り統計（累計回数・平均解決日数・原因フェーズ傾向・改善ポイント）を含む。

#### セクションG: ブロッカー・リスク

ブロッカー（作業停止中の Issue）、停滞中（7日以上進展なし）、同期ポイント待ち、識別済みリスクを一覧表示。各項目に担当者・影響範囲・見込み解消日を記載。

#### セクションH: メンバー・リソース状況

グループ別の人数・内訳（ロール・メンバー名）、直近のメンバー変動（着任・離任・移動）、グループ別稼働率（成果物確定数/人/週）を表示。

#### セクションI: Agent実績（実装フェーズ以降）

人間担当 vs Agent担当の完了比率、Agent品質指標（CI一発パス率・平均実装時間・エスカレーション率）、信頼度分布（supervised/semi-autonomous/autonomous の件数）、週次の改善推移を表示。

#### セクションJ: 直近の変更ログ

changelog から直近1週間のイベントを時系列で表示。成果物確定・フェーズ遷移・メンバー変動・手戻り・Agent活動を含む。

### 20.3 閲覧者別のデフォルト表示

ロールに応じてデフォルトで表示するセクションを設定する。全セクションを常時表示すると情報過多になるため。

| ロール | デフォルト表示セクション |
|---|---|
| 経営層・ステークホルダー | A, E, G |
| PM | A〜J（全セクション） |
| グループリード | A, B, C（自グループ）, D（自グループ）, F, G, I（自グループ） |
| 開発者 | B, D（自グループ）, I, J |
| 非エンジニア・起票者 | A, D（要求のみ） |

ダッシュボードURL にクエリパラメータ（`?role=pm` 等）またはトグルで切り替え可能にする。

### 20.4 定期レポート (`teraflow report generate`)

進捗会議用に特定時点のスナップショットを出力する。ダッシュボードの全セクションに加え、**前回レポートとの差分**を先頭に追加する。

差分セクションに含める内容:

```
■ 前回からの主な変化
  ・フェーズ遷移（どのグループがどこに進んだか）
  ・手戻りの発生・解決
  ・メンバー変動
  ・成果物確定数の増分

■ 今週の進捗
  ・各グループの進捗率の変化（前回→今回）

■ 見込完了の変化
  ・前回予測 vs 今回予測（改善 or 悪化）
  ・変化の主要因

■ 次週の見通し
  ・各グループの予定アクション

■ 対応が必要な事項
  ・ブロッカー・停滞中の Issue
  ・エスカレーションが必要な判断事項
```

対象者別にレポートの粒度を変える:

```
$ teraflow report generate --since 2026-06-08 --audience executive --format pdf
  → セクション A, E, G + 差分サマリ のみ

$ teraflow report generate --since 2026-06-08 --audience pm --format md
  → 全セクション + 差分詳細

$ teraflow report generate --since 2026-06-08 --audience lead --group backend --format md
  → 自グループ中心のセクション + 差分
```

### 20.5 データソース

| セクション | データソース |
|---|---|
| A: 全体サマリ | `project-state.yml` + `master-schedule.yml` |
| B: グループ別進捗 | `project-state.yml` + `docs/` ファイル数 |
| C: タイムライン | `project-state.yml` の `phase_history` |
| D: 成果物確定状況 | `docs/**/*.md` のファイル一覧 + frontmatter |
| E: スケジュール予測 | `master-schedule.yml` + 実績データ |
| F: 手戻り状況 | `rework-log.yml` + changelog |
| G: ブロッカー・リスク | GitHub Issues API + `groups.yml` の `sync_points` |
| H: メンバー・リソース | `groups.yml` + changelog |
| I: Agent実績 | changelog（`agent.*` イベント）+ CI実行結果 |
| J: 変更ログ | `.teraflow/changelog/YYYY-MM.jsonl` |

---

## 21. 設定ファイル統合

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
  sections: [A, B, C, D, E, F, G, H, I, J]    # 表示セクション（セクション20.2参照）
  default_views:                                # ロール別デフォルト表示
    executive: [A, E, G]
    pm: [A, B, C, D, E, F, G, H, I, J]
    lead: [A, B, C, D, F, G, I]
    developer: [B, D, I, J]
    requester: [A, D]
  health_thresholds:                            # 健全度判定基準（日数）
    green: 3
    yellow: 7

report:
  schedule: "weekly"                            # 自動生成スケジュール（weekly / biweekly / none）
  default_format: "md"                          # md / pdf
  default_audience: "pm"

notifications:
  slack_webhook: ""
```

---

