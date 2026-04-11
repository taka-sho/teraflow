package wave

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplateLoaderResolveAndRender(t *testing.T) {
	root := t.TempDir()
	tplDir := filepath.Join(root, "internal", "actions", "templates")
	mustWriteWaveFile(t, filepath.Join(tplDir, "wave-test.tmpl"), "hello {{.Name}}")

	loader := NewTemplateLoader(root)
	path, err := loader.Resolve("wave-test.tmpl")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !strings.HasSuffix(path, "wave-test.tmpl") {
		t.Fatalf("unexpected path: %s", path)
	}

	out, err := loader.Render("wave-test.tmpl", map[string]string{"Name": "world"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.TrimSpace(out) != "hello world" {
		t.Fatalf("unexpected render output: %q", out)
	}
}

func TestTemplateLoaderMissing(t *testing.T) {
	loader := NewTemplateLoader(t.TempDir())
	if _, err := loader.Resolve("missing"); err == nil {
		t.Fatal("expected missing template error")
	}
}
