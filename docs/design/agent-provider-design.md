---
codd:
  node_id: "design:agent-provider-plugin"
  title: "Agent Provider プラグイン設計書"
  depends_on:
    - id: "adr:009-agent-provider-plugin"
      relation: implements
    - id: "design:phase2-system"
      relation: extends
    - id: "adr:003-ai-integration"
      relation: extends
---

# Agent Provider プラグイン設計書

## 1. 概要

ADR-009に基づき、Agent実行環境のプラグイン化を設計する。既存のProvider interfaceを維持しつつ、設定ファイルからProviderを自動選択する Factory関数と、4つの内蔵Providerを追加する。

## 2. Provider Interface（変更なし）

```go
// internal/agent/agent.go — 既存。変更しない。
type Provider interface {
    Complete(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, int, error)
    Name() string
}
```

後方互換性を維持するため、interfaceの変更は行わない。

## 3. 対応プロバイダー一覧

| プロバイダー | provider名 | 認証方法 | 実装難易度 | 優先度 | 備考 |
|-------------|-----------|---------|-----------|-------|------|
| Anthropic API | `anthropic` | `ANTHROPIC_API_KEY` 環境変数 | 実装済み | P0 | 現行の`AnthropicProvider` |
| Claude Code CLI | `claude-code` | `claude` CLIがインストール済み+認証済み | M | P1 | `claude -p --model` で非対話実行 |
| OpenAI API | `openai` | `OPENAI_API_KEY` 環境変数 | S | P2 | Chat Completions API |
| GitHub Copilot | `copilot` | `gh auth` 認証済み + Copilot有効 | L | P3 | Copilot Extensions API（安定性未確認） |
| カスタム（外部コマンド） | `custom` | コマンドパス指定 | S | P1 | stdin→stdout のパイプ実行 |

## 4. teraflow.yml の agent セクション設計

### 4.1 スキーマ

```yaml
# .github/teraflow.yml
agent:
  provider: anthropic          # anthropic | claude-code | openai | copilot | custom
  model: claude-haiku-4-5-20251001  # プロバイダー固有のモデル名
  max_tokens: 4096             # デフォルト最大トークン数
  timeout: 120                 # API呼び出しタイムアウト（秒）
  custom_command: ""           # provider=custom時のコマンドパス（絶対パス必須）
  fallback: none               # none | anthropic — プロバイダー失敗時の動作
  trust_level: supervised      # supervised | autonomous — デフォルト信頼レベル
  rate_limit:
    max_calls_per_hour: 60     # 時間あたり最大呼び出し回数（0=無制限）
```

### 4.2 フォールバック動作

| `agent` セクション | `ai` セクション | 動作 |
|-------------------|----------------|------|
| 未設定 | `ai.default_provider: anthropic` | AnthropicProvider（ANTHROPIC_API_KEY必須） |
| 未設定 | 未設定 | エラー: "No AI provider configured" |
| `provider: claude-code` | — | ClaudeCodeProvider（claude CLI必須） |
| `provider: custom` | — | CustomProvider（custom_command必須） |
| `provider: openai` + `fallback: anthropic` | — | OpenAI失敗時にAnthropicへフォールバック |

### 4.3 設定例

```yaml
# 例1: Claude Code CLI（ローカル開発向け）
agent:
  provider: claude-code
  model: haiku
  max_tokens: 4096

# 例2: OpenAI + Anthropicフォールバック
agent:
  provider: openai
  model: gpt-4o
  max_tokens: 4096
  fallback: anthropic

# 例3: カスタムコマンド（社内LLM）
agent:
  provider: custom
  custom_command: /usr/local/bin/internal-llm-cli
  max_tokens: 2000

# 例4: Actions環境（API直接呼び出し）
agent:
  provider: anthropic
  model: claude-haiku-4-5-20251001
  max_tokens: 4096
  rate_limit:
    max_calls_per_hour: 100
```

