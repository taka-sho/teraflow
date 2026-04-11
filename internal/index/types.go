package index

import "time"

// Entry represents one CoDD node extracted from docs frontmatter.
type Entry struct {
	NodeID           string    `yaml:"node_id"`
	Title            string    `yaml:"title"`
	Path             string    `yaml:"path"`
	DependsOn        []string  `yaml:"depends_on,omitempty"`
	Phase            string    `yaml:"phase,omitempty"`
	Wave             int       `yaml:"wave,omitempty"`
	Modules          []string  `yaml:"modules,omitempty"`
	ReviewRequired   string    `yaml:"review_required,omitempty"`
	Verifies         []string  `yaml:"verifies,omitempty"`
	VerifiedBy       []string  `yaml:"verified_by,omitempty"`
	ChangeImpact     string    `yaml:"change_impact,omitempty"`
	Tags             []string  `yaml:"tags,omitempty"`
	Status           string    `yaml:"status,omitempty"`
	UpdatedAt        time.Time `yaml:"updated_at"`
	ContentHash      string    `yaml:"content_hash"`
	SummaryAvailable bool      `yaml:"summary_available"`
}

// Index is the persisted index.yml structure.
type Index struct {
	Version     string    `yaml:"version"`
	GeneratedAt time.Time `yaml:"generated_at"`
	Entries     []Entry   `yaml:"entries"`
}
