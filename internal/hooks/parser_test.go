package hooks

import (
	"testing"

	"github.com/taka-sho/teraflow/internal/config"
)

func TestParserParse(t *testing.T) {
	p := NewParser()
	cfg := &config.TeraflowConfig{
		Hooks: map[string][]config.HookActionCfg{
			string(EventDiscussionCreated): {
				{
					Action: "agent.run",
					Skill:  "docs-review",
					Conditions: &config.HookConditionsCfg{
						NotAuthor:  []string{"bot"},
						Categories: []string{"idea"},
						Labels:     []string{"needs-review"},
						Paths:      []string{"docs/"},
					},
				},
			},
		},
	}

	hookCfg := p.Parse(cfg)
	actions := hookCfg[EventDiscussionCreated]
	if len(actions) != 1 {
		t.Fatalf("actions len=%d, want 1", len(actions))
	}
	if actions[0].Action != "agent.run" {
		t.Fatalf("action=%q, want agent.run", actions[0].Action)
	}
	if actions[0].Skill != "docs-review" {
		t.Fatalf("skill=%q, want docs-review", actions[0].Skill)
	}
	if actions[0].Conditions == nil {
		t.Fatalf("conditions=nil, want non-nil")
	}
}

func TestMatchHooks_NotAuthor(t *testing.T) {
	p := NewParser()
	hookCfg := HookConfig{
		EventDiscussionCreated: {
			{
				Action: "agent.run",
				Conditions: &HookConditions{
					NotAuthor: []string{"bot-user"},
				},
			},
		},
	}

	skipped := p.MatchHooks(hookCfg, HookContext{
		Event:  EventDiscussionCreated,
		Author: "bot-user",
	})
	if len(skipped) != 0 {
		t.Fatalf("matched len=%d, want 0 for blocked author", len(skipped))
	}

	matched := p.MatchHooks(hookCfg, HookContext{
		Event:  EventDiscussionCreated,
		Author: "human-user",
	})
	if len(matched) != 1 {
		t.Fatalf("matched len=%d, want 1 for allowed author", len(matched))
	}
}

func TestMatchHooks_Categories(t *testing.T) {
	p := NewParser()
	hookCfg := HookConfig{
		EventDiscussionComment: {
			{
				Action: "agent.run",
				Conditions: &HookConditions{
					Categories: nil,
				},
			},
			{
				Action: "agent.run",
				Conditions: &HookConditions{
					Categories: []string{"bug", "idea"},
				},
			},
		},
	}

	all := p.MatchHooks(hookCfg, HookContext{
		Event:    EventDiscussionComment,
		Category: "anything",
	})
	if len(all) != 1 {
		t.Fatalf("matched len=%d, want 1 for empty categories", len(all))
	}

	hookCfg[EventDiscussionComment][1].Conditions.Categories = []string{"design", "idea"}
	match := p.MatchHooks(hookCfg, HookContext{
		Event:    EventDiscussionComment,
		Category: "design",
	})
	if len(match) != 2 {
		t.Fatalf("matched len=%d, want 2 for matched category", len(match))
	}

	noMatch := p.MatchHooks(hookCfg, HookContext{
		Event:    EventDiscussionComment,
		Category: "incident",
	})
	if len(noMatch) != 1 {
		t.Fatalf("matched len=%d, want 1 for category mismatch", len(noMatch))
	}
}

func TestMatchHooks_Labels(t *testing.T) {
	p := NewParser()
	hookCfg := HookConfig{
		EventPush: {
			{
				Action: "agent.run",
				Conditions: &HookConditions{
					Labels: []string{"backend", "urgent"},
				},
			},
		},
	}

	withIntersection := p.MatchHooks(hookCfg, HookContext{
		Event:  EventPush,
		Labels: []string{"docs", "urgent"},
	})
	if len(withIntersection) != 1 {
		t.Fatalf("matched len=%d, want 1 for label intersection", len(withIntersection))
	}

	noIntersection := p.MatchHooks(hookCfg, HookContext{
		Event:  EventPush,
		Labels: []string{"docs", "frontend"},
	})
	if len(noIntersection) != 0 {
		t.Fatalf("matched len=%d, want 0 for label no intersection", len(noIntersection))
	}
}

func TestMatchHooks_Paths(t *testing.T) {
	p := NewParser()
	hookCfg := HookConfig{
		EventPROpened: {
			{
				Action: "agent.run",
				Conditions: &HookConditions{
					Paths: []string{"internal/hooks/parser.go", "cmd/setup_actions.go"},
				},
			},
		},
	}

	withIntersection := p.MatchHooks(hookCfg, HookContext{
		Event: EventPROpened,
		Paths: []string{"README.md", "cmd/setup_actions.go"},
	})
	if len(withIntersection) != 1 {
		t.Fatalf("matched len=%d, want 1 for path intersection", len(withIntersection))
	}

	noIntersection := p.MatchHooks(hookCfg, HookContext{
		Event: EventPROpened,
		Paths: []string{"README.md", "docs/guide/hooks.md"},
	})
	if len(noIntersection) != 0 {
		t.Fatalf("matched len=%d, want 0 for path no intersection", len(noIntersection))
	}
}

func TestMatchHooks_CompositeConditions(t *testing.T) {
	p := NewParser()
	hookCfg := HookConfig{
		EventDiscussionCreated: {
			{
				Action: "agent.run",
				Conditions: &HookConditions{
					NotAuthor:  []string{"bot-user"},
					Categories: []string{"bug"},
					Labels:     []string{"critical", "urgent"},
					Paths:      []string{"internal/hooks/parser.go"},
				},
			},
		},
	}

	matched := p.MatchHooks(hookCfg, HookContext{
		Event:    EventDiscussionCreated,
		Author:   "human-user",
		Category: "bug",
		Labels:   []string{"urgent"},
		Paths:    []string{"internal/hooks/parser.go"},
	})
	if len(matched) != 1 {
		t.Fatalf("matched len=%d, want 1 for full composite match", len(matched))
	}

	skipped := p.MatchHooks(hookCfg, HookContext{
		Event:    EventDiscussionCreated,
		Author:   "human-user",
		Category: "bug",
		Labels:   []string{"urgent"},
		Paths:    []string{"README.md"},
	})
	if len(skipped) != 0 {
		t.Fatalf("matched len=%d, want 0 when one composite condition fails", len(skipped))
	}
}
