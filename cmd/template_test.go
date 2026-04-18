package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	discopkg "github.com/taka-sho/teraflow/internal/discovery"
	"gopkg.in/yaml.v3"
)

func buildTemplateCmd(t *testing.T, tmpdir string) *cobra.Command {
	t.Helper()
	configPath := filepath.Join(tmpdir, ".github", "teraflow.yml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := &cobra.Command{Use: "teraflow", SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().String("config", configPath, "")
	root.PersistentFlags().String("format", "text", "")
	root.PersistentFlags().BoolP("verbose", "v", false, "")
	root.AddCommand(newTemplateCmd())
	return root
}

func TestTemplateInit(t *testing.T) {
	tmp := t.TempDir()
	root := buildTemplateCmd(t, tmp)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"template", "init"})
	if err := root.Execute(); err != nil {
		t.Fatalf("init error: %v", err)
	}

	templatePath := filepath.Join(tmp, ".teraflow", "discovery", "requirement-template.yaml")
	if _, err := os.Stat(templatePath); err != nil {
		t.Fatalf("template file not created: %v", err)
	}

	data, _ := os.ReadFile(templatePath)
	var tmpl discopkg.RequirementTemplate
	if err := yaml.Unmarshal(data, &tmpl); err != nil {
		t.Fatalf("parse template: %v", err)
	}
	if len(tmpl.Items) == 0 {
		t.Fatal("template has no items")
	}
}

func TestTemplateInitForce(t *testing.T) {
	tmp := t.TempDir()
	root := buildTemplateCmd(t, tmp)
	root.SetArgs([]string{"template", "init"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	// second init without --force should fail
	root2 := buildTemplateCmd(t, tmp)
	root2.SetArgs([]string{"template", "init"})
	if err := root2.Execute(); err == nil {
		t.Fatal("expected error without --force")
	}

	// with --force should succeed
	root3 := buildTemplateCmd(t, tmp)
	root3.SetArgs([]string{"template", "init", "--force"})
	if err := root3.Execute(); err != nil {
		t.Fatalf("force init failed: %v", err)
	}
}

func TestTemplateList(t *testing.T) {
	tmp := t.TempDir()
	root := buildTemplateCmd(t, tmp)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"template", "list"})
	if err := root.Execute(); err != nil {
		t.Fatalf("list error: %v", err)
	}

	outStr := out.String()
	if !strings.Contains(outStr, "ID") || !strings.Contains(outStr, "CATEGORY") {
		t.Fatalf("unexpected list output: %s", outStr)
	}
	if !strings.Contains(outStr, "required") {
		t.Fatalf("expected 'required' in output: %s", outStr)
	}
}

func TestTemplateListCategoryFilter(t *testing.T) {
	tmp := t.TempDir()
	root := buildTemplateCmd(t, tmp)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"template", "list", "--category", "optional"})
	if err := root.Execute(); err != nil {
		t.Fatalf("list --category error: %v", err)
	}

	outStr := out.String()
	if strings.Contains(outStr, "required") {
		t.Fatalf("should not contain required fields: %s", outStr)
	}
}

func TestTemplateValidate(t *testing.T) {
	tmp := t.TempDir()
	root := buildTemplateCmd(t, tmp)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"template", "validate"})
	if err := root.Execute(); err != nil {
		t.Fatalf("validate error: %v", err)
	}
	if !strings.Contains(out.String(), "valid") {
		t.Fatalf("unexpected validate output: %s", out.String())
	}
}

