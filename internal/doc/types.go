package doc

// Reply is a reply to a discussion comment.
type Reply struct {
	Author    string
	Body      string
	CreatedAt string
}

// DocSection is a section in a CoDD document.
type DocSection struct {
	Heading string `json:"heading" yaml:"heading"`
	Body    string `json:"body" yaml:"body"`
}

// CoDDDocument represents generated CoDD document metadata and body.
type CoDDDocument struct {
	NodeID    string       `yaml:"node_id"`
	Title     string       `yaml:"title"`
	DependsOn []string     `yaml:"depends_on,omitempty"`
	Status    string       `yaml:"status"`
	Source    string       `yaml:"source"`
	CreatedAt string       `yaml:"created_at"`
	UpdatedAt string       `yaml:"updated_at"`
	Summary   string       `yaml:"summary,omitempty"`
	Sections  []DocSection `yaml:"sections,omitempty"`
	Body      string       `yaml:"-"`
}

type GenerateRequest struct {
	DiscussionID string
	ConfigPath   string
	ProjectRoot  string
	DryRun       bool
	OutputDir    string
}

type GenerateResult struct {
	Document    *CoDDDocument
	FilePath    string
	IndexUpdate bool
	PRBranch    string
}

type DiscussionData struct {
	Number    int
	Title     string
	Body      string
	Comments  []Comment
	Labels    []string
	CreatedAt string
}

type Comment struct {
	Author    string
	Body      string
	CreatedAt string
	IsAnswer  bool
	Replies   []Reply
}
