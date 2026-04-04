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
