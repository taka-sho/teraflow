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
	indexpkg "github.com/taka-sho/teraflow/internal/index"
	"github.com/taka-sho/teraflow/internal/pipeline"
	"github.com/taka-sho/teraflow/internal/validate"
)

func TestImplementCommandRequiresDesign(t *testing.T) {
	root := newRootCmd("test")
	root.SetArgs([]string{"implement"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "--design is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestImplementCommandDryRunWithoutProvider(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "index.yml"), "version: \"1\"\nentries: []\n")
	mustWrite(t, filepath.Join(tmp, "docs", "design", "d.md"), `---
codd:
  node_id: detail:d
  title: d
  modules:
    - internal/a.go
---
# d
`)

	oldLoad := implementLoadConfig
	oldResolve := implementResolveProviderForType
	oldProvider := implementNewProviderFromConfig
	oldEngine := implementNewEngine
	t.Cleanup(func() {
		implementLoadConfig = oldLoad
		implementResolveProviderForType = oldResolve
		implementNewProviderFromConfig = oldProvider
		implementNewEngine = oldEngine
	})

	implementLoadConfig = func(string) (*cfgpkg.TeraflowConfig, error) { return &cfgpkg.TeraflowConfig{}, nil }
	implementResolveProviderForType = func(*cfgpkg.TeraflowConfig, string) (string, string) { return "anthropic", "" }
	implementNewProviderFromConfig = func(agent.ProviderConfig) (agent.Provider, error) {
		return nil, errors.New("no key")
	}
	implementNewEngine = func(_ agent.Provider, _ *validate.Validator, projectRoot string, _ *indexpkg.Index) *pipeline.ImplementEngine {
		return pipeline.NewImplementEngine(nil, nil, nil, projectRoot, nil)
	}

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "implement", "--design", "docs/design/d.md", "--dry-run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("implement dry-run failed: %v", err)
	}
	if !strings.Contains(out.String(), "Warning: provider init skipped for dry-run") {
		t.Fatalf("expected warning, got: %s", out.String())
	}
}

func TestImplementCommandFailsOnModuleFailure(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "index.yml"), "version: \"1\"\nentries: []\n")
	mustWrite(t, filepath.Join(tmp, "docs", "design", "d.md"), `---
codd:
  node_id: detail:d
  title: d
  modules:
    - internal/fail.go
---
# d
`)

	oldLoad := implementLoadConfig
	oldResolve := implementResolveProviderForType
	oldProvider := implementNewProviderFromConfig
	t.Cleanup(func() {
		implementLoadConfig = oldLoad
		implementResolveProviderForType = oldResolve
		implementNewProviderFromConfig = oldProvider
	})
	implementLoadConfig = func(string) (*cfgpkg.TeraflowConfig, error) { return &cfgpkg.TeraflowConfig{}, nil }
	implementResolveProviderForType = func(*cfgpkg.TeraflowConfig, string) (string, string) { return "custom", "" }
	implementNewProviderFromConfig = func(agent.ProviderConfig) (agent.Provider, error) {
		return &failingCmdProvider{}, nil
	}

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "implement", "--design", "docs/design/d.md"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "failed module") {
		t.Fatalf("expected module failure, got: %v", err)
	}
}

type failingCmdProvider struct{}

func (f *failingCmdProvider) Complete(_ context.Context, _ string, _ string, _ int) (string, int, error) {
	return "", 0, errors.New("boom")
}

func (f *failingCmdProvider) Name() string { return "fake" }
