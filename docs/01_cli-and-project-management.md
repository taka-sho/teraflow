---
codd:
  node_id: "req:cli-project-mgmt"
  title: "CLIツール概要・プロジェクト初期化・ステージ/フェーズ/継続改善サイクル管理"
  depends_on:
    - id: "req:teraflow-overview"
      relation: derives_from
---

## 3. CLIツール概要

### 3.1 ツール名・インストール

```bash
# インストール（配布方法は実装時に決定。pip / npm / go install 等）
pip install teraflow

# プロジェクト初期化
teraflow init --project-name "my-project"
```

### 3.2 コマンド体系

```
teraflow
├── init                    # プロジェクト初期化（設定ファイル・ディレクトリ・テンプレート生成）
│
├── stage
│   ├── current             # 現在のステージ状態表示（複数active対応）
│   ├── advance             # 次ステージへ遷移（ゲート条件チェック付き）
│   ├── activate            # ステージの有効化（保守等の並行ステージ）
│   └── freeze              # ステージの凍結（廃止決定時等）
│
├── phase
│   ├── current             # 現在フェーズの表示（--group でグループ指定可）
│   ├── advance             # 次フェーズへ遷移（--group 必須）
│   ├── skip                # フェーズのスキップ（許可されたステージのみ）
│   └── status              # フェーズ内の進捗状況表示
│
├── group
│   ├── propose             # 要件分析からグループ分割案を提示
│   ├── define              # 仮グループを手動定義
│   ├── finalize            # グループを確定（ゲート条件化）
│   ├── rebalance           # 人員変動時の再編成分析
│   ├── status              # グループ別の現在状態
│   ├── list                # グループ一覧
│   └── member
│       ├── add             # メンバー追加（changelog自動追記）
│       ├── remove          # メンバー離任（changelog自動追記）
│       ├── transfer        # グループ間移動（changelog自動追記）
│       └── list            # 現在のメンバー一覧
│
├── role
│   ├── list                # 全ロールとアクション一覧
│   ├── show                # 特定ユーザーのロール・権限表示
│   └── check               # 特定アクションの実行可否チェック
│
├── cycle
│   ├── create              # 改善サイクルの作成（継続的改善ステージ用）
│   ├── list                # サイクル一覧表示
│   └── status              # サイクル内の進捗表示
│
├── schedule
│   ├── show                # マスタースケジュール表示
│   ├── predict             # 完了予測の算出・表示
│   └── set                 # スケジュールの設定・更新
│
├── rework
│   ├── create              # 手戻りIssueの起票
│   ├── approve             # 手戻りの承認（→フェーズ後退PR作成）
│   ├── status              # 手戻り状況の表示
│   └── stats               # 手戻り統計の表示
│
├── incident
│   ├── create              # 障害報告Issueの起票（運用ステージ用）
│   ├── status              # 障害対応状況
│   └── stats               # 障害統計（MTTR等）
│
├── agent
│   ├── assign              # Issueへのagent割り当て
│   ├── status              # Agent稼働状況
│   └── stats               # Agent実績統計
│
├── log
│   ├── show                # 変更履歴表示（フィルタ付き）
│   │   ├── --last N        # 直近N件
│   │   ├── --member X      # メンバー指定
│   │   ├── --group X       # グループ指定
│   │   ├── --type X        # イベントタイプ指定
│   │   ├── --file X        # ファイル指定
│   │   ├── --from DATE     # 期間指定（開始）
│   │   └── --to DATE       # 期間指定（終了）
│   └── stats               # 期間別イベント統計
│
├── dashboard
│   ├── generate            # ダッシュボードHTML生成（全セクション）
│   ├── deploy              # GitHub Pagesへデプロイ
│   └── config              # 表示セクションのカスタマイズ
│
├── report
│   ├── generate            # 定期レポート生成（進捗会議用）
│   │   ├── --since DATE    # 前回レポートからの差分起点
│   │   ├── --format md|pdf # 出力形式
│   │   └── --audience      # 対象者（executive/pm/lead/dev）
│   └── schedule            # レポート自動生成のスケジュール設定
│
├── harness
│   ├── check               # ハーネス設定の健全性チェック
│   └── score               # 定期スコアリング実行
│
└── setup
    ├── actions             # GitHub Actionsワークフロー群の生成
    ├── labels              # GitHubラベルの一括作成
    ├── templates           # Issue/Discussionテンプレートの生成
    └── codeowners          # CODEOWNERS ファイル生成（groups.yml ベース）
```

