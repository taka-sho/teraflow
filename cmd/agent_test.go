package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/agent"
)

func TestAgentStatus(t *testing.T) {
	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"agent", "status"})

	if err := root.Execute(); err != nil {
		t.Fatalf("agent status: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "requirements") {
		t.Fatalf("expected agent types in output: %s", got)
	}
}

func TestAgentStatusJSON(t *testing.T) {
	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"agent", "status", "--format", "json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("agent status --format json: %v", err)
	}

	if !strings.Contains(out.String(), "available_types") {
		t.Fatalf("expected JSON output: %s", out.String())
	}
}

func TestAgentAssignMissingType(t *testing.T) {
	root := newRootCmd("test")
	root.SetArgs([]string{"agent", "assign", "some input"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --type is missing")
	}
}

func TestAgentAssignMissingAPIKey(t *testing.T) {
	root := newRootCmd("test")
	root.SetArgs([]string{"agent", "assign", "--type", "review", "some input"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when API key is missing")
	}
	if !strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAgentAssignMissingInput(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "dummy")

	root := newRootCmd("test")
	root.SetArgs([]string{"agent", "assign", "--type", "review"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when input is missing")
	}
	if !strings.Contains(err.Error(), "input required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAgentAssignInputFileReadError(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "dummy")

	root := newRootCmd("test")
	root.SetArgs([]string{"agent", "assign", "--type", "review", "--input-file", "missing.txt"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when input file cannot be read")
	}
	if !strings.Contains(err.Error(), "read input file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAgentAssignInputFileUnknownType(t *testing.T) {
	tmp := t.TempDir()
	inputFile := filepath.Join(tmp, "input.txt")
	if err := os.WriteFile(inputFile, []byte("hello from file"), 0o644); err != nil {
		t.Fatalf("write input file: %v", err)
	}

	root := newRootCmd("test")
	root.SetArgs([]string{
		"agent", "assign",
		"--type", "unknown",
		"--input-file", inputFile,
		"--api-key", "dummy",
		"--trust-level=",
	})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected unknown type error")
	}
	if !strings.Contains(err.Error(), "unknown agent type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAgentAssignRequiresUserWhenGitHubEnforcementEnabled(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, `version: "1"
project:
  name: "my-project"
  description: ""
  repository: ""
ai:
  default_provider: anthropic
rbac:
  enabled: true
  github_enforcement: true
harness:
  score_threshold: 70
`)

	root := newRootCmd("test")
	root.SetArgs([]string{
		"agent", "assign",
		"--type", "unknown",
		"--config", cfgPath,
		"--api-key", "dummy",
		"request",
	})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --user is missing under github_enforcement")
	}
	if !strings.Contains(err.Error(), "--user flag is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAgentAssignAcceptsUserFlag(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, `version: "1"
project:
  name: "my-project"
  description: ""
  repository: ""
ai:
  default_provider: anthropic
rbac:
  enabled: true
  github_enforcement: true
harness:
  score_threshold: 70
`)

	root := newRootCmd("test")
	root.SetArgs([]string{
		"agent", "assign",
		"--type", "unknown",
		"--config", cfgPath,
		"--user", "octocat",
		"--api-key", "dummy",
		"request",
	})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected unknown type error")
	}
	if !strings.Contains(err.Error(), "unknown agent type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAgentAssignSuccessTextAndJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"generated output"}],"usage":{"input_tokens":2,"output_tokens":3}}`))
	}))
	t.Cleanup(server.Close)

	oldBaseURL := agent.BaseURL
	agent.BaseURL = server.URL
	t.Cleanup(func() { agent.BaseURL = oldBaseURL })

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"agent", "assign", "--type", "review", "--api-key", "dummy", "request"})
	if err := root.Execute(); err != nil {
		t.Fatalf("agent assign text failed: %v", err)
	}

	textOut := out.String()
	if !strings.Contains(textOut, "Agent: review") || !strings.Contains(textOut, "generated output") {
		t.Fatalf("unexpected text output: %s", textOut)
	}
	if !strings.Contains(textOut, "Tokens used: 5") {
		t.Fatalf("expected token usage in output: %s", textOut)
	}

	out.Reset()
	root = newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--format", "json", "agent", "assign", "--type", "review", "--api-key", "dummy", "request"})
	if err := root.Execute(); err != nil {
		t.Fatalf("agent assign json failed: %v", err)
	}

	jsonOut := out.String()
	if !strings.Contains(jsonOut, `"type": "review"`) || !strings.Contains(jsonOut, `"tokens_used": 5`) {
		t.Fatalf("unexpected json output: %s", jsonOut)
	}
}

func TestAgentAssignUsesConfigAssignmentProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer openai-key" {
			http.Error(w, "missing or wrong auth header", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"openai output"}}],"usage":{"total_tokens":9}}`))
	}))
	t.Cleanup(server.Close)

	oldOpenAIBaseURL := agent.OpenAIBaseURL
	oldAnthropicBaseURL := agent.BaseURL
	agent.OpenAIBaseURL = server.URL
	agent.BaseURL = server.URL
	t.Cleanup(func() {
		agent.OpenAIBaseURL = oldOpenAIBaseURL
		agent.BaseURL = oldAnthropicBaseURL
	})

	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, `version: "1"
