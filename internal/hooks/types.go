package hooks

type HookEvent string

const (
	EventDiscussionCreated HookEvent = "on_discussion_created"
	EventDiscussionComment HookEvent = "on_discussion_comment"
	EventConfirmation      HookEvent = "on_confirmation"
	EventPush              HookEvent = "on_push"
	EventPROpened          HookEvent = "on_pr_opened"
)

type HookAction struct {
	Action     string          `yaml:"action"`
	Skill      string          `yaml:"skill,omitempty"`
	Conditions *HookConditions `yaml:"conditions,omitempty"`
}

type HookConditions struct {
	NotAuthor  []string `yaml:"not_author,omitempty"`
	Categories []string `yaml:"categories,omitempty"`
	Labels     []string `yaml:"labels,omitempty"`
	Paths      []string `yaml:"paths,omitempty"`
}

type HookConfig map[HookEvent][]HookAction

type HookContext struct {
	Event        HookEvent
	Author       string
	Category     string
	Labels       []string
	Paths        []string
	Input        string
	DiscussionID string
}

type HookResult struct {
	Event   HookEvent
	Matched []HookAction
	DryRun  bool
	Results []ActionResult
}

type ActionResult struct {
	Action  string
	Success bool
	Message string
}
