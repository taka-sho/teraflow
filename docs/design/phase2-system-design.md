---
codd:
  node_id: "design:phase2-system"
  title: "Phase2 システム設計 — GitHub.com連携"
  depends_on:
    - id: "req:phase2-github-integration"
      relation: implements
    - id: "adr:005-github-actions-design"
      relation: implements
    - id: "adr:006-event-driven"
      relation: implements
    - id: "adr:007-non-engineer-ui"
      relation: implements
    - id: "adr:008-phase1-coexistence"
      relation: implements
    - id: "design:system-overview"
      relation: extends
---

# Phase2 システム設計 — GitHub.com連携

## 概要

Phase2ではPhase1のローカルCLIアーキテクチャにGitHub.com連携層を追加し、非エンジニア（PM/PMO）がブラウザのみでプロジェクト管理を行える環境を構築する。設計はADR-005〜008の決定事項に基づく。

### 設計原則

1. **CLIバイナリ再利用**: Actions内でリリース済みteraflowバイナリを実行（ロジック二重実装防止）
2. **イベント駆動**: GitHub Webhookイベント → Actions trigger → teraflow CLI → state更新
3. **SSoT**: GitHub上の `.github/project-state.yml` が唯一の状態ソース
4. **楽観的並行制御**: git push をSSoTとし、競合はretry + 手動解決（W-005）

---

## 1. イベントフロー図

### 1.1 全体フロー

```
┌────────────────────┐     ┌────────────────────┐
│   GitHub.com UI    │     │  CLI (Engineer)     │
│   (PM/PMO)         │     │                     │
│                    │     │  teraflow <command>  │
│  Issue作成/close   │     │  → YAML更新          │
│  ラベル付与        │     │  → git commit+push   │
│  PRマージ          │     │                     │
│  Discussion投稿    │     │                     │
└────────┬───────────┘     └──────────┬──────────┘
         │ Webhook                     │ git push
         ▼                             ▼
┌────────────────────────────────────────────────┐
│              GitHub Repository (SSoT)           │
│                                                │
│  .github/project-state.yml  ← 状態ファイル      │
│  .github/teraflow.yml      ← 設定ファイル      │
│  .github/groups.yml        ← グループ定義      │
│  .teraflow/changelog/      ← 変更ログ(JSONL)   │
│  .teraflow/schedule.yml    ← スケジュール      │
└────────┬───────────────────────────────────────┘
         │ on: trigger (issues, pull_request, etc.)
         ▼
┌────────────────────────────────────────────────┐
│            GitHub Actions Runner               │
│                                                │
│  1. git pull origin main --rebase              │
│  2. Install teraflow (release binary)          │
│  3. teraflow <command> --config .github/...    │
│  4. git add → commit → push                    │
│     └─ conflict? → retry (max 3)               │
│     └─ 3回失敗 → Issueコメント通知             │
│  5. 結果通知（ラベル更新 / コメント投稿）       │
│                                                │
│  concurrency: teraflow-state-update            │
│  cancel-in-progress: false                     │
└────────────────────────────────────────────────┘
```

### 1.2 主要イベント別フロー

#### フェーズ開始（Issue作成 → phase start）

```
PM: Issueテンプレート「フェーズ開始申請」作成
  │ labels: [phase-transition, group:auth]
  ▼
GitHub: issues.opened イベント発火
  ▼
TF-003 (phase-transition.yml) 起動
  │
  ├── テンプレート判定（bodyからgroup/phase解析）
  ├── teraflow phase start <phase> --group <group>
  ├── git commit + push (.github/project-state.yml)
  └── Issueコメント: "✅ フェーズ <phase> を開始しました"
```

#### ステージ遷移（ラベル付与 → stage advance）

```
PM/Engineer: Issueにラベル stage:release 付与
  │ W-001: ラベル主判定
  ▼
GitHub: issues.labeled イベント発火
  ▼
TF-001 (phase-gate.yml) / TF-003 起動
  │
  ├── ゲート条件チェック（全グループの完了状態確認）
  ├── 条件OK → teraflow stage advance
  ├── 条件NG → Issueコメント: "❌ ゲート条件未達: group-api phase=impl"
  └── git commit + push
```

#### PRマージ（changelog追記）

