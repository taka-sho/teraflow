package skill

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	embeddedskills "github.com/taka-sho/teraflow/skills"
	"gopkg.in/yaml.v3"
)

// FileLoader loads skill definitions from YAML files.
type FileLoader struct {
	skillDir string
}

// NewFileLoader creates a loader rooted at the provided skill directory.
func NewFileLoader(skillDir string) *FileLoader {
	return &FileLoader{skillDir: skillDir}
}

// LoadAll loads all skills from skills/*.yml under the configured root.
func (l *FileLoader) LoadAll() ([]*Skill, error) {
	pattern := filepath.Join(l.skillDir, "*.yml")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob skill files: %w", err)
	}
	if len(paths) == 0 {
		return l.loadEmbeddedAll()
	}

	skills := make([]*Skill, 0, len(paths))
	for _, path := range paths {
		skill, err := l.Load(path)
		if err != nil {
			return nil, err
		}
		skills = append(skills, skill)
	}
	return skills, nil
}

// Load loads a single skill file.
func (l *FileLoader) Load(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read skill file %s: %w", path, err)
	}

	return l.unmarshalSkill(data, path)
}

func (l *FileLoader) loadEmbeddedAll() ([]*Skill, error) {
	entries, err := fs.ReadDir(embeddedskills.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("read embedded skills: %w", err)
	}

	skills := make([]*Skill, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yml" {
			continue
		}
		data, err := embeddedskills.FS.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read embedded skill file %s: %w", entry.Name(), err)
		}
		skill, err := l.unmarshalSkill(data, "embedded:"+entry.Name())
		if err != nil {
			return nil, err
		}
		skills = append(skills, skill)
	}

	if len(skills) == 0 {
		return nil, fmt.Errorf("no skill files found in %s or embedded assets", l.skillDir)
	}
	return skills, nil
}

func (l *FileLoader) unmarshalSkill(data []byte, path string) (*Skill, error) {
	var skill Skill
	if err := yaml.Unmarshal(data, &skill); err != nil {
		return nil, fmt.Errorf("parse skill file %s: %w", path, err)
	}

	if err := validateSkill(&skill, path); err != nil {
		return nil, err
	}

	return &skill, nil
}

// LoadByName loads all skills and returns the first with matching name.
func (l *FileLoader) LoadByName(name string) (*Skill, error) {
	skills, err := l.LoadAll()
	if err != nil {
		return nil, err
	}

	for _, skill := range skills {
		if skill.Name == name {
			return skill, nil
		}
	}

	return nil, fmt.Errorf("skill not found: %s", name)
}

func validateSkill(skill *Skill, path string) error {
	if skill.Name == "" {
		return fmt.Errorf("invalid skill file %s: name is required", path)
	}
	if skill.Version == "" {
		return fmt.Errorf("invalid skill file %s: version is required", path)
	}
	return nil
}
