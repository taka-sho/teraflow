---
codd:
  node_id: "design:discovery-doc-index"
  title: "Discovery ドキュメントインデックス設計書"
  depends_on:
    - id: "design:requirements-discovery"
  tags:
    - discovery
    - doc-index
    - context-window
  status: draft
---

# Discovery ドキュメントインデックス設計書

## 1. 概要

### 1.1 目的

現行の discovery agent（grill-me）は Discussion 本文と過去コメントのみをコンテキストとして質問ツリーを生成する。既存の docs/ 配下にある要件定義書・設計書・ADR 等の蓄積知識を参照しないため、以下の問題が発生する:

| 問題 | 具体例 |
|------|--------|
| **既知事項の再質問** | docs/00_overview.md に記載済みの対象プラットフォームを再度質問する |
| **矛盾の見逃し** | 既存 ADR の決定事項と矛盾する回答を検出できない |
| **カバレッジの盲点** | 既存文書がカバーしていない領域を特定できない |

本設計は、docs/ 配下の文書のインデックスを構築し、discovery agent が逐次読み込みで必要な文書のみを参照する仕組みを定義する。

### 1.2 スコープ

| 対象 | 対象外 |
|------|--------|
| docs/ 配下 Markdown のインデックス生成 | GraphRAG との統合（Phase 2+） |
| LLM による関連文書判定 | Embedding ベクトル検索 |
| チャンク単位の逐次読み込み | ビジネス領域分析 |
| BuildInitialTreeWithLLM への入力拡張 | 全文書の全文読み込み |
| GitHub Actions 内での実現 | ローカル専用機能 |

## 2. ドキュメントインデックス設計

### 2.1 インデックスの構造

```yaml
# .teraflow/doc-index.yaml
version: "1"
generated_at: "2026-04-12T20:00:00Z"
generator: "teraflow doc index"
docs:
  - path: "docs/00_overview.md"
    title: "プロジェクト概要"
    summary: "teraflow の目的、対象ユーザー、主要機能の概要"
    categories:
      - scope
      - functional
    keywords:
      - "CLI"
      - "CoDD"
      - "V字モデル"
      - "GitHub Actions"
    sections:
      - heading: "対象ユーザー"
        line_start: 15
        line_end: 32
        token_estimate: 200
      - heading: "主要機能"
        line_start: 34
        line_end: 89
        token_estimate: 650
    total_tokens: 1800
    codd_node_id: "docs:overview"

  - path: "docs/01_functional-requirements.md"
    title: "機能要件定義書"
    summary: "teraflow の全機能要件。コマンド体系、スキル、エージェント連携。"
    categories:
      - functional
      - scope
    keywords:
      - "init"
      - "stage"
      - "phase"
      - "discovery"
      - "agent"
    sections:
      - heading: "コマンド体系"
        line_start: 10
        line_end: 45
        token_estimate: 400
      - heading: "スキル定義"
        line_start: 47
        line_end: 120
        token_estimate: 800
    total_tokens: 3200
    codd_node_id: "docs:functional-requirements"
```

### 2.2 インデックスエントリのフィールド定義

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `path` | string | Yes | docs/ からの相対パス |
| `title` | string | Yes | 文書タイトル（H1 見出し or CoDD title） |
| `summary` | string | Yes | 1-2 文の概要（LLM が関連性判定に使用） |
| `categories` | []string | Yes | discovery カテゴリとの対応（scope, functional, non_functional, acceptance, risk, dependency, priority） |
| `keywords` | []string | Yes | 主要キーワード（全文検索では拾えない概念レベルのタグ） |
| `sections` | []Section | Yes | セクション単位のチャンク情報 |
| `total_tokens` | int | Yes | 文書全体の推定トークン数 |
| `codd_node_id` | string | No | CoDD ノード ID（frontmatter から取得） |

