package skill

import (
	"fmt"
	"os"
	"path/filepath"

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
