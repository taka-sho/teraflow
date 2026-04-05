package skill

type Skill struct {
	Name        string     `yaml:"name"`
	Version     string     `yaml:"version"`
	Description string     `yaml:"description"`
	Trigger     Trigger    `yaml:"trigger"`
	Prompts     Prompts    `yaml:"prompts"`
	Context     ContextCfg `yaml:"context"`
	Output      OutputCfg  `yaml:"output"`
	Options     Options    `yaml:"options"`
}

type Trigger struct {
	Labels     []string `yaml:"labels"`
	Categories []string `yaml:"categories"`
	AgentTypes []string `yaml:"agent_types"`
}

type Prompts struct {
	System  string `yaml:"system"`
	Confirm string `yaml:"confirm"`
}

type ContextCfg struct {
	Include          []string `yaml:"include"`
	Exclude          []string `yaml:"exclude"`
	MaxContextTokens int      `yaml:"max_context_tokens"`
}

type OutputCfg struct {
	Dialogue OutputFormat `yaml:"dialogue"`
	Confirm  OutputFormat `yaml:"confirm"`
}

type OutputFormat struct {
	Header string `yaml:"header"`
	Footer string `yaml:"footer"`
}

type Options struct {
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
}