---

## 4. プロジェクト初期化 (`teraflow init`)

### 4.1 生成されるディレクトリ構成

```
project-root/
├── .github/
│   ├── teraflow.yml                   # teraflow メイン設定
│   ├── project-state.yml              # ステージ・フェーズ状態
│   ├── master-schedule.yml            # マスタースケジュール
│   ├── groups.yml                     # グループ・メンバー・ロール定義
│   ├── CODEOWNERS                     # PR必須レビュアー（groups.ymlから自動生成）
│   ├── ISSUE_TEMPLATE/
│   │   ├── requirement.yml            # 要求起票（非エンジニア向け）
│   │   ├── requirement-definition.yml # 要件定義
│   │   ├── basic-design.yml           # 基本設計
│   │   ├── detailed-design.yml        # 詳細設計
│   │   ├── implementation.yml         # 実装
│   │   ├── test.yml                   # テスト
│   │   ├── rework.yml                 # 手戻り
│   │   ├── incident.yml               # 障害報告（運用ステージ用）
│   │   ├── inquiry.yml                # 問い合わせ（運用ステージ用）
│   │   ├── maintenance.yml            # 保守作業
│   │   └── improvement-request.yml    # 改善要求（継続的改善ステージ用）
│   ├── DISCUSSION_TEMPLATE/
│   │   └── requirement-discussion.yml # 要求相談（Discussion用）
│   └── workflows/
│       ├── teraflow-stage-guard.yml       # ステージ不一致Issue拒否
│       ├── teraflow-phase-guard.yml       # フェーズ不一致Issue拒否
│       ├── teraflow-permission-guard.yml  # ロール権限チェック
│       ├── teraflow-req-agent.yml         # 要求整理AIエージェント（Discussion上）
│       ├── teraflow-req-finalize.yml      # 要求確定→REQ file+Issue+PR生成
│       ├── teraflow-phase-agent.yml       # Issue上AI対話（要件定義〜テスト）
│       ├── teraflow-phase-finalize.yml    # 「確定」→成果物file+PR+Issueクローズ
│       ├── teraflow-index-update.yml      # 各フェーズindex.md自動再生成
│       ├── teraflow-implement-agent.yml   # 実装エージェント
│       ├── teraflow-ci-fix-agent.yml      # CI失敗自動修正
│       ├── teraflow-review-agent.yml      # 自動レビュー
│       ├── teraflow-review-fix-agent.yml  # レビュー指摘修正
│       ├── teraflow-conflict-agent.yml    # コンフリクト自動解消
│       ├── teraflow-stuck-monitor.yml     # パイプライン停滞監視
│       ├── teraflow-dashboard.yml         # ダッシュボード自動更新
│       ├── teraflow-rework-impact.yml     # 手戻り時影響分析
│       ├── teraflow-rework-approve.yml    # 手戻り承認→フェーズ後退
│       ├── teraflow-incident-agent.yml    # 障害調査エージェント
│       └── teraflow-maintenance-agent.yml # 保守作業エージェント
│
├── .teraflow/
│   └── changelog/                     # 変更ログ（月別JSONL）
│       ├── 2026-04.jsonl
│       ├── 2026-05.jsonl
│       └── ...
│
├── docs/
│   ├── shared/                        # グループ横断の成果物
│   │   ├── 01_requirements/
│   │   │   └── index.md
│   │   └── 03_basic-design/           # API IF定義等
│   ├── backend/                       # バックエンドグループ（例）
│   │   ├── 01_requirements/
│   │   ├── 02_requirement-definition/
│   │   ├── 03_basic-design/
│   │   ├── 04_detailed-design/
│   │   └── 05_test/
│   ├── frontend/                      # フロントエンドグループ（例）
│   │   └── ...
│   ├── infra/                         # インフラグループ（例）
│   │   └── ...
│   ├── 06_migration/                  # 移行・リリース計画
│   ├── 07_operation/                  # 運用手順書・ランブック
│   ├── 08_maintenance/               # 保守記録
│   ├── golden-principles.md           # Terasoluna黄金原則
│   ├── rework-log.yml                 # 手戻り記録（自動追記）
│   └── incident-log.yml              # 障害記録（自動追記）
│
├── src/
│   ├── app/                           # アプリケーション層
│   ├── domain/                        # ドメイン層
│   └── infra/                         # インフラ層
│
├── tests/
├── CLAUDE.md
└── README.md
```

