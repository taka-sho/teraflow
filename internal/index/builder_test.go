package index_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/taka-sho/teraflow/internal/index"
)

func TestBuilderBuildScansCoDDFrontmatter(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "docs", "alpha.md"), mustReadFile(t, filepath.Join("testdata", "docs", "alpha.md")))
	mustWriteFile(t, filepath.Join(root, "docs", "beta.md"), mustReadFile(t, filepath.Join("testdata", "docs", "beta.md")))
	mustWriteFile(t, filepath.Join(root, "docs", "no-frontmatter.md"), mustReadFile(t, filepath.Join("testdata", "docs", "no-frontmatter.md")))
	mustWriteFile(t, filepath.Join(root, ".teraflow", "summaries", "req_alpha.txt"), []byte("summary"))

	builder := index.NewBuilder(root)
	idx, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if got, want := len(idx.Entries), 2; got != want {
		t.Fatalf("Build() entries = %d, want %d", got, want)
	}

	alpha := idx.FindByNodeID("req:alpha")
	if alpha == nil {
		t.Fatal("alpha entry not found")
	}
	if !alpha.SummaryAvailable {
		t.Fatal("alpha SummaryAvailable = false, want true")
	}
	if len(alpha.DependsOn) != 1 || alpha.DependsOn[0] != "req:base" {
		t.Fatalf("alpha DependsOn = %#v, want [req:base]", alpha.DependsOn)
	}
	if len(alpha.Tags) != 2 {
		t.Fatalf("alpha tags count = %d, want 2", len(alpha.Tags))
	}

	beta := idx.FindByNodeID("design:beta")
	if beta == nil {
		t.Fatal("beta entry not found")
	}
	if len(beta.DependsOn) != 1 || beta.DependsOn[0] != "req:alpha" {
		t.Fatalf("beta DependsOn = %#v, want [req:alpha]", beta.DependsOn)
	}
}

func TestBuilderSaveAndLoadIndex(t *testing.T) {
	root := t.TempDir()
	builder := index.NewBuilder(root)

	idx := &index.Index{Version: "1", Entries: []index.Entry{{NodeID: "req:x", Path: "docs/x.md"}}}
	if err := builder.Save(idx); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := builder.LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex() error = %v", err)
	}
	if got, want := len(loaded.Entries), 1; got != want {
		t.Fatalf("LoadIndex() entries = %d, want %d", got, want)
	}
	if loaded.Entries[0].NodeID != "req:x" {
		t.Fatalf("LoadIndex() node_id = %q, want req:x", loaded.Entries[0].NodeID)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
