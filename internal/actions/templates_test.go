package actions_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/actions"
)

func generatedTemplatesWithVersion(t *testing.T, version string) map[string][]byte {
	t.Helper()

	tmp := t.TempDir()
	if err := actions.GenerateWorkflows(tmp, version); err != nil {
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

func TestTemplateNoHardcodedVersion(t *testing.T) {
	templates := generatedTemplatesWithVersion(t, "LATEST_TAG")
	hardcodedVersion := regexp.MustCompile(`go install.*teraflow@v[0-9]`)

	for name, content := range templates {
		if hardcodedVersion.Match(content) {
			t.Errorf("template %s contains hardcoded teraflow version", name)
		}
	}
}

func TestTemplateGitHubOutputDelimiter(t *testing.T) {
	templates := generatedTemplatesWithVersion(t, "LATEST_TAG")
	legacyDelimiter := regexp.MustCompile(`echo\s+"output<<EOF"\s*>>`)
	newlineAfterCat := regexp.MustCompile(`echo\s+""\s*>>\s*"?\$GITHUB_OUTPUT"?`)

	for name, content := range templates {
		s := string(content)
		if legacyDelimiter.MatchString(s) {
			t.Errorf("template %s contains legacy GITHUB_OUTPUT delimiter", name)
		}

		lines := strings.Split(s, "\n")
		for i, line := range lines {
			if !strings.Contains(line, "cat ") || !strings.Contains(line, "GITHUB_OUTPUT") {
				continue
			}
			found := false
			for j := i + 1; j < len(lines) && j <= i+4; j++ {
				if strings.TrimSpace(lines[j]) == "" {
					continue
				}
				if newlineAfterCat.MatchString(lines[j]) {
					found = true
				}
				break
			}
			if !found {
				t.Errorf("template %s missing newline guard after cat to GITHUB_OUTPUT (line %d)", name, i+1)
			}
		}
	}
}

func TestTemplateGitConfigBeforeCommit(t *testing.T) {
	templates := generatedTemplates(t)

	for name, content := range templates {
		s := string(content)
		commitIdx := strings.Index(s, "git commit")
		if commitIdx == -1 {
			continue
		}
		configIdx := strings.Index(s, "git config user.name")
		if configIdx == -1 || configIdx > commitIdx {
			t.Errorf("template %s commits without prior git config user.name", name)
		}
	}
}

func TestTemplateAPIKeyCheck(t *testing.T) {
	templates := generatedTemplates(t)
	keyCheckAny := regexp.MustCompile(`\[\s*-n "\$ANTHROPIC_API_KEY"\s*\]\s*\|\|\s*\[\s*-n "\$OPENAI_API_KEY"\s*\]|\[\s*-z "\$ANTHROPIC_API_KEY"\s*\]\s*&&\s*\[\s*-z "\$OPENAI_API_KEY"\s*\]`)

	for name, content := range templates {
		s := string(content)
		if !strings.Contains(s, "name: Check API key") {
			continue
		}
		if !strings.Contains(s, "ANTHROPIC_API_KEY") || !strings.Contains(s, "OPENAI_API_KEY") {
			t.Errorf("template %s check api key step must reference both API keys", name)
		}
		if !keyCheckAny.MatchString(s) {
			t.Errorf("template %s check api key step must use OR-logic across both API keys", name)
		}
	}
}

func TestTemplateDiscoveryInputIncludesBody(t *testing.T) {
	templates := generatedTemplates(t)
	reqAgent, ok := templates["teraflow-req-agent"]
	if !ok {
		t.Fatal("teraflow-req-agent template not generated")
	}

	s := string(reqAgent)
	if !strings.Contains(s, "${{ github.event.discussion.title }}") {
		t.Fatal("discovery input missing discussion.title")
	}
	if !strings.Contains(s, "${{ github.event.discussion.body }}") {
		t.Fatal("discovery input missing discussion.body")
	}
}

func TestTemplateDiscoveryUsesStructuredSummary(t *testing.T) {
	templates := generatedTemplates(t)
	reqAgent, ok := templates["teraflow-req-agent"]
	if !ok {
		t.Fatal("teraflow-req-agent template not generated")
	}

	s := string(reqAgent)
	if !strings.Contains(s, "discussion-${DISC_NUM}-summary.yaml") {
		t.Fatal("discovery flow missing summary file path")
	}
	if !strings.Contains(s, "## 構造化サマリ") {
		t.Fatal("discovery input missing structured summary section")
	}
	if strings.Contains(s, "## 現在の CoDD ドラフト") {
		t.Fatal("discovery input should not include raw draft content")
	}
}

func TestTemplateDiscoveryCommentGuard(t *testing.T) {
	templates := generatedTemplates(t)
	reqAgent, ok := templates["teraflow-req-agent"]
	if !ok {
		t.Fatal("teraflow-req-agent template not generated")
	}

	s := string(reqAgent)
	if !strings.Contains(s, "count_numbered_questions") {
		t.Fatal("discovery flow missing numbered-question guard")
	}
	if !strings.Contains(s, "is_footer_only") {
		t.Fatal("discovery flow missing footer-only detection")
	}
	if !strings.Contains(s, "question_count < 3") {
		t.Fatal("discovery flow missing minimum-question fallback condition")
	}
	if !strings.Contains(s, "build_fallback_comment") {
		t.Fatal("discovery flow missing fallback comment builder")
	}
}

func TestAllTemplatesRenderWithoutError(t *testing.T) {
	templates := generatedTemplates(t)
	if got, want := len(templates), len(actions.WorkflowNames); got != want {
		t.Fatalf("rendered template count mismatch: got %d, want %d", got, want)
	}
}
