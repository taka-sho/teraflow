---
codd:
  node_id: "design:v-model-pipeline"
  title: "V字モデル準拠 SLCP-JCF 全フェーズパイプライン統合アーキテクチャ設計書"
  depends_on:
    - id: "design:graphrag-codd"
    - id: "design:requirements-discovery"
    - id: "design:rbac-gate-design"
    - id: "design:cicd-workflows"
    - id: "design:cli-interface"
    - id: "design:internal-packages"
  status: "draft"
  modules:
    - "internal/pipeline"
    - "internal/wave"
    - "internal/validate"
    - "cmd/generate"
    - "cmd/implement"
    - "cmd/validate"
    - "cmd/impact"
    - "cmd/plan"
  review_required: "approve"
  created_at: "2026-04-12"
  updated_at: "2026-04-12"
---

# V字モデル準拠 SLCP-JCF 全フェーズパイプライン統合アーキテクチャ設計書

> cmd_172 — teraflow 独自実装（codd-dev 非依存）

## 1. V字モデルのフェーズ定義と Wave 構成

### 1.1 フェーズ定義

teraflow は一般的な V字モデルに準拠し、以下の 8 フェーズを定義する。

| # | フェーズ | V字位置 | 主な成果物 | 対応するSLCP-JCFアクティビティ |
|---|---------|---------|-----------|-------------------------------|
| 1 | 要件定義 (Requirements) | 左1 | CoDD 要件文書 | ソフトウェア要件定義 |
| 2 | 基本設計 (Basic Design) | 左2 | ADR, システム設計書 | ソフトウェア方式設計 |
| 3 | 詳細設計 (Detailed Design) | 左3 | DB/API/UI 設計書 | ソフトウェア詳細設計 |
| 4 | 実装 (Implementation) | 底 | ソースコード, ビルド成果物 | ソフトウェアコード作成 |
| 5 | 単体テスト (Unit Test) | 右3 | テスト結果, カバレッジ | 単体テスト |
| 6 | 結合テスト (Integration Test) | 右2 | 結合テスト結果 | 結合テスト |
| 7 | システムテスト (System Test) | 右1 | ST結果, 性能計測 | システムテスト |
| 8 | 受入テスト (Acceptance Test) | 右0 | 受入判定結果 | 受入テスト |

### 1.2 V字モデルの対称性

左側（設計）と右側（検証）は対称関係にある。各設計フェーズが対応するテストフェーズの検証基準を生成する。

```
要件定義 ─────────────────────────────── 受入テスト
  │                                         │
  └─ acceptance_criteria ──────────────────→ │ 受入判定基準
  
基本設計 ─────────────────────────────── システムテスト
  │                                         │
  └─ system_test_spec ─────────────────────→ │ ST仕様
  
詳細設計 ─────────────────────────────── 結合テスト
  │                                         │
  └─ integration_test_spec ────────────────→ │ IT仕様
  
実装 ─────────────────────────────────── 単体テスト
  │                                         │
  └─ unit_test_spec ───────────────────────→ │ UT仕様
```

### 1.3 Wave 構成

各設計フェーズ内の成果物生成を Wave（段階）に分割する。Wave は依存順で直列実行される。

**基本設計フェーズの Wave:**

| Wave | 成果物 | 入力 | review_required デフォルト |
|------|--------|------|--------------------------|
| Wave 1 | 受入条件 + ADR | 要件文書 | review |
| Wave 2 | システム設計書 | ADR + 要件 | review |
| Wave 3 | DB設計 + API設計 | システム設計書 | approve |
| Wave 4 | UI/UX 設計（該当時） | API設計 + 要件 | review |
| Wave 5 | 実装計画 | Wave 1-4 全成果物 | approve |

**詳細設計フェーズの Wave:**

| Wave | 成果物 | 入力 | review_required デフォルト |
|------|--------|------|--------------------------|
| Wave 1 | モジュール分割 + IF定義 | 基本設計書 | review |
| Wave 2 | データフロー + シーケンス図 | Wave 1 | review |
| Wave 3 | テスト仕様（UT/IT） | Wave 1-2 | approve |

**実装フェーズの Wave:**

| Wave | 成果物 | 入力 | review_required デフォルト |
|------|--------|------|--------------------------|
| Wave 1 | スケルトンコード + テストハーネス | 詳細設計書 | auto |
| Wave 2 | モジュール実装（並列） | Wave 1 + 詳細設計書 | review |
| Wave 3 | 結合 + リファクタ | Wave 2 全モジュール | review |


## 2. フェーズ遷移の仕組み

### 2.1 フェーズ遷移トリガー

フェーズ遷移は Issue ラベル変更によって発火する。既存の `teraflow-phase-transition.yml` を拡張する。

```
[Issue] ラベル "phase: basic-design" 付与
  │
  ├─ teraflow-phase-transition.yml 発火
  │   ├─ teraflow phase start basic-design
  │   ├─ project-state.yml 更新
  │   └─ CoDD 整合性チェック（後述 §7）
  │
  ├─ teraflow-wave-generate.yml 発火（新規ワークフロー）
  │   ├─ Wave 1 成果物 AI 生成
  │   ├─ PR 作成（review_required に応じて処理）
  │   ├─ Wave 1 完了後 → Wave 2 発火
  │   └─ ...Wave N まで順次
  │
  └─ teraflow-phase-gate.yml（PR 時にゲート検証）
```

