package discovery

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// HistoryEntry records a single template change event.
type HistoryEntry struct {
	Timestamp  string            `yaml:"timestamp"`
	Action     string            `yaml:"action"` // "add" | "remove" | "modify" | "init"
	FieldID    string            `yaml:"field_id"`
	SectionID  string            `yaml:"section_id,omitempty"`
	Details    map[string]string `yaml:"details,omitempty"`
	Source     string            `yaml:"source"` // "cli" | "manual" | "recommend_accept"
	Repository string            `yaml:"repository,omitempty"`
}

// TemplateHistory is the persisted history file format.
type TemplateHistory struct {
	Version string         `yaml:"version"`
	Entries []HistoryEntry `yaml:"entries"`
}

// LocalHistoryPath returns the path to the local history file.
func LocalHistoryPath(repoPath string) string {
	return filepath.Join(repoPath, ".teraflow", "discovery", "template-history.yaml")
}

// GlobalHistoryPath returns the path to the global history file.
// Defined as a var to allow test overrides.
var GlobalHistoryPath = func() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(dir, "teraflow", "template-history-global.yaml")
}

// AppendLocalHistory appends entries to both local and global history files.
func AppendLocalHistory(repoPath string, entries []HistoryEntry) error {
	if len(entries) == 0 {
		return nil
	}
	if err := appendToHistory(LocalHistoryPath(repoPath), entries, ""); err != nil {
		return err
	}
	repo := repoPath
	globalEntries := make([]HistoryEntry, len(entries))
	copy(globalEntries, entries)
	for i := range globalEntries {
		globalEntries[i].Repository = repo
	}
	return appendToHistory(GlobalHistoryPath(), globalEntries, "")
}

func appendToHistory(path string, entries []HistoryEntry, _ string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	h, err := readHistory(path)
	if err != nil {
		return err
	}
	h.Entries = append(h.Entries, entries...)
	return writeHistory(path, h)
}

// DetectAndRecordChanges compares prev and curr templates and records diffs.
func DetectAndRecordChanges(repoPath string, prev, curr RequirementTemplate, source string) ([]HistoryEntry, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	prevMap := make(map[string]TemplateItem, len(prev.Items))
	for _, it := range prev.Items {
		prevMap[it.ID] = it
	}
	currMap := make(map[string]TemplateItem, len(curr.Items))
	for _, it := range curr.Items {
		currMap[it.ID] = it
	}

	var entries []HistoryEntry

	// detect removed fields
	for id := range prevMap {
		if _, ok := currMap[id]; !ok {
			entries = append(entries, HistoryEntry{
				Timestamp: now,
				Action:    "remove",
				FieldID:   id,
				Source:    source,
			})
		}
	}

	// detect added and modified fields
	for id, curr := range currMap {
		p, existed := prevMap[id]
		if !existed {
			details := map[string]string{
				"name":     curr.Name,
				"category": curr.Category,
			}
			entries = append(entries, HistoryEntry{
				Timestamp: now,
				Action:    "add",
				FieldID:   id,
				Details:   details,
				Source:    source,
			})
		} else {
			details := detectFieldDiff(p, curr)
			if len(details) > 0 {
				entries = append(entries, HistoryEntry{
					Timestamp: now,
					Action:    "modify",
					FieldID:   id,
					Details:   details,
					Source:    source,
				})
			}
		}
	}

	if len(entries) == 0 {
		return nil, nil
	}
	if err := AppendLocalHistory(repoPath, entries); err != nil {
		return entries, err
	}
	return entries, nil
}

func detectFieldDiff(prev, curr TemplateItem) map[string]string {
	d := map[string]string{}
	if prev.Name != curr.Name {
		d["name_before"] = prev.Name
		d["name_after"] = curr.Name
	}
	if prev.Category != curr.Category {
		d["category_before"] = prev.Category
		d["category_after"] = curr.Category
	}
	if prev.Description != curr.Description {
		d["description_changed"] = "true"
	}
	if prev.PromptHint != curr.PromptHint {
		d["prompt_hint_changed"] = "true"
	}
	return d
}

// ReadLocalHistory reads the local template history.
func ReadLocalHistory(repoPath string) (TemplateHistory, error) {
	return readHistory(LocalHistoryPath(repoPath))
}

// ReadGlobalHistory reads the global template history.
func ReadGlobalHistory() (TemplateHistory, error) {
	return readHistory(GlobalHistoryPath())
}

func readHistory(path string) (TemplateHistory, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return TemplateHistory{Version: "1", Entries: []HistoryEntry{}}, nil
	}
	if err != nil {
		return TemplateHistory{}, err
	}
	var h TemplateHistory
	if err := yaml.Unmarshal(data, &h); err != nil {
		return TemplateHistory{}, err
	}
	if h.Version == "" {
		h.Version = "1"
	}
	if h.Entries == nil {
		h.Entries = []HistoryEntry{}
	}
	return h, nil
}

func writeHistory(path string, h TemplateHistory) error {
	data, err := yaml.Marshal(h)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// SyncToGlobal copies local history entries that are not yet in the global history.
func SyncToGlobal(repoPath string) error {
	local, err := ReadLocalHistory(repoPath)
	if err != nil {
		return err
	}
	global, err := ReadGlobalHistory()
	if err != nil {
		return err
	}

	type key struct{ ts, fieldID, repo string }
	existing := make(map[key]struct{}, len(global.Entries))
	for _, e := range global.Entries {
		existing[key{e.Timestamp, e.FieldID, e.Repository}] = struct{}{}
	}

	var newEntries []HistoryEntry
	for _, e := range local.Entries {
		e.Repository = repoPath
		if _, dup := existing[key{e.Timestamp, e.FieldID, e.Repository}]; !dup {
			newEntries = append(newEntries, e)
		}
	}
	if len(newEntries) == 0 {
		return nil
	}
	global.Entries = append(global.Entries, newEntries...)
	return writeHistory(GlobalHistoryPath(), global)
}
