package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/discovery"
	"gopkg.in/yaml.v3"
)

func TestDocIndexGeneratesFile(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, "docs", "requirements.md"), "# Requirements\n\n## Scope\nbody\n")

	var out bytes.Buffer
	root := newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "doc", "index"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doc index failed: %v", err)
	}

	if !strings.Contains(out.String(), "Doc index generated:") {
		t.Fatalf("unexpected output: %s", out.String())
	}
	data, err := osReadFile(filepath.Join(tmp, ".teraflow", "doc-index.yaml"))
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	var idx discovery.DocIndex
	if err := yamlUnmarshal(data, &idx); err != nil {
		t.Fatalf("parse yaml: %v", err)
	}
	if len(idx.Docs) != 1 {
		t.Fatalf("docs=%d, want 1", len(idx.Docs))
	}
	if idx.Docs[0].Path != "requirements.md" {
		t.Fatalf("path=%q, want requirements.md", idx.Docs[0].Path)
	}
}

func TestDocIndexRequiresForceWhenOutputExists(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, "docs", "requirements.md"), "# Requirements\n")
	outputPath := filepath.Join(tmp, ".teraflow", "doc-index.yaml")
	mustWrite(t, outputPath, "version: \"old\"\n")

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "doc", "index"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when output already exists")
	}
	if !strings.Contains(err.Error(), "use --force") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDocIndexForceOverwritesExistingFile(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, "docs", "requirements.md"), "# Requirements\n")
	outputPath := filepath.Join(tmp, ".teraflow", "doc-index.yaml")
	mustWrite(t, outputPath, "version: \"old\"\n")

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "doc", "index", "--force"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doc index --force failed: %v", err)
	}
	data, err := osReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if strings.Contains(string(data), "version: \"old\"") {
		t.Fatalf("output file was not overwritten: %s", string(data))
	}
}

func TestDocIndexJSONOutput(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, "docs", "requirements.md"), "# Requirements\n")

	var out bytes.Buffer
	root := newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "--format", "json", "doc", "index"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doc index json failed: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("parse json output: %v", err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("status=%v, want ok", payload["status"])
	}
}

var osReadFile = func(path string) ([]byte, error) { return os.ReadFile(path) }
var yamlUnmarshal = func(data []byte, out any) error { return yaml.Unmarshal(data, out) }

func TestDocIndexGenerationFailure(t *testing.T) {
	oldGenerate := docIndexGenerateFn
	t.Cleanup(func() { docIndexGenerateFn = oldGenerate })
	docIndexGenerateFn = func(context.Context, string, discovery.LLMGenerator) (*discovery.DocIndex, error) {
		return nil, errors.New("boom")
	}

	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "doc", "index"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected generation error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDocIndexOutputStatError(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, "docs", "requirements.md"), "# Requirements\n")

	blocker := filepath.Join(tmp, "blocker")
	mustWrite(t, blocker, "x")
	outputPath := filepath.Join(blocker, "doc-index.yaml")

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "doc", "index", "--output", outputPath})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected stat error")
	}
	if !strings.Contains(err.Error(), "check output file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDocIndexWriteFailure(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, "docs", "requirements.md"), "# Requirements\n")

	outputDir := filepath.Join(tmp, ".teraflow")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("mkdir output dir: %v", err)
	}

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "doc", "index", "--output", outputDir, "--force"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected write error")
	}
	if !strings.Contains(err.Error(), "write doc index") {
		t.Fatalf("unexpected error: %v", err)
	}
}