**注**: グループ確定前（要求整理〜要件定義前半）は `docs/shared/` のみを使用する。グループ確定後にグループ別ディレクトリが生成される。

### 4.2 初期化時のインタラクティブ設問

```
$ teraflow init

Project name: my-order-system
Description: 注文管理システム

Starting stage (default: 初期開発):
  [x] 初期開発（新規プロジェクト）
  [ ] 継続的改善（既存プロジェクトへの導入）

AI Agent features:
  [x] 要求整理エージェント（Discussion上でのAI対話）
  [x] 実装エージェント（Issue→PR自動生成）
  [ ] 自動レビュー（後から有効化可能）
  [ ] CI失敗自動修正（後から有効化可能）
  [ ] 障害調査エージェント（運用ステージで有効化）
  [ ] 保守作業エージェント（保守ステージで有効化）

GitHub repository URL: https://github.com/org/my-order-system

Initializing...
  ✓ Directory structure created
  ✓ Configuration files generated
  ✓ Issue templates generated (stage: 初期開発)
  ✓ GitHub Actions workflows generated
  ✓ CLAUDE.md generated
  ✓ Golden Principles generated

Next steps:
  1. git add -A && git commit -m "chore: teraflow init"
  2. git push origin main
  3. teraflow setup labels
  4. teraflow setup actions
  5. teraflow schedule set
```

---

## 5. ステージ管理

### 5.1 ステージ遷移 (`teraflow stage advance`)

```
$ teraflow stage advance

Active stages: 初期開発
Target stage:  移行・リリース

Checking stage gate conditions...
  ✓ All phases in 初期開発 are complete
  ✓ Gate Issue #100 "初期開発完了判定" is closed
  ✓ リリース判定チェックリスト完了

Ready to advance. Create PR? (y/n)
```

### 5.2 並行ステージの有効化 (`teraflow stage activate`)

```
$ teraflow stage activate 保守

Active stages:
  ③ 運用: active
  ④ 継続的改善: active
  ⑤ 保守: active  ← 有効化

New Issue templates enabled:
  ✓ maintenance.yml

New workflows enabled:
  ✓ teraflow-maintenance-agent.yml
```

### 5.3 ステージの凍結 (`teraflow stage freeze`)

サービス終了決定時等に使用する。凍結されたステージでは新規Issueの作成が拒否される。

```
$ teraflow stage freeze 継続的改善

⚠️  Freezing stage: 継続的改善
  - No new Issues will be accepted
  - Existing open Issues will remain (close manually if needed)
  - Existing cycles will not accept new phases

Confirm? (y/n)
```

### 5.4 ステージごとのゲート条件

