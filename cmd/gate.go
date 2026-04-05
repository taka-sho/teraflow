package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/audit"
	"github.com/taka-sho/teraflow/internal/config"
	"github.com/taka-sho/teraflow/internal/gate"
	"github.com/taka-sho/teraflow/internal/rbac"
	"github.com/taka-sho/teraflow/internal/state"
	"gopkg.in/yaml.v3"
)

func newGateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gate",
		Short: "Manage process gates",
	}
	cmd.AddCommand(newGateApproveCmd())
	return cmd
}

func newGateApproveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "approve <process_name>",
		Short: "Approve a gate for the given process",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			processName := args[0]
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			currentUser, _ := rbac.CurrentUser()
			if cfg.RBAC.Enabled {
				if currentUser == "" {
					return errors.New("rbac enabled but current user could not be determined")
				}
				engine := rbac.NewEngine(cfg.RBAC)
				required := "gate.approve." + processName
				if !engine.CheckPermission(currentUser, required) {
					return fmt.Errorf("permission denied: missing %s", required)
				}
			}

			rule, err := loadGateRule(configPath, processName)
			if err != nil {
				return err
			}
			projectPath := filepath.Dir(filepath.Dir(configPath))

			ok, messages, err := gate.Evaluate(rule, projectPath)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("gate conditions not met: %s", strings.Join(messages, "; "))
			}

			s, err := state.LoadState(configPath)
			if err != nil {
				return err
			}
			if err := markSLCPProcessCompleted(s, processName); err != nil {
				return err
			}
			if err := state.SaveState(configPath, s); err != nil {
				return err
			}

			if err := audit.Append(configPath, audit.AuditEntry{
				User:   currentUser,
				Action: "gate.approve",
				Target: processName,
				Result: "approved",
			}); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Gate approved: %s\n", processName)
			return nil
		},
	}
}

type gateRuleFile struct {
	GateRules map[string]gateRuleConfig `yaml:"gate_rules"`
}

type gateRuleConfig struct {
	ApproverRole string                `yaml:"approver_role"`
	Conditions   []gateConditionConfig `yaml:"conditions"`
}

type gateConditionConfig struct {
	Type      gate.ConditionType `yaml:"type"`
	Value     string             `yaml:"value,omitempty"`
	Path      string             `yaml:"path,omitempty"`
	Threshold float64            `yaml:"threshold,omitempty"`
}

func loadGateRule(configPath, processName string) (gate.GateRule, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return gate.GateRule{}, fmt.Errorf("read config: %w", err)
	}

	var raw gateRuleFile
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return gate.GateRule{}, fmt.Errorf("parse config for gate rules: %w", err)
	}

	ruleCfg, ok := raw.GateRules[processName]
	if !ok {
		return gate.GateRule{}, fmt.Errorf("gate rule not found for process: %s", processName)
	}

	rule := gate.GateRule{
		ProcessName:  processName,
		ApproverRole: ruleCfg.ApproverRole,
		Conditions:   make([]gate.Condition, 0, len(ruleCfg.Conditions)),
	}

	for _, c := range ruleCfg.Conditions {
		value := strings.TrimSpace(c.Value)
		if value == "" && strings.TrimSpace(c.Path) != "" {
			value = c.Path
		}
		if value == "" && c.Threshold != 0 {
			value = strconv.FormatFloat(c.Threshold, 'f', -1, 64)
		}
		rule.Conditions = append(rule.Conditions, gate.Condition{Type: c.Type, Value: value})
	}

	return rule, nil
}

func markSLCPProcessCompleted(s *state.ProjectState, processName string) error {
	targets := slcpTargets(processName)
	now := time.Now().Format(time.RFC3339)

	for i := range s.SLCPJCF.Processes {
		name := s.SLCPJCF.Processes[i].Name
		if !containsString(targets, name) {
			continue
		}
		s.SLCPJCF.Processes[i].Status = "completed"
		s.SLCPJCF.Processes[i].CompletedAt = now
		s.SLCPJCF.CurrentProcess = name
		s.SLCPJCF.UpdatedAt = now
		return nil
	}

	return fmt.Errorf("slcp_jcf process not found for: %s", processName)
}

func slcpTargets(processName string) []string {
	m := map[string][]string{
		"planning":         {"企画プロセス", "企画"},
		"requirements":     {"要件定義プロセス", "要件定義"},
		"basic_design":     {"システム設計プロセス", "システム設計"},
		"detailed_design":  {"ソフトウェア設計プロセス", "ソフトウェア設計"},
		"implementation":   {"ソフトウェア構築プロセス", "ソフトウェア構築", "構築"},
		"testing":          {"ソフトウェアテストプロセス", "ソフトウェアテスト", "テスト"},
		"integration_test": {"システム結合テスト"},
		"operation":        {"運用・保守プロセス", "運用保守"},
	}
	if targets, ok := m[processName]; ok {
		return targets
	}
	return []string{processName}
}

func containsString(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