## 5. Provider 実装仕様

### 5.1 ProviderConfig 型

```go
// internal/agent/config.go（新規）

// ProviderConfig はteraflow.ymlのagentセクションを表す
type ProviderConfig struct {
    Provider      string `yaml:"provider"`
    Model         string `yaml:"model"`
    MaxTokens     int    `yaml:"max_tokens"`
    Timeout       int    `yaml:"timeout"`
    CustomCommand string `yaml:"custom_command"`
    Fallback      string `yaml:"fallback"`
    TrustLevel    string `yaml:"trust_level"`
    RateLimit     struct {
        MaxCallsPerHour int `yaml:"max_calls_per_hour"`
    } `yaml:"rate_limit"`
}

// NewProviderFromConfig は設定に基づいてProviderを生成する
func NewProviderFromConfig(cfg ProviderConfig) (Provider, error) {
    switch cfg.Provider {
    case "anthropic", "":
        apiKey := os.Getenv("ANTHROPIC_API_KEY")
        if apiKey == "" {
            return nil, fmt.Errorf("E6001: ANTHROPIC_API_KEY not set")
        }
        return NewAnthropicProvider(apiKey, cfg.Model), nil

    case "claude-code":
        return NewClaudeCodeProvider(cfg.Model), nil

    case "openai":
        apiKey := os.Getenv("OPENAI_API_KEY")
        if apiKey == "" {
            return nil, fmt.Errorf("E6001: OPENAI_API_KEY not set")
        }
        return NewOpenAIProvider(apiKey, cfg.Model), nil

    case "copilot":
        return NewCopilotProvider(), nil

    case "custom":
        if cfg.CustomCommand == "" {
            return nil, fmt.Errorf("E6003: custom_command is required for provider=custom")
        }
        return NewCustomProvider(cfg.CustomCommand), nil

    default:
        return nil, fmt.Errorf("E6003: unknown agent provider: %s", cfg.Provider)
    }
}
```

### 5.2 ClaudeCodeProvider

```go
// internal/agent/provider_claude_code.go（新規）

// ClaudeCodeProvider はClaude Code CLI を使うProvider実装
type ClaudeCodeProvider struct {
    model string
}

func NewClaudeCodeProvider(model string) *ClaudeCodeProvider {
    if model == "" {
        model = "haiku"
    }
    return &ClaudeCodeProvider{model: model}
}

func (p *ClaudeCodeProvider) Name() string { return "claude-code" }

func (p *ClaudeCodeProvider) Complete(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, int, error) {
    // claude -p "prompt" --model <model> で非対話実行
    // systemPrompt + userPrompt を結合してpromptとして渡す
    prompt := fmt.Sprintf("System: %s\n\nUser: %s", systemPrompt, userPrompt)

    args := []string{"-p", prompt, "--model", p.model}

    cmd := exec.CommandContext(ctx, "claude", args...)
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return "", 0, fmt.Errorf("E6002: claude CLI failed: %s: %w", stderr.String(), err)
    }

    // Claude Code CLIはトークン使用量を返さないため0を返す
    return stdout.String(), 0, nil
}

// HealthCheck はclaude CLIの存在と認証状態を確認する
func (p *ClaudeCodeProvider) HealthCheck() error {
    _, err := exec.LookPath("claude")
    if err != nil {
        return fmt.Errorf("claude CLI not found in PATH")
    }
    // claude --version で動作確認
    cmd := exec.Command("claude", "--version")
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("claude CLI not working: %w", err)
    }
    return nil
}
```

### 5.3 OpenAIProvider

