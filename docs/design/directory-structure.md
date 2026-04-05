---
codd:
  node_id: "design:directory-structure"
  title: "ディレクトリ構造設計"
  depends_on:
    - id: "design:system-overview"
      relation: derives_from
    - id: "adr:001-language"
      relation: derives_from
---

# ディレクトリ構造設計

## 概要

teraflowリポジトリのディレクトリ構造を定義する。
Go標準のプロジェクトレイアウトに準拠しつつ、teraflow固有の要件を反映する。

## リポジトリ構造

```
teraflow/
├── main.go                          # エントリポイント
├── go.mod
├── go.sum
├── Makefile                         # ビルド・テスト・リリース
├── .goreleaser.yml                  # GoReleaser設定（バイナリ配布）
├── README.md
├── LICENSE
│
├── cmd/                             # CLIコマンド定義（cobra）
│   ├── root.go                      # ルートコマンド + グローバルフラグ
│   ├── version.go                   # teraflow version
│   ├── init.go                      # teraflow init
│   ├── stage/                       # teraflow stage *
│   │   ├── stage.go                 # 親コマンド
│   │   ├── current.go
│   │   ├── advance.go
│   │   ├── activate.go
│   │   └── freeze.go
│   ├── phase/                       # teraflow phase *
│   │   ├── phase.go
│   │   ├── current.go
│   │   ├── advance.go
│   │   ├── skip.go
│   │   └── status.go
│   ├── group/                       # teraflow group *
│   │   ├── group.go
│   │   ├── propose.go
│   │   ├── define.go
│   │   ├── finalize.go
│   │   ├── rebalance.go
│   │   ├── status.go
│   │   ├── list.go
│   │   └── member/                  # teraflow group member *
│   │       ├── member.go
│   │       ├── add.go
│   │       ├── remove.go
│   │       ├── transfer.go
│   │       └── list.go
│   ├── role/                        # teraflow role *
│   │   ├── role.go
│   │   ├── list.go
│   │   ├── show.go
│   │   └── check.go
│   ├── cycle/                       # teraflow cycle *
│   │   ├── cycle.go
│   │   ├── create.go
│   │   ├── list.go
│   │   └── status.go
│   ├── schedule/                    # teraflow schedule *
│   │   ├── schedule.go
│   │   ├── show.go
│   │   ├── predict.go
│   │   └── set.go
│   ├── rework/                      # teraflow rework *
│   │   ├── rework.go
│   │   ├── create.go
│   │   ├── approve.go
│   │   ├── status.go
│   │   └── stats.go
│   ├── incident/                    # teraflow incident *
│   │   ├── incident.go
│   │   ├── create.go
│   │   ├── status.go
│   │   └── stats.go
│   ├── changelog/                   # teraflow changelog *
│   │   ├── changelog.go
│   │   ├── show.go
│   │   └── stats.go
│   ├── dashboard/                   # teraflow dashboard *
│   │   ├── dashboard.go
│   │   └── generate.go
│   ├── harness/                     # teraflow harness *
│   │   ├── harness.go
│   │   └── score.go
│   └── setup/                       # teraflow setup *
│       ├── setup.go
│       ├── labels.go
│       ├── actions.go
│       ├── templates.go
│       └── codeowners.go
│
├── internal/                        # 内部パッケージ（外部からimport不可）
│   ├── core/                        # ビジネスロジック
│   │   ├── project.go               # ProjectState管理
│   │   ├── stage.go                 # StageManager
│   │   ├── phase.go                 # PhaseManager
│   │   ├── group.go                 # GroupManager
│   │   ├── role.go                  # RoleManager
│   │   ├── cycle.go                 # CycleManager
│   │   ├── schedule.go              # ScheduleManager
│   │   ├── rework.go                # ReworkManager
│   │   ├── incident.go              # IncidentManager
│   │   ├── changelog.go             # ChangelogManager
│   │   ├── dashboard.go             # DashboardGenerator
│   │   ├── harness.go               # HarnessScorer
│   │   ├── gate.go                  # GateEvaluator（ステージ遷移ゲート）
│   │   └── setup.go                 # SetupExecutor
│   │
│   ├── model/                       # データモデル（構造体定義）
│   │   ├── project_state.go         # ProjectState, Stage, PhaseHistory
│   │   ├── group.go                 # Group, Member, SyncPoint
│   │   ├── schedule.go              # Schedule, Prediction
│   │   ├── config.go                # TeraflowConfig (teraflow.yml)
│   │   ├── changelog.go             # ChangelogEntry (30種のイベント型)
│   │   ├── rework.go                # ReworkRequest, ImpactReport
│   │   ├── incident.go              # Incident, IncidentStats
│   │   ├── role.go                  # Role, Permission, ActionMatrix
│   │   ├── label.go                 # Label定義 (50+)
│   │   └── template.go              # IssueTemplate (11種)
│   │
│   ├── store/                       # データ永続化
│   │   ├── yaml_store.go            # YAML R/W (project-state, groups, schedule等)
│   │   ├── changelog_writer.go      # JSONL追記
│   │   ├── config_loader.go         # teraflow.yml読み込み（Viper統合）
│   │   └── frontmatter.go           # Markdown frontmatter パーサー
│   │
│   ├── ai/                          # AI連携
│   │   ├── provider.go              # Provider interface
│   │   ├── config.go                # プロバイダ設定
│   │   ├── providers/
│   │   │   ├── anthropic.go         # Claude API
│   │   │   ├── openai.go            # OpenAI API
│   │   │   ├── gemini.go            # Gemini API
│   │   │   └── local.go             # ローカルLLM (Ollama)
│   │   └── prompt/
│   │       ├── builder.go           # プロンプト構築
│   │       └── templates/           # embed.FS でバイナリ埋め込み
│   │           ├── group_propose.tmpl
│   │           ├── group_rebalance.tmpl
│   │           ├── schedule_predict.tmpl
│   │           ├── impact_analysis.tmpl
│   │           ├── harness_score.tmpl
│   │           └── dashboard_summary.tmpl
│   │
│   ├── codd/                        # CoDD Adapter (B-002)
│   │   ├── analyzer.go              # 依存グラフ構築・影響分析
│   │   ├── graph.go                 # DependencyGraph, Node, Edge
│   │   └── validator.go             # frontmatter整合性チェック
│   │
│   ├── github/                      # GitHub連携 (B-004)
│   │   ├── gh.go                    # ghコマンドラッパー
│   │   ├── issue.go                 # Issue操作
│   │   ├── pr.go                    # PR操作
│   │   ├── label.go                 # Label操作
│   │   └── discussion.go            # Discussion操作
│   │
│   └── output/                      # 出力フォーマッター
│       ├── table.go                 # テーブル出力
│       ├── json.go                  # JSON出力
│       └── yaml.go                  # YAML出力
│
├── testdata/                        # テスト用フィクスチャ
│   ├── project-state/               # project-state.ymlのテストデータ
│   ├── groups/                      # groups.ymlのテストデータ
│   ├── schedule/                    # master-schedule.ymlのテストデータ
│   └── frontmatter/                 # frontmatter解析テスト用Markdown
│
├── docs/                            # 要件定義・設計書（codd frontmatter付き）
│   ├── index.md                     # ドキュメントインデックス
│   ├── guide/                       # ガイド（運用・概念・要件整理）
│   │   ├── concepts.md              # req:teraflow-overview
│   │   ├── 01_cli-and-project-management.md
│   │   ├── 02_github-labels-issues.md
│   │   ├── 03_schedule-rework-operations.md
│   │   ├── 04_group-roles-changelog.md
│   │   └── 05_traceability-cicd-roadmap.md
│   ├── requirements/                # 要件文書
│   │   ├── 06_non-functional-requirements.md
│   │   └── 07_phase2-requirements.md
│   ├── tutorial/                    # ハンズオンチュートリアル
│   │   ├── 01-first-project.md
│   │   └── 05-ci-cd-integration.md
│   ├── adr/                         # Architecture Decision Records
│   │   ├── ADR-001-implementation-language.md
│   │   ├── ADR-002-cli-framework.md
│   │   ├── ADR-003-ai-integration.md
│   │   └── ADR-004-data-github-integration.md
│   └── design/                      # 設計書
│       ├── 04_implementation-engineering.md
│       ├── system-overview.md
│       ├── directory-structure.md
│       └── command-dataflow.md
│
└── .teraflow/                       # teraflow管理データ（ユーザーPJ用テンプレート）
    └── templates/                   # initで生成されるテンプレート群
        ├── teraflow.yml.tmpl
        ├── project-state.yml.tmpl
        └── labels.yml.tmpl
```

