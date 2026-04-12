package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// TemplateQuestion defines one discovery question template.
type TemplateQuestion struct {
	ID             string   `yaml:"id" json:"id"`
	Question       string   `yaml:"question" json:"question"`
	Category       string   `yaml:"category" json:"category"`
	Recommendation string   `yaml:"recommendation,omitempty" json:"recommendation,omitempty"`
	DependsOn      []string `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
}

// DecisionTreeTemplate defines category-based initial tree templates.
type DecisionTreeTemplate struct {
	Version    string                        `yaml:"version" json:"version"`
	Categories map[string][]TemplateQuestion `yaml:"categories" json:"categories"`
}

// LLMGenerator generates initial tree seed from discussion text.
type LLMGenerator interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

// TreeBuildOptions controls optional context for LLM-based tree generation.
type TreeBuildOptions struct {
	DocContext string
}

func DefaultTemplate() DecisionTreeTemplate {
	return DecisionTreeTemplate{
		Version: "1",
		Categories: map[string][]TemplateQuestion{
			"scope": {
				{ID: "scope.target_users", Question: "この機能の対象ユーザーは誰ですか？", Category: "scope"},
				{ID: "scope.boundary", Question: "この機能のスコープ外は何ですか？", Category: "scope"},
				{ID: "scope.platform", Question: "対象プラットフォームは？（Web/Mobile/API/CLI等）", Category: "scope"},
			},
			"functional": {
				{ID: "func.happy_path", Question: "正常系のメインフローを教えてください", Category: "functional"},
				{ID: "func.error_handling", Question: "エラー時の振る舞いは？（バリデーション失敗、外部API障害等）", Category: "functional", Recommendation: "エラーメッセージを表示し、入力を保持してリトライ可能にする"},
				{ID: "func.edge_cases", Question: "考慮すべきエッジケースはありますか？", Category: "functional"},
			},
			"non_functional": {
				{ID: "nfr.performance", Question: "レスポンスタイム要件は？", Category: "non_functional", Recommendation: "p95 < 500ms"},
				{ID: "nfr.availability", Question: "可用性要件は？（SLA等）", Category: "non_functional", Recommendation: "99.9%"},
				{ID: "nfr.scalability", Question: "想定ユーザー数/トラフィック量は？", Category: "non_functional"},
			},
			"acceptance": {
				{ID: "ac.done_definition", Question: "この要件の「完了」の定義は？", Category: "acceptance"},
				{ID: "ac.test_scenarios", Question: "必須のテストシナリオは？", Category: "acceptance", Recommendation: "正常系E2E + 異常系バリデーション + パフォーマンステスト"},
			},
			"risk": {
				{ID: "risk.security", Question: "セキュリティ上の懸念事項は？", Category: "risk"},
				{ID: "risk.data_migration", Question: "既存データへの影響はありますか？", Category: "risk"},
				{ID: "risk.rollback", Question: "ロールバック計画は必要ですか？", Category: "risk", Recommendation: "Feature flag で段階的リリース"},
			},
			"dependency": {
				{ID: "dep.external", Question: "外部依存（API/ミドルウェア/認証基盤）はありますか？", Category: "dependency"},
				{ID: "dep.integration", Question: "既存システムとの統合ポイントはどこですか？", Category: "dependency"},
			},
			"priority": {
				{ID: "prio.phase1", Question: "Phase 1 で必須の要件はどれですか？", Category: "priority"},
				{ID: "prio.phase2", Question: "Phase 2 以降に回せる要件はどれですか？", Category: "priority"},
			},
		},
	}
}

func LoadTemplate(path string) (DecisionTreeTemplate, error) {
	if strings.TrimSpace(path) == "" {
		return DefaultTemplate(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return DecisionTreeTemplate{}, fmt.Errorf("read template %s: %w", path, err)
	}
	var tmpl DecisionTreeTemplate
	if err := yaml.Unmarshal(data, &tmpl); err != nil {
		return DecisionTreeTemplate{}, fmt.Errorf("parse template %s: %w", path, err)
	}
	if strings.TrimSpace(tmpl.Version) == "" {
		tmpl.Version = "1"
	}
	if len(tmpl.Categories) == 0 {
		return DecisionTreeTemplate{}, fmt.Errorf("template has no categories")
	}
	return tmpl, nil
}

func BuildInitialTree(tmpl DecisionTreeTemplate, categories []string) []Branch {
	if len(categories) == 0 {
		for name := range tmpl.Categories {
			categories = append(categories, name)
		}
		sort.Strings(categories)
	}

	out := make([]Branch, 0)
	for _, category := range categories {
		questions := tmpl.Categories[category]
		for _, q := range questions {
			out = append(out, Branch{
				ID:             strings.TrimSpace(q.ID),
				Question:       strings.TrimSpace(q.Question),
				Category:       strings.TrimSpace(q.Category),
				Status:         StatusPending,
				Recommendation: strings.TrimSpace(q.Recommendation),
				DependsOn:      append([]string(nil), q.DependsOn...),
			})
		}
	}
	return out
}

func BuildInitialTreeWithLLM(ctx context.Context, llm LLMGenerator, discussionBody string, fallback DecisionTreeTemplate, opts TreeBuildOptions) ([]Branch, error) {
	if llm == nil {
		return BuildInitialTree(fallback, nil), nil
	}

	var promptBuilder strings.Builder
	promptBuilder.WriteString("投稿内容から要件探索の決定木を生成し、JSON配列で返してください。各要素は id/question/category/recommendation/depends_on を持つこと。")
	trimmedDocContext := strings.TrimSpace(opts.DocContext)
	if trimmedDocContext != "" {
		promptBuilder.WriteString("\n\n【既存文書コンテキスト】\n")
		promptBuilder.WriteString(trimmedDocContext)
	}
	promptBuilder.WriteString("\n\n【投稿内容】\n")
	promptBuilder.WriteString(strings.TrimSpace(discussionBody))

	prompt := promptBuilder.String()
	raw, err := llm.Generate(ctx, prompt)
	if err != nil {
		return BuildInitialTree(fallback, nil), nil
	}

	var rows []TemplateQuestion
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &rows); err != nil {
		return BuildInitialTree(fallback, nil), nil
	}

	out := make([]Branch, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.ID) == "" || strings.TrimSpace(row.Question) == "" {
			continue
		}
		category := strings.TrimSpace(row.Category)
		if category == "" {
			category = "functional"
		}
		out = append(out, Branch{
			ID:             strings.TrimSpace(row.ID),
			Question:       strings.TrimSpace(row.Question),
			Category:       category,
			Status:         StatusPending,
			Recommendation: strings.TrimSpace(row.Recommendation),
			DependsOn:      append([]string(nil), row.DependsOn...),
		})
	}
	if len(out) == 0 {
		return BuildInitialTree(fallback, nil), nil
	}
	return out, nil
}
