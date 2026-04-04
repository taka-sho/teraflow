---
codd:
  node_id: "adr:004-data-github"
  title: "ADR-004 データ管理・GitHub連携設計"
  depends_on:
    - id: "req:teraflow-overview"
      relation: implements
    - id: "req:github-labels-issues"
      relation: implements
    - id: "adr:001-language"
      relation: derives_from
---

# ADR-004: データ管理・GitHub連携設計

## ステータス

提案（Proposed）

## コンテキスト

teraflowはGitHubリポジトリをSingle Source of Truth（SSoT）として、全てのプロジェクトデータをYAMLファイルで管理する。GitHub操作はghコマンドラップ方式（B-004）で実装する。

### 設計制約

- **SSoT裁定**: 全PhaseにおいてGitHubリポジトリが正。CLIはローカルYAML操作→git push→GitHub上が正
- **B-004裁定**: GitHub操作はghコマンドラップ。直接API呼び出しはしない
- **B-004b裁定**: Phase1: CLIのissue/PR操作許容（エンジニア向け）。Phase2以降: GitHub.com主体
- **B-002裁定**: CoDD概念のみ採用。frontmatter解析はadapter patternで独自実装

## データファイル設計

### ファイル一覧と役割

```
.teraflow/                          # teraflow管理ディレクトリ
├── teraflow.yml                    # メイン設定ファイル（sec21）
├── project-state.yml               # プロジェクト状態（sec1.4, 5.2）
├── master-schedule.yml             # マスタースケジュール（sec11）
├── groups.yml                      # グループ定義（sec15.2）
├── changelog/                      # 変更ログ（sec16）
│   └── YYYY-MM.jsonl               # 月別JSONL
├── templates/                      # Issueテンプレート（sec9）
│   └── *.yml
└── labels.yml                      # ラベル定義（sec8）
```

### project-state.yml（プロジェクト状態）

```yaml
# .teraflow/project-state.yml
project:
  name: "my-project"
  created_at: "2026-04-01T00:00:00Z"

stages:
  - id: initial_development
    status: active          # active | completed | frozen
    started_at: "2026-04-01"
    groups:
      - id: group-auth
        current_phase: implementation
        phase_history:
          - phase: requirements_definition
            started_at: "2026-04-01"
            completed_at: "2026-04-05"
          - phase: external_design
            started_at: "2026-04-05"
            completed_at: "2026-04-10"
          - phase: implementation
            started_at: "2026-04-10"
      - id: group-api
        current_phase: external_design

lifecycle:
  current_stage: initial_development
  stage_history:
    - stage: initial_development
      activated_at: "2026-04-01"
```

### master-schedule.yml（スケジュール）

```yaml
# .teraflow/master-schedule.yml
schedule:
  baseline_date: "2026-04-01"
  target_completion: "2026-09-30"

groups:
  group-auth:
    phases:
      requirements_definition:
        planned_start: "2026-04-01"
        planned_end: "2026-04-05"
        actual_start: "2026-04-01"
        actual_end: "2026-04-05"
        planned_artifacts: 3
        confirmed_artifacts: 3
      external_design:
        planned_start: "2026-04-05"
        planned_end: "2026-04-15"
        actual_start: "2026-04-05"
        planned_artifacts: 5
        confirmed_artifacts: 2
  group-api:
    phases:
      # ...

predictions:
  last_updated: "2026-04-10T10:00:00Z"
  method: "weighted_average"     # CoDD impact反映
  group_predictions:
    group-auth:
      estimated_completion: "2026-08-15"
      confidence: 0.75
```

### groups.yml（グループ定義）

```yaml
# .teraflow/groups.yml
groups:
  - id: group-auth
    name: "認証基盤グループ"
    status: finalized          # proposed | defined | finalized
    scope:
      directories:
        - "src/auth/"
        - "src/middleware/auth/"
      labels:
        - "group:auth"
    members:
      - github: "user-a"
        role: developer
      - github: "user-b"
        role: reviewer
    sync_points:
      - depends_on: group-api
        phase: external_design
        reason: "API仕様の確定が必要"

  - id: group-api
    name: "API設計グループ"
    status: finalized
    scope:
      directories:
        - "src/api/"
      labels:
        - "group:api"
    members:
      - github: "user-c"
        role: developer
```

### teraflow.yml（メイン設定: sec21）

```yaml
# .teraflow/teraflow.yml
version: "1.0"
project_name: "my-project"

# ステージ遷移ゲート条件
gates:
  initial_development:
    advance_to_release:
      - all_groups_phase: "integration_test"
      - ci_status: passing
  continuous_improvement:
    advance_to_release:
      - regression_test: passing

# 成果物確定フロー設定
confirmation:
  trigger: label       # label | keyword（W-001ハイブリッド裁定）
  keyword_fallback: "確定"   # ラベル不可時のキーワード
  file_patterns:
    - "docs/**/*.md"
    - "src/**/*.ts"

# AI設定
ai:
  default_provider: anthropic
  providers:
    anthropic:
      model: claude-sonnet-4-20250514
    openai:
      model: gpt-4o

# ダッシュボード設定
dashboard:
  output_dir: ".teraflow/dashboard"
  format: html
```

