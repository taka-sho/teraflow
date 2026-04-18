package discovery

import (
	"fmt"
	"time"
)

// DiffEntry describes a single difference between two templates.
type DiffEntry struct {
	Action  string        // "add" | "remove" | "modify"
	FieldID string
	Field   *TemplateItem // nil if removed
}

// DiffFromDefault returns differences between current and the default template.
func DiffFromDefault(current RequirementTemplate) []DiffEntry {
	def := DefaultRequirementTemplate()

	defByID := make(map[string]TemplateItem, len(def.Items))
	for _, item := range def.Items {
		defByID[item.ID] = item
	}

	curByID := make(map[string]TemplateItem, len(current.Items))
	for _, item := range current.Items {
		curByID[item.ID] = item
	}

	var diffs []DiffEntry

	for _, cur := range current.Items {
		d, exists := defByID[cur.ID]
		if !exists {
			cp := cur
			diffs = append(diffs, DiffEntry{Action: "add", FieldID: cur.ID, Field: &cp})
			continue
		}
		if cur.Name != d.Name || cur.Category != d.Category {
			cp := cur
			diffs = append(diffs, DiffEntry{Action: "modify", FieldID: cur.ID, Field: &cp})
		}
	}

	for _, d := range def.Items {
		if _, exists := curByID[d.ID]; !exists {
			diffs = append(diffs, DiffEntry{Action: "remove", FieldID: d.ID, Field: nil})
		}
	}

	return diffs
}

// ValidateTemplate checks the template for structural correctness.
func ValidateTemplate(t RequirementTemplate) error {
	if len(t.Items) == 0 {
		return fmt.Errorf("template has no items")
	}

	seen := make(map[string]struct{}, len(t.Items))
	for i, item := range t.Items {
		if item.ID == "" {
			return fmt.Errorf("item at index %d: id is required", i)
		}
		if item.Name == "" {
			return fmt.Errorf("item %q: name is required", item.ID)
		}
		if item.Category == "" {
			return fmt.Errorf("item %q: category is required", item.ID)
		}
		if item.Category != TemplateCategoryRequired &&
			item.Category != TemplateCategoryRecommended &&
			item.Category != TemplateCategoryOptional {
			return fmt.Errorf("item %q: invalid category %q", item.ID, item.Category)
		}
		if _, dup := seen[item.ID]; dup {
			return fmt.Errorf("item %q: duplicate id", item.ID)
		}
		seen[item.ID] = struct{}{}
	}
	return nil
}

// AddField appends a new field to the template. Returns an error if the ID already exists.
func AddField(t *RequirementTemplate, field TemplateItem) error {
	for _, item := range t.Items {
		if item.ID == field.ID {
			return fmt.Errorf("field %q already exists", field.ID)
		}
	}
	field.Origin = "user_added"
	field.AddedAt = time.Now().UTC().Format(time.RFC3339)
	t.Items = append(t.Items, field)
	return nil
}

// RemoveField removes the field with fieldID from the template and returns it.
func RemoveField(t *RequirementTemplate, fieldID string) (TemplateItem, error) {
	for i, item := range t.Items {
		if item.ID == fieldID {
			t.Items = append(t.Items[:i], t.Items[i+1:]...)
			return item, nil
		}
	}
	return TemplateItem{}, fmt.Errorf("field %q not found", fieldID)
}