### 2.2 フェーズ遷移の前提条件

遷移には以下の条件を満たす必要がある（`teraflow phase start` 内で検証）:

| 条件 | 検証方法 |
|------|----------|
| 前フェーズ完了 | `project-state.yml` の phase status = completed |
| ゲート承認済み | 全 gate conditions が passed |
| CoDD 整合性 | `teraflow validate --phase` が exit 0 |
| RBAC 権限 | 遷移実行者が phase-transition 権限保持 |

### 2.3 フェーズ遷移状態マシン

```
requirements ──→ basic-design ──→ detailed-design ──→ implementation
     │                │                  │                   │
     │                │                  │                   │
     ▼                ▼                  ▼                   ▼
acceptance-test ◀── system-test ◀── integration-test ◀── unit-test
```

各遷移で `project-state.yml` が更新される:

```yaml
# .teraflow/project-state.yml
current_phase: "basic-design"
phase_history:
  - phase: "requirements"
    started_at: "2026-04-10T10:00:00Z"
    completed_at: "2026-04-11T15:00:00Z"
    gate_approved_by: "taka-sho"
    artifacts:
      - node_id: "req:feature-auth"
        path: "docs/requirements/feature-auth.md"
        review_required: "approve"
        status: "merged"
```


## 3. Wave ベース設計書生成

### 3.1 Wave 実行エンジン

Wave エンジンは `internal/wave/` パッケージとして実装する。

```go
// internal/wave/engine.go
type WaveEngine struct {
    provider   ai.Provider
    validator  *validate.Validator
    graphRAG   *graphbridge.Bridge
}

type WaveDefinition struct {
    Number       int                `yaml:"number"`
    Name         string             `yaml:"name"`
    ArtifactType string             `yaml:"artifact_type"`
    Template     string             `yaml:"template"`
    DependsOn    []int              `yaml:"depends_on"`     // 前提 Wave 番号
    Inputs       []string           `yaml:"inputs"`         // CoDD node_id
    ReviewReq    string             `yaml:"review_required"`
    Parallel     bool               `yaml:"parallel"`       // 並列実行可能か
}

type WaveResult struct {
    WaveNumber   int
    Artifacts    []GeneratedArtifact
    PRNumber     int
    Status       string  // "completed" | "pending_review" | "failed"
}

func (e *WaveEngine) ExecuteWave(ctx context.Context, phase string, wave WaveDefinition) (*WaveResult, error)
func (e *WaveEngine) ExecutePhase(ctx context.Context, phase string, waves []WaveDefinition) ([]*WaveResult, error)
```

### 3.2 Wave テンプレート

各 Wave の AI プロンプトテンプレートは `internal/actions/templates/` に追加する。

```
templates/
├── wave-acceptance-criteria.tmpl    # Wave 1: 受入条件
├── wave-adr.tmpl                    # Wave 1: ADR
├── wave-system-design.tmpl          # Wave 2: システム設計
├── wave-db-design.tmpl              # Wave 3: DB設計
├── wave-api-design.tmpl             # Wave 3: API設計
├── wave-ui-design.tmpl              # Wave 4: UI/UX設計
├── wave-implementation-plan.tmpl    # Wave 5: 実装計画
├── wave-module-split.tmpl           # 詳細設計 Wave 1
├── wave-dataflow.tmpl               # 詳細設計 Wave 2
├── wave-test-spec.tmpl              # 詳細設計 Wave 3
└── wave-skeleton.tmpl               # 実装 Wave 1
```

### 3.3 コンテキスト制御（U-shaped context の teraflow 版）

Wave 実行時の AI コンテキストは以下で構成する:

```
[System] Wave テンプレート（~500 tok）
[Context] 依存する CoDD 文書の frontmatter + summary（~1500 tok/文書）
[Context] GraphRAG 関連ノード検索結果（~1000 tok）
[Context] 前 Wave の成果物 summary（~500 tok）
[Input] 入力文書の全文（可変）
[Instruction] 生成指示（~300 tok）
```

1文書あたりのコンテキスト上限: 8000 tok。超過時は GraphRAG の summary を利用して圧縮する。


## 4. 各成果物の CoDD frontmatter スキーマ

### 4.1 拡張 CoDD frontmatter

V字モデルパイプライン用に CoDD frontmatter を拡張する。既存の `node_id`, `title`, `depends_on`, `status` に加え、以下のフィールドを追加する。

```yaml
---
codd:
  node_id: "design:auth-api"
  title: "認証API詳細設計"
  depends_on:
    - id: "design:auth-system"
      relation: implements
    - id: "req:feature-auth"
      relation: implements
  status: "draft"                     # draft | review | approved | implemented | tested
  
  # V-model 拡張フィールド
  phase: "detailed-design"            # V字モデルフェーズ
  wave: 3                             # 生成された Wave 番号
  modules:                            # 実装対象モジュール
    - "internal/auth/handler.go"
    - "internal/auth/middleware.go"
  review_required: "approve"          # auto | review | approve
  review_status: "pending"            # pending | approved | rejected | merged
  reviewer: ""                        # approve 時の承認者
  
  # 検証対称性
  verifies: "req:feature-auth"        # このフェーズが検証する上流文書
  verified_by: "test:it-auth"         # この文書を検証する下流テスト
  
  # 変更追跡
  change_impact: "green"              # green | amber | gray（§9 参照）
  last_validated_at: "2026-04-12"
---
```

