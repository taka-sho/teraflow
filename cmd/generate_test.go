package cmd

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/agent"
	cfgpkg "github.com/taka-sho/teraflow/internal/config"
	"github.com/taka-sho/teraflow/internal/graphbridge"
	indexpkg "github.com/taka-sho/teraflow/internal/index"
	"github.com/taka-sho/teraflow/internal/validate"
	"github.com/taka-sho/teraflow/internal/wave"
)

type fakeWaveExecutor struct {
	results []wave.WaveResult
	err     error
}

func (f *fakeWaveExecutor) ExecutePhase(_ context.Context, _ string, _ []wave.WaveDefinition, _ wave.ExecuteOptions) ([]wave.WaveResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.results, nil
}

func TestGeneratePhaseRequired(t *testing.T) {
	root := newRootCmd("test")
	root.SetArgs([]string{"generate"})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "--phase is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateDryRunContinuesWithoutProvider(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "index.yml"), "version: \"1\"\nentries: []\n")

	oldLoad := generateLoadConfig
	oldResolve := generateResolveProviderForType
	oldProvider := generateNewProviderFromConfig
	oldEngine := generateNewWaveEngine
	t.Cleanup(func() {
		generateLoadConfig = oldLoad
		generateResolveProviderForType = oldResolve
		generateNewProviderFromConfig = oldProvider
		generateNewWaveEngine = oldEngine
	})

	generateLoadConfig = func(string) (*cfgpkg.TeraflowConfig, error) { return &cfgpkg.TeraflowConfig{}, nil }
	generateResolveProviderForType = func(*cfgpkg.TeraflowConfig, string) (string, string) { return "anthropic", "" }
	generateNewProviderFromConfig = func(agent.ProviderConfig) (agent.Provider, error) {
		return nil, errors.New("no key")
	}
	generateNewWaveEngine = func(_ agent.Provider, _ *validate.Validator, _ *graphbridge.Bridge, _ *indexpkg.Index, _ string) waveExecutor {
		return &fakeWaveExecutor{results: []wave.WaveResult{{WaveNumber: 1, WaveName: "w1", Status: "dry_run"}}}
	}

	var out bytes.Buffer
	root := newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "generate", "--phase", "basic-design", "--dry-run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("generate dry-run failed: %v", err)
	}
	if !strings.Contains(out.String(), "Warning: provider init skipped for dry-run") {
		t.Fatalf("expected warning, got: %s", out.String())
	}
}
