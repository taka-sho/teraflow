package discovery

import (
	"testing"
)

func TestDiffFromDefault_NoChange(t *testing.T) {
	def := DefaultRequirementTemplate()
	diffs := DiffFromDefault(def)
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for default template, got %d", len(diffs))
	}
}

func TestDiffFromDefault_Add(t *testing.T) {
	tmpl := DefaultRequirementTemplate()
	tmpl.Items = append(tmpl.Items, TemplateItem{
		ID:       "custom_field",
		Name:     "カスタムフィールド",
		Category: TemplateCategoryOptional,
	})

	diffs := DiffFromDefault(tmpl)
	found := false
	for _, d := range diffs {
		if d.Action == "add" && d.FieldID == "custom_field" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'add' diff for custom_field")
	}
}

func TestDiffFromDefault_Remove(t *testing.T) {
	tmpl := DefaultRequirementTemplate()
	// remove first item
	removed := tmpl.Items[0].ID
	tmpl.Items = tmpl.Items[1:]

	diffs := DiffFromDefault(tmpl)
	found := false
	for _, d := range diffs {
		if d.Action == "remove" && d.FieldID == removed {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'remove' diff for %q", removed)
	}
}

func TestDiffFromDefault_Modify(t *testing.T) {
	tmpl := DefaultRequirementTemplate()
	tmpl.Items[0].Name = "変更後の名前"

	diffs := DiffFromDefault(tmpl)
	found := false
	for _, d := range diffs {
		if d.Action == "modify" && d.FieldID == tmpl.Items[0].ID {
			found = true
		}
	}
	if !found {
		t.Error("expected 'modify' diff")
	}
}

func TestValidateTemplate_Valid(t *testing.T) {
	tmpl := DefaultRequirementTemplate()
	if err := ValidateTemplate(tmpl); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateTemplate_EmptyItems(t *testing.T) {
	tmpl := RequirementTemplate{}
	if err := ValidateTemplate(tmpl); err == nil {
		t.Error("expected error for empty items")
	}
}

func TestValidateTemplate_MissingID(t *testing.T) {
	tmpl := RequirementTemplate{
		Items: []TemplateItem{{Name: "foo", Category: TemplateCategoryRequired}},
	}
	if err := ValidateTemplate(tmpl); err == nil {
		t.Error("expected error for missing id")
	}
}

func TestValidateTemplate_MissingName(t *testing.T) {
	tmpl := RequirementTemplate{
		Items: []TemplateItem{{ID: "foo", Category: TemplateCategoryRequired}},
	}
	if err := ValidateTemplate(tmpl); err == nil {
		t.Error("expected error for missing name")
	}
}

func TestValidateTemplate_MissingCategory(t *testing.T) {
	tmpl := RequirementTemplate{
		Items: []TemplateItem{{ID: "foo", Name: "Foo"}},
	}
	if err := ValidateTemplate(tmpl); err == nil {
		t.Error("expected error for missing category")
	}
}

func TestValidateTemplate_InvalidCategory(t *testing.T) {
	tmpl := RequirementTemplate{
		Items: []TemplateItem{{ID: "foo", Name: "Foo", Category: "invalid"}},
	}
	if err := ValidateTemplate(tmpl); err == nil {
		t.Error("expected error for invalid category")
	}
}

func TestValidateTemplate_DuplicateID(t *testing.T) {
	tmpl := RequirementTemplate{
		Items: []TemplateItem{
			{ID: "dup", Name: "A", Category: TemplateCategoryRequired},
			{ID: "dup", Name: "B", Category: TemplateCategoryRequired},
		},
	}
	if err := ValidateTemplate(tmpl); err == nil {
		t.Error("expected error for duplicate id")
	}
}

func TestAddField_Success(t *testing.T) {
	tmpl := DefaultRequirementTemplate()
	before := len(tmpl.Items)
	err := AddField(&tmpl, TemplateItem{
		ID:       "new_field",
		Name:     "新フィールド",
		Category: TemplateCategoryOptional,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tmpl.Items) != before+1 {
		t.Error("item not added")
	}
	added := tmpl.Items[len(tmpl.Items)-1]
	if added.Origin != "user_added" {
		t.Errorf("expected origin=user_added, got %q", added.Origin)
	}
	if added.AddedAt == "" {
		t.Error("expected added_at to be set")
	}
}

func TestAddField_DuplicateID(t *testing.T) {
	tmpl := DefaultRequirementTemplate()
	first := tmpl.Items[0]
	err := AddField(&tmpl, TemplateItem{
		ID:       first.ID,
		Name:     "duplicate",
		Category: TemplateCategoryRequired,
	})
	if err == nil {
		t.Error("expected error for duplicate id")
	}
}

func TestRemoveField_Success(t *testing.T) {
	tmpl := DefaultRequirementTemplate()
	targetID := tmpl.Items[0].ID
	before := len(tmpl.Items)

	removed, err := RemoveField(&tmpl, targetID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removed.ID != targetID {
		t.Errorf("expected removed id=%q, got %q", targetID, removed.ID)
	}
	if len(tmpl.Items) != before-1 {
		t.Error("item not removed")
	}
}

func TestRemoveField_NotFound(t *testing.T) {
	tmpl := DefaultRequirementTemplate()
	_, err := RemoveField(&tmpl, "nonexistent_id")
	if err == nil {
		t.Error("expected error for nonexistent id")
	}
}