### 4.2 フェーズ別 node_id 命名規則

| フェーズ | node_id プレフィクス | 例 |
|---------|--------------------|----|
| 要件定義 | `req:` | `req:feature-auth` |
| 基本設計 | `design:` | `design:auth-system` |
| 詳細設計 | `detail:` | `detail:auth-api` |
| 実装 | `impl:` | `impl:auth-handler` |
| 単体テスト | `test:ut-` | `test:ut-auth-handler` |
| 結合テスト | `test:it-` | `test:it-auth` |
| システムテスト | `test:st-` | `test:st-auth-flow` |
| 受入テスト | `test:at-` | `test:at-feature-auth` |

### 4.3 modules フィールド

`modules` は実装フェーズで並列コード生成する単位を定義する。

```yaml
modules:
  - "internal/auth/handler.go"      # 生成対象ファイル
  - "internal/auth/middleware.go"
  - "internal/auth/token.go"
```

- `teraflow implement` コマンドが modules を読み取り、各モジュールを並列に AI 生成する
- 依存関係がある場合は `depends_on` で制御（同一ファイル内の depends_on とは別管理）


## 5. review_required 設計

### 5.1 3段階レビューレベル

| レベル | 動作 | 用途 |
|--------|------|------|
| `auto` | AI 生成 → CI チェック通過 → 自動マージ | スケルトンコード、定型テスト、小規模変更 |
| `review` | AI 生成 → PR 作成 → 人間レビュー待ち | 設計書、ロジック変更、新機能 |
| `approve` | AI 生成 → PR 作成 → 承認者の明示的 approve 必須 | DB 設計、API 設計、セキュリティ関連、フェーズ遷移承認 |

### 5.2 AI 判定ロジック

review_required のデフォルト値は Wave 定義で設定されるが、以下の条件で自動昇格する:

```go
// internal/pipeline/review.go
func DetermineReviewLevel(artifact GeneratedArtifact, waveDefault string) string {
    level := waveDefault
    
    // 自動昇格ルール
    if artifact.TouchesSecurityModule() {
        level = "approve"  // セキュリティ関連は常に approve
    }
    if artifact.HasExternalAPIChange() {
        level = max(level, "review")  // 外部 API 変更は最低 review
    }
    if artifact.AffectedModuleCount() > 5 {
        level = max(level, "review")  // 広範囲変更は最低 review
    }
    if artifact.IsDBSchemaChange() {
        level = "approve"  // DB スキーマ変更は常に approve
    }
    
    return level
}
```

### 5.3 review_required と GitHub Actions の連携

```yaml
# ワークフロー内での分岐処理
- name: Handle review level
  run: |
    REVIEW_LEVEL=$(yq '.codd.review_required' "$ARTIFACT_PATH")
    case "$REVIEW_LEVEL" in
      auto)
        gh pr merge "$PR_NUMBER" --auto --squash
        ;;
      review)
        gh pr ready "$PR_NUMBER"
        # reviewers は CODEOWNERS から自動決定
        ;;
      approve)
        gh pr ready "$PR_NUMBER"
        gh pr edit "$PR_NUMBER" --add-label "needs-approval"
        # 承認者を明示的にリクエスト
        APPROVERS=$(teraflow rbac list --permission "gate.approve" --format csv)
        gh pr edit "$PR_NUMBER" --add-reviewer "$APPROVERS"
        ;;
    esac
```


## 6. 実装自動化

### 6.1 設計書→コード生成フロー

```
[詳細設計書] modules フィールド読み取り
  │
  ├─ Module A ──→ AI Code Gen ──→ PR-A (review_required: auto)
  ├─ Module B ──→ AI Code Gen ──→ PR-B (review_required: review)
  └─ Module C ──→ AI Code Gen ──→ PR-C (review_required: review)
  
  ↓ 全モジュール完了
  
[結合 PR] 全モジュール統合 ──→ PR-integrate (review_required: approve)
```

### 6.2 並列コード生成エンジン

既存の `teraflow-implement-agent.yml` を拡張する。

```go
// internal/pipeline/implement.go
type ImplementEngine struct {
    provider   ai.Provider
    wave       *wave.WaveEngine
    validator  *validate.Validator
}

type ImplementRequest struct {
    DesignDocPath string            // 詳細設計書パス
    Modules       []ModuleSpec      // 生成対象モジュール
    MaxParallel   int               // 最大並列数（デフォルト: 3）
    DryRun        bool
}

type ModuleSpec struct {
    Path          string            // 出力ファイルパス
    DesignSection string            // 設計書の対応セクション
    DependsOn     []string          // 依存モジュール（先に生成）
    TestSpec      string            // テスト仕様の node_id
}

func (e *ImplementEngine) Generate(ctx context.Context, req ImplementRequest) ([]ImplementResult, error) {
    // 1. 依存グラフから実行順序を決定
    // 2. 依存なしモジュールを並列生成
    // 3. 各モジュールの PR 作成
    // 4. テストスケルトン同時生成
    // 5. CoDD 整合性チェック
}
```

### 6.3 テスト自動生成

実装と同時にテストスケルトンを生成する。V字モデルの対称性により、設計段階で作成されたテスト仕様（§1.2）を入力として使用する。

```
[詳細設計 Wave 3: テスト仕様]
  │
  ├─ UT 仕様 → 実装 Wave 1 でテストスケルトン生成
  ├─ IT 仕様 → 結合テスト時にテストコード生成
  └─ ST 仕様 → システムテスト時にテストスクリプト生成
```


