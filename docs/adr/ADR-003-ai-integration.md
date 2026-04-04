---
codd:
  node_id: "adr:003-ai-integration"
  title: "ADR-003 AI基盤連携設計"
  depends_on:
    - id: "req:teraflow-overview"
      relation: implements
---

# ADR-003: AI基盤連携設計

## ステータス

提案（Proposed）

## コンテキスト

teraflowはAIエージェントによる自律開発パイプラインを段階的に導入する。
AI基盤の選定と連携設計にあたり、以下の制約が存在する:

- **B-003裁定**: 複数AI基盤対応。ベンダーロックイン対策必須
- **Phase1スコープ**: CLI上でのAI利用（グループ提案、影響分析、スケジュール予測等）
- **Phase2スコープ**: GitHub Actions上の10エージェント（実装、レビュー、CI修正等）
- **ハーネスエンジニアリング**: Context供給 / アーキテクチャ制約 / フィードバックループの3層

## AI利用箇所（Phase1）

| 機能 | AI利用内容 | 入力 | 出力 |
|------|-----------|------|------|
| `group propose` | 要件分析からグループ分割案提示 | 要件定義ファイル群 | グループ定義YAML |
| `group rebalance` | 人員変動時の再編成分析 | groups.yml + 変更ログ | 再編案 |
| `schedule predict` | 完了予測の算出 | master-schedule.yml + 変更ログ | 予測結果 |
| `rework create` | 影響分析（CoDD adapter） | frontmatter依存グラフ | 影響範囲レポート |
| `harness score` | コード品質スコアリング | ソースコード + テスト結果 | スコア + 改善提案 |
| `dashboard generate` | サマリ生成 | プロジェクト状態データ | Markdownレポート |

## 設計方針: Provider Interface パターン

### アーキテクチャ

```
teraflow CLI
    │
    ├── internal/ai/
    │   ├── provider.go          # Provider interface定義
    │   ├── config.go            # プロバイダ設定の読み込み
    │   ├── providers/
    │   │   ├── anthropic.go     # Claude API (Anthropic SDK)
    │   │   ├── openai.go        # OpenAI API (GPT, Codex)
    │   │   ├── gemini.go        # Google Gemini API
    │   │   └── local.go         # ローカルLLM (Ollama等)
    │   └── prompt/
    │       ├── templates/       # プロンプトテンプレート
    │       └── builder.go       # プロンプト構築
    │
    └── teraflow.yml             # プロバイダ設定
```

### Provider Interface

```go
// provider.go
type Provider interface {
    // Complete は単一プロンプトに対する応答を返す
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)

    // Name はプロバイダ名を返す（ログ・エラーメッセージ用）
    Name() string

    // Models は利用可能なモデル一覧を返す
    Models() []string
}

type CompletionRequest struct {
    Model       string            // モデル指定（省略時はプロバイダデフォルト）
    System      string            // システムプロンプト
    Messages    []Message         // 会話履歴
    MaxTokens   int               // 最大トークン数
    Temperature float64           // 生成温度
    Format      ResponseFormat    // 応答形式（text/json）
}

type CompletionResponse struct {
    Content     string            // 応答テキスト
    Model       string            // 使用されたモデル
    Usage       TokenUsage        // トークン使用量
    StopReason  string            // 停止理由
}

type TokenUsage struct {
    InputTokens  int
    OutputTokens int
}
```

### プロバイダ設定

```yaml
# teraflow.yml
ai:
  # デフォルトプロバイダ
  default_provider: anthropic

  # プロバイダ別設定
  providers:
    anthropic:
      model: claude-sonnet-4-20250514
      # API keyは環境変数 ANTHROPIC_API_KEY から取得
    openai:
      model: gpt-4o
      # API keyは環境変数 OPENAI_API_KEY から取得
    gemini:
      model: gemini-2.0-flash
      # API keyは環境変数 GOOGLE_API_KEY から取得
    local:
      endpoint: http://localhost:11434
      model: llama3

  # 機能別プロバイダ上書き（オプション）
  overrides:
    group_propose: anthropic    # グループ提案はClaudeを使用
    harness_score: openai       # スコアリングはGPTを使用
```

### 認証情報の管理

```
優先度（高→低）:
1. 環境変数 (ANTHROPIC_API_KEY, OPENAI_API_KEY 等)
2. teraflow.yml の api_key フィールド（非推奨: .gitignoreに含めること）
3. OS keychain (将来対応)

※ SSoT=GitHubリポジトリの原則により、API keyをリポジトリにコミットしてはならない。
  sec25（Secrets管理）で定義されるGitHub Secretsとは別管理（Phase1はローカル環境変数）。
```

## プロンプト設計

### テンプレート管理

```
internal/ai/prompt/templates/
├── group_propose.tmpl       # グループ提案プロンプト
├── group_rebalance.tmpl     # グループ再編プロンプト
├── schedule_predict.tmpl    # スケジュール予測プロンプト
├── impact_analysis.tmpl     # 影響分析プロンプト
├── harness_score.tmpl       # 品質スコアリングプロンプト
└── dashboard_summary.tmpl   # ダッシュボードサマリ生成
```

### テンプレート形式

Go text/template を使用:

```
{{.SystemContext}}

## タスク
{{.TaskDescription}}

## 入力データ
{{.InputData}}

## 出力形式
以下のYAML形式で出力してください:
{{.OutputSchema}}
```

### ハーネスエンジニアリング3層の実装

| 層 | Phase1での実装 |
|----|--------------|
| **Context供給** | プロンプトテンプレートに project-state.yml, groups.yml の情報を注入 |
| **アーキテクチャ制約** | 出力形式をYAML/JSONに強制。バリデーションで不正出力を検知 |
| **フィードバックループ** | Phase1では手動フィードバック。Phase2でActions経由の自動ループ |

## Phase2への拡張パス

Phase2では10エージェント（sec18-19）をGitHub Actions上で動作させる:

| エージェント | 対応Provider機能 |
|------------|----------------|
| 実装Agent (#9) | Complete (コード生成) |
| レビューAgent (#12) | Complete (コードレビュー) |
| CI修正Agent (#10) | Complete (エラー修正) |
| 停滞監視Agent (#14) | Complete (状況分析) |

Provider Interfaceは変更不要。Actions環境変数でAPI keyを渡し、同じinterfaceを呼び出す。

## 決定

1. **Provider Interfaceパターン**を採用し、プロバイダ差し替えを保証
2. **Phase1ではAnthropic (Claude)をデフォルト**とする（SDK成熟度、日本語性能）
3. **プロンプトテンプレート**はGoテンプレートで管理し、バイナリに埋め込む
4. **API keyは環境変数管理**が標準。設定ファイルへの記載は非推奨

## 影響

- 全AI利用コマンドはProvider Interface経由で呼び出す
- テスト時はMock Providerを使用（外部API依存なし）
- トークン使用量はTokenUsageで追跡可能（コスト管理用）
