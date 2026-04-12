package wave

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/graphbridge"
	"github.com/taka-sho/teraflow/internal/index"
	"github.com/taka-sho/teraflow/internal/validate"
)

func TestExecutePhaseNoWaves(t *testing.T) {
	engine := NewEngine(nil, nil, nil, &index.Index{}, t.TempDir())
	_, err := engine.ExecutePhase(context.Background(), "basic-design", nil, ExecuteOptions{})
	if err == nil || !strings.Contains(err.Error(), "no waves defined") {
		t.Fatalf("expected no waves error, got %v", err)
	}
}

func TestExecutePhaseSortsAndExecutes(t *testing.T) {
	root := t.TempDir()
	mustWriteWaveFile(t, filepath.Join(root, "internal", "actions", "templates", "wave-generic.tmpl"), "phase={{.Phase}} wave={{.Wave.Number}}")
	engine := NewEngine(&fakeWaveProvider{out: "generated"}, nil, nil, &index.Index{}, root)
	results, err := engine.ExecutePhase(context.Background(), "detailed-design", []WaveDefinition{
		{Number: 2, Name: "Second", ArtifactType: "adr", Template: "wave-generic.tmpl", DependsOn: []int{1}},
		{Number: 1, Name: "First", ArtifactType: "acceptance-criteria", Template: "wave-generic.tmpl"},
	}, ExecuteOptions{})
	if err != nil {
		t.Fatalf("ExecutePhase failed: %v", err)
	}
	if len(results) != 2 || results[0].WaveNumber != 1 || results[1].WaveNumber != 2 {
		t.Fatalf("unexpected results order: %+v", results)
	}
	if _, err := os.Stat(results[0].Artifacts[0].Path); err != nil {
		t.Fatalf("expected artifact file to exist: %v", err)
	}
}

func TestExecuteWaveRequiresProviderForNonDryRun(t *testing.T) {
	root := t.TempDir()
	mustWriteWaveFile(t, filepath.Join(root, "internal", "actions", "templates", "wave-generic.tmpl"), "ok")
	engine := NewEngine(nil, nil, nil, &index.Index{}, root)
	_, err := engine.ExecuteWave(context.Background(), "basic-design", WaveDefinition{
		Number:       1,
		Name:         "A",
		ArtifactType: "artifact",
		Template:     "wave-generic.tmpl",
	}, ExecuteOptions{})
	if err == nil || !strings.Contains(err.Error(), "provider is required") {
		t.Fatalf("expected provider-required error, got %v", err)
	}
}

func TestExecuteWavePendingReviewWhenValidationFails(t *testing.T) {
	root := t.TempDir()
	mustWriteWaveFile(t, filepath.Join(root, "internal", "actions", "templates", "wave-generic.tmpl"), "ok")
	idx := &index.Index{Entries: []index.Entry{
		{NodeID: "basic-design:existing", Title: "existing", Path: "docs/existing.md"},
	}}
	v := validate.NewValidator(idx, nil, root)
	engine := NewEngine(&fakeWaveProvider{out: "body"}, v, nil, idx, root)

	res, err := engine.ExecuteWave(context.Background(), "basic-design", WaveDefinition{
		Number:       1,
		Name:         "A",
		ArtifactType: "different-node",
		Template:     "wave-generic.tmpl",
	}, ExecuteOptions{TemplateOverride: "wave-generic.tmpl"})
	if err != nil {
		t.Fatalf("ExecuteWave failed: %v", err)
	}
	if res.Status != "pending_review" {
		t.Fatalf("status=%s, want pending_review", res.Status)
	}
}