```yaml
# .github/teraflow.yml（lifecycle セクション）

lifecycle:
  stages:
    初期開発:
      phases: [要求整理, 要件定義, 基本設計, 詳細設計, 実装, テスト]
      phase_skip_allowed: false
      gate_conditions:
        - type: all_phases_complete
        - type: gate_issue_closed
          issue_label: "gate:初期開発"

    移行・リリース:
      phases: [移行計画, 環境構築, データ移行, リハーサル, 本番切替, 安定化確認]
      phase_skip_allowed: false
      gate_conditions:
        - type: all_phases_complete
        - type: gate_issue_closed
          issue_label: "gate:移行・リリース"
      required_artifacts:
        - "docs/06_migration/migration-plan.md"
        - "docs/06_migration/rollback-plan.md"

    運用:
      phases: []                  # イベント駆動
      parallel: true              # 他ステージと並行稼働
      gate_conditions: []         # 終了条件なし

    継続的改善:
      phases: [要求整理, 要件定義, 基本設計, 詳細設計, 実装, テスト, リリース]
      phase_skip_allowed: true    # 小規模改修ではスキップ可
      parallel: true
      cycle_based: true           # Milestone単位でサイクル管理

    保守:
      phases: [調査, 計画, 実施, 検証]
      phase_skip_allowed: true
      parallel: true

    廃止:
      phases: [廃止計画, 告知・移行支援, サービス停止, アーカイブ]
      phase_skip_allowed: false
      gate_conditions:
        - type: all_phases_complete
        - type: gate_issue_closed
          issue_label: "gate:廃止"
      required_artifacts:
        - "docs/decommission-plan.md"
      on_activate:
        freeze_stages: [継続的改善, 保守]
```

---

## 6. フェーズ管理

### 6.1 フェーズ状態（ステージ内）

フェーズは現在アクティブなステージの中で管理される。フェーズの管理方式はステージによって異なる。

| ステージ | フェーズ管理方式 |
|---|---|
| 初期開発 | 順序厳守、スキップ不可 |
| 移行・リリース | 順序厳守、スキップ不可 |
| 運用 | フェーズなし（イベント駆動） |
| 継続的改善 | 順序あり、スキップ可、サイクル単位 |
| 保守 | 順序あり、スキップ可 |
| 廃止 | 順序厳守、スキップ不可 |

### 6.2 フェーズ遷移 (`teraflow phase advance`)

フェーズ遷移時、`project-state.yml` の `phase_history` に完了記録を追加し、次フェーズの開始記録を追加する。全ての遷移はPR経由で行い、履歴の改ざんを防止する。

```
$ teraflow phase advance

Stage: 初期開発
Current phase: 要求整理
Target phase:  要件定義

Checking gate conditions...
  ✓ All requirement files confirmed (15 files in docs/01_requirements/REQ-*.md)
  ✓ Requirement index exists (docs/01_requirements/index.md)
  ✓ Gate Issue #1 is closed

Ready to advance. Create PR? (y/n)
```

遷移時に `phase_history` に記録されるエントリ:

```yaml
# 完了したフェーズ
- phase: "要求整理"
  started_at: "2026-04-01"
  ended_at: "2026-04-18"       # 遷移実行日を自動記録
  result: "completed"

# 開始したフェーズ
- phase: "要件定義"
  started_at: "2026-04-21"     # 遷移実行日を自動記録
  ended_at: null
  result: null
```

**result の種類**:

| result | 意味 | 発生条件 |
|---|---|---|
| `completed` | 正常に完了し次フェーズへ進んだ | `teraflow phase advance` |
| `reverted` | 手戻りにより中断し前フェーズへ戻った | `teraflow rework approve` |
| `skipped` | スキップされた | `teraflow phase skip` |

### 6.3 フェーズスキップ (`teraflow phase skip`)

`phase_skip_allowed: true` のステージでのみ使用可能。

```
$ teraflow phase skip 基本設計

Stage: 継続的改善 (v1.3)
Skipping phase: 基本設計

⚠️  Skipping requires justification.
Reason: 既存画面のテキスト変更のみのため設計不要

Confirm? (y/n)
  ✓ Phase skipped. Recorded in phase history.
  Current phase: 詳細設計
```

### 6.4 ゲート条件の定義

