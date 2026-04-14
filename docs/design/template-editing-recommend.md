---
codd:
  node_id: "design:template-editing-recommend"
  title: "テンプレート編集機能 + 編集履歴レコメンド設計書"
  depends_on:
    - id: "design:goal-driven-discovery"
      relation: extends
  status: draft
  created: "2026-04-14"
  tags:
    - discovery
    - template
    - recommendation
    - editing
---

# テンプレート編集機能 + 編集履歴レコメンド設計書

## 1. 概要

cmd_225 が要件定義テンプレートの「定義と利用」を担当し、cmd_226 はテンプレートの「編集とレコメンド」を担当する。本設計書は cmd_225 設計書（`goal-driven-discovery.md`）のテンプレートスキーマと完全統合する。

### 1.1 cmd_225 との関係

| 責務 | cmd_225 | cmd_226 |
|------|---------|---------|
| テンプレートスキーマ定義 | ✅ `RequirementTemplate` 型 | 拡張フィールド追加 |
| デフォルトテンプレート | ✅ `DefaultRequirementTemplate()` | — |
| テンプレート読み込み | ✅ ファイル/teraflow.yml | — |
| テンプレート編集 | — | ✅ CLI + YAML 直接編集 |
| 編集履歴記録 | — | ✅ `template-history.yaml` |
| レコメンドエンジン | — | ✅ パターン分析 + 提案 |

## 2. 統合テンプレートスキーマ

### 2.1 cmd_225 基本スキーマ（再掲 + cmd_226 拡張）

cmd_225 で定義した `RequirementTemplate` を cmd_226 で拡張する。**追加フィールドのみ**記載。

```yaml
# .teraflow/discovery/requirement-template.yaml
version: "1"
metadata:                          # NEW (cmd_226)
  name: "default"                  # テンプレート名
  description: "汎用要件定義テンプレート"
  source: "builtin"                # builtin | user | recommended
  created_at: "2026-04-14T00:00:00Z"
  updated_at: "2026-04-14T00:00:00Z"

template:
  sections:
    - id: overview
      title: "プロジェクト概要"
      priority: required
      fields:
        - id: overview.purpose
          label: "プロジェクトの目的"
          type: text
          priority: required
          hint: "このプロジェクトが解決する課題は何か"
          prompt_hint: "このシステムが解決しようとしている課題を教えてください"  # NEW (cmd_226)
          origin: builtin          # NEW: builtin | user_added | recommended
          added_at: ""             # NEW: ユーザー追加の場合のタイムスタンプ
        # ... (cmd_225 で定義した全フィールド)
```

### 2.2 拡張フィールドの意味

| フィールド | 型 | 用途 |
|---|---|---|
| `metadata.name` | string | テンプレートの識別名 |
| `metadata.source` | enum | `builtin`=デフォルト, `user`=ユーザー作成, `recommended`=レコメンド由来 |
| `metadata.created_at` | timestamp | テンプレート作成日時 |
| `metadata.updated_at` | timestamp | 最終更新日時 |
| `prompt_hint` | string | LLM質問生成時に使うプロンプトヒント（`hint`はユーザー向け、`prompt_hint`はLLM向け） |
| `origin` | enum | `builtin`=デフォルト項目, `user_added`=ユーザー追加, `recommended`=レコメンドから採用 |
| `added_at` | timestamp | ユーザー/レコメンドで追加された日時 |

### 2.3 Go 型定義の統合

```go
// internal/discovery/template.go

// RequirementTemplate — cmd_225 で定義、cmd_226 で拡張
type RequirementTemplate struct {
    Version  string           `yaml:"version"`
    Metadata TemplateMetadata `yaml:"metadata,omitempty"`  // cmd_226 追加
    Template TemplateBody     `yaml:"template"`
}

type TemplateMetadata struct {
    Name        string `yaml:"name,omitempty"`
    Description string `yaml:"description,omitempty"`
    Source      string `yaml:"source,omitempty"`      // builtin | user | recommended
    CreatedAt   string `yaml:"created_at,omitempty"`
    UpdatedAt   string `yaml:"updated_at,omitempty"`
}

type TemplateBody struct {
    Sections []TemplateSection `yaml:"sections"`
}

type TemplateSection struct {
    ID       string          `yaml:"id"`
    Title    string          `yaml:"title"`
    Priority string          `yaml:"priority"`  // required | recommended | optional
    Fields   []TemplateField `yaml:"fields"`
}

type TemplateField struct {
    ID                    string `yaml:"id"`
    Label                 string `yaml:"label"`
    Type                  string `yaml:"type"`      // text | choice | multi_choice
    Priority              string `yaml:"priority"`
    Hint                  string `yaml:"hint,omitempty"`
    PromptHint            string `yaml:"prompt_hint,omitempty"`  // cmd_226 追加
    DefaultRecommendation string `yaml:"default_recommendation,omitempty"`
    Origin                string `yaml:"origin,omitempty"`       // cmd_226 追加
    AddedAt               string `yaml:"added_at,omitempty"`     // cmd_226 追加
}
```