## GitHub連携設計（ghコマンドラップ）

### 設計方針

```go
// internal/github/gh.go

// GH はghコマンドのラッパー
type GH struct {
    // WorkDir はghコマンドの実行ディレクトリ
    WorkDir string
}

// RunGH はghコマンドを実行し結果を返す
func (g *GH) RunGH(args ...string) (string, error) {
    cmd := exec.CommandContext(ctx, "gh", args...)
    cmd.Dir = g.WorkDir
    output, err := cmd.CombinedOutput()
    // ... エラーハンドリング
    return string(output), nil
}
```

### コマンドマッピング（Phase1）

| teraflow機能 | gh コマンド |
|-------------|-----------|
| `setup labels` | `gh label create` × N |
| `setup templates` | ファイル生成 + `git push` |
| `setup codeowners` | CODEOWNERS生成 + `git push` |
| `rework create` | `gh issue create --label "rework,group:X"` |
| `rework approve` | `gh pr create --base main` |
| `incident create` | `gh issue create --label "incident"` |
| `dashboard deploy` | `gh pages deploy` (手動Phase1, 自動Phase2) |

### gh依存チェック

```go
// teraflow init 実行時にghの存在とログイン状態を確認
func CheckGHAvailable() error {
    // 1. gh がインストールされているか
    if _, err := exec.LookPath("gh"); err != nil {
        return fmt.Errorf("gh CLI is not installed. Install: https://cli.github.com")
    }
    // 2. 認証済みか
    if err := runGH("auth", "status"); err != nil {
        return fmt.Errorf("gh is not authenticated. Run: gh auth login")
    }
    return nil
}
```

## SSoTフロー

```
[ユーザー操作]
    │
    ▼
[teraflow CLI]
    │
    ├──(1) ローカルYAML更新
    │      .teraflow/project-state.yml 等を更新
    │
    ├──(2) git commit
    │      変更をコミット（自動 or 手動）
    │
    ├──(3) git push
    │      GitHubリポジトリに反映
    │
    └──(4) gh コマンド（必要時）
           Issue/PR/Label操作

[GitHub リポジトリ] ← SSoT（正のデータ）
```

### 排他制御（W-005対策）

Phase1（ローカルCLI）では排他制御は不要（単一ユーザー操作前提）。
Phase2以降で並行操作が発生する場合:

- **YAML**: git のマージ機構に委ねる。コンフリクト発生時は手動解決
- **JSONL（変更ログ）**: 追記のみのためコンフリクトしにくい（sec16の設計が適切）
- **project-state.yml**: 楽観的ロック（last_updated比較）を検討

## CoDD Adapter（B-002準拠）

```go
// internal/codd/adapter.go

// Analyzer はfrontmatterベースの依存分析を行う
type Analyzer struct {
    // RootDir はドキュメントルート
    RootDir string
}

// ScanDependencies はfrontmatterからnodes/edgesを構築する
func (a *Analyzer) ScanDependencies() (*DependencyGraph, error) {
    // 1. docs/ 配下のMarkdownファイルを走査
    // 2. YAML frontmatter (codd: node_id, depends_on) をパース
    // 3. 有向グラフを構築
    // ※ codd-devには依存しない（B-002）。自前でfrontmatter解析
}

// Impact は指定ノードの変更影響範囲を返す
func (a *Analyzer) Impact(nodeID string) ([]AffectedNode, error) {
    // 1. 依存グラフ上で影響を受けるノードを探索
    // 2. Green(直接影響)/Amber(間接影響)/Gray(影響なし)で分類
}
```

## 決定

1. **SSoT = GitHubリポジトリ**: CLIはローカルYAML操作→git push→GitHub上が正
2. **ghコマンドラップ**: internal/github/gh.go で統一的にラップ
3. **データファイルは`.teraflow/`配下**: project-state.yml, groups.yml, master-schedule.yml等
4. **変更ログはJSONL**: 月別ファイルで追記のみ。コンフリクト耐性を確保
5. **CoDD Adapter**: frontmatter解析を自前実装。codd-devバイナリに依存しない

## 影響

- `teraflow init` は `.teraflow/` ディレクトリと初期YAML群を生成
- 全コマンドは `.teraflow/` 配下のYAMLを読み書きする
- GitHub操作が必要な機能は `internal/github/gh.go` 経由で実行
- Phase3でGitHub App化する際、gh→GitHub App APIへの移行パスが明確