```yaml
# .github/teraflow.yml（phases セクション）

phases:
  要求整理:
    gate_conditions:
      - type: files_exist
        path: "docs/01_requirements/REQ-*.md"
        description: "全要求が確定済みファイルとして存在する"
      - type: file_exists
        path: "docs/01_requirements/index.md"
      - type: no_open_issues
        label: "phase:要求整理"
        description: "作業中のIssueが残っていない"
      - type: gate_issue_closed
        issue_label: "gate:要求整理"

  要件定義:
    gate_conditions:
      - type: files_exist
        path: "docs/02_requirement-definition/REQDEF-*.md"
      - type: no_open_issues
        label: "phase:要件定義"
      - type: gate_issue_closed
        issue_label: "gate:要件定義"

  基本設計:
    gate_conditions:
      - type: files_exist
        path: "docs/03_basic-design/BD-*.md"
      - type: no_open_issues
        label: "phase:基本設計"
      - type: gate_issue_closed
        issue_label: "gate:基本設計"

  詳細設計:
    gate_conditions:
      - type: files_exist
        path: "docs/04_detailed-design/DD-*.md"
      - type: no_open_issues
        label: "phase:詳細設計"
      - type: gate_issue_closed
        issue_label: "gate:詳細設計"

  実装:
    gate_conditions:
      - type: no_open_issues
        label: "phase:実装"
      - type: gate_issue_closed
        issue_label: "gate:実装"

  テスト:
    gate_conditions:
      - type: files_exist
        path: "docs/05_test/TEST-*.md"
      - type: no_open_issues
        label: "phase:テスト"
      - type: gate_issue_closed
        issue_label: "gate:テスト"

  リリース:
    gate_conditions:
      - type: no_open_issues
        label: "phase:リリース"
      - type: gate_issue_closed
        issue_label: "gate:リリース"
      - type: regression_test_pass

  # 移行・リリースステージ固有フェーズ
  移行計画:
    gate_conditions:
      - type: file_exists
        path: "docs/06_migration/migration-plan.md"
      - type: file_exists
        path: "docs/06_migration/rollback-plan.md"

  # 保守ステージ固有フェーズ
  調査:
    gate_conditions:
      - type: gate_issue_closed
        issue_label: "gate:調査"

  # 廃止ステージ固有フェーズ
  廃止計画:
    gate_conditions:
      - type: file_exists
        path: "docs/decommission-plan.md"
```

### 6.5 Issue ガード（ステージ + フェーズの二重チェック）

GitHub Actions ワークフロー `teraflow-stage-guard.yml` + `teraflow-phase-guard.yml` で実現。

**チェック順序**:
1. Issue のテンプレートからステージを判定
2. 現在アクティブなステージに含まれるか確認（ステージガード）
3. ステージ内の現在フェーズと一致するか確認（フェーズガード）

**例外**:
- `rework`（手戻り）Issue はどのステージ・フェーズでも受け付ける
- `incident`（障害報告）Issue は運用ステージがアクティブであれば常に受け付ける
- `maintenance`（保守）Issue は保守ステージがアクティブであれば常に受け付ける

**手戻り承認後のフェーズガード**: 手戻りが承認されると `project-state.yml` のフェーズ自体が正式に戻るため、戻し先フェーズのIssueは通常通り受け付けられる。個別Issueに対する例外処理は不要。

---

## 7. 継続的改善サイクル管理

### 7.1 サイクルの作成 (`teraflow cycle create`)

継続的改善ステージでは、Milestone 単位で開発サイクルを管理する。

```
$ teraflow cycle create --name "v1.1 注文検索機能追加"

Stage: 継続的改善
Creating cycle: v1.1 注文検索機能追加

  ✓ Milestone "v1.1: 注文検索機能追加" created
  ✓ Gate Issues created for each phase
  ✓ Starting phase: 要求整理

Phases for this cycle:
  [x] 要求整理      ← current
  [ ] 要件定義
  [ ] 基本設計       (skippable)
  [ ] 詳細設計       (skippable)
  [ ] 実装
  [ ] テスト
  [ ] リリース
```

### 7.2 複数サイクルの並行管理

```
$ teraflow cycle list

  Stage: 継続的改善

  Cycle              Phase        Progress    Status
  v1.1 注文検索      テスト       █████████░  90%  active
  v1.2 管理画面改善  基本設計     ████░░░░░░  40%  active
  v1.3 CSV出力       要求整理     ██░░░░░░░░  15%  active
```

### 7.3 リリースフェーズ（継続的改善固有）

初期開発にはない「リリース」フェーズが追加される。稼働中のシステムへ変更を反映するための最終チェックフェーズ。

ゲート条件:
- リグレッションテスト全パス
- CoDD 影響分析で未対応の Amber/Green がないこと
- リリースチェックリスト完了

---

## 8. ラベル体系