```
Engineer: PRマージ（タイトル: "feat: add auth module"）
  ▼
GitHub: pull_request.closed (merged=true) イベント発火
  ▼
TF-006 (changelog-update.yml) 起動
  │
  ├── PRタイトルからtype解析（feat: → feature）
  ├── teraflow changelog add --type feature --summary "add auth module"
  ├── git commit + push (.teraflow/changelog/YYYY-MM.jsonl)
  └── PRコメント: "📝 changelog に記録しました"
```

#### フェーズ完了（Issue close → phase complete）

```
PM: Issue「フェーズ完了報告」をクローズ
  │ labels: [phase-complete, group:auth]
  ▼
GitHub: issues.closed イベント発火
  ▼
TF-003 (phase-transition.yml) 起動
  │
  ├── 成果物確定状態チェック（確定ラベル有無）
  ├── teraflow phase complete --group <group>
  ├── changelog記録
  ├── git commit + push
  └── Issueコメント: "✅ フェーズ <phase> 完了。次: <next_phase>"
```

#### Discussion AI対話（要件整理Agent）

```
PM: Discussion「要件議論」カテゴリに投稿
  ▼
GitHub: discussion.created イベント発火
  ▼
TF-004 (req-agent.yml) 起動
  │
  ├── Discussionカテゴリ判定（要件議論のみ対象）
  ├── teraflow agent assign --type requirements --context <discussion_body>
  ├── AI APIへプロンプト送信 → 要件整理結果を取得
  └── Discussionコメント: AI整理結果 + 確認質問
```

#### CI失敗（自動修正Agent）

```
CI: check_run completed (conclusion=failure)
  ▼
TF-008 (ci-fix-agent.yml) 起動
  │
  ├── 失敗ログ解析
  ├── teraflow agent assign --type ci-fix --context <failure_log>
  ├── AI APIで修正コード生成
  └── 修正PR自動作成 or Issueコメントで修正案提示
```

---

## 2. 提供Actionsワークフロー一覧

Phase2で追加する16本のワークフローを3カテゴリに分類。

### 2.1 カテゴリ別一覧

#### カテゴリA: ゲート・遷移系（コア操作）

| ID | ファイル名 | トリガー | teraflowコマンド | 処理概要 |
|----|-----------|---------|-----------------|---------|
| TF-001 | teraflow-phase-gate.yml | `on: pull_request` | `teraflow scan` | フェーズ/ステージ整合性チェック。PR内の変更が現在フェーズの成果物に適合するか検証 |
| TF-002 | teraflow-permission-guard.yml | `on: pull_request` | `teraflow role check` | CODEOWNERS + ロール権限チェック。権限不足時はレビュー要求 |
| TF-003 | teraflow-phase-transition.yml | `on: issues (closed, labeled)` | `teraflow phase start/complete/skip` | フェーズ遷移の自動実行。Issueテンプレートからgroup/phase解析 |
| TF-005 | teraflow-artifact-finalize.yml | `on: issues (labeled: 確定)` | `teraflow artifact finalize` | 成果物確定 → ファイルリンク生成 + 確定カウント更新 |

#### カテゴリB: AI Agent系

| ID | ファイル名 | トリガー | teraflowコマンド | 処理概要 |
|----|-----------|---------|-----------------|---------|
| TF-004 | teraflow-req-agent.yml | `on: discussion (created, comment)` | `teraflow agent assign --type requirements` | Discussion上のAI要件整理。要件議論カテゴリのみ対象 |
| TF-007 | teraflow-implement-agent.yml | `on: issues (labeled: agent-implement)` | `teraflow agent assign --type implement` | 実装Agent起動。PR自動作成 |
| TF-008 | teraflow-ci-fix-agent.yml | `on: check_run (completed, failure)` | `teraflow agent assign --type ci-fix` | CI失敗の自動修正Agent |
| TF-009 | teraflow-review-agent.yml | `on: pull_request (opened, synchronize)` | `teraflow agent assign --type review` | 自動コードレビューAgent |
| TF-010 | teraflow-conflict-agent.yml | `on: pull_request (labeled: conflict)` | `teraflow agent assign --type conflict` | コンフリクト自動解消Agent |
| TF-014 | teraflow-incident-agent.yml | `on: issues (labeled: incident-*)` | `teraflow agent assign --type incident` | 障害調査Agent起動 |

