package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/agent"
	cfgpkg "github.com/taka-sho/teraflow/internal/config"
	docpkg "github.com/taka-sho/teraflow/internal/doc"
	indexpkg "github.com/taka-sho/teraflow/internal/index"
)

type fakeDocGenerator struct {
	res *docpkg.GenerateResult
	err error
}

type fakeProvider struct{}

func (f *fakeProvider) Complete(context.Context, string, string, int) (string, int, error) {
	return "", 0, nil
}

func (f *fakeProvider) Name() string {
	return "fake"
}

func (f *fakeDocGenerator) Generate(_ context.Context, _ docpkg.GenerateRequest) (*docpkg.GenerateResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.res, nil
}

func TestDocGenerateRequiresDiscussionFlag(t *testing.T) {
	root := newRootCmd("test")
	root.SetArgs([]string{"doc", "generate"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected missing required flag error")
	}
	if !strings.Contains(err.Error(), "required flag") || !strings.Contains(err.Error(), "discussion") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDocGenerateConfigError(t *testing.T) {
	oldLoad := docLoadConfig
	t.Cleanup(func() { docLoadConfig = oldLoad })
	docLoadConfig = func(path string) (*cfgpkg.TeraflowConfig, error) {
		return nil, errors.New("boom config")
	}

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", "/tmp/missing.yml", "doc", "generate", "--discussion", "42"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected config error")
	}
	if !strings.Contains(err.Error(), "boom config") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDocGenerateDryRun(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	oldLoad := docLoadConfig
	oldResolve := docResolveProviderForType
	oldNewProvider := docNewProviderFromConfig
	oldNewGenerator := docNewGenerator
	t.Cleanup(func() {
		docLoadConfig = oldLoad
		docResolveProviderForType = oldResolve
		docNewProviderFromConfig = oldNewProvider
		docNewGenerator = oldNewGenerator
	})

	docLoadConfig = func(path string) (*cfgpkg.TeraflowConfig, error) {
		return &cfgpkg.TeraflowConfig{}, nil
	}
	docResolveProviderForType = func(cfg *cfgpkg.TeraflowConfig, agentType string) (string, string) {
		return "anthropic", "claude-haiku-4-5-20251001"
	}
	docNewProviderFromConfig = func(cfg agent.ProviderConfig) (agent.Provider, error) {
		return &fakeProvider{}, nil
	}
	docNewGenerator = func(provider agent.Provider, projectRoot string, dryRun bool) docGenerator {
		return &fakeDocGenerator{res: &docpkg.GenerateResult{Document: &docpkg.CoDDDocument{
			NodeID: "req-auth",
			Title:  "Auth",
			Status: "review",
			Body:   "# Summary\n\nhello",
		}}}
	}

	var out bytes.Buffer
	root := newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "doc", "generate", "--discussion", "12", "--dry-run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doc generate dry-run failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "codd:") || !strings.Contains(got, "node_id: req-auth") || !strings.Contains(got, "# Summary") {
		t.Fatalf("unexpected dry-run output: %s", got)
	}
}

func TestDocListCommand(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	mustWrite(t, filepath.Join(tmp, "docs", "requirements", "req-auth.md"), `---
codd:
  node_id: req-auth
  title: Auth requirement
  status: confirmed
  depends_on: [req-base]
---
# Auth
`)
	mustWrite(t, filepath.Join(tmp, "docs", "design", "design-api.md"), `---
codd:
  node_id: design-api
  title: API design
  status: review
---
# API
`)

	builder := indexpkg.NewBuilder(tmp)
	idx, err := builder.Build()
	if err != nil {
		t.Fatalf("build index: %v", err)
	}
	if err := builder.Save(idx); err != nil {
		t.Fatalf("save index: %v", err)
	}

	var textOut bytes.Buffer
	root := newRootCmd("test")
	root.SetOut(&textOut)
	root.SetErr(&textOut)
	root.SetArgs([]string{"--config", cfgPath, "doc", "list", "--category", "requirements", "--status", "confirmed"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doc list text failed: %v", err)
	}
	if !strings.Contains(textOut.String(), "NODE_ID") || !strings.Contains(textOut.String(), "req-auth") || strings.Contains(textOut.String(), "design-api") {
		t.Fatalf("unexpected list text output: %s", textOut.String())
	}

	var jsonOut bytes.Buffer
	root = newRootCmd("test")
	root.SetOut(&jsonOut)
	root.SetErr(&jsonOut)
	root.SetArgs([]string{"--config", cfgPath, "--format", "json", "doc", "list"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doc list json failed: %v", err)
	}

	var rows []map[string]any
	if err := json.Unmarshal(jsonOut.Bytes(), &rows); err != nil {
		t.Fatalf("parse json output: %v\nraw=%s", err, jsonOut.String())
	}
	if len(rows) != 2 {
		t.Fatalf("json rows=%d, want 2", len(rows))
	}
}
