package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractFrontMatter(t *testing.T) {
	content := `---
code: TF-TEST
category: cli
---
body`
	fm, ok := extractFrontMatter(content)
	if !ok {
		t.Fatal("expected front matter to be extracted")
	}
	if !strings.Contains(fm, "TF-TEST") {
		t.Fatalf("unexpected front matter: %q", fm)
	}

	if _, ok := extractFrontMatter("no frontmatter"); ok {
		t.Fatal("expected no front matter for plain content")
	}
}

func TestConstName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"TF-CL01", "CodeTFCL01"},
		{"tf-x9", "CodeTFX9"},
		{"---", "CodeUnknown"},
	}
	for _, tc := range cases {
		if got := constName(tc.in); got != tc.want {
			t.Fatalf("constName(%q)=%q want=%q", tc.in, got, tc.want)
		}
	}
}

func TestLoadEntriesSortsAndDefaults(t *testing.T) {
	tmp := t.TempDir()
	mustWriteGenFile(t, filepath.Join(tmp, "b.md"), `---
error_code: TF-B
category: runtime
message_template: "B happened"
---
`)
	mustWriteGenFile(t, filepath.Join(tmp, "a.md"), `---
code: TF-A
category: cli
exit_code: 0
message: "A happened"
---
`)
	mustWriteGenFile(t, filepath.Join(tmp, "skip.md"), `---
code: ""
category: ""
---
`)

	entries, err := loadEntries(tmp)
	if err != nil {
		t.Fatalf("loadEntries error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries len=%d want=2", len(entries))
	}
	if entries[0].Code != "TF-A" || entries[1].Code != "TF-B" {
		t.Fatalf("expected sorted entries by code, got: %+v", entries)
	}
	if entries[0].ExitCode != 1 {
		t.Fatalf("expected exit code default to 1, got %d", entries[0].ExitCode)
	}
	if entries[1].Template != "B happened" {
		t.Fatalf("expected message_template fallback, got %q", entries[1].Template)
	}
}

func TestLoadEntriesParseError(t *testing.T) {
	tmp := t.TempDir()
	mustWriteGenFile(t, filepath.Join(tmp, "bad.md"), `---
code: TF-X
category: cli
exit_code: [broken
---
`)
	_, err := loadEntries(tmp)
	if err == nil || !strings.Contains(err.Error(), "parse frontmatter") {
		t.Fatalf("expected parse frontmatter error, got: %v", err)
	}
}

func TestWriteCatalog(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "internal", "errors", "catalog_gen.go")
	entries := []entry{
		{Code: "TF-CL01", ConstName: "CodeTFCL01", Category: "cli", ExitCode: 2, Template: "oops"},
	}

	if err := writeCatalog(out, entries); err != nil {
		t.Fatalf("writeCatalog failed: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "CodeTFCL01") || !strings.Contains(text, `Template: "oops"`) {
		t.Fatalf("unexpected catalog output: %s", text)
	}
}

func mustWriteGenFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
