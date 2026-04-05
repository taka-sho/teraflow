package skill_test

import (
	"testing"

	"github.com/taka-sho/teraflow/internal/skill"
)

func TestDefaultSelectorSelectAgentTypeFirst(t *testing.T) {
	selector := &skill.DefaultSelector{}
	skills := []*skill.Skill{
		{Name: "label-match", Trigger: skill.Trigger{Labels: []string{"urgent"}}},
		{Name: "agent-match", Trigger: skill.Trigger{AgentTypes: []string{"review"}}},
	}

	got := selector.Select(skills, skill.SelectContext{AgentType: "review", Labels: []string{"urgent"}, Category: "ops"})
	if got == nil || got.Name != "agent-match" {
		t.Fatalf("Select() = %v, want agent-match", got)
	}
}

func TestDefaultSelectorSelectByLabel(t *testing.T) {
	selector := &skill.DefaultSelector{}
	skills := []*skill.Skill{
		{Name: "category", Trigger: skill.Trigger{Categories: []string{"ops"}}},
		{Name: "label", Trigger: skill.Trigger{Labels: []string{"urgent"}}},
	}

	got := selector.Select(skills, skill.SelectContext{Labels: []string{"urgent"}, Category: "ops"})
	if got == nil || got.Name != "label" {
		t.Fatalf("Select() = %v, want label", got)
	}
}

func TestDefaultSelectorSelectByCategory(t *testing.T) {
	selector := &skill.DefaultSelector{}
	skills := []*skill.Skill{
		{Name: "category", Trigger: skill.Trigger{Categories: []string{"ops"}}},
	}

	got := selector.Select(skills, skill.SelectContext{Category: "ops"})
	if got == nil || got.Name != "category" {
		t.Fatalf("Select() = %v, want category", got)
	}
}

func TestDefaultSelectorSelectNoMatchReturnsNil(t *testing.T) {
	selector := &skill.DefaultSelector{}
	skills := []*skill.Skill{
		{Name: "unmatched", Trigger: skill.Trigger{Labels: []string{"low"}}},
	}

	got := selector.Select(skills, skill.SelectContext{Labels: []string{"urgent"}, Category: "ops", AgentType: "review"})
	if got != nil {
		t.Fatalf("Select() = %v, want nil", got)
	}
}
