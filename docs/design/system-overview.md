---
codd:
  node_id: "design:system-overview"
  title: "システム構成・コンポーネント間関係"
  depends_on:
    - id: "adr:001-language"
      relation: derives_from
    - id: "adr:002-cli-framework"
      relation: derives_from
    - id: "adr:003-ai-integration"
      relation: derives_from
    - id: "adr:004-data-github"
      relation: derives_from
---

# システム構成・コンポーネント間の関係

## 概要

teraflow Phase1は、エンジニア向けローカルCLIとして以下のコンポーネントで構成される。
Go + cobra で実装し、ghコマンドラップでGitHub操作を行う。

## コンポーネント一覧（Phase1）

```
┌─────────────────────────────────────────────────────────┐
│                    teraflow CLI                          │
│                                                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │ cmd/     │  │ cmd/     │  │ cmd/     │  ...         │
│  │ init     │  │ stage    │  │ phase    │              │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘              │
│       │              │              │                    │
│  ┌────▼──────────────▼──────────────▼────┐              │
│  │          internal/core/               │              │
│  │   ProjectState / StageManager /       │              │
│  │   PhaseManager / GroupManager         │              │
│  └────┬──────────┬──────────┬────────────┘              │
│       │          │          │                            │
│  ┌────▼────┐ ┌───▼────┐ ┌──▼──────┐ ┌───────────┐     │
│  │internal/│ │internal│ │internal/│ │ internal/ │     │
│  │ store/  │ │ /ai/   │ │ codd/   │ │ github/   │     │
│  │         │ │        │ │         │ │           │     │
│  │ YAML    │ │Provider│ │Adapter  │ │ GH Wrap   │     │
│  │ R/W     │ │Interface│ │(B-002) │ │ (B-004)   │     │
│  └────┬────┘ └───┬────┘ └──┬──────┘ └─────┬─────┘     │
│       │          │          │              │            │
└───────┼──────────┼──────────┼──────────────┼────────────┘
        │          │          │              │
   ┌────▼────┐ ┌───▼────┐ ┌──▼──────┐ ┌────▼─────┐
   │.teraflow│ │AI API  │ │docs/    │ │ gh CLI   │
   │ /*.yml  │ │(Claude │ │*.md     │ │ → GitHub │
   │ /*.jsonl│ │ etc.)  │ │frontmat.│ │   API    │
   └─────────┘ └────────┘ └─────────┘ └──────────┘
```

## コンポーネント詳細

### 1. cmd/ — CLIコマンド層

| 役割 | cobra のコマンド定義。ユーザー入力のパース・バリデーション |
|------|-------|
| 依存先 | internal/core/ |
| 責務 | フラグ解析、引数バリデーション、出力フォーマット（table/json/yaml）、エラー表示 |
| 設計原則 | ビジネスロジックを持たない。core層に処理を委譲 |

Phase1で実装するコマンド群:
- `init` — プロジェクト初期化
- `stage current/advance/activate/freeze` — ステージ管理
- `phase current/advance/skip/status` — フェーズ管理
- `group propose/define/finalize/rebalance/status/list/member *` — グループ管理
- `role list/show/check` — ロール管理
- `cycle create/list/status` — サイクル管理
- `schedule show/predict/set` — スケジュール管理
- `rework create/approve/status/stats` — 手戻り管理
- `incident create/status/stats` — 障害管理
- `changelog show/stats` — 変更ログ
- `dashboard generate` — ダッシュボード生成
- `setup labels/actions/templates/codeowners` — GitHub初期設定

### 2. internal/core/ — ビジネスロジック層

| 役割 | プロジェクト管理のドメインロジック |
|------|------|
| 依存先 | internal/store/, internal/ai/, internal/codd/, internal/github/ |
| 責務 | ステージ遷移ロジック、フェーズ管理、グループ操作、ゲート条件評価、スケジュール予測 |

主要コンポーネント:

```go
// ProjectState — プロジェクト全体の状態管理
type ProjectState struct {
    store  *store.YAMLStore
    state  *model.ProjectState
}

// StageManager — ステージ遷移・ゲート条件
type StageManager struct {
    state  *ProjectState
    gates  *GateEvaluator
}

// PhaseManager — フェーズ管理（グループ別）
type PhaseManager struct {
    state  *ProjectState
    groups *GroupManager
    log    *ChangelogWriter
}

// GroupManager — グループ管理
type GroupManager struct {
    state  *ProjectState
    ai     ai.Provider        // group propose/rebalance用
    log    *ChangelogWriter
}

// ScheduleManager — スケジュール予測
type ScheduleManager struct {
    state  *ProjectState
    ai     ai.Provider        // predict用
    codd   *codd.Analyzer     // 影響分析
}

// ReworkManager — 手戻り管理
type ReworkManager struct {
    state  *ProjectState
    codd   *codd.Analyzer     // 影響分析
    gh     *github.GH         // Issue/PR操作
    log    *ChangelogWriter
}
```

### 3. internal/store/ — データ永続化層

| 役割 | YAMLファイルの読み書き |
|------|------|
| 依存先 | なし（go-yaml, ファイルシステムのみ） |
| 責務 | .teraflow/ 配下のYAMLファイル群のCRUD操作、JSONL変更ログの追記 |

```go
// YAMLStore — YAML R/W
type YAMLStore struct {
    RootDir string   // .teraflow/ のパス
}

func (s *YAMLStore) LoadProjectState() (*model.ProjectState, error)
func (s *YAMLStore) SaveProjectState(state *model.ProjectState) error
func (s *YAMLStore) LoadGroups() (*model.Groups, error)
func (s *YAMLStore) SaveGroups(groups *model.Groups) error
func (s *YAMLStore) LoadSchedule() (*model.Schedule, error)
func (s *YAMLStore) SaveSchedule(schedule *model.Schedule) error

// ChangelogWriter — JSONL追記
type ChangelogWriter struct {
    Dir string   // .teraflow/changelog/
}

func (w *ChangelogWriter) Append(entry model.ChangelogEntry) error
```

### 4. internal/ai/ — AI連携層

| 役割 | LLMプロバイダとの通信 |
|------|------|
| 依存先 | 外部API（Anthropic, OpenAI等） |
| 責務 | Provider Interface実装、プロンプト構築、応答パース |
| 設計 | ADR-003に詳述 |

### 5. internal/codd/ — CoDD Adapter層

| 役割 | frontmatterベースの依存分析 |
|------|------|
| 依存先 | docs/ 配下のMarkdownファイル |
| 責務 | frontmatter解析、依存グラフ構築、影響分析 |
| 設計原則 | B-002: codd-devバイナリに依存しない。YAML frontmatterの解析を自前実装 |

```go
// Analyzer — CoDD adapter
type Analyzer struct {
    RootDir string
}

func (a *Analyzer) ScanDependencies() (*DependencyGraph, error)
func (a *Analyzer) Impact(nodeID string) ([]AffectedNode, error)
func (a *Analyzer) Validate() ([]ValidationError, error)
```

### 6. internal/github/ — GitHub連携層

| 役割 | ghコマンドのラッパー |
|------|------|
| 依存先 | gh CLI（外部バイナリ） |
| 責務 | Issue/PR/Label操作、認証チェック |
| 設計 | ADR-004に詳述 |

## コンポーネント間の依存関係

```
cmd/ ──depends──▶ internal/core/
                      │
                      ├──depends──▶ internal/store/
                      ├──depends──▶ internal/ai/
                      ├──depends──▶ internal/codd/
                      └──depends──▶ internal/github/
```

依存の方向は常に上位→下位。循環依存なし。

### 依存ルール

1. **cmd/ → core/のみ**: cmd層はstore/ai/codd/github/を直接呼ばない
2. **core/ → 4つの下位層**: core層が各下位層を統合
3. **下位層は相互依存しない**: store/ai/codd/github/は互いに独立
4. **外部依存は下位層で吸収**: AI API, gh CLI, ファイルシステムへのアクセスは下位層に閉じる

## Phase2以降の拡張

| Phase | 追加コンポーネント |
|-------|-----------------|
| Phase2 | internal/actions/ — GitHub Actionsワークフロー生成・管理 |
| Phase2 | internal/agent/ — 10エージェントの実行制御 |
| Phase3 | internal/app/ — GitHub App認証・Webhook処理 |
| Phase4 | internal/api/ — SaaS Web API |

Phase2以降もコンポーネントは internal/ 配下に追加され、core/が統合する構造は維持される。
