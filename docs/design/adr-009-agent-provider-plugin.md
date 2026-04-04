---
codd:
  node_id: "adr:009-agent-provider-plugin"
  title: "ADR-009: Agent実行環境プラグイン化アーキテクチャ"
  depends_on:
    - id: "adr:003-ai-integration"
      relation: extends
    - id: "design:phase2-system"
      relation: extends
    - id: "adr:005-github-actions-design"
      relation: extends
---

# ADR-009: Agent実行環境プラグイン化アーキテクチャ

## ステータス

提案（Proposed）

## コンテキスト

Phase2のAgent Pipeline（TF-004〜014）は現在 `AnthropicProvider` 固定で実装されている。殿の裁定 B-003（マルチベンダーAI、ベンダーロックイン対策）に基づき、Claude Code / OpenAI / GitHub Copilot / カスタムAgentへの差し替えを可能にする必要がある。

### 現状

```go
// internal/agent/agent.go
type Provider interface {
    Complete(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, int, error)
    Name() string
}

// internal/agent/provider.go — AnthropicProvider のみ実装
type AnthropicProvider struct { ... }
```

- Provider interface は既に定義済み（B-003準拠）
- しかし `NewAgentManager` に渡すProvider生成は呼び出し側に任されており、設定ファイルからのProvider自動選択機構がない
- teraflow.yml には `ai.default_provider: anthropic` があるが、agentセクションは未定義

## 検討したオプション

### Option A: 設定ファイルベースのプロバイダー選択（シンプル）

```yaml
# .github/teraflow.yml
agent:
  provider: anthropic
  model: claude-haiku-4-5-20251001
```

teraflow.yml の `agent.provider` フィールドでプロバイダーを選択。`NewProviderFromConfig(cfg)` でProvider interfaceを返す Factory関数を追加。

**利点**: 最小限のコード変更、設定ファイル1箇所の変更で切り替え可能
**欠点**: プロバイダー追加時にteraflow本体のコード変更が必要

### Option B: プラグインインターフェース拡張（柔軟）

Go pluginパッケージを使い、`.so` ファイルで外部Providerをロード。

**利点**: teraflow本体を変更せずにProvider追加可能
**欠点**: Goのpluginは Linux限定（macOS/Windows非対応）、CGO依存、デバッグ困難、実用例が少ない

### Option C: 外部コマンド実行型（最大汎用性）

`provider: custom` + `custom_command` で任意の外部コマンドをProviderとして実行。stdin/stdoutでプロンプト/レスポンスを受け渡し。

**利点**: 言語非依存、任意のツールをProvider化可能
**欠点**: プロセス起動コスト、エラーハンドリングの複雑化、セキュリティリスク

## 決定

**Option A（設定ファイルベース）をメイン方式とし、Option C（外部コマンド）をCustomProviderとして併用する。**

### 理由

1. **Option A で95%のユースケースをカバー**: Anthropic / OpenAI / Claude Code CLI は teraflow 内蔵Providerとして提供。設定変更のみで切り替え可能
2. **Option C で残り5%をカバー**: 内蔵Provider以外のLLM（ローカルLLM、社内API等）はCustomProviderで対応
3. **Option B は却下**: Go pluginの制約（Linux限定、CGO依存）はteraflowのクロスプラットフォーム方針に反する
4. **後方互換性**: `agent` セクション未設定時は `ai.default_provider` にフォールバック。既存ユーザーの設定変更不要

### 代替案（却下）

| 案 | 却下理由 |
|----|---------|
| Go plugin（.so） | Linux限定、CGO依存。5プラットフォームビルド（GoReleaser）と矛盾 |
| WebAssembly plugin | Go→Wasm互換性が不安定。オーバーエンジニアリング |
| gRPC/RESTサーバー | Agentのためにサーバー起動は過剰。CLIツールとして不適切 |

## 影響

- Provider interface は変更しない（後方互換性維持）
- `NewProviderFromConfig()` Factory関数の追加（新規）
- teraflow.yml に `agent` セクション追加（オプショナル、既存設定は影響なし）
- `teraflow doctor` にProvider healthcheck 追加
- CustomProvider の外部コマンド実行はコマンドパスのバリデーション必須（インジェクション対策）
