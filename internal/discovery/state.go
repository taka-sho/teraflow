package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	StatusPending  = "pending"
	StatusAnswered = "answered"
	StatusSkipped  = "skipped"
	StatusBlocked  = "blocked"
	StatusResolved = "resolved"
)

// SessionState tracks discovery progress for one GitHub discussion.
// v3: PendingRecommendations/AcceptedRecommendations added for Phase2 template recommendations.
type SessionState struct {
	Version          string             `yaml:"version"`
	DiscussionNumber int                `yaml:"discussion_number"`
	Title            string             `yaml:"title"`
	CreatedAt        string             `yaml:"created_at"`
	UpdatedAt        string             `yaml:"updated_at"`
	Mode             string             `yaml:"mode,omitempty"`
	Tree             []Branch           `yaml:"tree"`
	Summary          Progress           `yaml:"summary"`
	Fulfillment      FulfillmentSummary `yaml:"fulfillment,omitempty"`
	// Phase2: テンプレートレコメンド（Branch.Recommendationとは別物）
	PendingRecommendations  []PresentedRecommendation  `yaml:"pending_recommendations,omitempty"`
	AcceptedRecommendations []AcceptedRecommendation   `yaml:"accepted_recommendations,omitempty"`
}

// Progress holds aggregate node counts.
type Progress struct {
	Total           int `yaml:"total" json:"total"`
	Answered        int `yaml:"answered" json:"answered"`
	Resolved        int `yaml:"resolved,omitempty" json:"resolved,omitempty"`
	Pending         int `yaml:"pending" json:"pending"`
	Skipped         int `yaml:"skipped" json:"skipped"`
	Blocked         int `yaml:"blocked,omitempty" json:"blocked,omitempty"`
	ProgressPercent int `yaml:"progress_percent" json:"progress_percent"`
}