## 7. CoDD 整合性チェック

### 7.1 チェックタイミング

| タイミング | トリガー | チェック内容 |
|-----------|---------|-------------|
| フェーズ遷移時 | `teraflow phase start` | 全依存ノードの status 検証 |
| PR 作成時 | `teraflow-phase-gate.yml` | 変更対象ノードの依存グラフ検証 |
| 手動実行 | `teraflow validate` | 指定範囲の完全性チェック |
| Wave 完了時 | Wave エンジン内部 | 生成成果物の frontmatter 整合性 |

### 7.2 依存グラフ validate

```go
// internal/validate/validator.go
type Validator struct {
    index    *index.Index
    graphRAG *graphbridge.Bridge
}

type ValidationResult struct {
    Valid       bool
    Errors      []ValidationError
    Warnings    []ValidationWarning
    ImpactNodes []string    // 影響を受ける下流ノード
}

type ValidationError struct {
    NodeID      string
    ErrorType   string    // "missing_dependency" | "circular" | "status_mismatch" | "orphan"
    Message     string
    Severity    string    // "error" | "warning"
}

func (v *Validator) ValidatePhaseTransition(from, to string) (*ValidationResult, error) {
    // 1. 現フェーズの全成果物が completed/approved か
    // 2. 依存グラフに循環がないか
    // 3. 全 depends_on ノードが存在するか
    // 4. verifies/verified_by の対称性が保たれているか
    // 5. GraphRAG の graph check と連携
}

func (v *Validator) ValidateArtifact(nodeid string) (*ValidationResult, error) {
    // 個別ノードの整合性チェック
}
```

### 7.3 整合性チェックの段階

```
Level 1: Schema Validation
  - frontmatter の必須フィールド存在確認
  - node_id の命名規則準拠
  - review_required の値が auto/review/approve のいずれか

Level 2: Reference Integrity
  - depends_on の全ノードが存在
  - verifies/verified_by の相互参照
  - modules のファイルパス妥当性

Level 3: Phase Consistency
  - 現フェーズの全成果物の status が適切
  - 前フェーズのゲート通過確認
  - 変更伝播の影響範囲確認（§9）

Level 4: Graph Consistency (GraphRAG 連携)
  - teraflow graph check との統合
  - ノード間の意味的整合性
  - impact 分析結果の反映
```


## 8. GraphRAG 連携

### 8.1 Phase 1-3 実装済み機能との統合

teraflow の GraphRAG は `internal/graph/` + `internal/graphbridge/` で Phase 1-3 が実装済み。V字モデルパイプラインは以下の機能と統合する。

| GraphRAG 機能 | V字モデルでの利用 |
|--------------|-----------------|
| `graph status` | フェーズ遷移前のグラフ健全性確認 |
| `graph check` | CoDD 整合性チェック Level 4 |
| `graph build` | Wave 完了時のインクリメンタルグラフ更新 |
| `graph search` | Wave テンプレートへのコンテキスト供給 |
| `graph impact` | 変更伝播の影響範囲分析 |

### 8.2 統合アーキテクチャ

```
                    ┌──────────────┐
                    │  Wave Engine │
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
              ▼            ▼            ▼
        ┌──────────┐ ┌──────────┐ ┌──────────┐
        │  search  │ │  impact  │ │  build   │
        │ (context)│ │(analysis)│ │ (update) │
        └────┬─────┘ └────┬─────┘ └────┬─────┘
             │            │            │
             └────────────┼────────────┘
                          │
                    ┌─────▼──────┐
                    │ graphbridge│
                    │  .Bridge   │
                    └─────┬──────┘
                          │
                    ┌─────▼──────┐
                    │  Python    │
                    │ subprocess │
                    │ (NetworkX) │
                    └────────────┘
```

### 8.3 Wave 実行時の GraphRAG 利用

```go
// Wave 実行時のコンテキスト取得
func (e *WaveEngine) buildContext(ctx context.Context, wave WaveDefinition) (*WaveContext, error) {
    // 1. 入力 CoDD 文書の読み込み
    inputs := e.loadInputDocuments(wave.Inputs)
    
    // 2. GraphRAG で関連ノード検索
    related, err := e.graphRAG.Search(ctx, graphbridge.SearchRequest{
        Query:    wave.Name,
        NodeIDs:  wave.Inputs,
        MaxNodes: 10,
    })
    
    // 3. impact 分析（変更がある場合）
    if wave.HasPriorChanges() {
        impact, err := e.graphRAG.Impact(ctx, graphbridge.ImpactRequest{
            ChangedNodes: wave.ChangedNodes(),
        })
        // impact 結果をコンテキストに追加
    }
    
    return &WaveContext{Inputs: inputs, Related: related}, nil
}
```

### 8.4 グラフ更新タイミング

| イベント | グラフ操作 |
|---------|-----------|
| Wave 成果物生成後 | `graph build --incremental`（新ノード追加） |
| PR マージ後 | `graph build`（確定ノードとしてステータス更新） |
| フェーズ遷移後 | `graph build --full`（フェーズ全体の整合性再構築） |


## 9. 変更伝播

### 9.1 変更バンド

上流文書が変更された場合、下流文書への影響を 3 段階で分類する。

