package actions_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/actions"
	"gopkg.in/yaml.v3"
)

func generatedTemplates(t *testing.T) map[string][]byte {
	t.Helper()

	tmp := t.TempDir()
	if err := actions.GenerateWorkflows(tmp); err != nil {
		t.Fatalf("GenerateWorkflows: %v", err)
	}

	templates := make(map[string][]byte, len(actions.WorkflowNames))
	for _, name := range actions.WorkflowNames {
		path := filepath.Join(tmp, name+".yml")
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read template %s: %v", name, err)
		}
		templates[name] = content
	}

	return templates
}

func TestTemplatesYAMLValid(t *testing.T) {
	templates := generatedTemplates(t)

	for name, content := range templates {
		name := name
		content := content
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var v interface{}
			if err := yaml.Unmarshal(content, &v); err != nil {
				t.Fatalf("template %s is invalid YAML: %v", name, err)
			}
		})
	}
}

func TestTemplatesNoExcessBlankLines(t *testing.T) {
	templates := generatedTemplates(t)
	re := regexp.MustCompile(`\n{3,}`)

	for name, content := range templates {
		if re.FindString(string(content)) != "" {
			t.Errorf("template %s has 3+ consecutive blank lines", name)
		}
	}
}

func TestTemplatesNoColumnZeroInRunBlock(t *testing.T) {
	templates := generatedTemplates(t)

	for name, content := range templates {
		lines := strings.Split(string(content), "\n")
		inRunBlock := false

		for i, line := range lines {
			trimmed := strings.TrimSpace(line)

			if !inRunBlock {
				if trimmed == "run: |" {
					inRunBlock = true
				}
				continue
			}

			if trimmed == "" {
				continue
			}
			if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				continue
			}
			if strings.Contains(line, ":") {
				inRunBlock = false
				continue
			}

			t.Errorf("template %s has column-zero line in run block at line %d: %q", name, i+1, line)
		}
	}
}
