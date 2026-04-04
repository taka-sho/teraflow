package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// ChangelogEntry is a single JSONL changelog record.
type ChangelogEntry struct {
	Type      string `json:"type"` // feat | fix | chore | docs | refactor
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

type changelogGenerateOptions struct {
	Output string
}

func newChangelogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "changelog",
		Short: "Manage changelog entries",
	}

	cmd.AddCommand(newChangelogAddCmd())
	cmd.AddCommand(newChangelogGenerateCmd())
	return cmd
}

func newChangelogAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <type> <message>",
		Short: "Append a changelog entry",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			entryType := strings.TrimSpace(args[0])
			switch entryType {
			case "feat", "fix", "chore", "docs", "refactor":
			default:
				return fmt.Errorf("type must be one of: feat, fix, chore, docs, refactor")
			}

			message := strings.TrimSpace(strings.Join(args[1:], " "))
			if message == "" {
				return fmt.Errorf("message is required")
			}

			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}

			dir := changelogDirPath(configPath)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("create changelog directory: %w", err)
			}

			now := time.Now()
			entry := ChangelogEntry{
				Type:      entryType,
				Message:   message,
				Timestamp: now.Format(time.RFC3339),
			}
			payload, err := json.Marshal(entry)
			if err != nil {
				return fmt.Errorf("marshal changelog entry: %w", err)
			}

			path := filepath.Join(dir, now.Format("2006-01")+".jsonl")
			file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err != nil {
				return fmt.Errorf("open changelog file: %w", err)
			}
			defer func() { _ = file.Close() }()

			if _, err := file.Write(append(payload, '\n')); err != nil {
				return fmt.Errorf("append changelog entry: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Changelog entry added: [%s] %s\n", entryType, message)
			return nil
		},
	}

	return cmd
}

func newChangelogGenerateCmd() *cobra.Command {
	opts := &changelogGenerateOptions{}
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate release notes from changelog JSONL files",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			dir := changelogDirPath(configPath)

			files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
			if err != nil {
				return fmt.Errorf("glob changelog files: %w", err)
			}
			sort.Strings(files)

			entriesByType := map[string][]ChangelogEntry{
				"feat":     {},
				"fix":      {},
				"chore":    {},
				"docs":     {},
				"refactor": {},
			}

			for _, filePath := range files {
				file, openErr := os.Open(filePath)
				if openErr != nil {
					return fmt.Errorf("open changelog file: %w", openErr)
				}

				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					line := strings.TrimSpace(scanner.Text())
					if line == "" {
						continue
					}

					var entry ChangelogEntry
					if err := json.Unmarshal([]byte(line), &entry); err != nil {
						_ = file.Close()
						return fmt.Errorf("parse changelog entry in %s: %w", filePath, err)
					}
					entriesByType[entry.Type] = append(entriesByType[entry.Type], entry)
				}
				if err := scanner.Err(); err != nil {
					_ = file.Close()
					return fmt.Errorf("scan changelog file: %w", err)
				}
				_ = file.Close()
			}

			notes := buildReleaseNotes(entriesByType)
			if strings.TrimSpace(opts.Output) == "" {
				fmt.Fprint(cmd.OutOrStdout(), notes)
				return nil
			}

			if err := os.MkdirAll(filepath.Dir(opts.Output), 0o755); err != nil {
				return fmt.Errorf("create output directory: %w", err)
			}
			if err := os.WriteFile(opts.Output, []byte(notes), 0o644); err != nil {
				return fmt.Errorf("write release notes: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Release notes generated: %s\n", opts.Output)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Output, "output", "", "Output file path")
	return cmd
}

func changelogDirPath(configPath string) string {
	root := filepath.Dir(filepath.Dir(configPath))
	return filepath.Join(root, ".teraflow", "changelog")
}

func buildReleaseNotes(entriesByType map[string][]ChangelogEntry) string {
	type sectionsDef struct {
		Type  string
		Title string
	}

	sections := []sectionsDef{
		{Type: "feat", Title: "Features"},
		{Type: "fix", Title: "Bug Fixes"},
		{Type: "chore", Title: "Chores"},
		{Type: "docs", Title: "Documentation"},
		{Type: "refactor", Title: "Refactoring"},
	}

	var b strings.Builder
	b.WriteString("# Release Notes\n\n")

	for _, sec := range sections {
		entries := entriesByType[sec.Type]
		if len(entries) == 0 {
			continue
		}
		b.WriteString("## ")
		b.WriteString(sec.Title)
		b.WriteString("\n")
		for _, entry := range entries {
			date := entry.Timestamp
			if t, err := time.Parse(time.RFC3339, entry.Timestamp); err == nil {
				date = t.Format("2006-01-02")
			} else if len(date) >= 10 {
				date = date[:10]
			}
			b.WriteString("- ")
			b.WriteString(entry.Message)
			b.WriteString(" (")
			b.WriteString(date)
			b.WriteString(")\n")
		}
		b.WriteString("\n")
	}

	if b.String() == "# Release Notes\n\n" {
		b.WriteString("_No changelog entries found._\n")
	}

	return b.String()
}
