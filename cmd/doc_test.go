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

func TestDocGenerateJSONOutputNoPR(t *testing.T) {
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
		return &fakeDocGenerator{res: &docpkg.GenerateResult{
			FilePath:    filepath.Join(projectRoot, "docs", "requirements", "req-auth.md"),
			IndexUpdate: true,
		}}
	}

	var out bytes.Buffer
	root := newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "--format", "json", "doc", "generate", "--discussion", "12"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doc generate json failed: %v", err)
	}

	var got docGenerateOutput
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("parse json output: %v\nraw=%s", err, out.String())
	}
	if got.FilePath != "docs/requirements/req-auth.md" {
		t.Fatalf("file_path=%q, want docs/requirements/req-auth.md", got.FilePath)
	}
	if !got.IndexUpdated {
		t.Fatal("index_updated=false, want true")
	}
	if got.PRURL != "" || got.PRBranch != "" || got.PRNumber != "" {
		t.Fatalf("unexpected PR fields in no-pr output: %+v", got)
	}
}

func TestDocGenerateJSONOutputCreatePR(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	oldLoad := docLoadConfig
	oldResolve := docResolveProviderForType
	oldNewProvider := docNewProviderFromConfig
	oldNewGenerator := docNewGenerator
	oldCreateDocPRFn := createDocPRFn
	t.Cleanup(func() {
		docLoadConfig = oldLoad
		docResolveProviderForType = oldResolve
		docNewProviderFromConfig = oldNewProvider
		docNewGenerator = oldNewGenerator
		createDocPRFn = oldCreateDocPRFn
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
	fakeGen := &fakeDocGenerator{res: &docpkg.GenerateResult{
		FilePath:    filepath.Join(tmp, "docs", "requirements", "req-auth.md"),
		IndexUpdate: true,
	}}
	docNewGenerator = func(provider agent.Provider, projectRoot string, dryRun bool) docGenerator {
		return fakeGen
	}
	createDocPRFn = func(projectRoot, discussion, generatedRelPath string) (branch, prNumber, prURL string, err error) {
		if generatedRelPath != "docs/requirements/req-auth.md" {
			t.Fatalf("generatedRelPath=%q, want docs/requirements/req-auth.md", generatedRelPath)
		}
		return "doc/discussion-12-12345", "99", "https://github.com/taka-sho/teraflow/pull/99", nil
	}

	var out bytes.Buffer
	root := newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "--format", "json", "doc", "generate", "--discussion", "12", "--create-pr"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doc generate json create-pr failed: %v", err)
	}

	var got docGenerateOutput
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("parse json output: %v\nraw=%s", err, out.String())
	}
	if got.PRURL != "https://github.com/taka-sho/teraflow/pull/99" {
		t.Fatalf("pr_url=%q, want https://github.com/taka-sho/teraflow/pull/99", got.PRURL)
	}
	if got.PRNumber != "99" || got.PRBranch != "doc/discussion-12-12345" {
		t.Fatalf("unexpected PR fields: %+v", got)
	}
	if got.FilePath != "docs/requirements/req-auth.md" || !got.IndexUpdated {
		t.Fatalf("unexpected core fields: %+v", got)
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

func TestDocHelpers(t *testing.T) {
	if matchesCategory("docs/requirements/a.md", "") != true {
		t.Fatal("empty category should match all")
	}
	if matchesCategory("docs/design/a.md", "requirements") {
		t.Fatal("design doc should not match requirements category")
	}
	if got := extractPRNumberFromPRURL("https://github.com/acme/x/pull/123"); got != "123" {
		t.Fatalf("unexpected pr number: %q", got)
	}
	if got := extractPRNumberFromPRURL("invalid"); got != "" {
		t.Fatalf("expected empty pr number, got: %q", got)
	}
}

func TestReadCurrentPhase(t *testing.T) {
	tmp := t.TempDir()
	mustWrite(t, filepath.Join(tmp, ".github", "project-state.yml"), "phases:\n  current: basic_design\n")

	if got := readCurrentPhase(tmp); got != "basic_design" {
		t.Fatalf("readCurrentPhase=%q, want basic_design", got)
	}
}

func TestBuildDocPRBodyIncludesRequiredElements(t *testing.T) {
	body := buildDocPRBody("42", "requirements", "docs/requirements/req-auth.md")

	checks := []string{
		"📍 現在のフェーズ: 要件定義（requirements）",
		"要件内容を確認し、問題なければ Approve→Merge してください。",
		"✅ Mergeすると → 要件が確定し、次フェーズに自動遷移します",
		"❌ Closeすると → この要件定義は破棄されます",
		"- `docs/requirements/req-auth.md`",
		"- `.teraflow/index.yml`",
		"Generated by `teraflow doc generate --discussion 42 --create-pr`.",
	}
	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Fatalf("PR body missing %q:\n%s", want, body)
		}
	}
}

func TestBuildDocPRBodyUnknownPhaseFallback(t *testing.T) {
	body := buildDocPRBody("7", "", "")

	checks := []string{
		"📍 現在のフェーズ: 不明（unknown）",
		"変更内容を確認し、受け入れ可能なら Approve→Merge してください。",
		"✅ Mergeすると → この変更が確定し、次の処理へ進みます",
		"❌ Closeすると → この提案は破棄されます",
		"- `.teraflow/index.yml`",
	}
	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Fatalf("fallback PR body missing %q:\n%s", want, body)
		}
	}
}

func TestRenderDocPreviewAndReadDocStatus(t *testing.T) {
	if _, err := renderDocPreview(nil); err == nil {
		t.Fatal("expected nil document error")
	}

	doc := &docpkg.CoDDDocument{NodeID: "req:a", Title: "A", Status: "review", Body: "# Body"}
	out, err := renderDocPreview(doc)
	if err != nil {
		t.Fatalf("renderDocPreview failed: %v", err)
	}
	if !strings.Contains(out, "codd:") || !strings.Contains(out, "# Body") {
		t.Fatalf("unexpected preview: %s", out)
	}

	tmp := t.TempDir()
	mustWrite(t, filepath.Join(tmp, "docs", "requirements", "a.md"), `---
codd:
  status: confirmed
---
`)
	status, err := readDocStatus(tmp, "docs/requirements/a.md")
	if err != nil || status != "confirmed" {
		t.Fatalf("readDocStatus status=%q err=%v", status, err)
	}

	mustWrite(t, filepath.Join(tmp, "docs", "requirements", "nofm.md"), "no frontmatter")
	status, err = readDocStatus(tmp, "docs/requirements/nofm.md")
	if err != nil || status != "" {
		t.Fatalf("readDocStatus no frontmatter status=%q err=%v", status, err)
	}

	mustWrite(t, filepath.Join(tmp, "docs", "requirements", "bad.md"), "---\n:\n---\n")
	if _, err := readDocStatus(tmp, "docs/requirements/bad.md"); err == nil {
		t.Fatal("expected parse frontmatter error")
	}
}
