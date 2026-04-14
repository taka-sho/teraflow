---
codd:
  node_id: "design:goal-driven-discovery"
  title: "ゴール駆動型 Discovery 根本再設計"
  depends_on:
    - id: "design:requirements-discovery"
      relation: supersedes
    - id: "design:discovery-phase-gate"
      relation: extends
  status: draft
  created: "2026-04-14"
  tags:
    - discovery
    - requirements
    - goal-driven
    - template
---

# ゴール駆動型 Discovery 根本再設計

## 1. エグゼクティブサマリー

### 1.1 現状の問題

現行 discovery は「LLM が自由に質問を生成する」方式。`DefaultTemplate()` に 7 カテゴリ × 約 20 問の固定テンプレートがあるが、これらは「汎用的な質問リスト」であり、**要件定義書に必要な情報を網羅的に収集する**という目的に対して以下の欠陥がある：

| 問題 | 影響 |
|------|------|
| ゴールが不明確 | 何を聞き終われば「完了」なのか定義がない |
| 充足度が不可視 | ユーザーが「あとどれくらい？」を把握できない |
| LLM 依存の質問品質 | モデル変更やプロンプト劣化で質問品質が不安定 |
| 確定タイミングが曖昧 | 必須項目が未回答でも「要求確定」できてしまう |

### 1.2 ゴール駆動型アプローチ

**「要件定義書テンプレート」を事前定義し、テンプレートの充足度をトラッキングしながら質問を生成する」**方式に転換する。

```
現行:  Discussion → LLM自由質問 → 回答蓄積 → 要求確定 → CoDD生成
提案:  Discussion → テンプレート充足度評価 → 未充足項目の質問生成 → 回答蓄積
       → 充足度更新 → 全必須充足で確定通知 → 要求確定 → CoDD生成
```

## 2. cmd_222〜225 依存関係整理

### 2.1 各 cmd の概要と現在の状態

