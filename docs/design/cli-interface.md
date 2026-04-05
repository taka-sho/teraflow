---
codd:
  node_id: "design:cli-interface"
  title: "CLIコマンドインターフェース設計"
  depends_on:
    - id: "adr:001-language"
      relation: derives_from
    - id: "adr:002-cli-framework"
      relation: derives_from
    - id: "design:command-dataflow"
      relation: derives_from
    - id: "req:cli-project-mgmt"
      relation: implements
---

# CLIコマンドインターフェース設計

## 1. Phase1対象コマンド一覧

cmd_081 4Phase機能マトリクスのPhase1列に基づく。Phase1はエンジニア向けローカルCLI。

| カテゴリ | コマンド | Phase1対応 |
|---------|---------|-----------|
| 初期化 | `teraflow init` | ○ |
| 初期設定 | `teraflow setup labels\|actions\|templates\|codeowners` | ○（gh経由） |
| ステージ管理 | `teraflow stage current\|advance\|activate\|freeze` | ○ |
| フェーズ管理 | `teraflow phase current\|advance\|skip\|status` | ○ |
| グループ管理 | `teraflow group propose\|define\|finalize\|rebalance\|status\|list` | ○ |
| グループメンバー | `teraflow group member add\|remove\|transfer\|list` | ○ |
| ロール管理 | `teraflow role list\|show\|check` | ○ |
| サイクル管理 | `teraflow cycle create\|list\|status` | ○ |
| スケジュール | `teraflow schedule show\|predict\|set` | ○ |
| 手戻り管理 | `teraflow rework create\|approve\|status\|stats` | ○（gh経由） |
| 障害管理 | `teraflow incident create\|status\|stats` | ○（gh経由） |
| 変更ログ | `teraflow log show\|stats` | ○ |
| ダッシュボード | `teraflow dashboard generate` | ○（HTML出力） |
| ハーネス | `teraflow harness score` | △（手動実行） |
| トレーサビリティ | `teraflow trace scan\|impact\|validate` | ○（CoDD Adapter） |
| ユーティリティ | `teraflow version`, `teraflow help` | ○ |

**Phase1対象外**: `dashboard deploy`（Phase2）、`agent *`（Phase2）、`report *`（Phase2）、Actions連携ワークフロー群

---

## 2. 共通オプション

### グローバルフラグ（全コマンド共通）

```
--config <path>     設定ファイルパス（デフォルト: .github/teraflow.yml）
--format <type>     出力形式: text | json（デフォルト: text）
--verbose, -v       詳細出力
--dry-run           実行せずに変更内容を表示（書き込み系コマンドのみ有効）
--no-color          カラー出力を無効化（CI環境等）
--quiet, -q         エラー以外の出力を抑制
--help, -h          ヘルプ表示
--version           バージョン表示（root のみ）
```

### 環境変数

```
TERAFLOW_CONFIG       --config と同等
TERAFLOW_FORMAT       --format と同等
TERAFLOW_NO_COLOR     --no-color と同等（"1" or "true"）
NO_COLOR              標準的なno-color環境変数（https://no-color.org/）
GH_TOKEN              ghコマンドの認証トークン（gh auth loginが優先）
```

---

## 3. コマンド別インターフェース定義

### 3.1 teraflow init

プロジェクトを初期化し、設定ファイル・ディレクトリ・テンプレートを生成する。

```
teraflow init [flags]

Flags:
  --name <string>          プロジェクト名（必須。対話モードではプロンプト表示）
  --stage <string>         開始ステージ（デフォルト: initial_development）
                           選択肢: initial_development | continuous_improvement
  --non-interactive        対話なしで実行（CI等。--name必須）
  --skip-github            GitHub操作（ラベル作成等）をスキップ
  --repo <string>          GitHubリポジトリURL（省略時は git remote から検出）

Examples:
  teraflow init
  teraflow init --name "my-project" --non-interactive
  teraflow init --name "my-project" --stage continuous_improvement --skip-github
```