project:
  name: "my-project"
  description: ""
  repository: ""
ai:
  default_provider: anthropic
assignments:
  review:
    provider: openai
harness:
  score_threshold: 70
`)

	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{
		"agent", "assign",
		"--type", "review",
		"--config", cfgPath,
		"--api-key", "openai-key",
		"request",
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("agent assign should use openai assignment: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "openai output") || !strings.Contains(got, "Tokens used: 9") {
		t.Fatalf("unexpected output: %s", got)
	}
}

func TestAgentAssignFallbackWithoutConfigUsesAnthropic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-key"); got != "anthropic-key" {
			http.Error(w, "missing or wrong anthropic key", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"anthropic fallback output"}],"usage":{"input_tokens":2,"output_tokens":4}}`))
	}))
	t.Cleanup(server.Close)

	oldBaseURL := agent.BaseURL
	agent.BaseURL = server.URL
	t.Cleanup(func() { agent.BaseURL = oldBaseURL })

	t.Setenv("ANTHROPIC_API_KEY", "anthropic-key")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{
		"agent", "assign",
		"--type", "review",
		"--config", filepath.Join(t.TempDir(), ".github", "missing.yml"),
		"request",
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("agent assign should fallback to anthropic: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "anthropic fallback output") || !strings.Contains(got, "Tokens used: 6") {
		t.Fatalf("unexpected output: %s", got)
	}
}

func TestAgentAssignUsesDefaultPromptWhenNoSkillMatchesType(t *testing.T) {
	var gotSystem string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			System string `json:"system"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		gotSystem = body.System

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	t.Cleanup(server.Close)

	oldBaseURL := agent.BaseURL
	agent.BaseURL = server.URL
	t.Cleanup(func() { agent.BaseURL = oldBaseURL })

	root := newRootCmd("test")
	root.SetArgs([]string{
		"agent", "assign",
		"--type", "implement",
		"--api-key", "dummy",
		"request",
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("agent assign should succeed: %v", err)
	}

	if !strings.Contains(gotSystem, "提供された要件に基づいてコードを実装してください。") {
		t.Fatalf("expected default implement prompt, got: %q", gotSystem)
	}
}

func TestAgentAssignUsesSkillPromptWhenSkillFlagSpecified(t *testing.T) {
	var gotSystem string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			System string `json:"system"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		gotSystem = body.System

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	t.Cleanup(server.Close)

	oldBaseURL := agent.BaseURL
	agent.BaseURL = server.URL
	t.Cleanup(func() { agent.BaseURL = oldBaseURL })

	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, `version: "1"
project:
  name: "my-project"
  description: ""
  repository: ""
ai:
  default_provider: anthropic
harness:
  score_threshold: 70
`)
	mustWrite(t, filepath.Join(tmp, "skills", "review.yml"), `name: code-review
version: "1"
description: "review"
trigger:
  labels: []
  categories: []
  agent_types: ["review"]
prompts:
  system: |
    SKILL_REVIEW_PROMPT
context:
  include: []
  exclude: []
  max_context_tokens: 0
output:
  dialogue:
    header: ""
    footer: ""
  confirm:
    header: ""
    footer: ""
options:
  max_tokens: 0
  temperature: 0.0
`)

	root := newRootCmd("test")
	root.SetArgs([]string{
		"agent", "assign",
		"--type", "review",
		"--config", cfgPath,
		"--skill", "code-review",
		"--api-key", "dummy",
		"request",
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("agent assign should succeed: %v", err)
	}

	if !strings.Contains(gotSystem, "SKILL_REVIEW_PROMPT") {
		t.Fatalf("expected skill prompt to be used, got: %q", gotSystem)
	}
}