#### カテゴリC: 定期実行・自動記録系

| ID | ファイル名 | トリガー | teraflowコマンド | 処理概要 |
|----|-----------|---------|-----------------|---------|
| TF-006 | teraflow-changelog-update.yml | `on: pull_request (closed, merged)` | `teraflow changelog add` | PRマージ時のchangelog自動追記 |
| TF-011 | teraflow-stuck-monitor.yml | `on: schedule (cron: 0 9 * * 1-5)` | `teraflow monitor stuck` | 平日9時に停滞Issue検出・通知 |
| TF-012 | teraflow-rework-impact.yml | `on: issues (labeled: rework-*)` | `teraflow rework create` | 手戻り検知・影響分析の自動実行 |
| TF-013 | teraflow-dashboard-deploy.yml | `on: push (main), schedule (daily)` | `teraflow dashboard generate` | ダッシュボード自動生成 + GitHub Pages デプロイ |
| TF-015 | teraflow-maintenance-agent.yml | `on: schedule (cron: 0 0 * * 0)` | `teraflow harness score` | 週次保守スコアリング + 閾値超過時にIssue起票 |
| TF-016 | teraflow-schedule-predict.yml | `on: schedule (cron: 0 9 * * 1)` | `teraflow schedule predict` | 週次スケジュール予測更新 |

### 2.2 ワークフロー共通構造

```yaml
name: teraflow-<operation>
on:
  <trigger>:
    types: [<subtypes>]

concurrency:
  group: teraflow-state-update
  cancel-in-progress: false

permissions:
  contents: write      # project-state.yml 更新のため
  issues: write        # コメント投稿、ラベル操作
  pull-requests: write # PR操作、レビュー
  discussions: write   # Discussion コメント（TF-004のみ）

jobs:
  teraflow:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          token: ${{ secrets.GITHUB_TOKEN }}

      - name: Install teraflow
        run: |
          VERSION=$(curl -s https://api.github.com/repos/taka-sho/teraflow/releases/latest | jq -r .tag_name)
          curl -sL "https://github.com/taka-sho/teraflow/releases/download/${VERSION}/teraflow_${VERSION#v}_linux_amd64.tar.gz" | tar xz
          chmod +x teraflow && sudo mv teraflow /usr/local/bin/

      - name: Pull latest
        run: git pull origin main --rebase

      - name: Execute teraflow
        run: teraflow <command> --config .github/teraflow.yml
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      - name: Commit and push with retry
        run: |
          git config user.name "teraflow[bot]"
          git config user.email "teraflow[bot]@users.noreply.github.com"
          git add .github/project-state.yml .teraflow/
          git diff --cached --quiet && exit 0  # 変更なしならスキップ
          git commit -m "teraflow: <operation description>"
          MAX_RETRY=3
          for i in $(seq 1 $MAX_RETRY); do
            if git push origin main; then break; fi
            if [ $i -eq $MAX_RETRY ]; then
              gh issue comment $ISSUE_NUMBER --body "⚠️ State update failed after $MAX_RETRY retries."
              exit 1
            fi
            git pull --rebase origin main
            sleep 5
          done
```

### 2.3 実行パターン分類

| パターン | ワークフロー | 特徴 |
|---------|------------|------|
| A: CLIコマンド実行 | TF-001,002,003,005,006,012 | teraflow CLIバイナリを直接実行。state更新 → commit → push |
| B: AI Agent起動 | TF-004,007,008,009,010,014 | teraflow agent assign → AI API呼び出し → PR/コメント生成 |
| C: 定期レポート | TF-011,013,015,016 | schedule cron → teraflow実行 → 結果をGitHub Pages / Issue / コメントに出力 |

---

## 3. CLI ↔ GitHub.com操作対応表

全Phase1 CLIコマンドに対するGitHub.com等価操作の完全マッピング。

### 3.1 完全対応表