## 設計判断

### cmd/ vs internal/core/ の分離

- **cmd/**: cobra のコマンド登録・フラグ定義・出力フォーマットのみ
- **internal/core/**: ビジネスロジック。テスト容易性のためcmd/から分離
- cmd/ のファイルは薄いラッパーに徹する:

```go
// cmd/stage/advance.go（例）
func newAdvanceCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "advance",
        Short: "Advance to next stage",
        RunE: func(cmd *cobra.Command, args []string) error {
            mgr := core.NewStageManager(store, gates)
            result, err := mgr.Advance()
            // ... output.Print(result, format)
        },
    }
}
```

### internal/ の使用

Go の `internal/` は外部パッケージからのimportを禁止する。
teraflowはCLIツールであり、ライブラリとして外部提供する想定はないため、全ロジックを `internal/` に配置する。

### model/ の独立

model/ は純粋なデータ構造の定義のみ。依存なし（standard library以外のimportなし）。
他の全パッケージから参照される共通型定義。

### testdata/ の配置

Go 標準の `testdata/` ディレクトリ（go tool がビルド対象から除外）を使用。
各パッケージ内ではなくトップレベルに統合し、テスト間で共有するフィクスチャを管理する。

### embed.FS によるテンプレート埋め込み

プロンプトテンプレート（internal/ai/prompt/templates/）はGoの `//go:embed` でバイナリに埋め込む。
配布時に外部ファイル依存なし = シングルバイナリの原則を維持。

## テスト構造

```
各パッケージに *_test.go を配置:

internal/core/stage_test.go          # ステージ遷移のユニットテスト
internal/core/phase_test.go          # フェーズ管理のユニットテスト
internal/core/group_test.go          # グループ操作のユニットテスト
internal/store/yaml_store_test.go    # YAML R/Wのユニットテスト
internal/codd/analyzer_test.go       # 依存分析のユニットテスト
internal/github/gh_test.go           # ghラッパーのユニットテスト（mock）
internal/ai/providers/*_test.go      # AIプロバイダのユニットテスト（mock）

# 統合テスト
cmd/init_test.go                     # teraflow init のE2E
cmd/stage/advance_test.go            # stage advance のE2E
```

テスト時のAI/GitHub外部依存はinterface経由でモック差し替え。
