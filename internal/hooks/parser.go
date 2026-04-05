package hooks

import "github.com/taka-sho/teraflow/internal/config"

type Parser struct{}

func NewParser() *Parser { return &Parser{} }

// Parse converts config hooks section to runtime HookConfig.
func (p *Parser) Parse(cfg *config.TeraflowConfig) HookConfig {
	out := HookConfig{}
	if cfg == nil || len(cfg.Hooks) == 0 {
		return out
	}

	for eventName, actions := range cfg.Hooks {
		if eventName == "" {
			continue
		}

		event := HookEvent(eventName)
		converted := make([]HookAction, 0, len(actions))
		for _, action := range actions {
			h := HookAction{
				Action: action.Action,
				Skill:  action.Skill,
			}
			if action.Conditions != nil {
				h.Conditions = &HookConditions{
					NotAuthor:  copyStrings(action.Conditions.NotAuthor),
					Categories: copyStrings(action.Conditions.Categories),
					Labels:     copyStrings(action.Conditions.Labels),
					Paths:      copyStrings(action.Conditions.Paths),
				}
			}
			converted = append(converted, h)
		}
		out[event] = converted
	}

	return out
}

// MatchHooks returns actions for ctx.Event that satisfy all conditions.
func (p *Parser) MatchHooks(hookCfg HookConfig, ctx HookContext) []HookAction {
	actions := hookCfg[ctx.Event]
	if len(actions) == 0 {
		return nil
	}

	matched := make([]HookAction, 0, len(actions))
	for _, action := range actions {
		if !matchesConditions(action.Conditions, ctx) {
			continue
		}
		matched = append(matched, action)
	}

	return matched
}

func matchesConditions(cond *HookConditions, ctx HookContext) bool {
	if cond == nil {
		return true
	}
	if contains(cond.NotAuthor, ctx.Author) {
		return false
	}
	if len(cond.Categories) > 0 && !contains(cond.Categories, ctx.Category) {
		return false
	}
	if len(cond.Labels) > 0 && !hasAnyIntersection(cond.Labels, ctx.Labels) {
		return false
	}
	if len(cond.Paths) > 0 && !hasAnyIntersection(cond.Paths, ctx.Paths) {
		return false
	}
	return true
}

func hasAnyIntersection(want, got []string) bool {
	for _, x := range want {
		if contains(got, x) {
			return true
		}
	}
	return false
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func copyStrings(src []string) []string {
	if len(src) == 0 {
		return nil
	}
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}
