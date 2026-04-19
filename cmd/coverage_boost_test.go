package cmd

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/state"
)

func TestCommandConfigFlagErrors(t *testing.T) {
	tests := []struct {
		name string
		run  func() error
	}{
		{name: "stage list", run: func() error { return newStageListCmd().RunE(newStageListCmd(), nil) }},
		{name: "stage status", run: func() error { return newStageStatusCmd().RunE(newStageStatusCmd(), nil) }},
		{name: "stage advance", run: func() error { return newStageAdvanceCmd().RunE(newStageAdvanceCmd(), nil) }},
		{name: "phase list", run: func() error { return newPhaseListCmd().RunE(newPhaseListCmd(), nil) }},
		{name: "phase complete", run: func() error { return newPhaseCompleteCmd().RunE(newPhaseCompleteCmd(), nil) }},
		{name: "phase start", run: func() error { return newPhaseStartCmd().RunE(newPhaseStartCmd(), []string{"requirements"}) }},
		{name: "incident create", run: func() error {
			c := newIncidentCreateCmd()
			_ = c.Flags().Set("title", "t")
			_ = c.Flags().Set("severity", "major")
			return c.RunE(c, nil)
		}},
		{name: "incident list", run: func() error { return newIncidentListCmd().RunE(newIncidentListCmd(), nil) }},
		{name: "incident close", run: func() error {
			c := newIncidentCloseCmd()
			_ = c.Flags().Set("id", "inc-001")
			return c.RunE(c, nil)
		}},
		{name: "rework create", run: func() error {
			c := newReworkCreateCmd()
			_ = c.Flags().Set("group", "g")
			_ = c.Flags().Set("target-phase", "requirements")
			_ = c.Flags().Set("reason", "r")
			return c.RunE(c, nil)
		}},
		{name: "rework list", run: func() error { return newReworkListCmd().RunE(newReworkListCmd(), nil) }},
		{name: "label list", run: func() error { return newLabelListCmd().RunE(newLabelListCmd(), nil) }},
		{name: "changelog add", run: func() error { return newChangelogAddCmd().RunE(newChangelogAddCmd(), []string{"feat", "msg"}) }},
		{name: "changelog generate", run: func() error { return newChangelogGenerateCmd().RunE(newChangelogGenerateCmd(), nil) }},
		{name: "dashboard show", run: func() error { return newDashboardShowCmd().RunE(newDashboardShowCmd(), nil) }},
		{name: "scan", run: func() error { return newScanCmd().RunE(newScanCmd(), nil) }},
		{name: "status", run: func() error { return newStatusCmd().RunE(newStatusCmd(), nil) }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if err == nil {
				t.Fatal("expected config flag error")
			}
			if !strings.Contains(err.Error(), "flag accessed but not defined: config") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestDiscussionListGHError(t *testing.T) {
	oldLookPath := ghLookPath
	oldExecCommand := ghExecCommand
	ghLookPath = func(file string) (string, error) {
		return "/usr/bin/gh", nil
	}
	ghExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "exit 1")
	}
	t.Cleanup(func() {
		ghLookPath = oldLookPath
		ghExecCommand = oldExecCommand
	})

	root := newRootCmd("test")
	root.AddCommand(newDiscussionCmd())
	root.SetArgs([]string{"discussion", "list"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected E5003 error")
	}
	if !strings.Contains(err.Error(), "E5003") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDiscussionSummarizeNoGH(t *testing.T) {
	oldLookPath := ghLookPath
	ghLookPath = func(file string) (string, error) {
		return "", errors.New("not found")
	}
	t.Cleanup(func() { ghLookPath = oldLookPath })

	root := newRootCmd("test")
	root.AddCommand(newDiscussionCmd())
	root.SetArgs([]string{"discussion", "summarize", "1"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected E5001 error")
	}
	if !strings.Contains(err.Error(), "E5001") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestSummarizeDiscussionReadBodyError(t *testing.T) {
	old := httpDoer
	httpDoer = func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(errReader{})}, nil
	}
	t.Cleanup(func() { httpDoer = old })

	_, err := summarizeDiscussionWithAnthropic("content", "key")
	if err == nil {
		t.Fatal("expected E6002 error")
	}
	if !strings.Contains(err.Error(), "E6002") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSummarizeDiscussionEmptyText(t *testing.T) {
	old := httpDoer
	httpDoer = func(req *http.Request) (*http.Response, error) {
		body := `{"content":[{"type":"text","text":"   "}]}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	}
	t.Cleanup(func() { httpDoer = old })

	_, err := summarizeDiscussionWithAnthropic("content", "key")
	if err == nil {
		t.Fatal("expected empty response error")
	}
}

func TestCreateOrUpdateLabelEditError(t *testing.T) {
	oldExec := ghExecCommand
	call := 0
	ghExecCommand = func(name string, args ...string) *exec.Cmd {
		call++
		if call == 1 {
			return exec.Command("sh", "-c", "echo 'already exists' >&2; exit 1")
		}
		return exec.Command("sh", "-c", "echo 'edit failed' >&2; exit 1")
	}
	t.Cleanup(func() { ghExecCommand = oldExec })

	err := createOrUpdateLabel(labelDefinition{Name: "x", Color: "FFFFFF", Description: "d"}, true)
	if err == nil {
		t.Fatal("expected E5003 error")
	}
	if !strings.Contains(err.Error(), "E5003") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadLabelsMissingFileFallsBackToDefault(t *testing.T) {
	labels, fromConfig, err := loadLabels("/path/does/not/exist.yml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fromConfig {
		t.Fatal("expected fromConfig=false")
	}
	if len(labels) == 0 {
		t.Fatal("expected default labels")
	}
}

func TestLoadLabelsEmptyListFallsBackToDefault(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "labels.yml")
	if err := os.WriteFile(path, []byte("labels: []\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	labels, fromConfig, err := loadLabels(path)
	if err != nil {
		t.Fatalf("load labels: %v", err)
	}
	if fromConfig {
		t.Fatal("expected fromConfig=false for empty labels")
	}
	if len(labels) == 0 {
		t.Fatal("expected default labels")
	}
}

func TestChangelogAddMkdirError(t *testing.T) {
	cmd := newRootCmd("test")
	cmd.AddCommand(newChangelogCmd())
	cmd.SetArgs([]string{"--config", "/dev/null/.github/teraflow.yml", "changelog", "add", "feat", "x"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected mkdir error")
	}
	if !strings.Contains(err.Error(), "create changelog directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChangelogGenerateScannerError(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	longLine := `{"type":"feat","message":"` + strings.Repeat("a", 70*1024) + `","timestamp":"2026-04-05T10:00:00Z"}`
	mustWrite(t, filepath.Join(tmp, ".teraflow", "changelog", "2026-04.jsonl"), longLine+"\n")

	cmd := newRootCmd("test")
	cmd.AddCommand(newChangelogCmd())
	cmd.SetArgs([]string{"--config", cfgPath, "changelog", "generate"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected scanner error")
	}
	if !strings.Contains(err.Error(), "scan changelog file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChangelogAddOpenFileError(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	dir := filepath.Join(tmp, ".teraflow", "changelog", time.Now().Format("2006-01")+".jsonl")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cmd := newRootCmd("test")
	cmd.AddCommand(newChangelogCmd())
	cmd.SetArgs([]string{"--config", cfgPath, "changelog", "add", "feat", "x"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected open file error")
	}
	if !strings.Contains(err.Error(), "open changelog file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChangelogAddAppendError(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	target := filepath.Join(tmp, ".teraflow", "changelog", time.Now().Format("2006-01")+".jsonl")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Symlink("/dev/full", target); err != nil {
		t.Skipf("symlink not available: %v", err)
	}
	cmd := newRootCmd("test")
	cmd.AddCommand(newChangelogCmd())
	cmd.SetArgs([]string{"--config", cfgPath, "changelog", "add", "feat", "x"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected append error")
	}
	if !strings.Contains(err.Error(), "append changelog entry") && !strings.Contains(err.Error(), "open changelog file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDashboardInvalidFormat(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "operation", "testing")
	root := newRootCmd("test")
	root.AddCommand(newDashboardCmd())
	root.SetArgs([]string{"--config", configPath, "--format", "xml", "dashboard", "show"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected unsupported format error")
	}
}

func TestPhaseStartInvalidFormat(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := setupTestProjectState(t, tmp, "initial_development", "requirements")
	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "--format", "yaml", "phase", "start", "implementation"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected unsupported format")
	}
}

func TestPhaseCompleteInvalidFormat(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := setupTestProjectState(t, tmp, "initial_development", "requirements")
	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "--format", "yaml", "phase", "complete"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected unsupported format")
	}
}

func TestStageStatusInvalidFormat(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := setupTestProjectState(t, tmp, "release", "requirements")
	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "--format", "yaml", "stage", "status"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected unsupported format")
	}
}

func TestStageAdvanceInvalidFormatAndNoProject(t *testing.T) {
	t.Run("invalid format", func(t *testing.T) {
		tmp := t.TempDir()
		cfgPath := setupTestProjectState(t, tmp, "release", "requirements")
		root := newRootCmd("test")
		root.SetArgs([]string{"--config", cfgPath, "--format", "yaml", "stage", "advance"})
		if err := root.Execute(); err == nil {
			t.Fatal("expected unsupported format")
		}
	})
	t.Run("no project", func(t *testing.T) {
		tmp := t.TempDir()
		cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
		mustWrite(t, cfgPath, "version: \"1\"\n")
		root := newRootCmd("test")
		root.SetArgs([]string{"--config", cfgPath, "stage", "advance", "--yes"})
		if err := root.Execute(); err == nil {
			t.Fatal("expected no project error")
		}
	})
}

func TestLabelListCommandInvalidConfig(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "broken.yml")
	mustWrite(t, cfgPath, ":\n  bad: [\n")
	root := newRootCmd("test")
	root.AddCommand(newLabelCmd())
	root.SetArgs([]string{"label", "list", "--config", cfgPath})
	if err := root.Execute(); err == nil {
		t.Fatal("expected config error")
	}
}

func TestLoadLabelsAllInvalidFallback(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "labels.yml")
	mustWrite(t, path, "labels:\n  - name: \"   \"\n    color: \"\"\n")
	labels, fromConfig, err := loadLabels(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fromConfig {
		t.Fatal("expected fromConfig=false")
	}
	if len(labels) == 0 {
		t.Fatal("expected default labels")
	}
}

func TestBuildDashboardSummaryLimitsAndCounts(t *testing.T) {
	s := &state.ProjectState{}
	s.Project.Name = "test"
	s.Lifecycle.CurrentStage = "operation"
	s.Phases.Current = "testing"

	log := &state.ReworkLog{
		Reworks: []state.ReworkEntry{
			{ID: "rw-001", CreatedAt: "2026-04-05T01:00:00", Status: "open"},
			{ID: "rw-002", CreatedAt: "2026-04-05T02:00:00", Status: "resolved"},
			{ID: "rw-003", CreatedAt: "2026-04-05T03:00:00", Status: "OPEN"},
			{ID: "rw-004", CreatedAt: "2026-04-05T04:00:00", Status: "closed"},
			{ID: "rw-005", CreatedAt: "2026-04-05T05:00:00", Status: "resolved"},
			{ID: "rw-006", CreatedAt: "2026-04-05T06:00:00", Status: "open"},
		},
	}

	summary := buildDashboardSummary(s, log)
	if summary.Rework.Total != 6 || summary.Rework.Open != 3 || summary.Rework.Resolved != 3 {
		t.Fatalf("unexpected rework counts: %+v", summary.Rework)
	}
	if len(summary.RecentActivity) != 5 {
		t.Fatalf("expected 5 recent entries, got %d", len(summary.RecentActivity))
	}
	if summary.RecentActivity[0].ID != "rw-006" || summary.RecentActivity[4].ID != "rw-002" {
		t.Fatalf("unexpected recent ordering: %+v", summary.RecentActivity)
	}
}

func TestStageStatusWithStartedAt(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := setupTestProjectState(t, tmp, "release", "requirements")
	mustWrite(t, filepath.Join(tmp, ".github", "project-state.yml"), "project:\n  name: test\nlifecycle:\n  current_stage: release\n  started_at: \"2026-04-05\"\nphases:\n  current: requirements\n")
	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "stage", "status"})
	if err := root.Execute(); err != nil {
		t.Fatalf("stage status failed: %v", err)
	}
	if !strings.Contains(out.String(), "Started: 2026-04-05") {
		t.Fatalf("expected started_at output, got:\n%s", out.String())
	}
}

func TestStatusInvalidStateError(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".github", "project-state.yml"), ":\n  bad: [\n")
	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "status"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected parse error")
	}
}

type inputErrReader struct{}

func (inputErrReader) Read([]byte) (int, error) { return 0, errors.New("stdin failed") }

func TestAskForConfirmationEOFAndError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader("yes"))
	ok, err := askForConfirmation(cmd, "prompt")
	if err != nil || !ok {
		t.Fatalf("expected yes with EOF, ok=%v err=%v", ok, err)
	}

	cmd.SetIn(inputErrReader{})
	_, err = askForConfirmation(cmd, "prompt")
	if err == nil {
		t.Fatal("expected read error")
	}
}

func TestLoadStateOrNotProjectErrInvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".github", "project-state.yml"), ":\n  bad: [\n")

	_, err := loadStateOrNotProjectErr(cfgPath)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestLoadLifecycleStartedAtReadErrorAndNonMap(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	if _, err := loadLifecycleStartedAt(cfgPath); err == nil {
		t.Fatal("expected read error")
	}

	mustWrite(t, filepath.Join(tmp, ".github", "project-state.yml"), "lifecycle: plain-string\n")
	started, err := loadLifecycleStartedAt(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if started != "" {
		t.Fatalf("expected empty started_at, got %q", started)
	}
}

func TestStageAdvancePromptYes(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := setupTestProjectState(t, tmp, "initial_development", "requirements")
	root := newRootCmd("test")
	root.SetIn(strings.NewReader("yes\n"))
	root.SetArgs([]string{"--config", cfgPath, "stage", "advance"})
	if err := root.Execute(); err != nil {
		t.Fatalf("stage advance with prompt yes failed: %v", err)
	}
}

func TestPhaseListNoProject(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "phase", "list"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected notProject error")
	}
}

func TestPhaseCompleteNoProject(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "phase", "complete"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected notProject error")
	}
}

func TestIncidentCreateListCloseParseError(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "incident-log.yml"), ":\n  bad: [\n")

	root := newRootCmd("test")
	root.SetArgs([]string{"incident", "create", "--title", "x", "--severity", "major", "--config", cfgPath})
	if err := root.Execute(); err == nil {
		t.Fatal("expected create parse error")
	}
	root = newRootCmd("test")
	root.SetArgs([]string{"incident", "list", "--config", cfgPath})
	if err := root.Execute(); err == nil {
		t.Fatal("expected list parse error")
	}
	root = newRootCmd("test")
	root.SetArgs([]string{"incident", "close", "--id", "inc-001", "--config", cfgPath})
	if err := root.Execute(); err == nil {
		t.Fatal("expected close parse error")
	}
}

func TestReworkCreateListParseError(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "rework-log.yml"), ":\n  bad: [\n")

	root := newRootCmd("test")
	root.SetArgs([]string{"rework", "create", "--group", "g", "--target-phase", "requirements", "--reason", "r", "--config", cfgPath})
	if err := root.Execute(); err == nil {
		t.Fatal("expected create parse error")
	}
	root = newRootCmd("test")
	root.SetArgs([]string{"rework", "list", "--config", cfgPath})
	if err := root.Execute(); err == nil {
		t.Fatal("expected list parse error")
	}
}

func TestScanMissingDocsShowsRequiredMissing(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".github", "project-state.yml"), "project:\n  name: test\n")
	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"scan", "--config", cfgPath})
	if err := root.Execute(); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if !strings.Contains(out.String(), "✗ docs/") {
		t.Fatalf("expected missing docs marker, got:\n%s", out.String())
	}
}

func TestStatusInvalidFormat(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := setupTestProjectState(t, tmp, "release", "testing")
	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "--format", "xml", "status"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected unsupported format error")
	}
}

func TestSummarizeDiscussionMarshalAndRequestErrors(t *testing.T) {
	oldMarshal := discussionJSONMarshal
	oldReq := discussionNewRequest
	discussionJSONMarshal = func(v any) ([]byte, error) {
		return nil, errors.New("marshal failed")
	}
	t.Cleanup(func() {
		discussionJSONMarshal = oldMarshal
		discussionNewRequest = oldReq
	})
	if _, err := summarizeDiscussionWithAnthropic("content", "key"); err == nil {
		t.Fatal("expected marshal error")
	}

	discussionJSONMarshal = oldMarshal
	discussionNewRequest = func(method, url string, body io.Reader) (*http.Request, error) {
		return nil, errors.New("request failed")
	}
	if _, err := summarizeDiscussionWithAnthropic("content", "key"); err == nil {
		t.Fatal("expected request error")
	}
}