func TestTemplateValidateInvalid(t *testing.T) {
	tmp := t.TempDir()
	templatePath := filepath.Join(tmp, ".teraflow", "discovery", "requirement-template.yaml")
	if err := os.MkdirAll(filepath.Dir(templatePath), 0o755); err != nil {
		t.Fatal(err)
	}
	// write template with no items
	tmpl := discopkg.RequirementTemplate{Version: "1"}
	data, _ := yaml.Marshal(tmpl)
	if err := os.WriteFile(templatePath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	root := buildTemplateCmd(t, tmp)
	root.SetArgs([]string{"template", "validate"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected validation error for empty template")
	}
}

func TestTemplateDiffNoChanges(t *testing.T) {
	tmp := t.TempDir()
	root := buildTemplateCmd(t, tmp)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"template", "diff"})
	if err := root.Execute(); err != nil {
		t.Fatalf("diff error: %v", err)
	}
	if !strings.Contains(out.String(), "No changes from default") {
		t.Fatalf("unexpected diff output: %s", out.String())
	}
}

func TestTemplateDiffWithChanges(t *testing.T) {
	tmp := t.TempDir()
	// init template then add a field
	root := buildTemplateCmd(t, tmp)
	root.SetArgs([]string{"template", "init"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	root2 := buildTemplateCmd(t, tmp)
	root2.SetArgs([]string{"template", "add", "custom_field",
		"--label", "カスタムフィールド", "--category", "optional"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("add error: %v", err)
	}

	root3 := buildTemplateCmd(t, tmp)
	var out bytes.Buffer
	root3.SetOut(&out)
	root3.SetArgs([]string{"template", "diff"})
	if err := root3.Execute(); err != nil {
		t.Fatalf("diff error: %v", err)
	}
	if !strings.Contains(out.String(), "custom_field") {
		t.Fatalf("expected custom_field in diff: %s", out.String())
	}
}

func TestTemplateAdd(t *testing.T) {
	tmp := t.TempDir()
	root := buildTemplateCmd(t, tmp)
	root.SetArgs([]string{"template", "init"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	root2 := buildTemplateCmd(t, tmp)
	var out bytes.Buffer
	root2.SetOut(&out)
	root2.SetArgs([]string{"template", "add", "new_field",
		"--label", "新フィールド", "--category", "optional", "--hint", "some hint"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("add error: %v", err)
	}
	if !strings.Contains(out.String(), "new_field") {
		t.Fatalf("expected new_field in output: %s", out.String())
	}

	// verify field is persisted
	templatePath := filepath.Join(tmp, ".teraflow", "discovery", "requirement-template.yaml")
	data, _ := os.ReadFile(templatePath)
	var tmpl discopkg.RequirementTemplate
	yaml.Unmarshal(data, &tmpl) //nolint:errcheck
	found := false
	for _, item := range tmpl.Items {
		if item.ID == "new_field" {
			found = true
		}
	}
	if !found {
		t.Fatal("new_field not persisted in template")
	}
}

func TestTemplateRemove(t *testing.T) {
	tmp := t.TempDir()
	// init and add a field, then remove it
	root := buildTemplateCmd(t, tmp)
	root.SetArgs([]string{"template", "init"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	root2 := buildTemplateCmd(t, tmp)
	root2.SetArgs([]string{"template", "add", "to_remove",
		"--label", "削除予定", "--category", "optional"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("add error: %v", err)
	}

	root3 := buildTemplateCmd(t, tmp)
	var out bytes.Buffer
	root3.SetOut(&out)
	root3.SetArgs([]string{"template", "remove", "to_remove", "--force"})
	if err := root3.Execute(); err != nil {
		t.Fatalf("remove error: %v", err)
	}
	if !strings.Contains(out.String(), "to_remove") {
		t.Fatalf("expected to_remove in output: %s", out.String())
	}

	// verify removed
	templatePath := filepath.Join(tmp, ".teraflow", "discovery", "requirement-template.yaml")
	data, _ := os.ReadFile(templatePath)
	var tmpl discopkg.RequirementTemplate
	yaml.Unmarshal(data, &tmpl) //nolint:errcheck
	for _, item := range tmpl.Items {
		if item.ID == "to_remove" {
			t.Fatal("to_remove still present after remove")
		}
	}
}