**Section フィールド:**

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `heading` | string | Yes | セクション見出し |
| `line_start` | int | Yes | 開始行番号 |
| `line_end` | int | Yes | 終了行番号 |
| `token_estimate` | int | Yes | セクションの推定トークン数 |

### 2.3 インデックス生成方式

**ハイブリッド方式**（自動生成 + LLM サマリ補完）を採用する。

```text
Phase 1: 静的解析（LLM 不要）
  ├─ docs/ 配下の .md ファイルを走査
  ├─ CoDD frontmatter から node_id, title, tags を抽出
  ├─ Markdown H2 見出しからセクション分割 + 行番号記録
  ├─ トークン数を推定（文字数 × 0.4 の近似）
  └─ 出力: path, title, sections, total_tokens, codd_node_id

Phase 2: LLM サマリ生成（オプション）
  ├─ 各文書の先頭 ~500 トークンを LLM に渡す
  ├─ summary（1-2 文）と categories, keywords を生成
  └─ 出力: summary, categories, keywords を補完

Phase 3: 統合
  └─ .teraflow/doc-index.yaml に書き出し
```

**Phase 1 のみでも動作する。** LLM サマリがない場合、categories は CoDD tags から推定し、keywords は H2 見出しテキストから抽出する。

### 2.4 生成コマンド

```bash
# 静的解析のみ（LLM 不要、CI 向け）
teraflow doc index --no-llm

# LLM サマリ付き（初回推奨）
teraflow doc index

# 特定ディレクトリのみ
teraflow doc index --path docs/requirements/
```

### 2.5 更新タイミング

| トリガー | 方法 | 備考 |
|---------|------|------|
| `teraflow doc generate` 実行時 | 自動（post-hook） | 新規文書生成後にインデックス再生成 |
| CI パイプライン | `teraflow doc index --no-llm` | PR マージ時 |
| 手動 | `teraflow doc index` | 初回セットアップ時 |

### 2.6 格納場所

```text
.teraflow/
├── doc-index.yaml          ← NEW: ドキュメントインデックス
├── index.yml               # CoDD index（既存）
├── summaries/              # AI 要約キャッシュ（既存）
└── discovery/              # grill-me セッション（既存）
```

`.teraflow/doc-index.yaml` は git 管理対象とする。理由:
- チームメンバーがインデックスを共有できる
- CI で自動更新 + commit する想定
- LLM サマリの再生成コスト削減（差分更新）

## 3. 逐次読み込みフロー設計

### 3.1 全体フロー

```text
Discussion コメント受信
    │
    ▼
[Step 0] インデックス読み込み
    .teraflow/doc-index.yaml を読む（~1000-2000 tok）
    │
    ▼
[Step 1] 関連文書判定（LLM）
    インデックス全体 + Discussion 本文 + 最新コメント を LLM に渡す
    → 「このDiscussionに関連する文書はどれか」を判定
    → 関連度スコア付きで最大5文書を選択
    │
    ▼
[Step 2] チャンク読み込み
    選択された文書の該当セクションのみを読み込み
    → line_start〜line_end の範囲を抽出
    → トークン予算内に収まるよう調整
    │
    ▼
[Step 3] コンテキスト注入
    読み込んだチャンク + Discussion 本文 + 既存 state
    → BuildInitialTreeWithLLM (初回)
    → 通常の grill-me 質問生成 (2回目以降)
```

### 3.2 Step 1: 関連文書判定

**LLM プロンプト:**

```text
以下はプロジェクトのドキュメントインデックスです。
Discussion の内容に関連する文書を最大5件選び、
関連するセクションを特定してください。

## インデックス
{doc-index.yaml の docs 配列（path, title, summary, categories, keywords のみ）}

## Discussion
タイトル: {discussion_title}
本文: {discussion_body}
最新コメント: {latest_comment}

## 出力形式（JSON）
[
  {
    "path": "docs/00_overview.md",
    "relevance": "high",
    "reason": "対象ユーザーのスコープ定義に関連",
    "sections": ["対象ユーザー", "主要機能"]
  }
]
```