| CLIコマンド | GitHub.com操作 | 自動化ワークフロー | 対応区分 |
|------------|---------------|-----------------|---------|
| **初期化系** | | | |
| `teraflow init` | — | — | CLI専用 |
| `teraflow setup labels` | — | — | CLI推奨 |
| `teraflow setup actions` | — | — | CLI推奨 |
| `teraflow setup templates` | — | — | CLI推奨 |
| `teraflow setup codeowners` | — | — | CLI推奨 |
| **ステージ管理** | | | |
| `teraflow stage current` | Projectsボード閲覧 | — | 閲覧のみ |
| `teraflow stage advance` | ラベル `stage:XXX` 付与 | TF-001 | 完全代替 |
| `teraflow stage activate` | ラベル `stage:activate` 付与 | TF-001 | 完全代替 |
| `teraflow stage freeze` | ラベル `stage:freeze` 付与 | TF-001 | 完全代替 |
| **フェーズ管理** | | | |
| `teraflow phase current` | Projectsボード閲覧 | — | 閲覧のみ |
| `teraflow phase advance` | Issue「フェーズ完了報告」close | TF-003 | 完全代替 |
| `teraflow phase skip` | Issue「フェーズスキップ申請」作成 | TF-003 | 完全代替 |
| `teraflow phase status` | Projectsボード + Issue一覧 | — | 閲覧のみ |
| **グループ管理** | | | |
| `teraflow group propose` | — | — | CLI推奨（AI分析） |
| `teraflow group define` | — | — | CLI推奨 |
| `teraflow group finalize` | — | — | CLI推奨 |
| `teraflow group rebalance` | — | — | CLI推奨（AI分析） |
| `teraflow group status` | Projectsボード（フィルタ） | — | 閲覧のみ |
| `teraflow group list` | groups.yml 閲覧 | — | 閲覧のみ |
| `teraflow group member add` | groups.yml 編集 → PR | TF-006 | PR経由 |
| `teraflow group member remove` | groups.yml 編集 → PR | TF-006 | PR経由 |
| **ロール管理** | | | |
| `teraflow role list` | CODEOWNERS閲覧 | — | 閲覧のみ |
| `teraflow role show` | CODEOWNERS閲覧 | — | 閲覧のみ |
| `teraflow role check` | PR時自動チェック | TF-002 | 自動実行 |
| **サイクル管理** | | | |
| `teraflow cycle create` | Issue「サイクル作成」 | TF-003 | 完全代替 |
| `teraflow cycle list` | Projectsボード閲覧 | — | 閲覧のみ |
| `teraflow cycle status` | Projectsボード閲覧 | — | 閲覧のみ |
| **手戻り管理** | | | |
| `teraflow rework create` | Issue「手戻り申請」作成 | TF-012 | 完全代替 |
| `teraflow rework approve` | ラベル `rework-approved` 付与 | TF-012 | 完全代替 |
| `teraflow rework status` | Issue一覧（reworkラベルフィルタ） | — | 閲覧のみ |
| `teraflow rework stats` | ダッシュボード（GitHub Pages） | TF-013 | 自動生成 |
| **障害管理** | | | |
| `teraflow incident create` | Issue「インシデント報告」作成 | TF-014 | 完全代替 |
| `teraflow incident status` | Issue一覧（incidentラベルフィルタ） | — | 閲覧のみ |
| `teraflow incident stats` | ダッシュボード（GitHub Pages） | TF-013 | 自動生成 |
| **スケジュール** | | | |
| `teraflow schedule show` | ダッシュボード（GitHub Pages） | TF-013 | 自動生成 |
| `teraflow schedule predict` | 週次自動実行 | TF-016 | 自動実行 |
| `teraflow schedule set` | schedule.yml 編集 → PR | TF-006 | PR経由 |
| **変更ログ** | | | |
| `teraflow changelog add` | PRマージ（タイトル規則） | TF-006 | 自動実行 |
| `teraflow changelog show` | ダッシュボード + changelog閲覧 | — | 閲覧のみ |
| `teraflow changelog stats` | ダッシュボード（GitHub Pages） | TF-013 | 自動生成 |
| **ダッシュボード** | | | |
| `teraflow dashboard generate` | 自動デプロイ | TF-013 | 自動実行 |
| **保守** | | | |
| `teraflow harness score` | 週次自動実行 | TF-015 | 自動実行 |
| `teraflow harness report` | ダッシュボード（GitHub Pages） | TF-013 | 自動生成 |
| **トレーサビリティ** | | | |
| `teraflow trace scan` | PR時自動実行 | CI-006 | 自動実行 |
| `teraflow trace impact` | — | — | CLI専用 |
| `teraflow trace graph` | — | — | CLI専用 |
| **環境チェック** | | | |
| `teraflow doctor` | — | — | CLI専用 |
| **AI操作** | | | |
| `teraflow agent assign` | Issue/Discussionからラベル付与 | TF-004,007,008,009,010,014 | 完全代替 |
| `teraflow agent status` | Actions実行ログ閲覧 | — | 閲覧のみ |

