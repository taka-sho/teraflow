package discovery

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DraftMetadata maps discovery state into CoDD frontmatter.
type DraftMetadata struct {
	NodeID    string
	Title     string
	DependsOn []string
	Status    string
	Source    string
}

// Draft stores CoDD sections and can be rebuilt repeatedly from state.
type Draft struct {
	Meta     DraftMetadata
	Sections map[string][]string
}

func NewDraft(meta DraftMetadata) *Draft {
	if strings.TrimSpace(meta.Status) == "" {
		meta.Status = "draft"
	}
	return &Draft{
		Meta:     meta,
		Sections: make(map[string][]string),
	}
}

// BuildDraftFromState regenerates a full markdown draft from the current discovery state.
func BuildDraftFromState(state *SessionState, meta DraftMetadata) (string, error) {
	if state == nil {
		return "", fmt.Errorf("state is nil")
	}

	if strings.TrimSpace(meta.Title) == "" {
		meta.Title = state.Title
	}
	if strings.TrimSpace(meta.NodeID) == "" {
		meta.NodeID = fmt.Sprintf("discussion-%d", state.DiscussionNumber)
	}
	if strings.TrimSpace(meta.Source) == "" {
		meta.Source = fmt.Sprintf("discussion:#%d", state.DiscussionNumber)
	}

	draft := NewDraft(meta)
	nodes := flattenBranches(state.Tree)
	for _, node := range nodes {
		heading := sectionHeadingByCategory(node.Category)
		line := formatBranchLine(node)
		draft.Sections[heading] = append(draft.Sections[heading], line)
	}

	state.RecalculateSummary()
	return draft.Render(state.Summary), nil
}

func (d *Draft) Render(progress Progress) string {
	type coddFrontmatter struct {
		NodeID    string   `yaml:"node_id"`
		Title     string   `yaml:"title"`
		DependsOn []string `yaml:"depends_on,omitempty"`
		Status    string   `yaml:"status"`
		Source    string   `yaml:"source,omitempty"`
	}
	type wrapper struct {
		CoDD coddFrontmatter `yaml:"codd"`
	}

	content, _ := yaml.Marshal(wrapper{CoDD: coddFrontmatter{
		NodeID:    strings.TrimSpace(d.Meta.NodeID),
		Title:     strings.TrimSpace(d.Meta.Title),
		DependsOn: d.Meta.DependsOn,
		Status:    strings.TrimSpace(d.Meta.Status),
		Source:    strings.TrimSpace(d.Meta.Source),
	}})

	var b strings.Builder
	b.WriteString("---\n")
	b.Write(content)
	b.WriteString("---\n\n")
	b.WriteString("# ")
	b.WriteString(strings.TrimSpace(d.Meta.Title))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("> 進捗: %d/%d 分岐解決済み (%d%%)\n", progress.Answered, progress.Total, progress.ProgressPercent))
	b.WriteString(fmt.Sprintf("> 最終更新: %s\n\n", time.Now().UTC().Format(time.RFC3339)))

	ordered := []string{
		"背景・課題",
		"スコープ",
		"機能要件",
		"非機能要件",
		"受入条件",
		"リスク・未確定事項",
		"依存関係・前提条件",
		"優先度・フェーズ",
	}
	for _, heading := range ordered {
		b.WriteString("## ")
		b.WriteString(heading)
		b.WriteString("\n\n")
		lines := append([]string(nil), d.Sections[heading]...)
		if len(lines) == 0 {
			b.WriteString("（未記入）\n\n")
			continue
		}
		sort.Strings(lines)
		for _, line := range lines {
			b.WriteString("- ")
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}

func formatBranchLine(node Branch) string {
	q := strings.TrimSpace(node.Question)
	if q == "" {
		q = node.ID
	}
	switch normalizeStatus(node.Status) {
	case StatusAnswered:
		answer := strings.TrimSpace(node.Answer)
		if answer == "" {
			answer = "（回答済み）"
		}
		return fmt.Sprintf("%s: %s", q, answer)
	case StatusSkipped:
		reason := strings.TrimSpace(node.SkipReason)
		if reason == "" {
			reason = "理由未記入"
		}
		return fmt.Sprintf("%s: %s (skipped)", q, reason)
	case StatusBlocked:
		return fmt.Sprintf("%s: **保留（依存未解決）**", q)
	default:
		return fmt.Sprintf("%s: **未確定**", q)
	}
}

func sectionHeadingByCategory(category string) string {
	switch strings.TrimSpace(strings.ToLower(category)) {
	case "scope":
		return "スコープ"
	case "functional":
		return "機能要件"
	case "non_functional":
		return "非機能要件"
	case "acceptance":
		return "受入条件"
	case "risk":
		return "リスク・未確定事項"
	case "dependency":
		return "依存関係・前提条件"
	case "priority":
		return "優先度・フェーズ"
	default:
		return "背景・課題"
	}
}

func flattenBranches(nodes []Branch) []Branch {
	out := make([]Branch, 0)
	var walk func([]Branch)
	walk = func(items []Branch) {
		for _, node := range items {
			out = append(out, node)
			if len(node.Children) > 0 {
				walk(node.Children)
			}
		}
	}
	walk(nodes)
	return out
}
