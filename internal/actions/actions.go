package actions

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed templates/*.yml
var templateFS embed.FS

// WorkflowNames は生成する標準ワークフロー名
var WorkflowNames = []string{
	"teraflow-phase-transition",
	"teraflow-phase-gate",
	"teraflow-permission-guard",
	"teraflow-req-agent",
	"teraflow-artifact-finalize",
	"teraflow-changelog-update",
	"teraflow-coverage",
	"teraflow-rbac",
	"teraflow-implement-agent",
	"teraflow-ci-fix-agent",
	"teraflow-review-agent",
	"teraflow-conflict-agent",
	"teraflow-stuck-monitor",
	"teraflow-rework-impact",
	"teraflow-dashboard-deploy",
	"teraflow-incident-agent",
	"teraflow-maintenance-agent",
	"teraflow-schedule-predict",
	"teraflow-push",
	"teraflow-validate",
}

// GenerateWorkflows はワークフローYAMLを targetDir に生成する
func GenerateWorkflows(targetDir string) error {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("create workflows directory: %w", err)
	}

	return fs.WalkDir(templateFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		data, err := templateFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read template %s: %w", path, err)
		}

		dest := filepath.Join(targetDir, d.Name())
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return fmt.Errorf("write workflow %s: %w", dest, err)
		}

		return nil
	})
}

// ListTemplates は利用可能なテンプレート名の一覧を返す
func ListTemplates() ([]string, error) {
	entries, err := templateFS.ReadDir("templates")
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}