**判定基準:**
- `categories` の一致: Discussion の質問カテゴリとインデックスの categories の重なり
- `keywords` の一致: Discussion 本文中のキーワードとの一致
- `summary` の意味的関連性: LLM による判断

**出力:** 関連文書リスト（最大5件）+ 読むべきセクション名

### 3.3 Step 2: チャンク読み込み

```go
// ReadDocChunks reads specific sections from documents based on index.
func ReadDocChunks(index *DocIndex, selections []DocSelection, budget int) ([]DocChunk, error) {
    chunks := make([]DocChunk, 0)
    remaining := budget

    for _, sel := range selections {
        entry := index.Find(sel.Path)
        if entry == nil {
            continue
        }
        for _, sectionName := range sel.Sections {
            section := entry.FindSection(sectionName)
            if section == nil {
                continue
            }
            if section.TokenEstimate > remaining {
                // トークン予算超過 → このセクションはスキップ
                continue
            }
            content, err := readLines(sel.Path, section.LineStart, section.LineEnd)
            if err != nil {
                continue
            }
            chunks = append(chunks, DocChunk{
                Path:    sel.Path,
                Section: sectionName,
                Content: content,
                Tokens:  section.TokenEstimate,
            })
            remaining -= section.TokenEstimate
        }
    }
    return chunks, nil
}
```

**チャンク読み込みの原則:**
1. セクション単位で読む（全文ではない）
2. トークン予算を厳守
3. 関連度が高い文書のセクションを優先
4. 予算超過時はスキップ（部分読み込みはしない）

### 3.4 Step 3: コンテキスト注入

BuildInitialTreeWithLLM のプロンプトを拡張:

```go
func BuildInitialTreeWithLLM(
    ctx context.Context,
    llm LLMGenerator,
    discussionBody string,
    docContext string,        // NEW: インデックスから読み込んだ文書チャンク
    fallback DecisionTreeTemplate,
) ([]Branch, error) {
    prompt := fmt.Sprintf(
        `投稿内容から要件探索の決定木を生成し、JSON配列で返してください。
各要素は id/question/category/recommendation/depends_on を持つこと。

## 既存ドキュメントからの参照情報
%s

## 重要ルール
- 上記の既存ドキュメントで既に確定している事項は質問せず、recommendation に「（既存文書で確定済み: {内容}）」と記載すること
- 既存ドキュメントの内容と矛盾する可能性がある箇所を優先的に質問すること
- 既存ドキュメントがカバーしていない領域を重点的に質問すること

## Discussion 投稿内容
%s`,
        docContext,
        strings.TrimSpace(discussionBody),
    )
    // ... 以降は既存ロジック
}
```

## 4. 既存フローとの統合

### 4.1 統合ポイント一覧

| 統合先 | 変更内容 | 影響度 |
|--------|---------|--------|
| `internal/discovery/tree.go` | `BuildInitialTreeWithLLM` に docContext パラメータ追加 | 中 |
| `internal/discovery/` (新規) | `docindex.go` — インデックス読み込み・チャンク抽出 | 新規 |
| `cmd/doc.go` | `doc index` サブコマンド追加 | 小 |
| `skills/discovery.yml` | context.include にインデックス参照追加 | 小 |
| `teraflow-req-agent.yml` | discovery ステップにインデックス読み込み追加 | 中 |

### 4.2 tree.go の変更

**現行シグネチャ:**
```go
func BuildInitialTreeWithLLM(ctx context.Context, llm LLMGenerator, discussionBody string, fallback DecisionTreeTemplate) ([]Branch, error)
```

**変更後シグネチャ:**
```go
func BuildInitialTreeWithLLM(ctx context.Context, llm LLMGenerator, discussionBody string, opts TreeBuildOptions) ([]Branch, error)

type TreeBuildOptions struct {
    Fallback   DecisionTreeTemplate
    DocContext string // インデックスから読み込んだ文書チャンク（空文字列 = 参照なし）
}
```