#### 生成物

```
.github/
├── teraflow.yml                  # メイン設定ファイル
├── project-state.yml             # プロジェクト状態
├── master-schedule.yml           # マスタースケジュール（空テンプレート）
├── groups.yml                    # グループ定義（空）
├── CODEOWNERS                    # コードオーナー設定
├── ISSUE_TEMPLATE/               # Issueテンプレート 11種
│   ├── requirement.yml
│   ├── requirement-definition.yml
│   ├── basic-design.yml
│   ├── detailed-design.yml
│   ├── implementation.yml
│   ├── test.yml
│   ├── rework.yml
│   ├── incident.yml
│   ├── inquiry.yml
│   ├── maintenance.yml
│   └── improvement-request.yml
└── DISCUSSION_TEMPLATE/
    └── requirement-discussion.yml

.teraflow/
└── changelog/                    # 変更ログディレクトリ

docs/
├── shared/
│   └── 01_requirements/
│       └── index.md
└── golden-principles.md          # Terasoluna黄金原則
```

#### 対話モード出力例

```
$ teraflow init

  teraflow — project initializer

  Project name: my-order-system
  Description: 注文管理システム
  Starting stage: 初期開発

  Detecting GitHub repository... https://github.com/org/my-order-system

  Creating files...
    ✓ .github/teraflow.yml
    ✓ .github/project-state.yml
    ✓ .github/master-schedule.yml
    ✓ .github/groups.yml
    ✓ .github/ISSUE_TEMPLATE/ (11 templates)
    ✓ .teraflow/changelog/
    ✓ docs/shared/01_requirements/index.md
    ✓ docs/guide/golden-principles.md

  Next steps:
    1. git add -A && git commit -m "chore: teraflow init"
    2. git push origin main
    3. teraflow setup labels    # GitHubラベル作成
    4. teraflow schedule set    # スケジュール初期設定
```

#### JSON出力（--format json）

```json
{
  "status": "success",
  "project": {
    "name": "my-order-system",
    "stage": "initial_development",
    "repository": "https://github.com/org/my-order-system"
  },
  "files_created": [
    ".github/teraflow.yml",
    ".github/project-state.yml",
    "..."
  ]
}
```

#### 終了コード

| コード | 意味 |
|--------|------|
| 0 | 成功 |
| 1 | 既にteraflowプロジェクトが初期化済み（.github/teraflow.yml 存在） |
| 2 | ghコマンド未インストール or 未認証（--skip-github なし時） |

---

### 3.2 teraflow setup

GitHub上にラベル・テンプレート・ワークフロー・CODEOWNERSを設定する。`gh`コマンド経由（B-004）。

```
teraflow setup labels [flags]
teraflow setup templates [flags]
teraflow setup actions [flags]
teraflow setup codeowners [flags]

Flags (共通):
  --force               既存を上書き
  --dry-run             実行せず変更内容を表示

teraflow setup labels:
  50+のGitHubラベルを一括作成。ステージ/フェーズ/グループ/分類ラベルを含む。
  → gh label create "stage:initial-dev" --color "0E8A16" --description "..."

teraflow setup templates:
  .github/ISSUE_TEMPLATE/ と .github/DISCUSSION_TEMPLATE/ を再生成。

teraflow setup actions:
  .github/workflows/ にワークフロー群を生成。
  ※ Phase1ではワークフロー定義ファイルの生成のみ。Actions自体はPhase2で有効化。

teraflow setup codeowners:
  groups.yml からCODEOWNERSを自動生成。

Examples:
  teraflow setup labels
  teraflow setup labels --dry-run
  teraflow setup codeowners --force
```

---

### 3.3 teraflow stage

ステージの状態表示・遷移を管理する。

