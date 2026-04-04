package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestChangelogAdd(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	command := newRootCmd("test")
	command.AddCommand(newChangelogCmd())
	var out bytes.Buffer
	command.SetOut(&out)
	command.SetErr(&out)
	command.SetArgs([]string{"changelog", "add", "feat", "incident command added", "--config", cfgPath})

	if err := command.Execute(); err != nil {
		t.Fatalf("changelog add execute error: %v", err)
	}

	path := filepath.Join(tmp, ".teraflow", "changelog", time.Now().Format("2006-01")+".jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read changelog file: %v", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	count := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		count++
		var entry ChangelogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("json parse error: %v", err)
		}
		if entry.Type != "feat" {
			t.Fatalf("unexpected type: %s", entry.Type)
		}
		if entry.Message != "incident command added" {
			t.Fatalf("unexpected message: %s", entry.Message)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 jsonl line, got %d", count)
	}
}

func TestChangelogAddInvalidType(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	command := newRootCmd("test")
	command.AddCommand(newChangelogCmd())
	var out bytes.Buffer
	command.SetOut(&out)
	command.SetErr(&out)
	command.SetArgs([]string{"changelog", "add", "unknown", "some message", "--config", cfgPath})

	err := command.Execute()
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
	if !strings.Contains(err.Error(), "type must be one of") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChangelogAddEmptyMessage(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	command := newRootCmd("test")
	command.AddCommand(newChangelogCmd())
	command.SetArgs([]string{"changelog", "add", "feat", "  ", "--config", cfgPath})

	err := command.Execute()
	if err == nil {
		t.Fatal("expected error for empty message")
	}
	if !strings.Contains(err.Error(), "message is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChangelogGenerateEmpty(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	command := newRootCmd("test")
	command.AddCommand(newChangelogCmd())
	var out bytes.Buffer
	command.SetOut(&out)
	command.SetErr(&out)
	command.SetArgs([]string{"changelog", "generate", "--config", cfgPath})

	if err := command.Execute(); err != nil {
		t.Fatalf("changelog generate (empty): %v", err)
	}

	if !strings.Contains(out.String(), "No changelog entries found.") {
		t.Fatalf("expected no entries message, got: %s", out.String())
	}
}

func TestChangelogGenerateNonRFC3339Timestamp(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	dir := filepath.Join(tmp, ".teraflow", "changelog")
	// Use a non-RFC3339 timestamp that is >= 10 chars
	mustWrite(t, filepath.Join(dir, "2026-04.jsonl"), `{"type":"chore","message":"cleanup old code","timestamp":"2026-04-05 10:00:00"}`+"\n")

	command := newRootCmd("test")
	command.AddCommand(newChangelogCmd())
	var out bytes.Buffer
	command.SetOut(&out)
	command.SetErr(&out)
	command.SetArgs([]string{"changelog", "generate", "--config", cfgPath})

	if err := command.Execute(); err != nil {
		t.Fatalf("changelog generate non-RFC3339: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "## Chores") {
		t.Fatalf("expected Chores section, got: %s", got)
	}
	if !strings.Contains(got, "cleanup old code") {
		t.Fatalf("expected message in output, got: %s", got)
	}
}

func TestChangelogGenerateWithOutput(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	dir := filepath.Join(tmp, ".teraflow", "changelog")
	mustWrite(t, filepath.Join(dir, "2026-04.jsonl"), "{\"type\":\"feat\",\"message\":\"new feature\",\"timestamp\":\"2026-04-05T10:00:00Z\"}\n")

	outFile := filepath.Join(tmp, "release-notes.md")

	command := newRootCmd("test")
	command.AddCommand(newChangelogCmd())
	var out bytes.Buffer
	command.SetOut(&out)
	command.SetErr(&out)
	command.SetArgs([]string{"changelog", "generate", "--config", cfgPath, "--output", outFile})

	if err := command.Execute(); err != nil {
		t.Fatalf("changelog generate --output failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}
	if !strings.Contains(string(data), "new feature") {
		t.Fatalf("output file missing content: %s", string(data))
	}
}

func TestChangelogGenerate(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	dir := filepath.Join(tmp, ".teraflow", "changelog")
	mustWrite(t, filepath.Join(dir, "2026-04.jsonl"), "{\"type\":\"feat\",\"message\":\"incident command added\",\"timestamp\":\"2026-04-05T10:00:00Z\"}\n{\"type\":\"fix\",\"message\":\"close incident bug fixed\",\"timestamp\":\"2026-04-05T11:00:00Z\"}\n")

	command := newRootCmd("test")
	command.AddCommand(newChangelogCmd())
	var out bytes.Buffer
	command.SetOut(&out)
	command.SetErr(&out)
	command.SetArgs([]string{"changelog", "generate", "--config", cfgPath})

	if err := command.Execute(); err != nil {
		t.Fatalf("changelog generate execute error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "# Release Notes") {
		t.Fatalf("missing header in output: %s", got)
	}
	if !strings.Contains(got, "## Features") || !strings.Contains(got, "incident command added") {
		t.Fatalf("missing feature section in output: %s", got)
	}
	if !strings.Contains(got, "## Bug Fixes") || !strings.Contains(got, "close incident bug fixed") {
		t.Fatalf("missing bug fix section in output: %s", got)
	}
}

func TestChangelogGenerateWithDateFormats(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	dir := filepath.Join(tmp, ".teraflow", "changelog")
	mustWrite(t, filepath.Join(dir, "2026-04.jsonl"), strings.Join([]string{
		`{"type":"feat","message":"feature A","timestamp":"2026-04-05T10:00:00Z"}`,
		`{"type":"fix","message":"bug fix B","timestamp":"2026-04-05 11:00:00"}`,
		`{"type":"chore","message":"cleanup C","timestamp":"2026-04-05"}`,
		`{"type":"docs","message":"docs D","timestamp":"2026-04-06T09:00:00Z"}`,
		`{"type":"refactor","message":"refactor E","timestamp":"2026-04-07"}`,
	}, "\n")+"\n")

	command := newRootCmd("test")
	command.AddCommand(newChangelogCmd())
	var out bytes.Buffer
	command.SetOut(&out)
	command.SetErr(&out)
	command.SetArgs([]string{"changelog", "generate", "--config", cfgPath})

	if err := command.Execute(); err != nil {
		t.Fatalf("changelog generate with date formats: %v", err)
	}

	got := out.String()
	for _, section := range []string{"## Features", "## Bug Fixes", "## Chores", "## Documentation", "## Refactoring"} {
		if !strings.Contains(got, section) {
			t.Fatalf("missing section %q in output: %s", section, got)
		}
	}
	for _, expected := range []string{"feature A (2026-04-05)", "bug fix B (2026-04-05)", "cleanup C (2026-04-05)", "docs D (2026-04-06)", "refactor E (2026-04-07)"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("missing entry %q in output: %s", expected, got)
		}
	}
}

func TestChangelogGenerateInvalidJSONLine(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "changelog", "2026-04.jsonl"), "{invalid-json}\n")

	command := newRootCmd("test")
	command.AddCommand(newChangelogCmd())
	command.SetArgs([]string{"changelog", "generate", "--config", cfgPath})

	err := command.Execute()
	if err == nil {
		t.Fatal("expected error for invalid changelog JSON line")
	}
	if !strings.Contains(err.Error(), "parse changelog entry") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChangelogGenerateOpenFileError(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	dir := filepath.Join(tmp, ".teraflow", "changelog")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir changelog dir: %v", err)
	}
	if err := os.Symlink(filepath.Join(tmp, "not-found.jsonl"), filepath.Join(dir, "broken.jsonl")); err != nil {
		// On platforms where symlink creation is restricted, skip gracefully.
		if errorsIsPermission(err) {
			t.Skipf("symlink not supported in this environment: %v", err)
		}
		t.Fatalf("create symlink: %v", err)
	}

	command := newRootCmd("test")
	command.AddCommand(newChangelogCmd())
	command.SetArgs([]string{"changelog", "generate", "--config", cfgPath})

	err := command.Execute()
	if err == nil {
		t.Fatal("expected open changelog file error")
	}
	if !strings.Contains(err.Error(), "open changelog file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChangelogGenerateOutputDirCreationError(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "changelog", "2026-04.jsonl"), "{\"type\":\"feat\",\"message\":\"x\",\"timestamp\":\"2026-04-05T10:00:00Z\"}\n")

	command := newRootCmd("test")
	command.AddCommand(newChangelogCmd())
	command.SetArgs([]string{"changelog", "generate", "--config", cfgPath, "--output", "/dev/null/release-notes.md"})

	err := command.Execute()
	if err == nil {
		t.Fatal("expected error when output directory cannot be created")
	}
	if !strings.Contains(err.Error(), "create output directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChangelogGenerateOutputWriteError(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "changelog", "2026-04.jsonl"), "{\"type\":\"feat\",\"message\":\"x\",\"timestamp\":\"2026-04-05T10:00:00Z\"}\n")

	command := newRootCmd("test")
	command.AddCommand(newChangelogCmd())
	command.SetArgs([]string{"changelog", "generate", "--config", cfgPath, "--output", tmp})

	err := command.Execute()
	if err == nil {
		t.Fatal("expected error when output path is a directory")
	}
	if !strings.Contains(err.Error(), "write release notes") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func errorsIsPermission(err error) bool {
	return os.IsPermission(err) || strings.Contains(err.Error(), "operation not permitted")
}
