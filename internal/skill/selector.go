package skill

// SelectContext is the selection input for matching an active skill.
type SelectContext struct {
	Labels    []string
	Category  string
	AgentType string
}

// DefaultSelector selects skills by AgentType, then Label, then Category.
type DefaultSelector struct{}

// Select returns the first matched skill by priority or nil when unmatched.
func (s *DefaultSelector) Select(skills []*Skill, ctx SelectContext) *Skill {
	if len(skills) == 0 {
		return nil
	}

	if ctx.AgentType != "" {
		for _, skill := range skills {
			if contains(skill.Trigger.AgentTypes, ctx.AgentType) {
				return skill
			}
		}
	}

	if len(ctx.Labels) > 0 {
		for _, skill := range skills {
			if hasAny(skill.Trigger.Labels, ctx.Labels) {
				return skill
			}
		}
	}

	if ctx.Category != "" {
		for _, skill := range skills {
			if contains(skill.Trigger.Categories, ctx.Category) {
				return skill
			}
		}
	}

	return nil
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func hasAny(items []string, targets []string) bool {
	for _, target := range targets {
		if contains(items, target) {
			return true
		}
	}
	return false
}
