package discovery

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	TemplateCategoryRequired    = "required"
	TemplateCategoryRecommended = "recommended"
	TemplateCategoryOptional    = "optional"
)

// TemplateItem defines one requirement field in the requirement template.
type TemplateItem struct {
	ID                    string            `yaml:"id" json:"id"`
	Name                  string            `yaml:"name" json:"name"`
	Category              string            `yaml:"category" json:"category"`
	Description           string            `yaml:"description,omitempty" json:"description,omitempty"`
	PromptHint            string            `yaml:"prompt_hint,omitempty" json:"prompt_hint,omitempty"`
	Metadata              map[string]string `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	Origin                string            `yaml:"origin,omitempty" json:"origin,omitempty"`
	AddedAt               string            `yaml:"added_at,omitempty" json:"added_at,omitempty"`
	DefaultRecommendation string            `yaml:"default_recommendation,omitempty" json:"default_recommendation,omitempty"`
	DependsOn             []string          `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
}

// RequirementTemplate is the normalized discovery template model.
type RequirementTemplate struct {
	Version  string            `yaml:"version,omitempty" json:"version,omitempty"`
	Metadata map[string]string `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	Items    []TemplateItem    `yaml:"items" json:"items"`
}

type legacyRequirementTemplateFile struct {
	Version  string            `yaml:"version"`
	Metadata map[string]string `yaml:"metadata,omitempty"`
	Items    []TemplateItem    `yaml:"items"`
	Template struct {
		Sections []legacyTemplateSection `yaml:"sections"`
	} `yaml:"template"`
}

type legacyTemplateSection struct {
	ID       string              `yaml:"id"`
	Title    string              `yaml:"title"`
	Priority string              `yaml:"priority"`
	Fields   []legacyTemplateRow `yaml:"fields"`
}

type legacyTemplateRow struct {
	ID                    string `yaml:"id"`
	Label                 string `yaml:"label"`
	Priority              string `yaml:"priority"`
	Hint                  string `yaml:"hint"`
	PromptHint            string `yaml:"prompt_hint,omitempty"`
	Origin                string `yaml:"origin,omitempty"`
	AddedAt               string `yaml:"added_at,omitempty"`
	DefaultRecommendation string `yaml:"default_recommendation,omitempty"`
}

// NewDefaultTemplate returns the built-in requirement template (31 fields).
func NewDefaultTemplate() RequirementTemplate {
	items := []TemplateItem{
		{ID: "project_overview", Name: "プロジェクト概要", Category: TemplateCategoryRequired, Description: "プロジェクトの目的と全体像", PromptHint: "このプロジェクトが解決する課題と期待成果を教えてください", Origin: "builtin"},
		{ID: "business_background", Name: "背景・動機", Category: TemplateCategoryRequired, Description: "この取り組みの背景と必要性", PromptHint: "なぜ今この取り組みが必要なのかを説明してください", Origin: "builtin"},
		{ID: "success_criteria", Name: "成功基準", Category: TemplateCategoryRequired, Description: "何をもって成功とするか", PromptHint: "成功判定に使う指標や条件を教えてください", Origin: "builtin"},
		{ID: "stakeholders", Name: "主要ステークホルダー", Category: TemplateCategoryRequired, Description: "意思決定に関わる関係者", PromptHint: "関係者とそれぞれの役割を教えてください", Origin: "builtin"},
		{ID: "target_users", Name: "対象ユーザー", Category: TemplateCategoryRequired, Description: "この機能を利用する主対象", PromptHint: "主な利用者像を具体的に教えてください", Origin: "builtin"},
		{ID: "functional_requirements", Name: "機能要件", Category: TemplateCategoryRequired, Description: "提供すべき機能の一覧", PromptHint: "最低限必要な機能を列挙してください", Origin: "builtin"},
		{ID: "core_features", Name: "必須機能", Category: TemplateCategoryRequired, Description: "Phase 1で必ず提供する機能", PromptHint: "初期リリースで必須となる機能を教えてください", Origin: "builtin"},
		{ID: "user_flow", Name: "正常系フロー", Category: TemplateCategoryRequired, Description: "ユーザーが辿る主要フロー", PromptHint: "ユーザー操作の主要な流れを説明してください", Origin: "builtin"},
		{ID: "non_functional_requirements", Name: "非機能要件", Category: TemplateCategoryRequired, Description: "性能・可用性・運用品質の要件", PromptHint: "性能や品質に関する要件を教えてください", Origin: "builtin"},
		{ID: "security_requirements", Name: "セキュリティ要件", Category: TemplateCategoryRequired, Description: "認証・認可・データ保護の要件", PromptHint: "守るべきセキュリティ要件を教えてください", Origin: "builtin"},
		{ID: "performance_requirements", Name: "パフォーマンス要件", Category: TemplateCategoryRequired, Description: "応答時間や処理性能の基準", PromptHint: "性能目標（例: p95）を教えてください", Origin: "builtin", DefaultRecommendation: "p95 < 500ms"},
		{ID: "constraints", Name: "制約条件", Category: TemplateCategoryRequired, Description: "技術・運用・組織上の制約", PromptHint: "前提となる制約条件を教えてください", Origin: "builtin"},
		{ID: "timeline", Name: "期限・マイルストーン", Category: TemplateCategoryRequired, Description: "必達期限と段階目標", PromptHint: "期限と主要マイルストーンを教えてください", Origin: "builtin"},
		{ID: "acceptance_criteria", Name: "受入条件", Category: TemplateCategoryRequired, Description: "完了判定の条件", PromptHint: "受入時に満たすべき条件を教えてください", Origin: "builtin"},

		{ID: "decision_makers", Name: "意思決定者", Category: TemplateCategoryRecommended, Description: "最終判断を行う責任者", PromptHint: "承認権限を持つ担当者を教えてください", Origin: "builtin"},
		{ID: "affected_teams", Name: "影響を受けるチーム", Category: TemplateCategoryRecommended, Description: "導入影響がある関連チーム", PromptHint: "影響範囲となるチームを教えてください", Origin: "builtin"},
		{ID: "error_handling", Name: "エラー処理方針", Category: TemplateCategoryRecommended, Description: "障害発生時の振る舞い", PromptHint: "異常時の挙動と復旧方針を教えてください", Origin: "builtin", DefaultRecommendation: "エラーメッセージを表示し、入力を保持してリトライ可能にする"},
		{ID: "availability_requirements", Name: "可用性要件", Category: TemplateCategoryRecommended, Description: "SLAや稼働率の目標", PromptHint: "可用性目標（例: 99.9%）を教えてください", Origin: "builtin", DefaultRecommendation: "99.9%"},
		{ID: "scalability_requirements", Name: "想定規模", Category: TemplateCategoryRecommended, Description: "想定ユーザー数・トラフィック", PromptHint: "想定する利用規模を教えてください", Origin: "builtin"},
		{ID: "budget_constraints", Name: "予算制約", Category: TemplateCategoryRecommended, Description: "予算・リソース上限", PromptHint: "予算や工数の制約を教えてください", Origin: "builtin"},
		{ID: "test_scenarios", Name: "必須テストシナリオ", Category: TemplateCategoryRecommended, Description: "検証すべき重要シナリオ", PromptHint: "必須テスト観点を教えてください", Origin: "builtin", DefaultRecommendation: "正常系E2E + 異常系バリデーション + パフォーマンステスト"},
		{ID: "priority_phasing", Name: "優先順位とフェーズ分け", Category: TemplateCategoryRecommended, Description: "段階的リリース計画", PromptHint: "Must/Should/Couldの優先度を教えてください", Origin: "builtin"},
		{ID: "integration_points", Name: "統合ポイント", Category: TemplateCategoryRecommended, Description: "既存システムとの接続点", PromptHint: "他システムとの連携点を教えてください", Origin: "builtin"},
		{ID: "identified_risks", Name: "特定済みリスク", Category: TemplateCategoryRecommended, Description: "実装前に把握しているリスク", PromptHint: "現時点で懸念しているリスクを教えてください", Origin: "builtin"},

		{ID: "edge_cases", Name: "エッジケース", Category: TemplateCategoryOptional, Description: "例外系ケースの考慮", PromptHint: "想定される例外ケースがあれば教えてください", Origin: "builtin"},
		{ID: "data_migration", Name: "データ移行", Category: TemplateCategoryOptional, Description: "既存データ移行の有無", PromptHint: "データ移行が必要か教えてください", Origin: "builtin"},
		{ID: "rollback_plan", Name: "ロールバック計画", Category: TemplateCategoryOptional, Description: "障害時の切り戻し戦略", PromptHint: "切り戻し手順が必要なら教えてください", Origin: "builtin", DefaultRecommendation: "Feature flag で段階的リリース"},
		{ID: "design_direction", Name: "デザイン方針", Category: TemplateCategoryOptional, Description: "UI/UXの方向性", PromptHint: "UI/UX方針があれば教えてください", Origin: "builtin"},
		{ID: "accessibility", Name: "アクセシビリティ要件", Category: TemplateCategoryOptional, Description: "配慮すべきアクセシビリティ", PromptHint: "必要なアクセシビリティ要件を教えてください", Origin: "builtin"},
		{ID: "operations_monitoring", Name: "監視・アラート", Category: TemplateCategoryOptional, Description: "運用監視の要件", PromptHint: "必要な監視指標と通知条件を教えてください", Origin: "builtin"},
		{ID: "deployment_strategy", Name: "デプロイ方針", Category: TemplateCategoryOptional, Description: "リリース戦略", PromptHint: "デプロイ方式と展開戦略を教えてください", Origin: "builtin"},
	}

	return RequirementTemplate{
		Version: "1",
		Metadata: map[string]string{
			"name":        "default",
			"description": "goal-driven discovery default template",
			"source":      "builtin",
		},
		Items: items,
	}
}

// DefaultRequirementTemplate keeps compatibility with design naming.
func DefaultRequirementTemplate() RequirementTemplate {
	return NewDefaultTemplate()
}

// DefaultTemplate keeps backward compatibility with the old API name.
func DefaultTemplate() DecisionTreeTemplate {
	return NewDefaultTemplate()
}

// LoadRequirementTemplate loads a custom template or returns default.
// If path is empty, it prefers .teraflow/requirement-template.yml.
func LoadRequirementTemplate(path string) (RequirementTemplate, error) {
	if strings.TrimSpace(path) != "" {
		return loadRequirementTemplateFile(path)
	}

	for _, candidate := range []string{
		filepath.FromSlash(".teraflow/requirement-template.yml"),
		filepath.FromSlash(".teraflow/requirement-template.yaml"),
		filepath.FromSlash(".teraflow/discovery/requirement-template.yaml"),
	} {
		if _, err := os.Stat(candidate); err == nil {
			return loadRequirementTemplateFile(candidate)
		}
	}

	return NewDefaultTemplate(), nil
}

// LoadTemplate keeps backward compatibility with the old API name.
func LoadTemplate(path string) (DecisionTreeTemplate, error) {
	return LoadRequirementTemplate(path)
}

func loadRequirementTemplateFile(path string) (RequirementTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RequirementTemplate{}, fmt.Errorf("read template %s: %w", path, err)
	}

	var raw legacyRequirementTemplateFile
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return RequirementTemplate{}, fmt.Errorf("parse template %s: %w", path, err)
	}

	tmpl := RequirementTemplate{
		Version:  strings.TrimSpace(raw.Version),
		Metadata: raw.Metadata,
		Items:    append([]TemplateItem(nil), raw.Items...),
	}
	if tmpl.Version == "" {
		tmpl.Version = "1"
	}

	if len(tmpl.Items) == 0 && len(raw.Template.Sections) > 0 {
		tmpl.Items = flattenLegacySections(raw.Template.Sections)
	}
	if len(tmpl.Items) == 0 {
		return RequirementTemplate{}, errors.New("template has no items")
	}

	for i := range tmpl.Items {
		tmpl.Items[i].ID = strings.TrimSpace(tmpl.Items[i].ID)
		tmpl.Items[i].Name = strings.TrimSpace(tmpl.Items[i].Name)
		tmpl.Items[i].Category = normalizeTemplateCategory(tmpl.Items[i].Category)
		if tmpl.Items[i].Origin == "" {
			tmpl.Items[i].Origin = "user"
		}
		if tmpl.Items[i].ID == "" || tmpl.Items[i].Name == "" {
			return RequirementTemplate{}, fmt.Errorf("template item at index %d is invalid", i)
		}
	}
	return tmpl, nil
}

func flattenLegacySections(sections []legacyTemplateSection) []TemplateItem {
	items := make([]TemplateItem, 0)
	for _, section := range sections {
		for _, field := range section.Fields {
			category := strings.TrimSpace(field.Priority)
			if category == "" {
				category = section.Priority
			}
			items = append(items, TemplateItem{
				ID:                    strings.TrimSpace(field.ID),
				Name:                  strings.TrimSpace(field.Label),
				Category:              normalizeTemplateCategory(category),
				Description:           strings.TrimSpace(field.Hint),
				PromptHint:            strings.TrimSpace(field.PromptHint),
				Origin:                strings.TrimSpace(field.Origin),
				AddedAt:               strings.TrimSpace(field.AddedAt),
				DefaultRecommendation: strings.TrimSpace(field.DefaultRecommendation),
				Metadata: map[string]string{
					"section_id":    strings.TrimSpace(section.ID),
					"section_title": strings.TrimSpace(section.Title),
				},
			})
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	return items
}

func normalizeTemplateCategory(in string) string {
	switch strings.ToLower(strings.TrimSpace(in)) {
	case TemplateCategoryRequired:
		return TemplateCategoryRequired
	case TemplateCategoryRecommended:
		return TemplateCategoryRecommended
	case TemplateCategoryOptional:
		return TemplateCategoryOptional
	default:
		return TemplateCategoryOptional
	}
}

// DecisionTreeTemplate remains as compatibility alias.
type DecisionTreeTemplate = RequirementTemplate