```
teraflow stage current [flags]
teraflow stage advance [flags]
teraflow stage activate <stage-name> [flags]
teraflow stage freeze <stage-name> [flags]

Subcommands:
  current     現在のアクティブステージを表示
  advance     次ステージへ遷移（ゲート条件チェック付き）
  activate    ステージを有効化（保守等の並行ステージ）
  freeze      ステージを凍結（廃止決定時）

teraflow stage current:
  Flags: なし（グローバルフラグのみ）

teraflow stage advance:
  Flags:
    --force         ゲート条件未達でも遷移（警告表示。非推奨）
    --yes, -y       確認プロンプトをスキップ

teraflow stage activate <stage-name>:
  Arguments:
    stage-name      有効化するステージ名
                    選択肢: operation | maintenance | continuous_improvement | retirement

teraflow stage freeze <stage-name>:
  Arguments:
    stage-name      凍結するステージ名
  Flags:
    --reason <string>   凍結理由（変更ログに記録）
```

#### 出力例

```
$ teraflow stage current

  Active Stages:
    ● 初期開発 (initial_development)  — started: 2026-04-01
    ● 保守 (maintenance)              — started: 2026-07-01

$ teraflow stage advance

  Current: 初期開発 (initial_development)
  Target:  移行・リリース (release)

  Gate conditions:
    ✓ All groups completed: integration_test
    ✓ Gate Issue #100 closed
    ✗ CI status: failing (2 jobs)

  ✗ Gate conditions not met. Fix the above issues before advancing.
  Exit code: 10

$ teraflow stage advance --format json
{
  "status": "blocked",
  "current_stage": "initial_development",
  "target_stage": "release",
  "gate_conditions": [
    {"condition": "all_groups_phase", "value": "integration_test", "met": true},
    {"condition": "gate_issue_closed", "value": "#100", "met": true},
    {"condition": "ci_status", "value": "passing", "met": false, "detail": "2 jobs failing"}
  ]
}
```

---

### 3.4 teraflow phase

グループ単位のフェーズ管理。

```
teraflow phase current [flags]
teraflow phase advance [flags]
teraflow phase skip [flags]
teraflow phase status [flags]

Subcommands:
  current     現在フェーズの表示
  advance     次フェーズへ遷移
  skip        フェーズのスキップ（許可されたステージのみ）
  status      フェーズ内の進捗状況

共通Flags:
  --group, -g <string>    対象グループID（advance/skipでは必須、current/statusでは省略時に全グループ表示）

teraflow phase advance:
  Flags:
    --group, -g <string>  対象グループID（必須）
    --force               成果物未確定でも遷移
    --yes, -y             確認プロンプトスキップ

teraflow phase skip:
  Flags:
    --group, -g <string>  対象グループID（必須）
    --phase <string>      スキップするフェーズ名
    --reason <string>     スキップ理由（必須）

teraflow phase status:
  Flags:
    --group, -g <string>  グループ指定（省略時は全グループ）
    --detail              詳細表示（成果物一覧含む）
```

#### 出力例

```
$ teraflow phase current

  Group          Stage          Phase                Status
  ─────          ─────          ─────                ──────
  group-auth     初期開発       実装 (implementation)  in_progress
  group-api      初期開発       基本設計 (basic_design) in_progress
  group-infra    初期開発       要件定義 (req_def)     in_progress

$ teraflow phase advance --group group-auth

  Group: group-auth
  Current: 基本設計 (basic_design)
  Target:  詳細設計 (detailed_design)

  Pre-conditions:
    ✓ Planned artifacts: 5/5 confirmed
    ✓ Sync point: group-api basic_design — completed
    ✓ Role check: user-a has 'developer' role

  Advance group-auth to 詳細設計? (y/n): y

  ✓ Phase advanced: basic_design → detailed_design
  ✓ Changelog entry recorded
  ✓ master-schedule.yml updated

$ teraflow phase status --group group-auth --detail

  Group: group-auth — 基本設計 (basic_design)

  Artifacts:
    Status   File                          Confirmed
    ──────   ────                          ─────────
    ✓        docs/group-auth/03_basic-design/api-spec.md     2026-04-10
    ✓        docs/group-auth/03_basic-design/db-schema.md    2026-04-11
    ✗        docs/group-auth/03_basic-design/sequence.md     —

  Progress: 2/3 confirmed (67%)
  Sync points:
    ● group-api basic_design — waiting (current: req_def)
```

