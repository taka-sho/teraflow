package wave

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/index"
)

type fakeWaveProvider struct {
	out string
}

func (p *fakeWaveProvider) Complete(context.Context, string, string, int) (string, int, error) {
	return p.out, 10, nil
}

func (p *fakeWaveProvider) Name() string { return "fake" }

func TestExecuteWaveDryRunWithoutProvider(t *testing.T) {
	root := t.TempDir()
	mustWriteWaveFile(t, filepath.Join(root, "internal", "actions", "templates", "wave-acceptance-criteria.tmpl"), "phase={{.Phase}} wave={{.Wave.Number}}")
	idx := &index.Index{}

	engine := NewEngine(nil, nil, nil, idx, root)
	res, err := engine.ExecuteWave(context.Background(), "basic-design", WaveDefinition{
		Number:       1,
		Name:         "Acceptance Criteria",
		ArtifactType: "acceptance-criteria",
		Template:     "wave-acceptance-criteria.tmpl",
	}, ExecuteOptions{DryRun: true})
	if err != nil {
		t.Fatalf("ExecuteWave dry-run failed: %v", err)
	}
	if res.Status != "dry_run" {
		t.Fatalf("status=%s, want dry_run", res.Status)
	}
	if len(res.Artifacts) != 1 {
		t.Fatalf("artifacts=%d, want 1", len(res.Artifacts))
	}
}

func TestExecutePhaseDependencyValidation(t *testing.T) {
	root := t.TempDir()
	mustWriteWaveFile(t, filepath.Join(root, "internal", "actions", "templates", "wave-acceptance-criteria.tmpl"), "ok")
	engine := NewEngine(&fakeWaveProvider{out: "# ok"}, nil, nil, &index.Index{}, root)
	_, err := engine.ExecutePhase(context.Background(), "basic-design", []WaveDefinition{
		{Number: 2, Name: "B", ArtifactType: "x", Template: "wave-acceptance-criteria.tmpl", DependsOn: []int{1}},
	}, ExecuteOptions{})
	if err == nil || !strings.Contains(err.Error(), "depends on unfinished") {
		t.Fatalf("expected dependency error, got %v", err)
	}
}

func mustWriteWaveFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