### 3.2 対応区分サマリ

| 対応区分 | コマンド数 | 説明 |
|---------|----------|------|
| CLI専用 | 8 | init, setup×4, doctor, trace impact/graph |
| CLI推奨 | 4 | group propose/define/finalize/rebalance（AI分析を伴うため） |
| 完全代替 | 12 | Issueテンプレート/ラベルで完全に等価操作可能 |
| 自動実行 | 8 | schedule/cron/PRマージで自動実行 |
| 閲覧のみ | 12 | Projectsボード/GitHub Pages/ファイル閲覧で情報取得 |
| PR経由 | 3 | YAML直接編集→PR→マージで更新 |

---

## 4. データ整合性設計（W-005対応）

### 4.1 並行制御モデル

Phase2ではCLI（エンジニア）とActions（GitHub.comユーザー操作起点）の2系統が同一の `.github/project-state.yml` を読み書きする。ADR-008に基づく楽観的並行制御を実装する。

```
              ┌──────────────────────────────────────────┐
              │         .github/project-state.yml        │
              │              (SSoT on GitHub)             │
              └──────────┬──────────────┬────────────────┘
                         │              │
              ┌──────────▼──────┐  ┌────▼─────────────┐
              │  CLI User       │  │  GitHub Actions   │
              │  (Engineer)     │  │  (PM操作起点)     │
              │                 │  │                   │
              │ git pull        │  │ git pull --rebase │
              │ teraflow <cmd>  │  │ teraflow <cmd>    │
              │ git commit      │  │ git commit        │
              │ git push        │  │ git push (retry)  │
              │   └→ conflict?  │  │   └→ retry ×3     │
              │     手動解決    │  │   └→ 3回失敗:通知 │
              └─────────────────┘  └───────────────────┘
```

### 4.2 競合シナリオと対応策

| シナリオ | 発生条件 | 検知方法 | 対応策 |
|---------|---------|---------|-------|
| Actions同士 | 複数イベント同時発火 | concurrency group | 逐次実行で競合回避 |
| CLI vs Actions | CLI操作中にActionsが起動 | git push 失敗 | Actions: 3回retry。CLI: 手動pull+再操作 |
| CLI vs CLI | 2名のエンジニアが同時操作 | git push 失敗 | 手動pull+再操作（通常のgitフロー） |

### 4.3 Actions側のリトライ実装

```bash
# 全teraflowワークフロー共通のpush処理
MAX_RETRY=3
for i in $(seq 1 $MAX_RETRY); do
  git pull --rebase origin main
  teraflow <command> --config .github/teraflow.yml
  git add .github/project-state.yml .teraflow/
  git diff --cached --quiet && { echo "No changes"; exit 0; }
  git commit -m "teraflow: <operation>"
  if git push origin main; then
    echo "Push succeeded on attempt $i"
    break
  fi
  if [ $i -eq $MAX_RETRY ]; then
    gh issue comment $ISSUE_NUMBER \
      --body "⚠️ State update failed after $MAX_RETRY retries. Manual intervention required."
    exit 1
  fi
  echo "Push failed on attempt $i, retrying in 5s..."
  sleep 5
done
```

### 4.4 concurrencyグループ設計

```yaml
# 全teraflow Actionsワークフロー共通
concurrency:
  group: teraflow-state-update
  cancel-in-progress: false
```

- **group名**: `teraflow-state-update` — 全ワークフローで統一
- **cancel-in-progress: false**: 先行ジョブをキャンセルしない（データ不整合防止）
- **効果**: Actions間の同時実行を排除。キューに入った順に逐次実行

### 4.5 データファイル別の競合リスク