**後方互換性:** `DocContext` が空文字列の場合は従来と同じ動作。既存テストは `TreeBuildOptions{Fallback: tmpl}` に変更するだけで通る。

### 4.3 discovery_input.txt の構成変更

**現行:**
```text
## Discussion 本文（初回投稿）
{discussion_body}

## 構造化サマリ
{summary yaml}

## 最新コメント
{latest_comment}

## モード
{skill_mode}
```

**変更後:**
```text
## Discussion 本文（初回投稿）
{discussion_body}

## 既存ドキュメント参照                    ← NEW
{関連文書のチャンク（Step 2 の出力）}

## 構造化サマリ
{summary yaml}

## 最新コメント
{latest_comment}

## モード
{skill_mode}
```

### 4.4 req-agent ワークフローの変更

Discovery grill-me response ステップの前に、インデックス読み込みステップを追加:

```yaml
      - name: Load doc-index and select relevant docs
        if: startsWith(steps.mode.outputs.mode, 'discovery')
        id: doc_context
        env:
          DISC_BODY: ${{ github.event.discussion.body }}
          DISC_TITLE: ${{ github.event.discussion.title }}
          LATEST_COMMENT: ${{ github.event.comment.body }}
        run: |
          DOC_INDEX=".teraflow/doc-index.yaml"
          if [ ! -f "$DOC_INDEX" ]; then
            # インデックス未生成 → 静的解析で生成
            teraflow doc index --no-llm
          fi

          # Step 1: 関連文書判定 + Step 2: チャンク読み込み
          teraflow doc context \
            --index "$DOC_INDEX" \
            --discussion-title "$DISC_TITLE" \
            --discussion-body "$DISC_BODY" \
            --latest-comment "$LATEST_COMMENT" \
            --budget 2000 \
            --format text \
            > /tmp/doc_context.txt 2>/dev/null || echo "" > /tmp/doc_context.txt
```

## 5. コンテキストウィンドウ管理

### 5.1 トークン予算配分

```text
discovery agent の入力トークン予算: ~8000 tok

┌──────────────────────────────────────────────────┐
│ 1. システムプロンプト                             │  ~500 tok (固定)
├──────────────────────────────────────────────────┤
│ 2. インデックスサマリ（Step 1 入力）              │  ~800 tok (可変)
│    path + title + summary のみ（sections 除外）   │
├──────────────────────────────────────────────────┤
│ 3. 関連文書チャンク（Step 2 出力）                │  ~2000 tok (予算制限)
│    最大5文書 × 関連セクションのみ                 │
├──────────────────────────────────────────────────┤
│ 4. 状態ファイル（未解決分岐のみ）                 │  ~500-1500 tok
├──────────────────────────────────────────────────┤
│ 5. CoDD ドラフト（現在のスナップショット）        │  ~1000-2000 tok
├──────────────────────────────────────────────────┤
│ 6. 最新コメント（ユーザーの回答）                 │  ~200-500 tok
└──────────────────────────────────────────────────┘
合計: ~5000-7300 tok（入力側。8000 tok 予算内）
```

### 5.2 インデックスサイズの上限

| 制約 | 値 | 根拠 |
|------|-----|------|
| インデックス最大エントリ数 | 50 文書 | docs/ 配下の現実的な文書数上限 |
| Step 1 入力サイズ | ~800 tok | path + title + summary × 50 ≈ 16 tok/entry × 50 |
| Step 2 出力サイズ（予算） | 2000 tok | 全体予算 8000 tok の 25% |
| 1文書あたりの最大チャンク | 3 セクション | 過度な読み込み防止 |
| 1セクションの最大トークン | 800 tok | 2000 tok 予算で最低2文書は読める保証 |