| バンド | 状態 | 意味 | アクション |
|--------|------|------|-----------|
| **Green** | 整合 | 上流変更なし、または変更の影響なし | なし |
| **Amber** | 要確認 | 上流が変更されたが、影響範囲が限定的 | 自動レビュー生成 → 人間確認 |
| **Gray** | 要再生成 | 上流の重大変更により整合性が破綻 | 下流文書の再生成が必要 |

### 9.2 変更伝播フロー

```
[要件文書 req:feature-auth 変更]
  │
  ├─ teraflow impact --node req:feature-auth
  │   ├─ GraphRAG graph impact 実行
  │   ├─ depends_on 逆引きで下流ノード特定
  │   └─ 各下流ノードの change_impact 判定
  │
  ├─ 結果:
  │   ├─ design:auth-system    → Amber (受入条件の一部変更)
  │   ├─ detail:auth-api       → Gray  (API 仕様が非互換変更)
  │   └─ test:at-feature-auth  → Gray  (受入条件変更)
  │
  └─ アクション:
      ├─ Amber ノード: Issue 作成 "Review impact on design:auth-system"
      └─ Gray ノード: Wave 再実行キュー登録
```

### 9.3 impact 分析エンジン

```go
// internal/pipeline/impact.go
type ImpactAnalyzer struct {
    index    *index.Index
    graphRAG *graphbridge.Bridge
}

type ImpactResult struct {
    ChangedNode   string
    AffectedNodes []AffectedNode
    RegenRequired []string         // Gray: 再生成必要なノード
    ReviewNeeded  []string         // Amber: レビュー必要なノード
}

type AffectedNode struct {
    NodeID    string
    Band      string    // "green" | "amber" | "gray"
    Reason    string
    Distance  int       // 変更元からのグラフ距離
}

func (a *ImpactAnalyzer) Analyze(ctx context.Context, changedNodeID string) (*ImpactResult, error) {
    // 1. GraphRAG graph impact で影響ノード取得
    // 2. 変更の種類（追加/修正/削除）に応じてバンド判定
    // 3. depends_on チェーンを辿り、間接影響も検出
    // 4. distance に応じて影響度を減衰
}
```

### 9.4 自動レビュー生成（Amber バンド）

Amber バンドのノードに対して、AI が変更影響のサマリーを生成し、Issue にコメントする。

```
## Impact Review Required

**Changed**: `req:feature-auth` (要件変更: OAuth2 スコープ追加)
**Affected**: `design:auth-system` (Band: Amber)

### 影響分析
- 既存の Bearer Token 認証は影響なし
- OAuth2 スコープ拡張部分のみ設計追記が必要
- 推定影響度: 低（設計書の §3.2 に追記のみ）

### 推奨アクション
- [ ] design:auth-system §3.2 にスコープ定義を追記
- [ ] 関連テスト仕様 test:it-auth にスコープテスト追加
```


## 10. ワークフロー設計

### 10.1 GitHub Actions 一覧

既存ワークフローの拡張と新規ワークフローを以下に示す。

| # | ワークフロー名 | 種別 | トリガー | 処理内容 |
|---|--------------|------|---------|---------|
| 1 | `teraflow-phase-transition.yml` | 既存拡張 | Issue labeled | フェーズ遷移 + CoDD 整合性チェック |
| 2 | `teraflow-phase-gate.yml` | 既存拡張 | PR opened/sync | ゲート検証 + review_required チェック |
| 3 | `teraflow-wave-generate.yml` | **新規** | workflow_dispatch / phase-transition 完了 | Wave ベース成果物 AI 生成 |
| 4 | `teraflow-implement-agent.yml` | 既存拡張 | Issue labeled | 並列コード生成（modules 単位） |
| 5 | `teraflow-req-agent.yml` | 既存 | Discussion created/comment | 要件定義対話（discovery 連携） |
| 6 | `teraflow-impact-check.yml` | **新規** | CoDD 文書変更時 | 変更伝播の影響分析 + バンド判定 |
| 7 | `teraflow-validate.yml` | **新規** | PR / schedule | CoDD 整合性の定期検証 |
| 8 | `teraflow-wave-review.yml` | **新規** | Wave PR 作成時 | review_required に応じた自動処理 |
| 9 | `teraflow-test-generate.yml` | **新規** | 実装 PR マージ後 | テスト仕様からテストコード生成 |
| 10 | `teraflow-graph-sync.yml` | **新規** | CoDD 文書変更時 | GraphRAG インクリメンタル更新 |

### 10.2 新規ワークフロー詳細

#### teraflow-wave-generate.yml

```yaml
name: Wave Generate
on:
  workflow_dispatch:
    inputs:
      phase:
        description: "Target phase"
        required: true
      wave_number:
        description: "Wave number (0 = all)"
        default: "0"
  workflow_run:
    workflows: ["Phase Transition"]
    types: [completed]

jobs:
  generate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install teraflow
        run: go install ./...
      - name: Determine waves
        run: teraflow plan waves --phase "${{ inputs.phase }}"
      - name: Execute wave
        run: |
          teraflow generate --phase "${{ inputs.phase }}" \
            --wave "${{ inputs.wave_number }}" \
            --create-pr
      - name: Handle review level
        run: |
          # review_required に応じて PR 処理（§5.3 参照）
```

#### teraflow-impact-check.yml