**互換性:** `Metadata`, `PromptHint`, `Origin`, `AddedAt` はすべて `omitempty` のため、cmd_225 のテンプレートファイルをそのまま読み込み可能。

## 3. テンプレート編集機能

### 3.1 編集方法の比較

| 案 | 概要 | 利点 | 欠点 |
|---|---|---|---|
| **案1: YAML直接編集** | ユーザーがファイルを直接編集 | 最もシンプル、IDE補完可能 | バリデーションなし、書式ミスしやすい |
| **案2: CLIコマンド** | `teraflow template add/remove/list` | バリデーション付き、操作が明確 | CLI実装コスト |
| **案3: Discussion上botに指示** | botがテンプレートを更新 | ノンエンジニア向き | 実装複雑、precision問題 |
| **案1+2** | YAML + CLI | 両方の利点 | 実装コスト中 |

### 3.2 推奨: **案1+2（YAML直接編集 + CLIコマンド）**

**理由:**
1. YAML 直接編集は**必須**。テンプレートファイルはリポジトリにコミットするため、PR レビューの対象にしやすい。
2. CLI コマンドは**バリデーション付き編集**を提供。スキーマ違反を事前検出。
3. 案3 は precision が低く（LLM が意図と異なる編集をするリスク）、実装コストに見合わない。

### 3.3 テンプレートファイルの保存場所と優先順位

```
優先度1: teraflow.yml の discovery.template で指定されたパス
優先度2: .teraflow/discovery/requirement-template.yaml
優先度3: Go コード内 DefaultRequirementTemplate()
```

これは cmd_225 設計書（§3.3）で定義済み。cmd_226 では変更なし。

### 3.4 CLI コマンド設計

```
teraflow template list                     # 現在のテンプレート項目一覧
teraflow template show                     # テンプレート全体を YAML で表示
teraflow template add <section_id> <field>  # フィールド追加
teraflow template remove <field_id>        # フィールド削除
teraflow template validate                 # バリデーション（スキーマ準拠チェック）
teraflow template init                     # デフォルトテンプレートをファイルに出力
teraflow template diff                     # デフォルトとの差分表示
```

#### 3.4.1 `teraflow template list`

```
$ teraflow template list
Source: .teraflow/discovery/requirement-template.yaml

Section: プロジェクト概要 (overview) [required]
  ✅ overview.purpose      プロジェクトの目的     required  builtin
  ✅ overview.background   背景・動機             required  builtin
  ✅ overview.success      成功基準               required  builtin
  🆕 overview.compliance   コンプライアンス要件   required  user_added  (2026-04-10)

Section: ステークホルダー (stakeholders) [required]
  ...

Total: 33 fields (required: 15, recommended: 10, optional: 8)
  Modified from default: +2 added, -1 removed
```

#### 3.4.2 `teraflow template add`

```
$ teraflow template add functional --id func.i18n \
    --label "国際化要件" \
    --priority recommended \
    --hint "対応言語、ロケール、翻訳フローを記載"

Added: func.i18n (国際化要件) to section 'functional' as recommended
Saved: .teraflow/discovery/requirement-template.yaml
```

内部処理:
1. テンプレートファイル読み込み（なければ `template init` でデフォルト生成）
2. ID 重複チェック
3. フィールド追加（`origin: user_added`, `added_at: now`）
4. バリデーション
5. ファイル書き込み
6. 編集履歴記録

#### 3.4.3 `teraflow template remove`

```
$ teraflow template remove func.edge_cases

Removed: func.edge_cases (エッジケース) from section 'functional'
Saved: .teraflow/discovery/requirement-template.yaml
```

**安全策:** `priority: required` のフィールド削除時は確認プロンプト表示。

#### 3.4.4 `teraflow template validate`

```
$ teraflow template validate

Validating .teraflow/discovery/requirement-template.yaml...
✅ Schema valid
✅ All field IDs unique
✅ All sections have at least one field
⚠️  Section 'ux' has no required fields (may be intentional)
✅ No circular dependencies

Result: PASS (1 warning)
```