### 5.3 予算超過時の振る舞い

```text
予算 2000 tok に対して選択文書の合計が超過した場合:

1. 関連度順にソート（high → medium → low）
2. 上位から順にセクションを読み込み
3. 予算残量 < 次のセクションの token_estimate → そのセクションをスキップ
4. 予算残量 < 100 tok → 読み込み終了
5. 読めなかったセクションは「※ トークン予算により省略」とマーク
```

### 5.4 インデックスが存在しない場合

インデックス未生成でも discovery は動作する（従来どおり）。

```text
.teraflow/doc-index.yaml が存在しない場合:
  → Step 1-2 をスキップ
  → docContext = ""（空文字列）
  → BuildInitialTreeWithLLM は従来と同じ動作
```

## 6. ユースケース

### 6.1 初回 discovery: 既存文書のカバー範囲把握

```text
[シナリオ] ユーザーが「ユーザー認証機能」の Discussion を新規作成

1. インデックスから関連文書を判定:
   - docs/01_functional-requirements.md → "認証" キーワード一致
   - docs/06_non-functional-requirements.md → "セキュリティ" カテゴリ一致

2. チャンク読み込み:
   - 機能要件書の「認証」セクション（既存の認証方式定義）
   - 非機能要件書の「セキュリティ要件」セクション

3. 質問ツリー生成時の効果:
   ✅ 「対象プラットフォームは？」→ 既存文書に「Web + CLI」と記載済み
      → recommendation: "（既存文書で確定済み: Web + CLI）"
      → resolved_by: "auto" で自動解決
   ✅ 「認証方式は？」→ 既存文書に記載なし → pending として質問
   ✅ 「SLA要件は？」→ 非機能要件書に "99.9%" と記載済み
      → recommendation: "（既存文書で確定済み: 99.9%）"
```

### 6.2 2回目以降: 矛盾検出・漏れ検出

```text
[シナリオ] ユーザーが「JWT の有効期限は24時間」と回答

1. 既存文書チャンクに「セッション有効期限: 1時間」と記載あり
2. LLM が矛盾を検出:
   → "既存の非機能要件書ではセッション有効期限が1時間と定義されています。
      JWT有効期限を24時間にすると矛盾しますが、要件を更新しますか？"
3. ユーザーの回答を待ち、矛盾解消後に進行
```

### 6.3 確定済み事項の参照提示

```text
[シナリオ] 質問ツリーに「データベースは何を使いますか？」がある

1. インデックスからデータベース関連の記述を検出:
   - ADR-003: "PostgreSQL を採用"
2. 質問を自動解決:
   → status: "answered"
   → answer: "PostgreSQL（ADR-003 で決定済み）"
   → resolved_by: "auto"
3. ユーザーには質問せず、ドラフトに反映
```

## 7. 実装パッケージ構成

```text
internal/discovery/
├── state.go          # 既存: SessionState
├── tree.go           # 既存: BuildInitialTreeWithLLM（変更: opts パラメータ）
├── draft.go          # 既存: BuildDraftFromState
└── docindex.go       # NEW: DocIndex, ReadDocChunks, SelectRelevantDocs

cmd/
├── doc.go            # 既存: doc サブコマンド群
└── doc_index.go      # NEW: doc index サブコマンド
```

### 7.1 docindex.go の主要型