// Branch is one decision tree node.
type Branch struct {
	ID             string   `yaml:"id" json:"id"`
	Question       string   `yaml:"question" json:"question"`
	Category       string   `yaml:"category,omitempty" json:"category,omitempty"`
	Status         string   `yaml:"status" json:"status"`
	Recommendation string   `yaml:"recommendation,omitempty" json:"recommendation,omitempty"`
	Answer         string   `yaml:"answer,omitempty" json:"answer,omitempty"`
	SkipReason     string   `yaml:"skip_reason,omitempty" json:"skip_reason,omitempty"`
	DependsOn      []string `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	ResolvedAt     string   `yaml:"resolved_at,omitempty" json:"resolved_at,omitempty"`
	ResolvedBy     string   `yaml:"resolved_by,omitempty" json:"resolved_by,omitempty"`
	Children       []Branch `yaml:"children,omitempty" json:"children,omitempty"`
}

// PresentedRecommendation はユーザーに提示されたレコメンドを表す（Phase2）
type PresentedRecommendation struct {
	Index       int    `yaml:"index"`        // ユーザー返信時の番号 (1始まり)
	FieldID     string `yaml:"field_id"`
	Name        string `yaml:"name"`
	Category    string `yaml:"category"`
	Type        string `yaml:"type"`         // promote_to_default / delete_warning
	PresentedAt string `yaml:"presented_at"` // RFC3339 UTC
}

// AcceptedRecommendation はユーザーが採用したレコメンドを表す（Phase2）
type AcceptedRecommendation struct {
	FieldID    string `yaml:"field_id"`
	AcceptedAt string `yaml:"accepted_at"` // RFC3339 UTC
}

// FulfillmentSummary stores template fulfillment in state v2.
type FulfillmentSummary struct {
	Map                  FulfillmentMap `yaml:"map,omitempty" json:"map,omitempty"`
	RequiredTotal        int            `yaml:"required_total" json:"required_total"`
	RequiredFulfilled    int            `yaml:"required_fulfilled" json:"required_fulfilled"`
	RecommendedTotal     int            `yaml:"recommended_total,omitempty" json:"recommended_total,omitempty"`
	RecommendedFulfilled int            `yaml:"recommended_fulfilled,omitempty" json:"recommended_fulfilled,omitempty"`
	MissingRequiredIDs   []string       `yaml:"missing_required_ids,omitempty" json:"missing_required_ids,omitempty"`
}

func NewSessionState(discussionNumber int, title string) *SessionState {
	now := time.Now().UTC().Format(time.RFC3339)
	return &SessionState{
		Version:          "3",
		DiscussionNumber: discussionNumber,
		Title:            strings.TrimSpace(title),
		CreatedAt:        now,
		UpdatedAt:        now,
		Mode:             "sequential",
		Tree:             []Branch{},
		Summary: Progress{
			ProgressPercent: 0,
		},
		Fulfillment: FulfillmentSummary{
			Map: FulfillmentMap{},
		},
	}
}

func LoadSessionState(path string) (*SessionState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state SessionState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if strings.TrimSpace(state.Version) == "" {
		state.Version = "1"
	}
	if state.Fulfillment.Map == nil {
		state.Fulfillment.Map = FulfillmentMap{}
	}
	state.RecalculateSummary()
	return &state, nil
}

func (s *SessionState) Save(path string) error {
	if s == nil {
		return fmt.Errorf("session state is nil")
	}
	s.RecalculateSummary()
	s.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if s.CreatedAt == "" {
		s.CreatedAt = s.UpdatedAt
	}
	if v := strings.TrimSpace(s.Version); v == "" || v == "1" || v == "2" {
		s.Version = "3"
	}
	if s.Fulfillment.Map == nil {
		s.Fulfillment.Map = FulfillmentMap{}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir for state file: %w", err)
	}
	content, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}
	return nil
}

// UpdateFulfillment calculates and stores fulfillment from the given template.
func (s *SessionState) UpdateFulfillment(tmpl RequirementTemplate) {
	if s == nil {
		return
	}
	result := AnalyzeGap(s, tmpl)
	missing := make([]string, 0, len(result.MissingRequired))
	for _, item := range result.MissingRequired {
		if id := strings.TrimSpace(item.ID); id != "" {
			missing = append(missing, id)
		}
	}
	s.Fulfillment = FulfillmentSummary{
		Map:                  result.Fulfillment,
		RequiredTotal:        result.RequiredTotal,
		RequiredFulfilled:    result.RequiredFulfilled,
		RecommendedTotal:     result.RecommendedTotal,
		RecommendedFulfilled: result.RecommendedFilled,
		MissingRequiredIDs:   missing,
	}
}

func (s *SessionState) RecalculateSummary() {
	if s == nil {
		return
	}
	progress := Progress{}
	walkBranches(s.Tree, func(node *Branch) {
		progress.Total++
		switch normalizeStatus(node.Status) {
		case StatusAnswered:
			progress.Answered++
		case StatusSkipped:
			progress.Skipped++
		case StatusBlocked:
			progress.Blocked++
		case StatusPending:
			progress.Pending++
		default:
			progress.Pending++
		}
	})
	progress.Resolved = progress.Answered
	if progress.Total > 0 {
		progress.ProgressPercent = int(float64(progress.Answered+progress.Skipped) / float64(progress.Total) * 100.0)
	}
	s.Summary = progress
}

func (s *SessionState) IsComplete() bool {
	if s == nil {
		return false
	}
	s.RecalculateSummary()
	return s.Summary.Pending == 0 && s.Summary.Blocked == 0
}

func (s *SessionState) CompletionReport() string {
	if s == nil {
		return ""
	}
	s.RecalculateSummary()
	return fmt.Sprintf(
		"要件探索完了: %d/%d 解決済み, %d スキップ, 進捗 %d%%",
		s.Summary.Answered, s.Summary.Total,
		s.Summary.Skipped, s.Summary.ProgressPercent,
	)
}

func (s *SessionState) FindBranch(id string) *Branch {
	if s == nil {
		return nil
	}
	for i := range s.Tree {
		if node := findBranchByID(&s.Tree[i], id); node != nil {
			return node
		}
	}
	return nil
}

func (s *SessionState) AddBranch(parentID string, branch Branch) error {
	if s == nil {
		return fmt.Errorf("session state is nil")
	}
	branch.ID = strings.TrimSpace(branch.ID)
	if branch.ID == "" {
		return fmt.Errorf("branch id is required")
	}
	if s.FindBranch(branch.ID) != nil {
		return fmt.Errorf("branch %q already exists", branch.ID)
	}
	branch.Status = normalizeStatus(branch.Status)

	if strings.TrimSpace(parentID) == "" {
		s.Tree = append(s.Tree, branch)
		s.RecalculateSummary()
		return nil
	}
	parent := s.FindBranch(parentID)
	if parent == nil {
		return fmt.Errorf("parent branch %q not found", parentID)
	}
	parent.Children = append(parent.Children, branch)
	s.RecalculateSummary()
	return nil
}

func (s *SessionState) MarkAnswered(id, answer, resolvedBy string) error {
	node := s.FindBranch(id)
	if node == nil {
		return fmt.Errorf("branch %q not found", id)
	}
	node.Status = StatusAnswered
	node.Answer = strings.TrimSpace(answer)
	node.SkipReason = ""
	node.ResolvedBy = strings.TrimSpace(resolvedBy)
	if node.ResolvedBy == "" {
		node.ResolvedBy = "user"
	}
	node.ResolvedAt = time.Now().UTC().Format(time.RFC3339)
	s.UpdateBlockedByDependencies()
	s.RecalculateSummary()
	return nil
}

func (s *SessionState) MarkSkipped(id, reason string) error {
	node := s.FindBranch(id)
	if node == nil {
		return fmt.Errorf("branch %q not found", id)
	}
	node.Status = StatusSkipped
	node.SkipReason = strings.TrimSpace(reason)
	node.Answer = ""
	node.ResolvedBy = "user"
	node.ResolvedAt = time.Now().UTC().Format(time.RFC3339)
	s.UpdateBlockedByDependencies()
	s.RecalculateSummary()
	return nil
}

func (s *SessionState) MarkBlocked(id string) error {
	node := s.FindBranch(id)
	if node == nil {
		return fmt.Errorf("branch %q not found", id)
	}
	node.Status = StatusBlocked
	s.RecalculateSummary()
	return nil
}

func (s *SessionState) PendingBranches() []Branch {
	if s == nil {
		return nil
	}
	out := make([]Branch, 0)
	walkBranches(s.Tree, func(node *Branch) {
		if normalizeStatus(node.Status) == StatusPending {
			out = append(out, *node)
		}
	})
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *SessionState) NextQuestions(limit int) []Branch {
	if s == nil {
		return nil
	}
	if limit <= 0 {
		limit = 1
	}
	available := make([]Branch, 0)
	walkBranches(s.Tree, func(node *Branch) {
		if normalizeStatus(node.Status) != StatusPending {
			return
		}
		if !s.DependenciesSatisfied(node.DependsOn) {
			return
		}
		available = append(available, *node)
	})
	sort.Slice(available, func(i, j int) bool { return available[i].ID < available[j].ID })
	if len(available) > limit {
		return available[:limit]
	}
	return available
}

func (s *SessionState) DependenciesSatisfied(dependsOn []string) bool {
	for _, dep := range dependsOn {
		node := s.FindBranch(dep)
		if node == nil {
			return false
		}
		status := normalizeStatus(node.Status)
		if status != StatusAnswered && status != StatusSkipped {
			return false
		}
	}
	return true
}

func (s *SessionState) UpdateBlockedByDependencies() {
	if s == nil {
		return
	}
	walkBranches(s.Tree, func(node *Branch) {
		status := normalizeStatus(node.Status)
		if status == StatusAnswered || status == StatusSkipped {
			return
		}
		if len(node.DependsOn) == 0 {
			node.Status = StatusPending
			return
		}
		if s.DependenciesSatisfied(node.DependsOn) {
			node.Status = StatusPending
			return
		}
		node.Status = StatusBlocked
	})
}

func normalizeStatus(status string) string {
	s := strings.ToLower(strings.TrimSpace(status))
	switch s {
	case StatusAnswered, StatusPending, StatusSkipped, StatusBlocked:
		return s
	case StatusResolved:
		return StatusAnswered
	default:
		return StatusPending
	}
}

func walkBranches(nodes []Branch, fn func(*Branch)) {
	for i := range nodes {
		fn(&nodes[i])
		if len(nodes[i].Children) > 0 {
			walkBranches(nodes[i].Children, fn)
		}
	}
}

func findBranchByID(node *Branch, id string) *Branch {
	if node == nil {
		return nil
	}
	if node.ID == id {
		return node
	}
	for i := range node.Children {
		if found := findBranchByID(&node.Children[i], id); found != nil {
			return found
		}
	}
	return nil
}