| cmd | 内容 | 状態 | 設計書 |
|-----|------|------|--------|
| cmd_222 | フェーズゲート追加 | ✅ CLOSED (v0.5.24, PR#233) | discovery-phase-gate.md |
| cmd_223 | 自動作成 PR の説明文改善 | 未着手 | — |
| cmd_224 | 要求確定後 bot 質問継続バグ修正 | 未着手 | — |
| cmd_225 | discovery 根本再設計（本設計書） | 設計中 | 本ファイル |

### 2.2 依存関係分析

```
cmd_222 (フェーズゲート) ✅ CLOSED
  ↓ 前提完了
cmd_225 (ゴール駆動型再設計) ← 本設計書
  ↓ 実装後に...
cmd_224 (要求確定後の質問停止)
cmd_223 (PR説明文改善)
```

#### cmd_224: cmd_225 で解決されるか？

**結論: cmd_225 の充足度判定で部分的に解決されるが、独立した先行実装も有効。**

- cmd_225 のゴール駆動設計では、全必須項目充足時に「要求確定可能」通知を出し、確定後は `session.confirmed = true` フラグで質問生成を停止する。これは cmd_224 の要件を包含する。
- ただし cmd_225 の実装規模は大きい（後述: Phase 1〜3）。cmd_224 は「`confirmed` フラグチェックを `NextQuestions()` に追加する」だけの小規模修正（1〜2 時間）で対応可能。
- **推奨: cmd_224 を先行実装。** cmd_225 Phase 1 で `confirmed` フラグの仕組みを活用するため、先行実装は無駄にならない。

#### cmd_223: cmd_225 実装後に変更内容が変わるか？

**結論: cmd_223 は先行実装可能。**

- PR 説明文は `BuildDraftFromState()` の出力を使って生成される。cmd_225 ではテンプレート充足度情報が追加されるが、PR 説明文の構造自体は変わらない。
- cmd_225 実装後に「充足度サマリー」を PR 説明に追加する拡張は必要だが、これは追加作業であり cmd_223 の作業を無効化しない。
- **推奨: cmd_223 は先行実装可能。** cmd_225 後に充足度表示を追記する小タスクを追加。

### 2.3 推奨実行順序

```
1. cmd_224 (要求確定後の質問停止)    ← 小規模、即効性あり、cmd_225 の前提にもなる
2. cmd_223 (PR説明文改善)            ← 独立、先行可能
3. cmd_225 Phase 1 (テンプレート基盤) ← 本設計の中核
4. cmd_225 Phase 2 (ギャップ分析)     ← Phase 1 依存
5. cmd_225 Phase 3 (自動完了判定)     ← Phase 2 依存
```

**cmd_224 と cmd_223 は並行実施可能。** cmd_225 の実装開始前に両方を完了させるのが理想。

## 3. 要件定義書テンプレート設計

### 3.1 テンプレート構造

```yaml
# .teraflow/discovery/requirement-template.yaml（デフォルト）
# ユーザーは teraflow.yml で上書き可能
version: "1"
template:
  sections:
    - id: overview
      title: "プロジェクト概要"
      priority: required    # required | recommended | optional
      fields:
        - id: overview.purpose
          label: "プロジェクトの目的"
          type: text         # text | choice | multi_choice
          priority: required
          hint: "このプロジェクトが解決する課題は何か"
        - id: overview.background
          label: "背景・動機"
          type: text
          priority: required
        - id: overview.success_criteria
          label: "成功基準"
          type: text
          priority: required
          hint: "何をもって成功とするか（定量的指標があれば尚良）"

    - id: stakeholders
      title: "ステークホルダー"
      priority: required
      fields:
        - id: stakeholders.target_users
          label: "対象ユーザー"
          type: text
          priority: required
        - id: stakeholders.decision_makers
          label: "意思決定者"
          type: text
          priority: recommended
        - id: stakeholders.affected_teams
          label: "影響を受けるチーム"
          type: text
          priority: optional

    - id: functional
      title: "機能要件"
      priority: required
      fields:
        - id: functional.core_features
          label: "必須機能"
          type: text
          priority: required
          hint: "ユーザーストーリー形式が望ましい"
        - id: functional.happy_path
          label: "正常系フロー"
          type: text
          priority: required
        - id: functional.error_handling
          label: "エラー処理方針"
          type: text
          priority: recommended
          default_recommendation: "エラーメッセージを表示し、入力を保持してリトライ可能にする"
        - id: functional.edge_cases
          label: "エッジケース"
          type: text
          priority: optional

    - id: non_functional
      title: "非機能要件"
      priority: required
      fields:
        - id: nfr.performance
          label: "パフォーマンス要件"
          type: text
          priority: required
          default_recommendation: "p95 < 500ms"
        - id: nfr.availability
          label: "可用性要件"
          type: text
          priority: recommended
          default_recommendation: "99.9%"
        - id: nfr.scalability
          label: "想定規模"
          type: text
          priority: recommended
        - id: nfr.security
          label: "セキュリティ要件"
          type: text
          priority: required

    - id: constraints
      title: "制約条件"
      priority: required
      fields:
        - id: constraints.technical
          label: "技術的制約"
          type: text
          priority: required
          hint: "言語、フレームワーク、インフラ等の制約"
        - id: constraints.budget
          label: "予算・リソース制約"
          type: text
          priority: recommended
        - id: constraints.timeline
          label: "期限"
          type: text
          priority: required

    - id: acceptance
      title: "成功基準・受入条件"
      priority: required
      fields:
        - id: acceptance.done_definition
          label: "完了の定義"
          type: text
          priority: required
        - id: acceptance.test_scenarios
          label: "必須テストシナリオ"
          type: text
          priority: recommended
          default_recommendation: "正常系E2E + 異常系バリデーション + パフォーマンステスト"

    - id: priorities
      title: "優先順位"
      priority: recommended
      fields:
        - id: priorities.moscow
          label: "MoSCoW分類"
          type: text
          priority: recommended
          hint: "Must/Should/Could/Won't に分類"
        - id: priorities.phasing
          label: "フェーズ分け"
          type: text
          priority: optional

    - id: existing_system
      title: "既存システムとの関係"
      priority: recommended
      fields:
        - id: existing.dependencies
          label: "外部依存"
          type: text
          priority: recommended
        - id: existing.integration
          label: "統合ポイント"
          type: text
          priority: recommended
        - id: existing.migration
          label: "データ移行"
          type: text
          priority: optional

    - id: risks
      title: "リスク・前提条件"
      priority: recommended
      fields:
        - id: risks.identified
          label: "特定済みリスク"
          type: text
          priority: recommended
        - id: risks.assumptions
          label: "前提条件"
          type: text
          priority: recommended
        - id: risks.rollback
          label: "ロールバック計画"
          type: text
          priority: optional
          default_recommendation: "Feature flag で段階的リリース"

    - id: ux
      title: "UI/UX期待"
      priority: optional
      fields:
        - id: ux.design_direction
          label: "デザイン方針"
          type: text
          priority: optional
        - id: ux.accessibility
          label: "アクセシビリティ要件"
          type: text
          priority: optional

    - id: operations
      title: "運用要件"
      priority: optional
      fields:
        - id: ops.monitoring
          label: "監視・アラート"
          type: text
          priority: optional
        - id: ops.deployment
          label: "デプロイ方針"
          type: text
          priority: optional
```

### 3.2 テンプレート項目の分類根拠

| 優先度 | セクション | 根拠 |
|--------|-----------|------|
| **required** | 概要, ステークホルダー, 機能要件, 非機能要件, 制約条件, 受入条件 | 要件定義書として最低限必要。これらが欠けると実装可否の判断ができない |
| **recommended** | 優先順位, 既存システム, リスク | 実装計画の精度に影響。欠けても要件定義は成立するが品質が下がる |
| **optional** | UI/UX, 運用 | プロジェクト種別による。全案件に必須ではない |

**必須フィールド数: 14** / 推奨: 10 / 任意: 7 / 合計: 31

### 3.3 ユーザーカスタマイズ

`teraflow.yml` で上書き可能にする:

```yaml
# .github/teraflow.yml
discovery:
  template: ".teraflow/discovery/custom-template.yaml"  # カスタムテンプレートパス
  # または inline で上書き
  template_overrides:
    sections:
      - id: overview
        fields:
          - id: overview.purpose
            priority: required  # デフォルトと同じでも明示可能
          - id: overview.compliance
            label: "コンプライアンス要件"
            type: text
            priority: required  # カスタム追加
```

**設計方針:**
1. デフォルトテンプレートは Go コード内に `DefaultRequirementTemplate()` としてハードコード
2. `.teraflow/discovery/requirement-template.yaml` があればそちらを優先読み込み
3. `teraflow.yml` の `discovery.template` で外部パス指定も可能
4. 優先順序: `teraflow.yml` 指定 > `.teraflow/discovery/requirement-template.yaml` > デフォルト

## 4. ギャップ分析エンジン設計

### 4.1 アーキテクチャ

```
                    ┌──────────────────┐
Discussion Comment  │  Workflow Entry  │
        │           │  (req-agent.yml) │
        ▼           └──────┬───────────┘
  ┌─────────────┐          │
  │ Load State  │◀─────────┘
  │ (state.go)  │
  └──────┬──────┘
         │
         ▼
  ┌──────────────────┐
  │ Load Template    │  ← DefaultRequirementTemplate() or custom
  │ (template.go)    │
  └──────┬───────────┘
         │
         ▼
  ┌──────────────────────┐
  │ Gap Analysis Engine  │  ← NEW
  │ (gap.go)             │
  │                      │
  │ 1. 各フィールドの     │
  │    充足度を評価       │
  │ 2. 未充足必須項目を   │
  │    特定              │
  │ 3. 質問を生成        │
  │ 4. メタデータ付与     │
  └──────┬───────────────┘
         │
         ▼
  ┌──────────────────┐
  │ Update State     │  充足度情報をstateに追記
  │ (state.go)       │
  └──────┬───────────┘
         │
         ▼
  ┌──────────────────┐
  │ Discussion Reply │  質問 + 充足度表示
  └──────────────────┘
```

### 4.2 ギャップ分析の処理フロー

各ラウンド（Discussion comment 受信時）に以下を実行:

```
1. テンプレート読み込み（全セクション・全フィールド）
2. 現在の state.Tree から回答済みフィールドを特定
   - Branch.ID とテンプレートフィールド ID のマッピング
   - 回答内容の妥当性は LLM で簡易判定（空回答や「未定」は未充足扱い）
3. 未充足フィールドのリスト作成
   - priority: required を優先
   - 依存関係（depends_on）が解決済みのもののみ
4. 未充足フィールドから質問を生成
   - テンプレートの hint/default_recommendation を LLM プロンプトに含める
   - 1 ラウンドあたり最大 3 問（バッチモードでは最大 5 問）
5. 充足度メタデータを state に記録
```

### 4.3 充足度評価ロジック

```go
// gap.go (新規ファイル)

type FulfillmentStatus string

const (
    Fulfilled   FulfillmentStatus = "fulfilled"
    Partial     FulfillmentStatus = "partial"
    Unfulfilled FulfillmentStatus = "unfulfilled"
    Skipped     FulfillmentStatus = "skipped"
)

// TemplateFieldStatus tracks one template field's fulfillment.
type TemplateFieldStatus struct {
    FieldID    string            `yaml:"field_id"`
    Label      string            `yaml:"label"`
    Priority   string            `yaml:"priority"`
    Status     FulfillmentStatus `yaml:"status"`
    BranchID   string            `yaml:"branch_id,omitempty"`  // 対応する Branch の ID
    Answer     string            `yaml:"answer,omitempty"`
}

// GapAnalysisResult holds the result of one gap analysis run.
type GapAnalysisResult struct {
    TotalFields       int                   `yaml:"total_fields"`
    RequiredTotal     int                   `yaml:"required_total"`
    RequiredFulfilled int                   `yaml:"required_fulfilled"`
    RecommendedTotal  int                   `yaml:"recommended_total"`
    RecommendedFulfilled int               `yaml:"recommended_fulfilled"`
    FieldStatuses     []TemplateFieldStatus `yaml:"field_statuses"`
    NextQuestions     []TemplateQuestion    `yaml:"next_questions"`
    CanConfirm        bool                  `yaml:"can_confirm"`
    Warnings          []string              `yaml:"warnings,omitempty"`
}

func AnalyzeGap(tmpl RequirementTemplate, state *SessionState) GapAnalysisResult {
    // 1. テンプレートの全フィールドを走査
    // 2. state.Tree から対応する Branch を検索（ID マッチング）
    // 3. 充足度を判定
    // 4. 未充足の required フィールドを NextQuestions に変換
    // 5. 全 required 充足時に CanConfirm = true
}
```

### 4.4 LLM プロンプトへのテンプレート情報の渡し方

既存の `BuildInitialTreeWithLLM()` を拡張し、テンプレート情報を含める:

```
【要件定義テンプレート】
以下のテンプレートに基づいて、未充足の項目について質問を生成してください。

■ 未充足の必須項目（優先的に質問すべき）:
- プロジェクトの目的 (overview.purpose): ヒント: このプロジェクトが解決する課題は何か
- セキュリティ要件 (nfr.security): 推奨回答: なし

■ 未充足の推奨項目:
- 意思決定者 (stakeholders.decision_makers)

■ 既に充足済みの項目（参考情報として）:
- 対象ユーザー: Webアプリの一般ユーザー
- 正常系フロー: ログイン → ダッシュボード → 操作

■ 充足度: 必須 4/14 (29%), 推奨 2/10 (20%)

【指示】
1. 未充足の必須項目から最大3問を生成
2. 各質問にはテンプレートの hint を参考に推奨回答を付与
3. 選択肢がある場合は a), b), c) 形式で提示
4. 回答形式は「番号 選択肢」（例: 1a 2b 3: 自由回答）
```

**既存 decision tree モデルとの統合:**
- `DefaultTemplate()` (tree.go) → `DefaultRequirementTemplate()` (template.go) に**置き換え**
- `Branch` 構造体はそのまま使用。Branch.ID とテンプレートフィールド ID を 1:1 マッピング
- `BuildInitialTree()` → `BuildInitialTreeFromTemplate()` にリネーム。テンプレートから Branch リストを生成
- `BuildInitialTreeWithLLM()` → LLM プロンプトにテンプレート充足度情報を含める形に拡張

## 5. 充足度トラッキング設計

### 5.1 state ファイル拡張

```yaml
# .teraflow/discovery/discussion-42.yaml
version: "2"  # バージョンアップ
discussion_number: 42
title: "ユーザー認証要件"
# ... 既存フィールド ...

# NEW: 充足度トラッキング
fulfillment:
  template_version: "1"
  last_analyzed: "2026-04-14T10:00:00Z"
  required:
    total: 14
    fulfilled: 6
    partial: 2
    unfulfilled: 6
  recommended:
    total: 10
    fulfilled: 3
    unfulfilled: 7
  optional:
    total: 7
    fulfilled: 0
    unfulfilled: 7
  can_confirm: false
  fields:
    - field_id: overview.purpose
      status: fulfilled
      branch_id: scope.target_users  # 対応する Branch
    - field_id: nfr.security
      status: unfulfilled
    # ...
```

### 5.2 ユーザーへの可視化

Discussion コメントに充足度バーを表示:

```markdown
## 📊 要件充足度

| 区分 | 充足 | 合計 | 進捗 |
|------|------|------|------|
| 必須 | 6 | 14 | ████████░░░░░░ 43% |
| 推奨 | 3 | 10 | ████░░░░░░░░░░ 30% |
| 任意 | 0 | 7 | ░░░░░░░░░░░░░░ 0% |

### ✅ 充足済み
- プロジェクトの目的: Webアプリのログイン機能
- 対象ユーザー: 一般ユーザー
- ...

### ❓ 次の質問（必須項目を優先）

1. セキュリティ要件は？（認証方式、データ暗号化、セッション管理等）
   推奨: OAuth 2.0 + PKCE, AES-256暗号化
   
2. 期限はいつですか？
   
3. パフォーマンス要件は？（レスポンスタイム、同時接続数等）
   推奨: p95 < 500ms
```

### 5.3 「要求確定可能」通知

全必須項目が充足された時点で自動通知:

```markdown
## ✅ 要求確定可能

必須項目がすべて充足されました！

| 区分 | 充足 | 合計 |
|------|------|------|
| 必須 | 14 | 14 | ✅ 100% |
| 推奨 | 7 | 10 | 70% |
| 任意 | 2 | 7 | 29% |

「要求確定」とコメントすると、要件定義書（CoDD ドキュメント）が生成されます。
推奨項目の未回答: 意思決定者, 可用性要件, MoSCoW分類
```

## 6. 自動完了判定設計

### 6.1 判定ロジック

```go
func (g *GapAnalysisResult) CanAutoConfirm() bool {
    return g.RequiredFulfilled == g.RequiredTotal
}

func (g *GapAnalysisResult) ConfirmationWarnings() []string {
    var warnings []string
    for _, f := range g.FieldStatuses {
        if f.Priority == "required" && f.Status != Fulfilled {
            warnings = append(warnings, fmt.Sprintf(
                "必須項目「%s」が未回答です", f.Label))
        }
    }
    return warnings
}
```

### 6.2 要求確定時のフロー変更

```
現行:
  「要求確定」コメント → 即座にCoDD生成

提案:
  「要求確定」コメント
    → ギャップ分析実行
    → 未充足必須項目あり？
       YES → 警告リスト + 「それでも確定しますか？(「強制確定」で続行)」
       NO  → CoDD生成（充足度メタデータ付き）
```

### 6.3 state ファイルの confirmed フラグ

```yaml
# 要求確定後
confirmed: true
confirmed_at: "2026-04-14T12:00:00Z"
confirmed_by: "user"
force_confirmed: false  # 必須未充足での強制確定
```

**cmd_224 との関連:** `confirmed: true` のセッションでは `NextQuestions()` が空を返す。これにより要求確定後の質問停止を実現。cmd_224 で先行実装する `confirmed` チェックは、cmd_225 でそのまま活用される。

## 7. 既存 E2E テストへの影響

### 7.1 影響分析

| E2E フェーズ | 影響 | 対応 |
|---|---|---|
| Phase A (setup) | なし | — |
| Phase B (initial) | **変更あり** | テンプレートベースの初期質問に変わる。質問数・形式が変化。V1〜V5 のバリデーション条件を更新。 |
| Phase C (rounds) | **変更あり** | 充足度表示が追加される。回答パース処理は既存を踏襲。V6〜V11 を更新。 |
| Phase D (confirm) | **変更あり** | 充足度チェック + 警告表示が追加。V12〜V18 を更新。 |

### 7.2 E2E 更新方針

- Phase B: 初期質問が「テンプレートの必須項目」から生成されることを検証
- Phase C: 充足度バーの表示を新バリデーション項目として追加
- Phase D: 全必須充足時の「要求確定可能」通知を検証
- **既存 V1〜V18 のうち、質問/回答構造に依存するものは条件を緩和**（テンプレート変更に追従しやすくする）

## 8. 実装フェーズ

### Phase 1: テンプレート基盤（見積: M）

| # | タスク | ファイル |
|---|--------|----------|
| 1-1 | `RequirementTemplate` 型定義 + `DefaultRequirementTemplate()` | `internal/discovery/template.go` (新規) |
| 1-2 | テンプレート読み込み（ファイル + teraflow.yml 上書き） | `internal/discovery/template.go` |
| 1-3 | `BuildInitialTreeFromTemplate()` — テンプレートから Branch 生成 | `internal/discovery/tree.go` (変更) |
| 1-4 | 既存 `DefaultTemplate()` を deprecated 化 | `internal/discovery/tree.go` |
| 1-5 | テンプレート構造のユニットテスト | `internal/discovery/template_test.go` (新規) |

### Phase 2: ギャップ分析エンジン（見積: L）

| # | タスク | ファイル |
|---|--------|----------|
| 2-1 | `GapAnalysisResult` 型定義 + `AnalyzeGap()` | `internal/discovery/gap.go` (新規) |
| 2-2 | state ファイルに `fulfillment` セクション追加 | `internal/discovery/state.go` (変更) |
| 2-3 | LLM プロンプトにテンプレート充足度情報を注入 | `internal/discovery/tree.go` (変更) |
| 2-4 | Discussion コメントに充足度バー表示 | `teraflow-req-agent.yml` (変更) |
| 2-5 | ギャップ分析のユニットテスト | `internal/discovery/gap_test.go` (新規) |

### Phase 3: 自動完了判定（見積: S）

| # | タスク | ファイル |
|---|--------|----------|
| 3-1 | `CanAutoConfirm()` + `ConfirmationWarnings()` | `internal/discovery/gap.go` (追記) |
| 3-2 | 「要求確定」フロー変更（警告 + 強制確定） | `teraflow-req-agent.yml` (変更) |
| 3-3 | `confirmed` フラグによる質問停止（cmd_224 先行実装を活用） | `internal/discovery/state.go` (変更) |
| 3-4 | E2E テスト更新 | `tests/e2e/` (変更) |

### 依存関係

```
cmd_224 (confirmed フラグ先行実装)
  ↓
Phase 1 (テンプレート基盤) → Phase 2 (ギャップ分析) → Phase 3 (自動完了判定)
                                                              ↓
                                                        E2E テスト更新
```

## 9. リスクと緩和策

| リスク | 影響 | 緩和策 |
|--------|------|--------|
| LLM 充足度判定の精度が低い | 未充足を充足と誤判定 → 不完全な要件定義書 | Phase 2 初期はルールベース（回答の有無 + 長さ）で判定。LLM 判定は Phase 2 後半で段階的に導入 |
| テンプレートが硬直的 | プロジェクト種別に合わないテンプレートがストレスになる | カスタムテンプレート機能を Phase 1 で実装。デフォルトは汎用的に |
| 既存ユーザーの混乱 | 質問形式の変化で既存ユーザーが戸惑う | 初回コメントに「テンプレートベースの質問に移行しました」と案内 |
| state ファイルの後方互換性 | version "1" の state を version "2" で読めない | `LoadSessionState()` でバージョン判定、v1 は fulfillment なしで動作 |
