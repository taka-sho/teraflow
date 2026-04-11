package validate

import (
	"testing"

	"github.com/taka-sho/teraflow/internal/index"
)

func TestValidateArtifactSchemaAndReference(t *testing.T) {
	idx := &index.Index{Entries: []index.Entry{
		{
			NodeID:         "req:ok",
			Title:          "Requirement",
			Path:           "docs/req.md",
			ReviewRequired: "review",
			DependsOn:      []string{"missing:node"},
			Modules:        []string{"../escape.go"},
		},
	}}
	v := NewValidator(idx, nil, t.TempDir())
	v.SetLevel(2)
	result := v.ValidateArtifact("req:ok")
	if result.Valid {
		t.Fatal("ValidateArtifact() valid = true, want false")
	}
	if len(result.Errors) == 0 {
		t.Fatal("expected reference errors")
	}
}

func TestValidateVerifiesReciprocity(t *testing.T) {
	idx := &index.Index{Entries: []index.Entry{
		{
			NodeID:   "design:a",
			Title:    "Design A",
			Path:     "docs/a.md",
			Verifies: []string{"req:b"},
		},
		{
			NodeID: "req:b",
			Title:  "Req B",
			Path:   "docs/b.md",
		},
	}}
	v := NewValidator(idx, nil, t.TempDir())
	v.SetLevel(2)
	result := v.ValidatePhaseTransition("", "")
	if len(result.Errors) == 0 {
		t.Fatal("expected verifies/verified_by mismatch error")
	}
}

func TestValidateLevel4WithoutGraphRAGWarns(t *testing.T) {
	idx := &index.Index{Entries: []index.Entry{{NodeID: "req:a", Title: "A", Path: "docs/a.md"}}}
	v := NewValidator(idx, nil, t.TempDir())
	v.SetLevel(4)
	result := v.ValidatePhaseTransition("", "")
	if len(result.Warnings) == 0 {
		t.Fatal("expected warning when GraphRAG unavailable")
	}
}
