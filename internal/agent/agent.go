package agent

import (
	"context"
	"fmt"
	"time"
)

// AgentType は Agent の種類を表す
type AgentType string

const (
	AgentTypeRequirements AgentType = "requirements"
	AgentTypeReview       AgentType = "review"
	AgentTypeImplement    AgentType = "implement"
	AgentTypeCIFix        AgentType = "ci-fix"
	AgentTypeConflict     AgentType = "conflict"
	AgentTypeIncident     AgentType = "incident"
	AgentTypeMaintenance  AgentType = "maintenance"
)

// TrustLevel はエージェントの信頼レベルを表す
type TrustLevel string

const (
	TrustLevelSupervised TrustLevel = "supervised" // 要人間レビュー
	TrustLevelAutonomous TrustLevel = "autonomous" // 自動マージ可
)

// AgentContext はエージェントに渡すコンテキスト情報
type AgentContext struct {
	Type         AgentType         `json:"type"`
	TrustLevel   TrustLevel        `json:"trust_level"`
	Input        string            `json:"input"`              // メイン入力（Issue body, PR diff等）
	Metadata     map[string]string `json:"metadata,omitempty"` // 追加情報
	ConfigPath   string            `json:"config_path"`
	MaxTokens    int               `json:"max_tokens,omitempty"`
	SystemPrompt string            `json:"system_prompt,omitempty"`
}

// AgentResult はエージェントの実行結果
type AgentResult struct {
	Type       AgentType `json:"type"`
	Output     string    `json:"output"`
	Success    bool      `json:"success"`
	Error      string    `json:"error,omitempty"`
	TokensUsed int       `json:"tokens_used,omitempty"`
	ExecutedAt time.Time `json:"executed_at"`
}

// Provider はAI API プロバイダーのインターフェース
type Provider interface {
	Complete(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, int, error)
	Name() string
}

// AgentManager は Agent の実行を管理する
type AgentManager struct {
	provider Provider
}

// NewAgentManager は AgentManager を作成する
func NewAgentManager(provider Provider) *AgentManager {
	return &AgentManager{provider: provider}
}

// Run はエージェントを実行する
func (m *AgentManager) Run(ctx context.Context, agentCtx AgentContext) (*AgentResult, error) {
	result := &AgentResult{
		Type:       agentCtx.Type,
		ExecutedAt: time.Now(),
	}

	systemPrompt := agentCtx.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = GetSystemPrompt(agentCtx.Type)
	}
	if systemPrompt == "" {
		return nil, fmt.Errorf("unknown agent type: %s", agentCtx.Type)
	}

	maxTokens := agentCtx.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	output, tokens, err := m.provider.Complete(ctx, systemPrompt, agentCtx.Input, maxTokens)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, fmt.Errorf("agent %s failed: %w", agentCtx.Type, err)
	}

	result.Output = output
	result.TokensUsed = tokens
	result.Success = true
	return result, nil
}

// GetSystemPrompt はエージェントタイプ別のシステムプロンプトを返す
func GetSystemPrompt(agentType AgentType) string {
	prompts := map[AgentType]string{
		AgentTypeRequirements: `あなたはソフトウェア要件コンサルタントです。
ユーザーと壁打ちしながら要件を深めてください。

通常応答時:
- 不明点や曖昧な部分があれば質問してください
- 既存の要件との矛盾や重複があれば指摘してください
- 代替案や改善案があれば提示してください
- 一方的に整理するのではなく、対話を通じて要件を固めてください

ユーザーが「要求確定」と発言した場合:
- 「📋 要件確定」ヘッダーで始める
- 会話全体を踏まえた最終要件を構造化して出力する
- 形式: マークダウンのリスト「- [ ] <要件>」

通常の対話応答は「💬 AI壁打ち」ヘッダーで始めること。`,

		AgentTypeReview: `あなたはシニアソフトウェアエンジニアです。
提供されたコードの差分（diff）をレビューし、問題点や改善提案を指摘してください。
観点: セキュリティ、パフォーマンス、可読性、テスト網羅性、設計の一貫性。
出力形式: マークダウン形式で、各指摘を severity（high/medium/low）付きで記載。`,

		AgentTypeImplement: `あなたはソフトウェアエンジニアです。
提供された要件に基づいてコードを実装してください。
既存のコードスタイルと一貫性を保ち、テストコードも含めてください。`,

		AgentTypeCIFix: `あなたはCI/CD の専門家です。
提供されたCI失敗ログを分析し、失敗原因と修正方法を提案してください。
可能な場合は具体的な修正コードも提示してください。`,

		AgentTypeConflict: `あなたはGitのコンフリクト解消の専門家です。
提供されたコンフリクト内容を分析し、最適な解消方法を提案してください。`,

		AgentTypeIncident: `あなたはSREエンジニアです。
提供されたインシデント情報を分析し、原因調査と対応策を提案してください。
5W1H形式で整理し、緊急度と影響範囲を明確にしてください。`,

		AgentTypeMaintenance: `あなたはソフトウェアメンテナンスの専門家です。
提供されたプロジェクト状態を分析し、保守スコアと改善提案を出力してください。
スコア基準: テストカバレッジ、ドキュメント整備度、技術的負債、依存関係の新鮮度。`,
	}
	return prompts[agentType]
}