```yaml
name: Impact Check
on:
  push:
    paths: ["docs/**/*.md"]
    branches: [main]

jobs:
  impact:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Detect changed CoDD documents
        id: changed
        run: |
          CHANGED=$(git diff --name-only HEAD~1 HEAD -- docs/ | xargs -I{} teraflow scan --file {} --format node_id)
          echo "nodes=$CHANGED" >> "$GITHUB_OUTPUT"
      - name: Run impact analysis
        run: |
          for NODE in ${{ steps.changed.outputs.nodes }}; do
            teraflow impact --node "$NODE" --output json >> impact-results.json
          done
      - name: Create issues for amber/gray
        run: teraflow impact --apply impact-results.json
```


## 11. CLI コマンド設計

### 11.1 新規コマンド一覧

既存コマンド（scan, phase, gate, doc, index, graph, rbac, hook, trace, summary）に加え、以下を追加する。

| コマンド | サブコマンド | 説明 | 入力 | 出力 |
|---------|------------|------|------|------|
| `generate` | `--phase --wave` | Wave ベース成果物生成 | CoDD 依存文書 | 設計書/コード + PR |
| `implement` | `--design --module --parallel` | 設計書→コード生成 | 詳細設計書 | ソースコード + PR |
| `validate` | `--phase --node --full` | CoDD 整合性検証 | Index + GraphRAG | 検証結果レポート |
| `impact` | `--node --apply` | 変更伝播分析 | 変更ノード | 影響分析結果 |
| `plan` | `waves --phase` | Wave 計画表示 | project-state | Wave 実行計画 |

### 11.2 コマンド詳細

#### teraflow generate

```
teraflow generate --phase basic-design --wave 1 [--create-pr] [--dry-run]

Flags:
  --phase      対象フェーズ (required)
  --wave       Wave 番号 (0=全Wave, default: 0)
  --create-pr  PR を自動作成
  --dry-run    生成結果をstdoutに出力（ファイル書き込みなし）
  --template   カスタムテンプレートパス（上書き用）

Workflow:
  1. project-state.yml から現フェーズ確認
  2. Wave 定義を読み込み
  3. 依存 CoDD 文書 + GraphRAG コンテキスト取得
  4. AI 生成実行
  5. frontmatter 自動付与
  6. teraflow validate 実行
  7. (--create-pr) PR 作成 + review_required 処理
```

#### teraflow implement

```
teraflow implement --design docs/design/auth-api.md [--module internal/auth/handler.go] [--parallel 3]

Flags:
  --design     詳細設計書パス (required)
  --module     特定モジュールのみ生成（省略時: modules 全体）
  --parallel   最大並列数 (default: 3)
  --with-test  テストスケルトン同時生成 (default: true)
  --create-pr  PR を自動作成
  --dry-run    生成結果をstdoutに出力

Workflow:
  1. 詳細設計書の modules フィールド読み取り
  2. 依存グラフから実行順序決定
  3. 並列対象モジュールを同時生成
  4. テストスケルトン生成
  5. teraflow validate 実行
  6. (--create-pr) モジュール別 PR + 統合 PR 作成
```

#### teraflow validate

```
teraflow validate [--phase basic-design] [--node design:auth-system] [--full] [--format json|text]

Flags:
  --phase    フェーズ単位の検証
  --node     個別ノードの検証
  --full     全ノードの完全検証
  --level    検証レベル (1-4, default: 3)
  --format   出力形式 (default: text)

Validation Levels:
  1: Schema (frontmatter 構造)
  2: References (depends_on 存在確認)
  3: Phase (フェーズ整合性)
  4: Graph (GraphRAG 意味的整合性)
```

#### teraflow impact

```
teraflow impact --node req:feature-auth [--apply] [--format json|text]

Flags:
  --node     変更されたノード (required)
  --apply    影響分析結果に基づき Issue 作成/ラベル付与
  --depth    分析深度 (default: 3)
  --format   出力形式

Output:
  - 影響を受けるノード一覧
  - 各ノードのバンド (green/amber/gray)
  - 推奨アクション
```

#### teraflow plan

```
teraflow plan waves --phase basic-design
teraflow plan phases
teraflow plan status

Subcommands:
  waves    指定フェーズの Wave 実行計画表示
  phases   全フェーズの進捗一覧
  status   現在の V字モデル全体状態
```


## 12. 段階的導入計画

### Phase 1: 基盤構築（M サイズ）

**目標**: validate + impact コマンドの実装

| サブタスク | 内容 | 工数 |
|-----------|------|------|
| 1-1 | `internal/validate/` パッケージ実装（Level 1-3） | M |
| 1-2 | `internal/pipeline/impact.go` 実装 | M |
| 1-3 | `cmd/validate.go` CLI 実装 | S |
| 1-4 | `cmd/impact.go` CLI 実装 | S |
| 1-5 | CoDD frontmatter 拡張（phase, wave, modules, review_required, verifies, verified_by, change_impact） | S |
| 1-6 | `teraflow-validate.yml` ワークフロー | S |
| 1-7 | テスト + ドキュメント | M |

**ゲート条件**: `teraflow validate --full` が既存 docs/ 全体で exit 0

### Phase 2: Wave エンジン（L サイズ）

**目標**: Wave ベース成果物生成の実装