チェック項目:
- YAML パース可能か
- 必須フィールド（id, label, priority）の存在
- ID の一意性
- priority の値が `required | recommended | optional` のいずれか
- セクションに 1 つ以上のフィールドがあるか

#### 3.4.5 `teraflow template init`

```
$ teraflow template init

Created: .teraflow/discovery/requirement-template.yaml (31 fields, 10 sections)
  Default template from teraflow v0.6.0
  Edit this file to customize your requirements template.
```

デフォルトテンプレートをファイルに出力。既存ファイルがある場合は `--force` が必要。

## 4. 編集履歴の記録

### 4.1 履歴ファイル構造

```yaml
# .teraflow/discovery/template-history.yaml
version: "1"
entries:
  - timestamp: "2026-04-10T15:30:00Z"
    action: add           # add | remove | modify | init
    field_id: overview.compliance
    section_id: overview
    details:
      label: "コンプライアンス要件"
      priority: required
    source: cli            # cli | manual | recommend_accept
    repository: "taka-sho/honya-flow"

  - timestamp: "2026-04-11T09:00:00Z"
    action: remove
    field_id: func.edge_cases
    section_id: functional
    details:
      label: "エッジケース"
      priority: optional
    source: cli
    repository: "taka-sho/honya-flow"

  - timestamp: "2026-04-12T14:00:00Z"
    action: add
    field_id: overview.compliance
    section_id: overview
    details:
      label: "コンプライアンス要件"
      priority: required
    source: manual
    repository: "taka-sho/another-project"
```

### 4.2 記録タイミング

| 操作 | トリガー | 記録方法 |
|------|----------|----------|
| `teraflow template add` | CLI 実行 | 自動記録 |
| `teraflow template remove` | CLI 実行 | 自動記録 |
| YAML 直接編集 | `teraflow template validate` 実行時 or discovery 開始時 | 差分検出 → 自動記録 |
| レコメンド採用 | discovery コメントで「採用」 | 自動記録 (`source: recommend_accept`) |

### 4.3 差分検出（YAML 直接編集対応）

ユーザーが YAML を直接編集した場合、前回の状態との差分を検出して履歴に記録する。

```go
// internal/discovery/history.go

func DetectChanges(previous, current RequirementTemplate) []HistoryEntry {
    prevFields := flattenFields(previous)
    currFields := flattenFields(current)

    var entries []HistoryEntry

    // 追加検出
    for id, field := range currFields {
        if _, exists := prevFields[id]; !exists {
            entries = append(entries, HistoryEntry{
                Action:  "add",
                FieldID: id,
                // ...
            })
        }
    }

    // 削除検出
    for id := range prevFields {
        if _, exists := currFields[id]; !exists {
            entries = append(entries, HistoryEntry{
                Action:  "remove",
                FieldID: id,
                // ...
            })
        }
    }

    // 変更検出（priority, label 等の変更）
    for id, curr := range currFields {
        prev, exists := prevFields[id]
        if !exists { continue }
        if prev.Priority != curr.Priority || prev.Label != curr.Label {
            entries = append(entries, HistoryEntry{
                Action:  "modify",
                FieldID: id,
                // ...
            })
        }
    }

    return entries
}
```

### 4.4 クロスプロジェクト履歴

レコメンドエンジンがクロスプロジェクトの傾向を分析するため、履歴をグローバルに集約する仕組みが必要。

**保存場所:** `~/.config/teraflow/template-history-global.yaml`

```yaml
# ~/.config/teraflow/template-history-global.yaml
version: "1"
entries:
  # 各リポジトリの template-history.yaml からマージされたエントリ
  - timestamp: "2026-04-10T15:30:00Z"
    action: add
    field_id: overview.compliance
    repository: "taka-sho/honya-flow"
    # ...
```

**同期タイミング:**
- `teraflow template add/remove` 実行時にローカル + グローバルの両方に記録
- `teraflow template sync` コマンドで手動同期（ローカル → グローバル）

## 5. レコメンドエンジン設計

### 5.1 レコメンドのトリガー

| トリガー | タイミング | アクション |
|----------|----------|----------|
| 新規 discovery 開始 | Discussion 作成 → 初期質問生成前 | テンプレートレコメンドを提示 |
| テンプレート編集後 | `teraflow template add/remove` | 関連レコメンドを表示（CLI 出力） |

### 5.2 レコメンドロジック

#### パターン 1: 反復追加パターン（Phase 1 実装）

```
条件: 過去 N プロジェクト（デフォルト N=3）で毎回同じフィールドを追加
提案: そのフィールドをデフォルトテンプレートに昇格
```

