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

type captureWaveExecutor struct {
	phase string
	defs  []wave.WaveDefinition
	opts  wave.ExecuteOptions
}

func (c *captureWaveExecutor) ExecutePhase(_ context.Context, phase string, defs []wave.WaveDefinition, opts wave.ExecuteOptions) ([]wave.WaveResult, error) {
	c.phase = phase
	c.defs = defs
	c.opts = opts
	return []wave.WaveResult{{WaveNumber: 1, WaveName: "cap", Status: "ok"}}, nil
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

func TestGenerateValidationAndProviderErrors(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "index.yml"), "version: \"1\"\nentries: []\n")

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "generate", "--phase", "basic-design", "--wave", "-1"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "--wave must be >= 0") {
		t.Fatalf("expected wave validation error, got: %v", err)
	}

	oldLoad := generateLoadConfig
	oldResolve := generateResolveProviderForType
	oldProvider := generateNewProviderFromConfig
	t.Cleanup(func() {
		generateLoadConfig = oldLoad
		generateResolveProviderForType = oldResolve
		generateNewProviderFromConfig = oldProvider
	})
	generateLoadConfig = func(string) (*cfgpkg.TeraflowConfig, error) { return &cfgpkg.TeraflowConfig{}, nil }
	generateResolveProviderForType = func(*cfgpkg.TeraflowConfig, string) (string, string) { return "anthropic", "x" }
	generateNewProviderFromConfig = func(agent.ProviderConfig) (agent.Provider, error) {
		return nil, errors.New("provider failed")
	}

	root = newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "generate", "--phase", "basic-design"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "create provider") {
		t.Fatalf("expected create provider error, got: %v", err)
	}
}

func TestGenerateWaveSelectionAndJSONOutput(t *testing.T) {
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
	generateResolveProviderForType = func(*cfgpkg.TeraflowConfig, string) (string, string) { return "anthropic", "x" }
	generateNewProviderFromConfig = func(agent.ProviderConfig) (agent.Provider, error) { return &fakeProvider{}, nil }

	capture := &captureWaveExecutor{}
	generateNewWaveEngine = func(_ agent.Provider, _ *validate.Validator, _ *graphbridge.Bridge, _ *indexpkg.Index, _ string) waveExecutor {
		return capture
	}

	var out bytes.Buffer
	root := newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "--format", "json", "generate", "--phase", "basic-design", "--wave", "2", "--template", "x.tpl", "--dry-run", "--create-pr"})
	if err := root.Execute(); err != nil {
		t.Fatalf("generate json failed: %v", err)
	}
	if !strings.Contains(out.String(), `"phase":"basic-design"`) || !strings.Contains(out.String(), `"wave_count":1`) {
		t.Fatalf("unexpected json output: %s", out.String())
	}
	if capture.phase != "basic-design" || len(capture.defs) != 1 || capture.defs[0].Number != 2 {
		t.Fatalf("unexpected captured defs: phase=%s defs=%+v", capture.phase, capture.defs)
	}
	if !capture.opts.DryRun || capture.opts.TemplateOverride != "x.tpl" {
		t.Fatalf("unexpected execute options: %+v", capture.opts)
	}

	root = newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "generate", "--phase", "basic-design", "--wave", "999"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "wave 999 not found") {
		t.Fatalf("expected unknown wave error, got: %v", err)
	}
}

func TestProviderTypeForPhase(t *testing.T) {
	if got := providerTypeForPhase("requirements"); got != "requirements" {
		t.Fatalf("requirements mapping mismatch: %q", got)
	}
	if got := providerTypeForPhase("basic_design"); got != "design" {
		t.Fatalf("design mapping mismatch: %q", got)
	}
	if got := providerTypeForPhase("something-else"); got != "implement" {
		t.Fatalf("default mapping mismatch: %q", got)
	}
}