| サブタスク | 内容 | 工数 |
|-----------|------|------|
| 2-1 | `internal/wave/` パッケージ実装 | L |
| 2-2 | Wave テンプレート作成（10 テンプレート） | M |
| 2-3 | `cmd/generate.go` CLI 実装 | M |
| 2-4 | `cmd/plan.go` CLI 実装 | S |
| 2-5 | `teraflow-wave-generate.yml` ワークフロー | M |
| 2-6 | `teraflow-wave-review.yml` ワークフロー | S |
| 2-7 | GraphRAG コンテキスト連携 | M |
| 2-8 | テスト + ドキュメント | M |

**ゲート条件**: 単一 Discussion から Wave 1-5 で設計書 5 本が自動生成され、PR が作成されること

### Phase 3: 実装自動化（L サイズ）

**目標**: 設計書→コード並列生成

| サブタスク | 内容 | 工数 |
|-----------|------|------|
| 3-1 | `internal/pipeline/implement.go` 実装 | L |
| 3-2 | `cmd/implement.go` CLI 拡張 | M |
| 3-3 | テスト自動生成連携 | M |
| 3-4 | `teraflow-implement-agent.yml` 拡張 | M |
| 3-5 | `teraflow-test-generate.yml` ワークフロー | M |
| 3-6 | テスト + ドキュメント | M |

**ゲート条件**: 詳細設計書から 3 モジュール並列生成 + テストスケルトン生成が動作すること

### Phase 4: 変更伝播 + 統合（M サイズ）

**目標**: 変更伝播の自動化と全体統合

| サブタスク | 内容 | 工数 |
|-----------|------|------|
| 4-1 | 変更伝播エンジン完成（Amber/Gray 自動処理） | M |
| 4-2 | `teraflow-impact-check.yml` ワークフロー | M |
| 4-3 | `teraflow-graph-sync.yml` ワークフロー | S |
| 4-4 | phase-transition 拡張（整合性チェック統合） | M |
| 4-5 | E2E テスト（全フェーズ通しテスト） | L |
| 4-6 | ドキュメント整備 | M |

**ゲート条件**: 要件定義変更→影響分析→下流 Amber/Gray 判定→Issue 自動作成が動作すること


## 13. 既存機能との統合

### 13.1 統合マトリクス

| 既存機能 | 統合方法 | 変更内容 |
|---------|---------|---------|
| `req-agent` (teraflow-req-agent.yml) | ラベルベース共存 | discovery ラベル時は grill-me モード、requirements ラベル時は従来モード。確認モードで `doc generate` 呼び出し |
| `doc generate` | Wave エンジンの基盤 | `--from-draft` フラグ追加（cmd_171）。`--phase --wave` フラグ追加。既存の Discussion → CoDD 生成パスは維持 |
| `phase-gate` (teraflow-phase-gate.yml) | 拡張 | `teraflow validate` の結果をゲート条件に追加。review_required チェック追加 |
| `phase-transition` (teraflow-phase-transition.yml) | 拡張 | CoDD 整合性チェック追加。Wave 生成ワークフローのトリガー追加 |
| `implement-agent` (teraflow-implement-agent.yml) | 拡張 | modules 単位の並列生成対応。詳細設計書からの自動読み取り |
| `scan` | 内部利用 | validate コマンドが内部で scan を呼び出し |
| `index` | 内部利用 | Wave エンジンが index を参照して依存解決 |
| `graph` (GraphRAG) | 深い統合 | §8 参照。コンテキスト供給、影響分析、グラフ更新 |
| `rbac` | 権限チェック | フェーズ遷移権限、approve 権限、ゲート承認権限 |
| `hook` | イベント連携 | フェーズ遷移時・Wave 完了時のフック発火 |
| `trace` | 追跡 | 全フェーズの成果物トレーサビリティ |
| `summary` | コンテキスト圧縮 | 大量 CoDD 文書の要約キャッシュ利用 |

### 13.2 既存コマンドへの変更（後方互換）

既存コマンドの動作は一切変更しない。新機能は以下の方法で追加する:

- **新フラグ追加**: `doc generate --phase --wave`, `doc generate --from-draft`
- **新コマンド追加**: `generate`, `implement`, `validate`, `impact`, `plan`
- **ワークフロー拡張**: 既存ワークフローに条件分岐を追加（既存パスは影響なし）

### 13.3 skills 連携

| スキル | V字モデルでの役割 |
|--------|-----------------|
| `req.yml` | 要件定義フェーズの対話エンジン |
| `discovery.yml` (cmd_171) | grill-me ベース要件探索 |
| `design.yml` | 基本設計フェーズの ADR 生成支援 |
| `review.yml` | review_required: review 時のレビュー支援 |
| `bugfix.yml` | テストフェーズでの障害分析 |
| `change-request.yml` | 変更伝播時の影響分析支援 |


## 14. grill-me（cmd_171）との接続

### 14.1 要件定義→設計への橋渡し

grill-me（cmd_171 設計書: `docs/design/requirements-discovery.md`）は要件定義フェーズの入力品質を向上させる。V字モデルパイプラインとの接続は以下の通り。

```
[Discussion 作成]
  │
  ├─ requirements ラベル → 従来 req-agent（壁打ちモード）
  │
  └─ discovery ラベル → grill-me モード
      │
      ├─ 決定木ベース質問（7カテゴリ）
      │   scope / functional / nfr / acceptance / risk / dependency / priority
      │
      ├─ .teraflow/discovery/{discussion-N}.yaml（状態ファイル）
      ├─ .teraflow/discovery/{discussion-N}-draft.md（CoDDドラフト累積）
      │
      └─ 全分岐 resolved 判定
          │
          ├─ doc generate --from-draft（AI 再構造化スキップ）
          │   → CoDD 要件文書 PR 作成
          │   → review_required: approve（要件確定は必ず人間承認）
          │
          └─ PR マージ → 要件定義フェーズ完了
              │
              └─ フェーズ遷移トリガー可能状態
                  │
                  └─ Issue でラベル "phase: basic-design" 付与
                      │
                      └─ V字モデルパイプライン発火（§2 参照）
```