---

### 3.5 teraflow group

グループの提案・定義・確定・メンバー管理。

```
teraflow group propose [flags]
teraflow group define [flags]
teraflow group finalize [flags]
teraflow group rebalance [flags]
teraflow group status [flags]
teraflow group list [flags]
teraflow group member add [flags]
teraflow group member remove [flags]
teraflow group member transfer [flags]
teraflow group member list [flags]

teraflow group propose:
  AI分析により要件からグループ分割案を提示
  Flags:
    --docs <path>         要件ファイルのディレクトリ（デフォルト: docs/shared/01_requirements/）
    --count <int>         提案するグループ数の目安（省略時はAI判断）

teraflow group define:
  仮グループを手動定義
  Flags:
    --id <string>         グループID（必須。英数字+ハイフン）
    --name <string>       グループ表示名（必須）
    --dirs <string>       担当ディレクトリ（カンマ区切り）

teraflow group finalize:
  仮グループを確定（以降のフェーズ遷移でグループ指定が必須に）
  Flags:
    --group, -g <string>  確定するグループID（省略時は全仮グループ）
    --yes, -y             確認プロンプトスキップ

teraflow group rebalance:
  人員変動時の再編成分析（AI利用）
  Flags:
    --reason <string>     再編理由（必須）

teraflow group member add:
  Flags:
    --group, -g <string>  グループID（必須）
    --user <string>       GitHubユーザー名（必須）
    --role <string>       ロール: developer | reviewer | requester | pm（デフォルト: developer）

teraflow group member remove:
  Flags:
    --group, -g <string>  グループID（必須）
    --user <string>       GitHubユーザー名（必須）
    --reason <string>     離任理由

teraflow group member transfer:
  Flags:
    --from <string>       移動元グループID（必須）
    --to <string>         移動先グループID（必須）
    --user <string>       GitHubユーザー名（必須）
```

---

### 3.6 teraflow rework

手戻りの起票・承認・状況確認。GitHub Issue/PR連携あり（B-004b: Phase1でCLIからのissue/PR操作許容）。

```
teraflow rework create [flags]
teraflow rework approve [flags]
teraflow rework status [flags]
teraflow rework stats [flags]

teraflow rework create:
  手戻りIssueを起票し、CoDD影響分析を実行
  Flags:
    --group, -g <string>       対象グループID（必須）
    --target-phase <string>    手戻り先フェーズ（必須）
    --reason <string>          手戻り理由（必須）
    --skip-impact              影響分析をスキップ
  出力: Issue URL、影響分析結果（Green/Amber/Gray件数）

teraflow rework approve:
  手戻りを承認し、フェーズを後退させるPRを作成
  Flags:
    --issue <int>              手戻りIssue番号（必須）
    --yes, -y                  確認プロンプトスキップ
  出力: PR URL、後退先フェーズ

teraflow rework status:
  Flags:
    --group, -g <string>       グループ指定（省略時は全グループ）
    --open                     未解決の手戻りのみ表示

teraflow rework stats:
  手戻り統計の表示
  Flags:
    --group, -g <string>       グループ指定
    --since <date>             集計開始日（YYYY-MM-DD）
```

#### 出力例