```go
package discovery

type DocIndex struct {
    Version     string     `yaml:"version"`
    GeneratedAt string     `yaml:"generated_at"`
    Generator   string     `yaml:"generator"`
    Docs        []DocEntry `yaml:"docs"`
}

type DocEntry struct {
    Path       string      `yaml:"path"`
    Title      string      `yaml:"title"`
    Summary    string      `yaml:"summary"`
    Categories []string    `yaml:"categories"`
    Keywords   []string    `yaml:"keywords"`
    Sections   []DocSection `yaml:"sections"`
    TotalTokens int        `yaml:"total_tokens"`
    CoddNodeID string      `yaml:"codd_node_id,omitempty"`
}

type DocSection struct {
    Heading       string `yaml:"heading"`
    LineStart     int    `yaml:"line_start"`
    LineEnd       int    `yaml:"line_end"`
    TokenEstimate int    `yaml:"token_estimate"`
}

type DocSelection struct {
    Path      string   `json:"path"`
    Relevance string   `json:"relevance"`
    Reason    string   `json:"reason"`
    Sections  []string `json:"sections"`
}

type DocChunk struct {
    Path    string
    Section string
    Content string
    Tokens  int
}
```

### 7.2 主要関数

```go
// LoadDocIndex reads the doc-index.yaml file.
func LoadDocIndex(path string) (*DocIndex, error)

// GenerateDocIndex scans docs/ and builds the index.
// If llm is nil, only static analysis is performed (Phase 1).
func GenerateDocIndex(ctx context.Context, docsDir string, llm LLMGenerator) (*DocIndex, error)

// SelectRelevantDocs asks LLM which docs are relevant to the discussion.
func SelectRelevantDocs(ctx context.Context, llm LLMGenerator, index *DocIndex, discussion string) ([]DocSelection, error)

// ReadDocChunks reads specific sections from selected documents within token budget.
func ReadDocChunks(index *DocIndex, selections []DocSelection, budget int) ([]DocChunk, error)

// FormatDocContext formats chunks into a string for LLM prompt injection.
func FormatDocContext(chunks []DocChunk) string
```

## 8. 将来拡張: GraphRAG 連携

本設計は Phase 1（テキストベースインデックス + LLM 関連判定）であり、GraphRAG との統合は Phase 2+ で検討する。

**拡張ポイント:**

| 拡張 | 変更箇所 | 概要 |
|------|---------|------|
| Embedding 検索 | `SelectRelevantDocs` | LLM 判定の代わりに or 併用で Embedding 類似度検索 |
| ナレッジグラフ参照 | `DocIndex` | CoDD ノード間の depends_on 関係をグラフとして活用 |
| 自動矛盾検出 | `BuildInitialTreeWithLLM` | GraphRAG の関係性情報から矛盾候補を事前抽出 |
| インデックス自動更新 | `GenerateDocIndex` | GraphRAG の差分検知トリガーで自動再生成 |

**Phase 1 → Phase 2 移行時の非破壊保証:**
- `DocIndex` の YAML スキーマは追加フィールドで拡張（既存フィールドは変更しない）
- `SelectRelevantDocs` は interface 経由で呼び出し、実装を差し替え可能にする
- `.teraflow/doc-index.yaml` のフォーマットは version フィールドで管理

## 9. サブタスク分解案

| # | タスク | 依存 | 見積 |
|---|--------|------|------|
| 1 | `internal/discovery/docindex.go` — DocIndex 型定義 + LoadDocIndex + GenerateDocIndex（静的解析のみ） | なし | S |
| 2 | `cmd/doc_index.go` — `teraflow doc index` サブコマンド | #1 | S |
| 3 | `internal/discovery/docindex.go` — SelectRelevantDocs + ReadDocChunks + FormatDocContext | #1 | M |
| 4 | `internal/discovery/tree.go` — BuildInitialTreeWithLLM の TreeBuildOptions 化 + docContext 対応 | #3 | S |
| 5 | `internal/actions/templates/teraflow-req-agent.yml` — doc context ステップ追加 | #2, #4 | M |
| 6 | `skills/discovery.yml` — プロンプト更新（既存文書参照ルール追加） | #4 | S |
| 7 | GenerateDocIndex の LLM サマリ補完（Phase 2 オプション） | #1 | M |
| 8 | テスト: docindex_test.go（静的解析 + チャンク読み込み） | #1, #3 | S |

注: S = 1足軽セッション, M = 2足軽セッション
