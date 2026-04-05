package audit

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/taka-sho/teraflow/internal/rbac"
	"gopkg.in/yaml.v3"
)

// AuditEntry is a single audit log record.
type AuditEntry struct {
	Timestamp string `yaml:"timestamp"`
	User      string `yaml:"user"`
	Action    string `yaml:"action"`
	Target    string `yaml:"target"`
	Result    string `yaml:"result"`
	Reason    string `yaml:"reason,omitempty"`
}

// AuditLog stores audit entries in YAML format.
type AuditLog struct {
	Entries []AuditEntry `yaml:"entries"`
}

const AuditLogPath = ".teraflow/audit-log.yml"

// AppendAudit appends an entry to the audit log.
func AppendAudit(projectPath string, entry AuditEntry) error {
	if projectPath == "" {
		return errors.New("project path is required")
	}

	logData, err := LoadAuditLog(projectPath)
	if err != nil {
		return err
	}
	logData.Entries = append(logData.Entries, entry)

	logPath := filepath.Join(projectPath, AuditLogPath)
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(logData)
	if err != nil {
		return err
	}
	return os.WriteFile(logPath, data, 0o644)
}

// LoadAuditLog loads the audit log from project directory.
func LoadAuditLog(projectPath string) (*AuditLog, error) {
	if projectPath == "" {
		return nil, errors.New("project path is required")
	}

	logPath := filepath.Join(projectPath, AuditLogPath)
	data, err := os.ReadFile(logPath)
	if errors.Is(err, os.ErrNotExist) {
		return &AuditLog{Entries: []AuditEntry{}}, nil
	}
	if err != nil {
		return nil, err
	}

	var logData AuditLog
	if err := yaml.Unmarshal(data, &logData); err != nil {
		return nil, err
	}
	if logData.Entries == nil {
		logData.Entries = []AuditEntry{}
	}
	return &logData, nil
}

// NewEntry creates a new AuditEntry with current timestamp and user.
func NewEntry(action, target, result string) AuditEntry {
	user, _ := rbac.CurrentUser()
	return AuditEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		User:      user,
		Action:    action,
		Target:    target,
		Result:    result,
	}
}

// Append appends an entry using config path for backward compatibility.
func Append(configPath string, entry AuditEntry) error {
	projectPath := filepath.Dir(filepath.Dir(configPath))
	return AppendAudit(projectPath, entry)
}
