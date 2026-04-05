---
codd:
  node_id: "req:phase2-github-integration"
  title: "teraflow Phase2 要件定義 — GitHub.com連携"
  depends_on:
    - id: "req:teraflow-overview"
      relation: derives_from
    - id: "req:cli-project-mgmt"
      relation: extends
    - id: "req:github-labels-issues"
      relation: extends
---

# teraflow Phase2 要件定義 — GitHub.com連携

## 概要

Phase2はPhase1（ローカルCLI）にGitHub.com連携を追加し、非エンジニア（PM/PMO）がブラウザからプロジェクト管理を行えるようにする。CLI操作と等価な操作がGitHub.com上から可能であることを保証する。

### Phase2の対象ユーザー

| ユーザー種別 | 操作手段 | 主な操作 |
|------------|---------|---------|
| エンジニア | CLI + GitHub.com | コード実装、テスト、CLI操作（Phase1と同等） |
| PM/PMO | GitHub.com のみ | フェーズ管理、手戻り承認、スケジュール確認 |
| 非エンジニア | GitHub.com のみ | 要件作成、Discussion参加、レポート閲覧 |

### 殿の裁定事項（本要件に反映済み）

- **B-004**: ghコマンドラップ方式
- **W-001**: ラベル主判定 + キーワード補完のハイブリッド方式
- **W-005**: YAML並行編集の排他制御
- **SSoT**: GitHub = Single Source of Truth

---

## セクション1: GitHub.com操作体系

非エンジニアがブラウザからプロジェクト管理操作を実行するための仕組み。

### 1.1 Issue駆動のフェーズ管理

| 操作 | GitHub.com上の操作 | 自動処理 |
|------|------------------|---------|
| フェーズ開始 | Issueテンプレート「フェーズ開始申請」作成 | Actions: project-state.yml更新、ラベル付与 |
| フェーズ完了 | Issue「フェーズ完了報告」のクローズ | Actions: フェーズ遷移、changelog記録 |
| フェーズスキップ | Issueテンプレート「フェーズスキップ申請」作成 | Actions: 理由記録、ゲート条件チェック |
| 成果物確定 | Issue/PRに `確定` ラベル付与（W-001主判定） | Actions: 確定カウント更新、成果物ファイルリンク |

### 1.2 ラベル駆動のステージ遷移（W-001ハイブリッド方式）

```
■ 主判定: ラベルベース
  ラベル stage:release → ステージ遷移Actionsトリガー → ゲート条件チェック → 遷移実行

■ 補完: キーワード（ラベル不可の場面のみ）
  Discussionコメントに「ステージ遷移: release」→ Actionsがキーワード検出 → 同上
```

### 1.3 Discussion上の要件確定フロー

| Discussionカテゴリ | 用途 | 自動処理 |
|-------------------|------|---------|
| 要件議論 | 要求整理・要件定義の議論 | AI対話Agent（#4）が要件整理を支援 |
| 設計議論 | 基本設計・詳細設計の議論 | 参照ドキュメントの自動リンク |
| 振り返り | フェーズ完了後のふりかえり | 統計情報の自動添付 |

### 1.4 PRマージ時のchangelog自動追記

```
PRタイトル規則:
  feat: ... → changelog type: feature
  fix: ...  → changelog type: bugfix
  docs: ... → changelog type: documentation
  refactor: → changelog type: refactoring

PRマージ → Actions → teraflow changelog add → .teraflow/changelog/YYYY-MM.jsonl 追記
```

---

## セクション2: GitHub Actionsワークフロー

Phase2で追加するteraflow専用Actionsワークフロー群。

### 2.1 ワークフロー一覧