func TestBuildContextReadsInputsAndAddsWarnings(t *testing.T) {
	root := t.TempDir()
	docPath := filepath.Join(root, "docs", "req.md")
	mustWriteWaveFile(t, docPath, strings.Repeat("x", 200))

	idx := &index.Index{Entries: []index.Entry{
		{NodeID: "req:a", Title: "A", Path: "docs/req.md"},
		{NodeID: "design:b", Title: "B", Path: "docs/design-b.md", DependsOn: []string{"req:a"}},
	}}
	engine := NewEngine(nil, nil, nil, idx, root)
	ctxData, warnings, err := engine.buildContext(context.Background(), "basic-design", WaveDefinition{
		Number: 1, Name: "Wave", Inputs: []string{"req:a", "missing:node"},
	})
	if err != nil {
		t.Fatalf("buildContext failed: %v", err)
	}
	if len(ctxData.Inputs) != 1 || ctxData.Inputs[0].NodeID != "req:a" {
		t.Fatalf("unexpected inputs: %+v", ctxData.Inputs)
	}
	if ctxData.ImpactSummary == "" {
		t.Fatal("expected impact summary to be populated")
	}
	if ctxData.TokenEstimate <= 0 {
		t.Fatalf("expected positive token estimate, got %d", ctxData.TokenEstimate)
	}
	if !containsText(warnings, "input node not found") || !containsText(warnings, "graphrag not available") {
		t.Fatalf("expected warnings for missing input and graphrag, got: %v", warnings)
	}
}

func TestBuildContextGraphRAGAvailableButQueryFails(t *testing.T) {
	root := t.TempDir()
	mustWriteWaveFile(t, filepath.Join(root, "graphrag", "pyproject.toml"), "[project]\nname='x'\n")
	idx := &index.Index{Entries: []index.Entry{{NodeID: "req:a", Title: "A", Path: "docs/a.md"}}}
	mustWriteWaveFile(t, filepath.Join(root, "docs", "a.md"), "hello")

	engine := NewEngine(nil, nil, graphbridge.New(root), idx, root)
	_, warnings, err := engine.buildContext(context.Background(), "basic-design", WaveDefinition{
		Number: 1, Name: "Wave", Inputs: []string{"req:a"},
	})
	if err != nil {
		t.Fatalf("buildContext failed: %v", err)
	}
	if !containsText(warnings, "graphrag query failed") {
		t.Fatalf("expected graphrag query warning, got: %v", warnings)
	}
}

func TestPathAndHelperFunctions(t *testing.T) {
	if got := buildArtifactPath("/tmp/p", "unit-test", WaveDefinition{Number: 1, Name: "API", ArtifactType: "api"}); !strings.Contains(got, "/docs/test/") {
		t.Fatalf("unexpected test dir path: %s", got)
	}
	if got := buildArtifactPath("/tmp/p", "requirements", WaveDefinition{Number: 1}); !strings.Contains(got, "/docs/requirements/") {
		t.Fatalf("unexpected requirements path: %s", got)
	}
	if got := nodeIDForWave("Detailed_Design", WaveDefinition{Number: 3, ArtifactType: "API Design"}); got != "detailed-design:api-design" {
		t.Fatalf("unexpected node id: %s", got)
	}
	if got := nodeIDForWave("basic-design", WaveDefinition{Number: 4}); got != "basic-design:wave-4" {
		t.Fatalf("unexpected fallback node id: %s", got)
	}
	if min(2, 3) != 2 || min(4, 1) != 1 {
		t.Fatal("min helper returned incorrect values")
	}
}

func TestContextHelpersAndDefaults(t *testing.T) {
	if got := compactLines("a\nb\nc", 2); got != "a\nb" {
		t.Fatalf("unexpected compactLines output: %q", got)
	}
	if got := compactLines("a\nb", 3); got != "a\nb" {
		t.Fatalf("unexpected compactLines passthrough: %q", got)
	}
	if got := estimateTokens(""); got != 0 {
		t.Fatalf("expected zero token estimate for empty input, got %d", got)
	}
	if got := trimToTokenBudget("abcdef", 0); got != "" {
		t.Fatalf("expected empty string for zero budget, got %q", got)
	}
	if got := trimToTokenBudget("abcdef", 1); got != "a..." {
		t.Fatalf("expected truncated text with ellipsis, got %q", got)
	}
	if defs := DefaultWaveDefinitions("basic_design"); len(defs) == 0 {
		t.Fatal("expected default definitions for basic_design")
	}
	if defs := DefaultWaveDefinitions("detailed_design"); len(defs) == 0 {
		t.Fatal("expected default definitions for detailed_design")
	}
	if defs := DefaultWaveDefinitions("unknown-phase"); defs != nil {
		t.Fatalf("expected nil for unknown phase, got: %+v", defs)
	}
}

func containsText(items []string, needle string) bool {
	for _, item := range items {
		if strings.Contains(item, needle) {
			return true
		}
	}
	return false
}