```go
func FindRepeatedAdditions(globalHistory []HistoryEntry, threshold int) []Recommendation {
    // field_id ごとに追加された repository をカウント
    addedInRepos := map[string]map[string]bool{}  // field_id -> set of repos
    for _, entry := range globalHistory {
        if entry.Action != "add" { continue }
        if addedInRepos[entry.FieldID] == nil {
            addedInRepos[entry.FieldID] = map[string]bool{}
        }
        addedInRepos[entry.FieldID][entry.Repository] = true
    }

    var recs []Recommendation
    for fieldID, repos := range addedInRepos {
        if len(repos) >= threshold {
            recs = append(recs, Recommendation{
                Type:    "promote_to_default",
                FieldID: fieldID,
                Reason:  fmt.Sprintf("過去%dプロジェクトで毎回追加されています", len(repos)),
                // ...
            })
        }
    }
    return recs
}
```

#### パターン 2: 削除後再追加パターン（Phase 1 実装）

```
条件: 同一リポジトリ内で削除 → 再追加された項目
提案: 削除時に警告「この項目は過去に削除後に再追加されています」
```

```go
func FindDeleteReaddPatterns(history []HistoryEntry) []Recommendation {
    // field_id ごとに remove → add の順序があるかチェック
    removed := map[string]bool{}
    var recs []Recommendation
    for _, entry := range history {
        if entry.Action == "remove" {
            removed[entry.FieldID] = true
        }
        if entry.Action == "add" && removed[entry.FieldID] {
            recs = append(recs, Recommendation{
                Type:    "delete_warning",
                FieldID: entry.FieldID,
                Reason:  "この項目は過去に削除後に再追加されています。本当に削除しますか？",
            })
        }
    }
    return recs
}
```

#### パターン 3: 類似プロジェクト参照（Phase 2 — 将来）

```
条件: プロジェクトカテゴリ（Web/CLI/API 等）が類似するリポジトリのテンプレート傾向
提案: 同カテゴリで頻出するフィールドを参照提案
```

Phase 2 ではプロジェクトメタデータ（teraflow.yml の `project.description` やリポジトリのトピックス）からカテゴリを推定し、類似プロジェクトのテンプレート傾向を分析する。

#### パターン 4: 業界標準提案（Phase 3 — 将来）

```
条件: LLM にプロジェクト概要を渡し、業界標準で推奨される項目を提案
提案: 「ヘルスケア系プロジェクトでは HIPAA 準拠項目が推奨されます」等
```

Phase 3 では LLM を活用した高度なレコメンドを実装。トークンコストがかかるため、明示的なオプトインが必要。

### 5.3 レコメンド結果の型定義

```go
// internal/discovery/recommend.go

type RecommendationType string

const (
    RecPromoteToDefault RecommendationType = "promote_to_default"
    RecDeleteWarning    RecommendationType = "delete_warning"
    RecSimilarProject   RecommendationType = "similar_project"
    RecIndustryStandard RecommendationType = "industry_standard"
)

type Recommendation struct {
    Type      RecommendationType `yaml:"type" json:"type"`
    FieldID   string             `yaml:"field_id" json:"field_id"`
    Label     string             `yaml:"label" json:"label"`
    Priority  string             `yaml:"priority" json:"priority"`
    Section   string             `yaml:"section" json:"section"`
    Reason    string             `yaml:"reason" json:"reason"`
    Source    string             `yaml:"source" json:"source"`  // history | similar | llm
    Confidence float64           `yaml:"confidence" json:"confidence"`  // 0.0 - 1.0
}

type RecommendationResult struct {
    Recommendations []Recommendation `yaml:"recommendations" json:"recommendations"`
    AnalyzedRepos   int              `yaml:"analyzed_repos" json:"analyzed_repos"`
    Timestamp       string           `yaml:"timestamp" json:"timestamp"`
}
```

### 5.4 Discussion での提示方法

discovery 開始時（初回 Discussion コメント）にレコメンドがある場合:

```markdown
## 💡 テンプレート提案

過去のプロジェクト分析に基づき、以下の項目の追加を提案します：

| # | 項目 | セクション | 理由 |
|---|------|-----------|------|
| 1 | コンプライアンス要件 | プロジェクト概要 | 過去3プロジェクトで毎回追加 |
| 2 | SLA定義 | 非機能要件 | 過去3プロジェクトで毎回追加 |

**採用する場合**: 番号を返信してください（例: `1,2` で両方採用、`1` のみも可）
**スキップ**: 何も返信せずに次の質問に進みます

---

## 📊 要件充足度
...（通常の discovery 質問に続く）
```