```
$ teraflow rework create --group group-auth --target-phase basic_design --reason "API仕様変更"

  Group: group-auth
  Current phase: implementation
  Target phase:  basic_design (2フェーズ後退)

  Running impact analysis...
    Green (直接影響):  3 files
    Amber (間接影響):  5 files
    Gray  (影響なし):  12 files

  Creating GitHub Issue...
  ✓ Issue #42: [Rework] group-auth: basic_design ← API仕様変更
    Labels: rework, group:auth, phase:basic-design
    URL: https://github.com/org/my-project/issues/42

$ teraflow rework approve --issue 42

  Issue #42: [Rework] group-auth: basic_design ← API仕様変更

  This will:
    - Revert group-auth from implementation → basic_design
    - Create a PR with state changes
    - Record phase reversion in changelog

  Proceed? (y/n): y

  ✓ Phase reverted: implementation → basic_design
  ✓ PR #43 created: rework(#42): revert group-auth to basic_design
    URL: https://github.com/org/my-project/pull/43
  ✓ Changelog entry recorded

$ teraflow rework stats

  Rework Statistics (2026-04-01 〜 2026-04-15)

  Group          Total   Open   Resolved   Avg Resolution
  ─────          ─────   ────   ────────   ──────────────
  group-auth     3       1      2          2.5 days
  group-api      1       0      1          1.0 days
  ─────          ─────   ────   ────────
  Total          4       1      3
```

---

### 3.7 teraflow schedule

スケジュール表示・予測・設定。

```
teraflow schedule show [flags]
teraflow schedule predict [flags]
teraflow schedule set [flags]

teraflow schedule show:
  マスタースケジュールを表示
  Flags:
    --group, -g <string>    グループ指定（省略時は全グループ）
    --gantt                 簡易ガントチャート表示（テキスト）

teraflow schedule predict:
  完了予測を算出・表示
  Flags:
    --ai                    AI予測を使用（デフォルト: 加重平均法）
    --group, -g <string>    グループ指定

teraflow schedule set:
  スケジュール初期値の設定
  Flags:
    --group, -g <string>      グループID（必須）
    --phase <string>          フェーズ名（必須）
    --start <date>            開始予定日 YYYY-MM-DD（必須）
    --end <date>              終了予定日 YYYY-MM-DD（必須）
    --artifacts <int>         計画成果物数
```

---

### 3.8 teraflow incident

障害の起票・状況確認（運用ステージ用）。

```
teraflow incident create [flags]
teraflow incident status [flags]
teraflow incident stats [flags]

teraflow incident create:
  障害報告Issueを起票
  Flags:
    --title <string>          障害タイトル（必須）
    --severity <string>       重要度: critical | major | minor（デフォルト: major）
    --description <string>    障害概要

teraflow incident status:
  Flags:
    --open                    未解決の障害のみ

teraflow incident stats:
  Flags:
    --since <date>            集計開始日
    --metric <string>         表示メトリクス: mttr | count | severity（デフォルト: all）
```

---

### 3.9 teraflow log

変更ログの表示・統計。

```
teraflow log show [flags]
teraflow log stats [flags]

teraflow log show:
  Flags:
    --last <int>              直近N件（デフォルト: 20）
    --member <string>         メンバー指定
    --group, -g <string>      グループ指定
    --type <string>           イベントタイプ（phase.advance, rework.create等）
    --from <date>             期間開始日
    --to <date>               期間終了日

teraflow log stats:
  Flags:
    --group, -g <string>      グループ指定
    --since <date>            集計開始日
    --by <string>             集計軸: type | member | group | week（デフォルト: type）
```

---

### 3.10 teraflow dashboard generate

ダッシュボードHTMLを生成する。

```
teraflow dashboard generate [flags]

Flags:
  --output, -o <path>       出力先（デフォルト: .teraflow/dashboard/index.html）
  --no-ai                   AIサマリ生成をスキップ（静的データのみ）
  --open                    生成後にブラウザで開く
```

---

### 3.11 teraflow trace

CoDD Adapter経由の依存分析（B-002: 自前実装）。

