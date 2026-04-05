package index

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Entry represents one document node from index.yml.
type Entry struct {
	NodeID      string `yaml:"node_id"`
	FilePath    string `yaml:"file_path"`
	Title       string `yaml:"title,omitempty"`
	ContentHash string `yaml:"content_hash"`
}

// Index is the top-level structure of index.yml.
type Index struct {
	Version string  `yaml:"version,omitempty"`
	Entries []Entry `yaml:"entries"`
}

// Load reads index data from a YAML file.
func Load(path string) (*Index, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}

	var idx Index
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parse index: %w", err)
	}
	if idx.Entries == nil {
		idx.Entries = []Entry{}
	}

	return &idx, nil
}