| ファイル | 更新頻度 | 競合リスク | 理由 |
|---------|---------|----------|------|
| `.github/project-state.yml` | 高 | **高** | ステージ/フェーズ遷移で頻繁に更新。主要な競合対象 |
| `.github/groups.yml` | 低 | 低 | グループ定義変更は稀 |
| `.github/teraflow.yml` | 低 | 低 | 設定変更は稀 |
| `.teraflow/changelog/*.jsonl` | 中 | **低** | 追記のみ（append）のため競合しにくい |
| `.teraflow/schedule.yml` | 低 | 低 | 週次予測更新のみ |

### 4.6 運用ガイドライン

1. **CLIユーザー向け**: 操作前に `git pull` を実施（ドキュメントに明記）
2. **高頻度同時操作の制限**: 10+ users同時フェーズ遷移は想定外。Phase3（GitHub App + DB）で対応
3. **失敗通知**: Actions側の3回リトライ失敗時、対象IssueにBot コメントで通知
4. **手動リカバリ**: 通知を受けたPM/エンジニアが手動で `git pull` → 再操作

---

## 5. Phase2追加コンポーネント

Phase1のシステム構成（design:system-overview）に以下を追加。

### 5.1 コンポーネント図（Phase2追加分）

```
┌─────────────────────────────────────────────────────────────┐
│                    teraflow CLI (Phase1 + Phase2)            │
│                                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │ cmd/     │  │ cmd/     │  │ cmd/     │  │ cmd/     │   │
│  │ phase    │  │ agent    │  │ setup    │  │ monitor  │   │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘   │
│       │              │              │              │         │
│  ┌────▼──────────────▼──────────────▼──────────────▼────┐   │
│  │          internal/core/ (Phase1)                     │   │
│  │   + AgentManager / MonitorManager (Phase2)           │   │
│  └────┬──────────┬──────────┬────────────┬──────────┐   │   │
│       │          │          │            │          │   │   │
│  ┌────▼────┐ ┌───▼────┐ ┌──▼──────┐ ┌───▼─────┐ ┌─▼───┐   │
│  │ store/  │ │ ai/    │ │ codd/   │ │ github/ │ │NEW: │   │
│  │         │ │        │ │         │ │         │ │     │   │
│  │ YAML    │ │Provider│ │Adapter  │ │GH Wrap  │ │acti-│   │
│  │ R/W     │ │Interf. │ │(B-002) │ │(B-004)  │ │ons/ │   │
│  └─────────┘ └────────┘ └─────────┘ └─────────┘ │     │   │
│                                                  │agent│   │
│                                                  └─────┘   │
└─────────────────────────────────────────────────────────────┘

Phase2 追加パッケージ:
  internal/actions/ — Actionsワークフロー生成・テンプレート管理
  internal/agent/   — 10 Agent（要件整理/実装/CI修正/レビュー/等）の実行制御
```

### 5.2 internal/actions/ — Actionsワークフロー管理

```go
// ActionsGenerator — ワークフローYAML生成
type ActionsGenerator struct {
    TemplateDir string  // embed.FSからのテンプレート
}

func (g *ActionsGenerator) GenerateWorkflows(config *model.TeraflowConfig) ([]WorkflowFile, error)
func (g *ActionsGenerator) ValidateWorkflow(path string) error
```

`teraflow setup actions` コマンドでリポジトリの `.github/workflows/` に16ワークフローファイルを生成。

### 5.3 internal/agent/ — Agent実行制御

```go
// AgentManager — AI Agent統合管理
type AgentManager struct {
    AI       ai.Provider
    GitHub   *github.GH
    Store    *store.YAMLStore
}

// AgentType — Phase2で提供する10種のAgent
type AgentType string
const (
    AgentRequirements AgentType = "requirements"  // TF-004
    AgentImplement    AgentType = "implement"      // TF-007
    AgentCIFix        AgentType = "ci-fix"         // TF-008
    AgentReview       AgentType = "review"         // TF-009
    AgentConflict     AgentType = "conflict"       // TF-010
    AgentStuckMonitor AgentType = "stuck-monitor"  // TF-011
    AgentReworkImpact AgentType = "rework-impact"  // TF-012
    AgentIncident     AgentType = "incident"       // TF-014
    AgentMaintenance  AgentType = "maintenance"    // TF-015
    AgentSchedule     AgentType = "schedule"       // TF-016
)

func (m *AgentManager) Assign(agentType AgentType, context AgentContext) (*AgentResult, error)
func (m *AgentManager) Status(runID string) (*AgentStatus, error)
```