```go
// internal/agent/provider_openai.go（新規）

// OpenAIProvider はOpenAI Chat Completions APIを使うProvider実装
type OpenAIProvider struct {
    apiKey     string
    model      string
    httpClient *http.Client
}

func NewOpenAIProvider(apiKey, model string) *OpenAIProvider {
    if model == "" {
        model = "gpt-4o-mini"
    }
    return &OpenAIProvider{
        apiKey:     apiKey,
        model:      model,
        httpClient: &http.Client{Timeout: 120 * time.Second},
    }
}

func (p *OpenAIProvider) Name() string { return "openai" }

func (p *OpenAIProvider) Complete(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, int, error) {
    // POST https://api.openai.com/v1/chat/completions
    reqBody := map[string]any{
        "model":      p.model,
        "max_tokens": maxTokens,
        "messages": []map[string]string{
            {"role": "system", "content": systemPrompt},
            {"role": "user", "content": userPrompt},
        },
    }

    body, _ := json.Marshal(reqBody)
    req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+p.apiKey)

    resp, err := p.httpClient.Do(req)
    if err != nil {
        return "", 0, fmt.Errorf("E6002: OpenAI API request failed: %w", err)
    }
    defer resp.Body.Close()

    respBody, _ := io.ReadAll(resp.Body)
    if resp.StatusCode != 200 {
        return "", 0, fmt.Errorf("E6002: OpenAI API returned %d: %s", resp.StatusCode, string(respBody))
    }

    var result struct {
        Choices []struct {
            Message struct {
                Content string `json:"content"`
            } `json:"message"`
        } `json:"choices"`
        Usage struct {
            TotalTokens int `json:"total_tokens"`
        } `json:"usage"`
    }
    json.Unmarshal(respBody, &result)

    if len(result.Choices) == 0 {
        return "", 0, fmt.Errorf("empty response from OpenAI API")
    }
    return result.Choices[0].Message.Content, result.Usage.TotalTokens, nil
}
```

### 5.4 CustomProvider

```go
// internal/agent/provider_custom.go（新規）

// CustomProvider は外部コマンドを使うProvider実装
type CustomProvider struct {
    command string
}

func NewCustomProvider(command string) *CustomProvider {
    return &CustomProvider{command: command}
}

func (p *CustomProvider) Name() string { return "custom" }

func (p *CustomProvider) Complete(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, int, error) {
    // セキュリティ: コマンドパスの検証
    if !filepath.IsAbs(p.command) {
        return "", 0, fmt.Errorf("E6003: custom_command must be an absolute path: %s", p.command)
    }
    if _, err := os.Stat(p.command); err != nil {
        return "", 0, fmt.Errorf("E6003: custom_command not found: %s", p.command)
    }

    // プロンプトをJSON形式でstdinに渡す
    input := map[string]any{
        "system_prompt": systemPrompt,
        "user_prompt":   userPrompt,
        "max_tokens":    maxTokens,
    }
    inputJSON, _ := json.Marshal(input)

    cmd := exec.CommandContext(ctx, p.command)
    cmd.Stdin = bytes.NewReader(inputJSON)
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return "", 0, fmt.Errorf("E6002: custom command failed: %s: %w", stderr.String(), err)
    }

    // stdoutの内容をそのままレスポンスとして返す
    return strings.TrimSpace(stdout.String()), 0, nil
}
```

## 6. Doctor コマンド拡張

`teraflow doctor` にProvider healthcheckカテゴリを追加する。

### 6.1 チェック項目

| チェック名 | 対象Provider | 確認内容 |
|-----------|-------------|---------|
| `agent.provider_configured` | 全て | teraflow.yml または ai.default_provider が設定されているか |
| `agent.anthropic_api_key` | anthropic | ANTHROPIC_API_KEY 環境変数の存在 |
| `agent.openai_api_key` | openai | OPENAI_API_KEY 環境変数の存在 |
| `agent.claude_cli_available` | claude-code | `claude --version` が成功するか |
| `agent.custom_command_exists` | custom | custom_command のパスが存在し実行可能か |
| `agent.copilot_auth` | copilot | `gh auth status` + Copilot有効化確認 |

### 6.2 出力例