| ID | ファイル | トリガー | 処理 | 対応機能マトリクス |
|----|---------|---------|------|-----------------|
| TF-001 | teraflow-phase-gate.yml | on: pull_request | フェーズ/ステージ整合性チェック | ステージ遷移ゲート(#1,2) |
| TF-002 | teraflow-permission-guard.yml | on: pull_request | ロール権限チェック（CODEOWNERS+Actions） | 権限ガード(#3) |
| TF-003 | teraflow-phase-transition.yml | on: issues (closed, labeled) | フェーズ遷移の自動記録 | Issueガード(#1,2) |
| TF-004 | teraflow-req-agent.yml | on: discussion (created, comment) | Discussion上AI要件整理 | Discussion上AI対話(#4) |
| TF-005 | teraflow-artifact-finalize.yml | on: issues (labeled: 確定) | 成果物確定→file+PR生成 | 成果物確定フロー(#5,7) |
| TF-006 | teraflow-changelog-update.yml | on: pull_request (closed, merged) | changelog自動追記 | changelog+Actions(#11) |
| TF-007 | teraflow-implement-agent.yml | on: issues (labeled: agent-implement) | 実装Agent起動 | 実装Agent(#9) |
| TF-008 | teraflow-ci-fix-agent.yml | on: check_run (completed, failure) | CI失敗自動修正Agent | CI修正Agent(#10) |
| TF-009 | teraflow-review-agent.yml | on: pull_request (opened, synchronize) | 自動レビューAgent | レビューAgent(#11,12) |
| TF-010 | teraflow-conflict-agent.yml | on: pull_request (labeled: conflict) | コンフリクト自動解消 | コンフリクト解消(#13) |
| TF-011 | teraflow-stuck-monitor.yml | on: schedule (cron: 0 9 * * 1-5) | 停滞Issue検出・通知 | 停滞監視(#14) |
| TF-012 | teraflow-rework-impact.yml | on: issues (labeled: rework-*) | 手戻り検知・影響分析 | rework+自動(#15,16) |
| TF-013 | teraflow-dashboard-deploy.yml | on: push (main), schedule (daily) | ダッシュボード自動デプロイ | dashboard deploy(#17) |
| TF-014 | teraflow-incident-agent.yml | on: issues (labeled: incident-*) | 障害調査Agent起動 | 障害調査Agent(#20) |
| TF-015 | teraflow-maintenance-agent.yml | on: schedule (cron: 0 0 * * 0) | 保守スコアリング・Issue起票 | 保守Agent(#21,22) |
| TF-016 | teraflow-schedule-predict.yml | on: schedule (cron: 0 9 * * 1) | 週次スケジュール予測更新 | schedule+自動(#6) |

### 2.2 ワークフロー実行パターン

```
パターン A: CLIバイナリ呼び出し
  ダウンロード → teraflow <command> 実行 → 結果をgit commit + push

パターン B: AI Agent起動
  teraflow agent assign → AI API呼び出し → PR/コメント生成

パターン C: 単純ファイル操作
  git checkout → ファイル更新 → git commit + push
```

### 2.3 teraflow CLIバイナリのActions内利用

```yaml
# 共通ステップ: teraflow CLIのインストール
- name: Install teraflow
  run: |
    curl -sL https://github.com/taka-sho/teraflow/releases/latest/download/teraflow_linux_amd64.tar.gz | tar xz
    chmod +x teraflow
    sudo mv teraflow /usr/local/bin/
```

---

## セクション3: Webhookイベント処理

### 3.1 イベント種別と処理マッピング

| GitHubイベント | サブタイプ | teraflow処理 | 対象ワークフロー |
|--------------|----------|-------------|----------------|
| issues | opened | フェーズ開始チェック（テンプレート判定） | TF-003 |
| issues | closed | フェーズ完了処理 | TF-003 |
| issues | labeled | ラベルに応じた自動処理分岐 | TF-003, TF-005, TF-012, TF-014 |
| pull_request | opened | フェーズゲートチェック + 自動レビュー | TF-001, TF-002, TF-009 |
| pull_request | closed (merged) | changelog追記 | TF-006 |
| pull_request | labeled: conflict | コンフリクト解消Agent | TF-010 |
| discussion | created | AI要件整理Agent（要件議論カテゴリのみ） | TF-004 |
| discussion | comment | AI応答生成 | TF-004 |
| check_run | completed (failure) | CI失敗修正Agent | TF-008 |
| schedule | cron | 定期実行（停滞監視、保守、スケジュール予測、ダッシュボード） | TF-011, TF-013, TF-015, TF-016 |

### 3.2 project-state.yml の自動更新（W-005排他制御）

```
GitHub Actions内でのproject-state.yml更新手順:

1. git pull origin main --rebase
2. teraflow <command> --config .github/teraflow.yml
   → .github/project-state.yml が更新される
3. git add .github/project-state.yml .teraflow/changelog/
4. git commit -m "teraflow: <operation description>"
5. git push origin main
   → 失敗時（conflict）: 最大3回リトライ（pull --rebase → push）
   → 3回失敗: Issueコメントで通知 + ワークフロー失敗
```

### 3.3 並行実行の制御

```yaml
# 全teraflow Actionsワークフロー共通
concurrency:
  group: teraflow-state-update
  cancel-in-progress: false  # 先行ジョブは完了まで待機
```

`concurrency` グループで同時実行を防止。`cancel-in-progress: false` で先行ジョブをキャンセルしない（データ不整合防止）。

---

## セクション4: 非エンジニア向けUI設計

### 4.1 Issueテンプレート（Phase2追加分）

| テンプレート | 用途 | 自動ラベル |
|------------|------|----------|
| フェーズ開始申請 | PM/PMOがフェーズ開始を申請 | `phase-transition`, `group:XXX` |
| フェーズ完了報告 | 成果物確定後のフェーズ完了報告 | `phase-complete`, `group:XXX` |
| フェーズスキップ申請 | フェーズをスキップする申請 | `phase-skip`, `group:XXX` |
| 手戻り申請 | 手戻りの申請（影響分析自動実行） | `rework-request`, `group:XXX` |
| インシデント報告 | 障害の報告（調査Agent自動起動） | `incident-report`, `severity:XXX` |
| Agent実装依頼 | AIによる実装を依頼 | `agent-implement`, `group:XXX` |

### 4.2 Discussionカテゴリ設計

```
.github/DISCUSSION_TEMPLATE/
├── requirement-discussion.yml    # 要件議論（AI対話Agent対応）
├── design-discussion.yml         # 設計議論
└── retrospective-discussion.yml  # 振り返り
```

### 4.3 GitHub Projects連携

```
カンバンボード構成:
  列: ステージ × フェーズ
  ┌─────────┬────────────┬──────────┬──────────┬──────────┐
  │ 要件定義 │ 基本設計   │ 詳細設計 │ 実装     │ テスト   │
  ├─────────┼────────────┼──────────┼──────────┼──────────┤
  │ Issue#1 │ Issue#5    │          │ Issue#10 │          │
  │ Issue#2 │ Issue#6    │          │          │          │
  └─────────┴────────────┴──────────┴──────────┴──────────┘

  フィルタ: グループ別（group:auth, group:api等）
  自動移動: フェーズ遷移時にIssueを次の列へ移動
```

### 4.4 CLI ↔ GitHub.com操作対応表（CLI代替マトリクス）

| CLI コマンド | GitHub.com等価操作 | 自動化ワークフロー |
|-------------|------------------|-----------------|
| `teraflow init` | — （初期化はCLI必須） | — |
| `teraflow setup labels` | — （ラベル作成はCLI推奨） | — |
| `teraflow stage current` | Projects ボード閲覧 | — |
| `teraflow stage advance` | ラベル `stage:XXX` 付与 | TF-001 |
| `teraflow phase current` | Projects ボード閲覧 | — |
| `teraflow phase advance` | Issueテンプレート「フェーズ完了報告」 | TF-003 |
| `teraflow phase skip` | Issueテンプレート「フェーズスキップ申請」 | TF-003 |
| `teraflow phase status` | Projects ボード + Issue一覧 | — |
| `teraflow group propose` | — （AI分析はCLI推奨） | — |
| `teraflow group define` | — （定義はCLI推奨） | — |
| `teraflow group finalize` | — （確定はCLI推奨） | — |
| `teraflow group member add` | groups.yml を直接編集→PR | TF-006 |
| `teraflow rework create` | Issueテンプレート「手戻り申請」 | TF-012 |
| `teraflow rework approve` | Issue `rework-approved` ラベル付与 | TF-012 |
| `teraflow incident create` | Issueテンプレート「インシデント報告」 | TF-014 |
| `teraflow schedule show` | ダッシュボード（GitHub Pages） | TF-013 |
| `teraflow schedule predict` | 週次自動実行（結果はダッシュボード） | TF-016 |
| `teraflow log show` | ダッシュボード + changelog閲覧 | — |
| `teraflow dashboard generate` | 自動デプロイ（GitHub Pages） | TF-013 |
| `teraflow harness score` | 週次自動実行 | TF-015 |
| `teraflow trace scan` | PR時自動実行（codd check） | CI-006 |
| `teraflow doctor` | — （環境チェックはCLI専用） | — |

**CLI必須操作**: init, setup, group propose/define/finalize, doctor
**GitHub.com完全代替**: stage advance, phase advance/skip, rework, incident, schedule, dashboard

---

## Phase2機能マトリクス対応表

| マトリクス項目（P2列） | 本要件でのカバー | ワークフローID |
|---------------------|---------------|--------------|
| ステージ遷移ゲート+Actions | セクション1.2, 2.1 | TF-001 |
| フェーズ管理+guard | セクション1.1, 2.1 | TF-003 |
| Issueガード(#1,2) | セクション2.1 | TF-001, TF-003 |
| Discussion上AI対話(#4) | セクション1.3, 2.1 | TF-004 |
| 成果物確定フロー(#5,7) | セクション1.1, 2.1 | TF-005 |
| schedule+自動 | セクション2.1 | TF-016 |
| rework+自動(#15,16) | セクション2.1, 3.1 | TF-012 |
| 障害調査Agent(#20) | セクション2.1 | TF-014 |
| 保守Agent(#21,22) | セクション2.1 | TF-015 |
| changelog+Actions | セクション1.4, 2.1 | TF-006 |
| 権限ガード(#3) | セクション2.1 | TF-002 |
| 実装Agent(#9) | セクション2.1 | TF-007 |
| CI修正/レビュー/修正Agent(#10-12) | セクション2.1 | TF-008, TF-009 |
| コンフリクト解消/停滞監視(#13,14) | セクション2.1 | TF-010, TF-011 |
| dashboard deploy(#17) | セクション2.1 | TF-013 |
