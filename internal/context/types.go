package context

// TokenBudget is the default token allocation for context assembly.
type TokenBudget struct {
	System       int `yaml:"system"`
	Conversation int `yaml:"conversation"`
	Summaries    int `yaml:"summaries"`
	FullDocument int `yaml:"full_document"`
	UserInput    int `yaml:"user_input"`
}

// DefaultBudget returns the standard token budget.
func DefaultBudget() TokenBudget {
	return TokenBudget{
		System:       500,
		Conversation: 2000,
		Summaries:    1000,
		FullDocument: 2000,
		UserInput:    500,
	}
}

// AssembledContext is the final packed prompt context.
type AssembledContext struct {
	SystemPrompt string
	Context      string
	UserInput    string
	TotalTokens  int
	IncludedDocs []string
}