```
Agent Provider Checks
  ✅ Provider configured: anthropic
  ✅ ANTHROPIC_API_KEY: set (sk-ant-...****)
  ⚠️  Fallback provider: none (no fallback configured)
  ✅ Rate limit: 60 calls/hour
```

## 7. フォールバック機構

```go
// internal/agent/fallback.go（新規）

// FallbackProvider は主Providerの失敗時にフォールバックProviderを使う
type FallbackProvider struct {
    primary  Provider
    fallback Provider
}

func NewFallbackProvider(primary, fallback Provider) *FallbackProvider {
    return &FallbackProvider{primary: primary, fallback: fallback}
}

func (p *FallbackProvider) Name() string {
    return fmt.Sprintf("%s (fallback: %s)", p.primary.Name(), p.fallback.Name())
}

func (p *FallbackProvider) Complete(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, int, error) {
    result, tokens, err := p.primary.Complete(ctx, systemPrompt, userPrompt, maxTokens)
    if err == nil {
        return result, tokens, nil
    }
    // primaryが失敗 → fallbackを試行
    return p.fallback.Complete(ctx, systemPrompt, userPrompt, maxTokens)
}
```

## 8. セキュリティ考慮事項

### 8.1 CustomProvider のインジェクション対策

| リスク | 対策 |
|-------|------|
| コマンドインジェクション | `custom_command` は絶対パス必須。`exec.Command(command)` で直接実行（shell経由しない） |
| パストラバーサル | `filepath.IsAbs()` で絶対パス検証。`os.Stat()` で存在確認 |
| 環境変数リーク | CustomProvider実行時に不要な環境変数を除外（`cmd.Env` で制限） |
| 長時間実行 | `context.WithTimeout` でタイムアウト制御（teraflow.yml の `timeout` 設定） |

### 8.2 API キーの管理

- 環境変数のみサポート（teraflow.ymlにAPIキーを書かない）
- doctor コマンドでAPIキーの存在確認時、値はマスク表示（`sk-ant-...****`）

## 9. 実装タスク分解（足軽向け）

| タスクID | 内容 | 見積 | 依存 | 優先度 |
|---------|------|------|------|-------|
| W1 | `internal/agent/config.go` — ProviderConfig型 + NewProviderFromConfig Factory関数 | M | なし | P0 |
| W2 | `internal/agent/provider_claude_code.go` — ClaudeCodeProvider実装 + HealthCheck | M | W1 | P1 |
| W3 | `internal/agent/provider_openai.go` — OpenAIProvider実装 | S | W1 | P2 |
| W4 | `internal/agent/provider_custom.go` — CustomProvider実装（セキュリティ検証込み） | S | W1 | P1 |
| W5 | `internal/agent/fallback.go` — FallbackProvider実装 | S | W1 | P2 |
| W6 | `internal/config/config.go` 拡張 — AgentConfig読み込み（teraflow.yml → ProviderConfig変換） | M | W1 | P0 |
| W7 | `cmd/doctor.go` 拡張 — Provider healthcheckカテゴリ追加（6チェック項目） | M | W1, W6 | P1 |
| W8 | ユニットテスト — 各Provider + Factory + Fallback + Config読み込み | M | W1〜W5 | P0 |
| W9 | E2Eテスト — `teraflow doctor` の Provider チェック確認 | S | W7, W8 | P1 |

**合計**: S×3, M×5, L×0（9タスク）

### 推奨実行順序

```
W1 (config.go) → W6 (config読み込み)
    │
    ├→ W2 (ClaudeCode) ─┐
    ├→ W3 (OpenAI)     ─┤→ W8 (テスト) → W9 (E2E)
    ├→ W4 (Custom)     ─┤
    └→ W5 (Fallback)   ─┘
                          └→ W7 (doctor拡張) → W9
```

W1 + W6 完了後、W2〜W5は並列実行可能。