### 14.2 grill-me 品質保証

grill-me で生成された CoDD ドラフトは以下の品質保証を受ける:

| チェック | タイミング | 方法 |
|---------|-----------|------|
| 7カテゴリ網羅性 | 全分岐 resolved 判定時 | 決定木の全分岐が answered/skipped |
| frontmatter 整合性 | `doc generate --from-draft` 時 | `teraflow validate --level 1` |
| 依存関係妥当性 | PR 作成時 | `teraflow-phase-gate.yml` |
| 受入条件の具体性 | 人間レビュー時 | review_required: approve |

### 14.3 skipped 分岐の扱い

grill-me で skipped された質問は、CoDD 要件文書のリスクセクションに自動追記される。これにより:

- 意図的に省略された要件が記録される
- 基本設計フェーズの Wave 1（受入条件生成）で、skipped 項目が「未確定リスク」として可視化される
- 後続フェーズで追加要件として復活可能

### 14.4 コンテキスト効率

grill-me の状態ファイルが「圧縮された記憶」として機能するため、要件定義→設計のフェーズ遷移時に:

- Discussion 全コメントの再読み込みが不要
- 状態ファイル（~2000 tok）+ ドラフト（~3000 tok）= ~5000 tok で全要件コンテキストを復元
- GraphRAG search と組み合わせても ~8000 tok 以内に収まる


## 付録 A: 用語集

| 用語 | 定義 |
|------|------|
| Wave | フェーズ内の成果物生成段階。依存順で直列実行される |
| Band | 変更伝播の影響度（Green/Amber/Gray） |
| Gate | フェーズ遷移の承認チェックポイント |
| CoDD | Coherence-Driven Development。YAML frontmatter 付き Markdown 文書 |
| GraphRAG | Graph-based Retrieval Augmented Generation。依存グラフの意味的検索 |
| SLCP-JCF | Software Life Cycle Process - Japan Common Frame |
| node_id | CoDD 文書の一意識別子（`prefix:name` 形式） |
| review_required | 成果物のレビューレベル（auto/review/approve） |

## 付録 B: 設定ファイル例

```yaml
# .teraflow/pipeline.yml
pipeline:
  version: "1.0"
  
  phases:
    requirements:
      waves: []  # 要件定義は grill-me / req-agent で対話的に実施
      gate:
        conditions:
          - "all requirements documents have status: approved"
          - "acceptance criteria defined for all features"
    
    basic-design:
      waves:
        - number: 1
          name: "acceptance-criteria-adr"
          artifact_type: "design"
          template: "wave-acceptance-criteria.tmpl"
          review_required: "review"
        - number: 2
          name: "system-design"
          artifact_type: "design"
          template: "wave-system-design.tmpl"
          depends_on: [1]
          review_required: "review"
        - number: 3
          name: "db-api-design"
          artifact_type: "design"
          template: "wave-db-design.tmpl"
          depends_on: [2]
          review_required: "approve"
        - number: 4
          name: "ui-design"
          artifact_type: "design"
          template: "wave-ui-design.tmpl"
          depends_on: [3]
          review_required: "review"
          optional: true
        - number: 5
          name: "implementation-plan"
          artifact_type: "plan"
          template: "wave-implementation-plan.tmpl"
          depends_on: [1, 2, 3, 4]
          review_required: "approve"
      gate:
        conditions:
          - "all wave artifacts have review_status: approved or merged"
          - "teraflow validate --phase basic-design exits 0"
    
    detailed-design:
      waves:
        - number: 1
          name: "module-split"
          template: "wave-module-split.tmpl"
          review_required: "review"
        - number: 2
          name: "dataflow"
          template: "wave-dataflow.tmpl"
          depends_on: [1]
          review_required: "review"
        - number: 3
          name: "test-spec"
          template: "wave-test-spec.tmpl"
          depends_on: [1, 2]
          review_required: "approve"
      gate:
        conditions:
          - "all modules defined in wave 1 output"
          - "test specifications exist for all modules"
    
    implementation:
      waves:
        - number: 1
          name: "skeleton"
          template: "wave-skeleton.tmpl"
          review_required: "auto"
          parallel: false
        - number: 2
          name: "module-implement"
          template: null  # AI 自由生成
          depends_on: [1]
          review_required: "review"
          parallel: true
        - number: 3
          name: "integration"
          depends_on: [2]
          review_required: "approve"
          parallel: false
      gate:
        conditions:
          - "all modules compile successfully"
          - "unit test coverage >= 80%"
  
  review_escalation:
    security_modules: ["internal/auth/", "internal/rbac/", "internal/crypto/"]
    db_schema_paths: ["migrations/", "internal/db/schema/"]
    external_api_patterns: ["**/api/**", "**/handler/**"]
  
  impact:
    max_depth: 3
    amber_threshold: 0.3   # 変更影響度 30% 以上で Amber
    gray_threshold: 0.7    # 変更影響度 70% 以上で Gray
```