```
teraflow trace scan [flags]
teraflow trace impact [flags]
teraflow trace validate [flags]

teraflow trace scan:
  docs/ 配下のfrontmatterから依存グラフを構築・表示
  Flags:
    --docs <path>          ドキュメントルート（デフォルト: docs/）

teraflow trace impact:
  指定ノードの変更影響範囲を表示
  Flags:
    --node <string>        対象node_id（必須）

teraflow trace validate:
  frontmatterの整合性チェック（循環依存、欠損参照等）
```

---

### 3.12 その他Phase1コマンド

```
teraflow role list                     全ロールとアクション一覧
teraflow role show --user <username>   ユーザーのロール・権限表示
teraflow role check --action <action> --user <username>  実行可否チェック

teraflow cycle create --name <string>  改善サイクル作成（継続改善ステージ用）
teraflow cycle list                    サイクル一覧
teraflow cycle status --id <string>    サイクル内進捗

teraflow harness score                 コード品質スコアリング（AI利用、手動実行）
  Flags:
    --path <string>        スコアリング対象パス（デフォルト: src/）

teraflow version                       バージョン表示
teraflow help [command]                ヘルプ表示
```

---

## 4. 出力形式

### テキスト出力（デフォルト）

- テーブル形式: Unicode罫線を使用したaligned columns
- ステータス記号: `✓`（成功）、`✗`（失敗）、`●`（アクティブ）、`○`（非アクティブ）
- カラー: 緑（成功）、赤（エラー/失敗）、黄（警告）、シアン（情報）
- ターミナル幅に応じて自動折り返し

### JSON出力（--format json）

全コマンドが `--format json` に対応。スクリプト連携やCI組み込みに使用。

```json
{
  "status": "success" | "error" | "blocked",
  "data": { ... },
  "errors": [
    {"code": "E1001", "message": "...", "detail": "..."}
  ]
}
```

---

## 5. エラーコード体系

### 体系

```
E0xxx — 一般エラー
E1xxx — 初期化・設定エラー
E2xxx — ステージ・フェーズ管理エラー
E3xxx — グループ・ロールエラー
E4xxx — 手戻り・障害管理エラー
E5xxx — GitHub連携エラー
E6xxx — AI連携エラー
E7xxx — CoDD/トレーサビリティエラー
```

### 主要エラーコード

| コード | 終了コード | メッセージ |
|--------|----------|-----------|
| E0001 | 1 | Not a teraflow project. Run `teraflow init` first. |
| E0002 | 1 | Configuration file not found: {path} |
| E0003 | 1 | Configuration file is invalid: {detail} |
| E1001 | 1 | Project already initialized. Use `--force` to reinitialize. |
| E2001 | 10 | Stage gate conditions not met. |
| E2002 | 10 | Phase advance blocked: artifacts not confirmed ({n}/{total}). |
| E2003 | 10 | Sync point not met: {group} {phase} is not completed. |
| E2004 | 10 | Phase skip not allowed in current stage. |
| E3001 | 11 | Group not found: {group_id}. |
| E3002 | 11 | User not found in group: {user} in {group_id}. |
| E3003 | 11 | Permission denied: {user} lacks role '{role}' for action '{action}'. |
| E3004 | 11 | Group is not finalized. Run `teraflow group finalize` first. |
| E4001 | 12 | Rework issue not found: #{issue_number}. |
| E4002 | 12 | Rework already approved for issue #{issue_number}. |
| E5001 | 20 | GitHub CLI (gh) is not installed. Install: https://cli.github.com |
| E5002 | 20 | GitHub CLI is not authenticated. Run: gh auth login |
| E5003 | 20 | GitHub API error: {detail} |
| E6001 | 30 | AI provider not configured. Set 'ai.default_provider' in teraflow.yml. |
| E6002 | 30 | AI API error: {detail}. Check API key and network. |
| E6003 | 30 | AI response parse error: unexpected format. |
| E7001 | 40 | Circular dependency detected: {node_a} ↔ {node_b}. |
| E7002 | 40 | Missing dependency reference: {node_id} depends on unknown '{ref}'. |

