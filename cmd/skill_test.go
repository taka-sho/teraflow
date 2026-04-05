package cmd

import (
	"bytes"
	"strings"
	"testing"
)

const skillTestDir = "../internal/skill/testdata/skills"

func TestSkillList(t *testing.T) {
	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"skill", "list", "--skills-dir", skillTestDir})

	if err := root.Execute(); err != nil {
		t.Fatalf("skill list failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "incident-skill") {
		t.Fatalf("expected incident-skill in output, got:\n%s", got)
	}
	if !strings.Contains(got, "review-skill") {
		t.Fatalf("expected review-skill in output, got:\n%s", got)
	}
}

func TestSkillShow(t *testing.T) {
	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"skill", "show", "review-skill", "--skills-dir", skillTestDir})

	if err := root.Execute(); err != nil {
		t.Fatalf("skill show failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "name: review-skill") {
		t.Fatalf("expected skill name in output, got:\n%s", got)
	}
	if !strings.Contains(got, "prompts.system: You are reviewer") {
		t.Fatalf("expected system prompt in output, got:\n%s", got)
	}
}

func TestSkillShowNotFound(t *testing.T) {
	root := newRootCmd("test")
	root.SetArgs([]string{"skill", "show", "missing-skill", "--skills-dir", skillTestDir})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for missing skill")
	}
	if !strings.Contains(err.Error(), "skill not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSkillValidate(t *testing.T) {
	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"skill", "validate", "--skills-dir", skillTestDir})

	if err := root.Execute(); err != nil {
		t.Fatalf("skill validate failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "All 2 skills are valid.") {
		t.Fatalf("unexpected output:\n%s", got)
	}
}

func TestSkillListJSON(t *testing.T) {
	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--format", "json", "skill", "list", "--skills-dir", skillTestDir})

	if err := root.Execute(); err != nil {
		t.Fatalf("skill list --format json failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, `"name":"incident-skill"`) && !strings.Contains(got, `"name": "incident-skill"`) {
		t.Fatalf("expected JSON name field, got:\n%s", got)
	}
	if !strings.Contains(got, `"labels"`) {
		t.Fatalf("expected labels in JSON output, got:\n%s", got)
	}
}