ユーザーが番号を返信した場合:
1. テンプレートにフィールド追加（`origin: recommended`）
2. 編集履歴に記録（`source: recommend_accept`）
3. 充足度トラッキングに反映

## 6. 実装フェーズ

### Phase 1: テンプレート編集基盤（見積: M）

| # | タスク | ファイル |
|---|--------|----------|
| 1-1 | `TemplateMetadata` 型追加（cmd_225 の `RequirementTemplate` 拡張） | `internal/discovery/template.go` |
| 1-2 | `teraflow template list/show/validate/init` CLI 実装 | `cmd/template.go` (新規) |
| 1-3 | `teraflow template add/remove` CLI 実装 | `cmd/template.go` |
| 1-4 | 編集履歴記録 (`HistoryEntry` 型 + ファイル I/O) | `internal/discovery/history.go` (新規) |
| 1-5 | 差分検出 (`DetectChanges()`) | `internal/discovery/history.go` |
| 1-6 | CLI ユニットテスト | `cmd/template_test.go` (新規) |

### Phase 2: レコメンドエンジン基盤（見積: M）

| # | タスク | ファイル |
|---|--------|----------|
| 2-1 | `Recommendation` 型定義 | `internal/discovery/recommend.go` (新規) |
| 2-2 | パターン 1（反復追加）実装 | `internal/discovery/recommend.go` |
| 2-3 | パターン 2（削除後再追加）実装 | `internal/discovery/recommend.go` |
| 2-4 | グローバル履歴同期 (`template sync`) | `internal/discovery/history.go` |
| 2-5 | Discovery 開始時のレコメンド提示 | `teraflow-req-agent.yml` 変更 |
| 2-6 | ユーザーのレコメンド採用処理 | `teraflow-req-agent.yml` 変更 |
| 2-7 | レコメンドユニットテスト | `internal/discovery/recommend_test.go` (新規) |

### Phase 3: 高度なレコメンド（見積: S — 将来）

| # | タスク | ファイル |
|---|--------|----------|
| 3-1 | パターン 3（類似プロジェクト参照） | `internal/discovery/recommend.go` |
| 3-2 | パターン 4（LLM 業界標準提案） | `internal/discovery/recommend.go` |

### 依存関係

```
cmd_225 Phase 1 (テンプレート基盤)
  ↓ RequirementTemplate 型が必要
cmd_226 Phase 1 (テンプレート編集)
  ↓ 編集履歴が必要
cmd_226 Phase 2 (レコメンドエンジン)
  ↓ 将来
cmd_226 Phase 3 (高度なレコメンド)
```

**cmd_225 Phase 1 完了後に cmd_226 Phase 1 を開始可能。** 両者を同一リリース（v0.7.0 等）に含めるのが理想。

## 7. cmd_225 との統合ポイント（まとめ）

| 統合ポイント | 変更対象 | 内容 |
|---|---|---|
| `RequirementTemplate` 型拡張 | `internal/discovery/template.go` | `Metadata`, `PromptHint`, `Origin`, `AddedAt` フィールド追加。全 `omitempty` で後方互換。 |
| テンプレート読み込みパス | `internal/discovery/template.go` | cmd_225 で定義した優先順位をそのまま使用。変更なし。 |
| `BuildInitialTreeFromTemplate()` | `internal/discovery/tree.go` | テンプレートにレコメンド採用フィールドがある場合も Branch 生成。変更なし。 |
| 充足度トラッキング | `internal/discovery/gap.go` | レコメンド採用で追加されたフィールドも充足度対象に含める。cmd_225 の `AnalyzeGap()` で自然対応。 |
| state ファイル | `internal/discovery/state.go` | 変更なし。Branch 構造は同一。 |

**結論:** cmd_226 は cmd_225 の型定義に `omitempty` フィールドを追加するのみで、cmd_225 の動作を一切壊さない。

## 8. リスクと緩和策

| リスク | 影響 | 緩和策 |
|--------|------|--------|
| グローバル履歴の肥大化 | `~/.config/teraflow/` にデータが蓄積 | エントリ上限（デフォルト 1000 件）+ 古いエントリの自動アーカイブ |
| レコメンド精度が低い | 不要な提案が多いとユーザーが無視する習慣がつく | Phase 1 はパターン 1（反復追加, threshold=3）のみ。precision > recall を優先 |
| CLI コマンド追加のリリース | `teraflow template` は新コマンドツリー | 別リリースが必要。cmd_225 Phase 1 のリリースに含める |
| YAML 直接編集とCLI の競合 | 両方で編集すると差分検出が複雑 | 差分検出は常にファイル状態ベース。CLI は write 後に差分検出と同じ処理を実行 |
