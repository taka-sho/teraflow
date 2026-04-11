package index

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Builder scans docs and generates .teraflow/index.yml.
type Builder struct {
	projectRoot string
	indexPath   string
	summaryDir  string
}

// NewBuilder creates an index builder rooted at projectRoot.
func NewBuilder(projectRoot string) *Builder {
	return &Builder{
		projectRoot: projectRoot,
		indexPath:   filepath.Join(projectRoot, ".teraflow", "index.yml"),
		summaryDir:  filepath.Join(projectRoot, ".teraflow", "summaries"),
	}
}

// Build scans docs/**/*.md and returns a generated index.
func (b *Builder) Build() (*Index, error) {
	docsRoot := filepath.Join(b.projectRoot, "docs")
	entries := make([]Entry, 0, 64)

	err := filepath.WalkDir(docsRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		meta, ok, err := parseFrontmatter(data)
		if err != nil {
			return fmt.Errorf("parse frontmatter %s: %w", path, err)
		}
		if !ok {
			return nil
		}

		st, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("stat %s: %w", path, err)
		}

		rel, err := filepath.Rel(b.projectRoot, path)
		if err != nil {
			return fmt.Errorf("relative path %s: %w", path, err)
		}
		rel = filepath.ToSlash(rel)

		h := sha256.Sum256(data)
		summaryPath := filepath.Join(b.summaryDir, sanitizeNodeID(meta.NodeID)+".txt")
		_, summaryErr := os.Stat(summaryPath)

		entries = append(entries, Entry{
			NodeID:           meta.NodeID,
			Title:            meta.Title,
			Path:             rel,
			DependsOn:        meta.DependsOn,
			Phase:            meta.Phase,
			Wave:             meta.Wave,
			Modules:          meta.Modules,
			ReviewRequired:   meta.ReviewRequired,
			Verifies:         meta.Verifies,
			VerifiedBy:       meta.VerifiedBy,
			ChangeImpact:     meta.ChangeImpact,
			Tags:             meta.Tags,
			Status:           meta.Status,
			UpdatedAt:        st.ModTime(),
			ContentHash:      hex.EncodeToString(h[:]),
			SummaryAvailable: summaryErr == nil,
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})

	return &Index{
		Version:     "1",
		GeneratedAt: time.Now().UTC(),
		Entries:     entries,
	}, nil
}

// LoadIndex reads .teraflow/index.yml.
func (b *Builder) LoadIndex() (*Index, error) {
	data, err := os.ReadFile(b.indexPath)
	if err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}

	var idx Index
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parse index: %w", err)
	}
	return &idx, nil
}

// Save writes .teraflow/index.yml.
func (b *Builder) Save(idx *Index) error {
	if idx == nil {
		return fmt.Errorf("index is nil")
	}
	if err := os.MkdirAll(filepath.Dir(b.indexPath), 0o755); err != nil {
		return fmt.Errorf("create index dir: %w", err)
	}

	data, err := yaml.Marshal(idx)
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}
	if err := os.WriteFile(b.indexPath, data, 0o644); err != nil {
		return fmt.Errorf("write index: %w", err)
	}
	return nil
}

type frontmatter struct {
	NodeID         string
	Title          string
	DependsOn      []string
	Phase          string
	Wave           int
	Modules        []string
	ReviewRequired string
	Verifies       []string
	VerifiedBy     []string
	ChangeImpact   string
	Tags           []string
	Status         string
}

func parseFrontmatter(data []byte) (frontmatter, bool, error) {
	body := strings.TrimPrefix(string(data), "\ufeff")
	if !strings.HasPrefix(body, "---") {
		return frontmatter{}, false, nil
	}

	lines := strings.Split(body, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return frontmatter{}, false, nil
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return frontmatter{}, false, nil
	}

	fmText := strings.Join(lines[1:end], "\n")
	type coddRaw struct {
		NodeID         string `yaml:"node_id"`
		Title          string `yaml:"title"`
		DependsOn      any    `yaml:"depends_on"`
		Phase          string `yaml:"phase"`
		Wave           int    `yaml:"wave"`
		Modules        any    `yaml:"modules"`
		ReviewRequired string `yaml:"review_required"`
		Verifies       any    `yaml:"verifies"`
		VerifiedBy     any    `yaml:"verified_by"`
		ChangeImpact   string `yaml:"change_impact"`
		Tags           any    `yaml:"tags"`
		Status         string `yaml:"status"`
	}
	var raw struct {
		Codd    coddRaw `yaml:"codd"`
		coddRaw `yaml:",inline"`
	}
	if err := yaml.Unmarshal([]byte(fmText), &raw); err != nil {
		return frontmatter{}, false, err
	}

	src := raw.Codd
	if strings.TrimSpace(src.NodeID) == "" {
		src = raw.coddRaw
	}
	if strings.TrimSpace(src.NodeID) == "" {
		return frontmatter{}, false, nil
	}

	return frontmatter{
		NodeID:         src.NodeID,
		Title:          src.Title,
		DependsOn:      normalizeDependsOn(src.DependsOn),
		Phase:          src.Phase,
		Wave:           src.Wave,
		Modules:        normalizeStringSlice(src.Modules),
		ReviewRequired: src.ReviewRequired,
		Verifies:       normalizeStringSlice(src.Verifies),
		VerifiedBy:     normalizeStringSlice(src.VerifiedBy),
		ChangeImpact:   src.ChangeImpact,
		Tags:           normalizeStringSlice(src.Tags),
		Status:         src.Status,
	}, true, nil
}

func normalizeDependsOn(v any) []string {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		switch typed := item.(type) {
		case string:
			if typed != "" {
				out = append(out, typed)
			}
		case map[string]any:
			if id, ok := typed["id"].(string); ok && id != "" {
				out = append(out, id)
			}
		}
	}
	return out
}

func normalizeStringSlice(v any) []string {
	switch typed := v.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if typed == "" {
			return nil
		}
		return []string{typed}
	default:
		return nil
	}
}

var sanitizeRE = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeNodeID(nodeID string) string {
	safe := sanitizeRE.ReplaceAllString(nodeID, "_")
	safe = strings.Trim(safe, "_")
	if safe == "" {
		return "node"
	}
	return safe
}