### 終了コード

| 範囲 | 意味 |
|------|------|
| 0 | 成功 |
| 1 | 一般エラー（設定不備、引数不正等） |
| 10-19 | ビジネスロジックエラー（ゲート未達、権限不足等） |
| 20-29 | 外部システムエラー（GitHub連携） |
| 30-39 | AI連携エラー |
| 40-49 | CoDD/トレーサビリティエラー |

---

## 6. 設定ファイル仕様（.github/teraflow.yml）

```yaml
# .github/teraflow.yml — teraflow メイン設定ファイル
version: "1"

project:
  name: "my-project"
  description: "注文管理システム"
  repository: "https://github.com/org/my-project"

# ステージ遷移ゲート条件
gates:
  initial_development:
    advance_to_release:
      - type: all_groups_phase
        value: integration_test
      - type: ci_status
        value: passing
      - type: gate_issue_closed
        value: true
  continuous_improvement:
    advance_to_release:
      - type: regression_test
        value: passing

# 成果物確定トリガー（W-001裁定: ハイブリッド方式）
confirmation:
  trigger: label                      # 主判定: ラベルベース
  keyword_fallback: "確定"            # ラベル不可の場面でのキーワード
  file_patterns:
    - "docs/**/*.md"
    - "src/**/*.ts"

# AI設定（B-003: マルチベンダー対応）
ai:
  default_provider: anthropic
  providers:
    anthropic:
      model: claude-sonnet-4-20250514
      # API key: 環境変数 ANTHROPIC_API_KEY
    openai:
      model: gpt-4o
      # API key: 環境変数 OPENAI_API_KEY
  overrides:                          # 機能別プロバイダ上書き（オプション）
    group_propose: anthropic
    schedule_predict: anthropic

# ダッシュボード設定
dashboard:
  output_dir: ".teraflow/dashboard"
  format: html                        # html | md
  sections:                           # 表示セクションの選択
    - project_overview
    - stage_progress
    - group_phase_matrix
    - schedule_prediction
    - recent_activity
    - rework_stats

# ハーネスエンジニアリング設定
harness:
  score_threshold: 70                 # スコア閾値
  auto_issue: false                   # Phase1では手動。Phase2でtrueに。
```

---

## 7. 設計ノート

### N-001: 設定ファイルパスの不整合

要件定義（sec 4.1）では設定ファイルを `.github/` 配下に配置しているが、ADR-004では `.teraflow/` 配下を設計している。本設計書では要件定義（sec 4.1）に準拠し `.github/` を採用した。

```
要件定義(sec 4.1):  .github/teraflow.yml, .github/project-state.yml
ADR-004:            .teraflow/teraflow.yml, .teraflow/project-state.yml
本設計書:           .github/ に準拠（要件定義が上流）
```

この不整合は殿の裁定を仰ぐべき事項。判断基準:
- `.github/` を使う利点: GitHub UIでの視認性、GitHub App連携時の標準パス
- `.teraflow/` を使う利点: `.github/` の肥大化防止、teraflow専用領域の明確化
- 変更ログは両案とも `.teraflow/changelog/`（JSONLは `.github/` に置くべきでない）

### N-002: --dry-run の適用範囲

書き込み系コマンド（init, stage advance, phase advance, rework create/approve, group define/finalize, member add/remove/transfer, schedule set, setup *）で有効。読み取り系コマンド（current, show, list, status, stats）では無視。

### N-003: 対話モードとnon-interactiveモード

`teraflow init` のみ対話モードを持つ。他のコマンドは全てフラグで完結する（CI/スクリプト連携を優先）。確認プロンプトが出るコマンド（advance, approve等）は `--yes` で省略可能。