func TestAgentAssignUsesEmbeddedSkillWhenSkillsDirMissing(t *testing.T) {
	var gotSystem string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			System string `json:"system"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		gotSystem = body.System

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	t.Cleanup(server.Close)

	oldBaseURL := agent.BaseURL
	agent.BaseURL = server.URL
	t.Cleanup(func() { agent.BaseURL = oldBaseURL })

	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, `version: "1"
project:
  name: "my-project"
  description: ""
  repository: ""
ai:
  default_provider: anthropic
harness:
  score_threshold: 70
`)

	root := newRootCmd("test")
	var stderr bytes.Buffer
	root.SetErr(&stderr)
	root.SetArgs([]string{
		"agent", "assign",
		"--type", "review",
		"--config", cfgPath,
		"--skill", "code-review",
		"--api-key", "dummy",
		"request",
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("agent assign should succeed with embedded skills: %v", err)
	}

	if stderr.String() != "" {
		t.Fatalf("unexpected stderr output: %q", stderr.String())
	}
	if !strings.Contains(gotSystem, "あなたはシニアソフトウェアエンジニアです。") {
		t.Fatalf("expected embedded review skill prompt, got: %q", gotSystem)
	}
}

func TestAgentAssignReturnsErrorWhenSkillNotFound(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, `version: "1"
project:
  name: "my-project"
  description: ""
  repository: ""
ai:
  default_provider: anthropic
harness:
  score_threshold: 70
`)
	mustWrite(t, filepath.Join(tmp, "skills", "review.yml"), `name: code-review
version: "1"
description: "review"
trigger:
  labels: []
  categories: []
  agent_types: ["review"]
prompts:
  system: |
    SKILL_REVIEW_PROMPT
context:
  include: []
  exclude: []
  max_context_tokens: 0
output:
  dialogue:
    header: ""
    footer: ""
  confirm:
    header: ""
    footer: ""
options:
  max_tokens: 0
  temperature: 0.0
`)

	root := newRootCmd("test")
	root.SetArgs([]string{
		"agent", "assign",
		"--type", "review",
		"--config", cfgPath,
		"--skill", "does-not-exist",
		"--api-key", "dummy",
		"request",
	})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when skill does not exist")
	}
	if !strings.Contains(err.Error(), "skill not found: does-not-exist") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAgentAssignBuildsAdditionalContextFromIndex(t *testing.T) {
	var gotUserPrompt string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if len(body.Messages) == 0 {
			t.Fatalf("expected at least one message")
		}
		gotUserPrompt = body.Messages[0].Content

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	t.Cleanup(server.Close)

	oldBaseURL := agent.BaseURL
	agent.BaseURL = server.URL
	t.Cleanup(func() { agent.BaseURL = oldBaseURL })

	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, `version: "1"
project:
  name: "my-project"
  description: ""
  repository: ""
ai:
  default_provider: anthropic
harness:
  score_threshold: 70
`)
	mustWrite(t, filepath.Join(tmp, "skills", "review.yml"), `name: code-review
version: "1"
description: "review"
trigger:
  labels: []
  categories: []
  agent_types: ["review"]
prompts:
  system: |
    SKILL_REVIEW_PROMPT
context:
  include: ["docs/*.md"]
  exclude: []
  max_context_tokens: 500
output:
  dialogue:
    header: ""
    footer: ""
  confirm:
    header: ""
    footer: ""
options:
  max_tokens: 0
  temperature: 0.0
`)
	mustWrite(t, filepath.Join(tmp, "docs", "design.md"), "# Design Doc\n\nindex based context")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "index.yml"), `version: "1"
generated_at: "2026-04-06T00:00:00Z"
entries:
  - node_id: "doc-1"
    title: "Design"
    path: "docs/design.md"
    updated_at: "2026-04-06T00:00:00Z"
    content_hash: "abc"
    summary_available: true
`)
	mustWrite(t, filepath.Join(tmp, ".teraflow", "summaries", "doc-1.txt"), "Summary from index")

	root := newRootCmd("test")
	root.SetArgs([]string{
		"agent", "assign",
		"--type", "review",
		"--config", cfgPath,
		"--api-key", "dummy",
		"request body",
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("agent assign should succeed: %v", err)
	}

	if !strings.Contains(gotUserPrompt, "## コンテキスト（関連文書）") {
		t.Fatalf("expected context wrapper in prompt: %q", gotUserPrompt)
	}
	if !strings.Contains(gotUserPrompt, "Summary from index") {
		t.Fatalf("expected assembled summary context: %q", gotUserPrompt)
	}
	if !strings.Contains(gotUserPrompt, "## ユーザー入力") || !strings.Contains(gotUserPrompt, "request body") {
		t.Fatalf("expected original input section in prompt: %q", gotUserPrompt)
	}
}

func TestAgentAssignKeepsBackwardCompatibilityWhenIndexMissing(t *testing.T) {
	var gotUserPrompt string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if len(body.Messages) == 0 {
			t.Fatalf("expected at least one message")
		}
		gotUserPrompt = body.Messages[0].Content

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	t.Cleanup(server.Close)

	oldBaseURL := agent.BaseURL
	agent.BaseURL = server.URL
	t.Cleanup(func() { agent.BaseURL = oldBaseURL })

	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, `version: "1"
project:
  name: "my-project"
  description: ""
  repository: ""
ai:
  default_provider: anthropic
harness:
  score_threshold: 70
`)
	mustWrite(t, filepath.Join(tmp, "skills", "review.yml"), `name: code-review
version: "1"
description: "review"
trigger:
  labels: []
  categories: []
  agent_types: ["review"]
prompts:
  system: |
    SKILL_REVIEW_PROMPT
context:
  include: ["docs/*.md"]
  exclude: []
  max_context_tokens: 500
output:
  dialogue:
    header: ""
    footer: ""
  confirm:
    header: ""
    footer: ""
options:
  max_tokens: 0
  temperature: 0.0
`)

	root := newRootCmd("test")
	root.SetArgs([]string{
		"agent", "assign",
		"--type", "review",
		"--config", cfgPath,
		"--api-key", "dummy",
		"request body",
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("agent assign should succeed: %v", err)
	}

	if gotUserPrompt != "request body" {
		t.Fatalf("expected backward-compatible prompt without index context, got: %q", gotUserPrompt)
	}
}
