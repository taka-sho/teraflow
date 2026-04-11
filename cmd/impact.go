package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/graphbridge"
	indexpkg "github.com/taka-sho/teraflow/internal/index"
	"github.com/taka-sho/teraflow/internal/pipeline"
)

var impactExecCommand = exec.Command
var impactLookPath = exec.LookPath

type impactApplyIssue struct {
	NodeID  string `json:"node_id"`
	Band    string `json:"band"`
	Title   string `json:"title"`
	URL     string `json:"url,omitempty"`
	Created bool   `json:"created"`
}

type impactApplyResult struct {
	Created []impactApplyIssue `json:"created"`
	Skipped []impactApplyIssue `json:"skipped,omitempty"`
}

func newImpactCmd() *cobra.Command {
	var depth int
	var nodeID string
	var applyPath string
	var output string

	cmd := &cobra.Command{
		Use:   "impact [node-id]",
		Short: "Analyze downstream impact from a changed CoDD node",
		RunE: func(cmd *cobra.Command, args []string) error {
			if output != "" {
				switch output {
				case "json", "text":
					_ = cmd.Root().PersistentFlags().Set("format", output)
				default:
					return fmt.Errorf("unsupported --output format: %s", output)
				}
			}

			if applyPath != "" {
				if len(args) > 0 || nodeID != "" {
					return fmt.Errorf("--apply cannot be combined with --node or positional node-id")
				}
				return runImpactApply(cmd, applyPath)
			}

			if len(args) > 1 {
				return fmt.Errorf("accepts at most 1 arg(s), received %d", len(args))
			}
			if nodeID == "" && len(args) == 1 {
				nodeID = args[0]
			}
			if strings.TrimSpace(nodeID) == "" {
				return fmt.Errorf("--node or positional node-id is required")
			}
			if depth < 1 {
				return fmt.Errorf("--depth must be >= 1")
			}

			projectRoot, err := projectRootFromCmd(cmd)
			if err != nil {
				return err
			}
			idx, err := indexpkg.NewBuilder(projectRoot).LoadIndex()
			if err != nil {
				return err
			}

			analyzer := pipeline.NewImpactAnalyzer(idx, graphbridge.New(projectRoot), projectRoot)
			result, err := analyzer.Analyze(context.Background(), nodeID, depth)
			if err != nil {
				return err
			}

			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			if format == "json" {
				return writeJSON(cmd, result)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Changed: %s\n", result.ChangedNode)
			if result.ChangeType != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Change type: %s\n", result.ChangeType)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Affected: %d (gray=%d amber=%d green=%d)\n", result.Summary["total"], result.Summary["gray"], result.Summary["amber"], result.Summary["green"])
			for _, node := range result.AffectedNodes {
				fmt.Fprintf(cmd.OutOrStdout(), "- %s band=%s distance=%d source=%s reason=%s\n", node.NodeID, colorBand(node.Band), node.Distance, node.Source, node.Reason)
			}
			for _, w := range result.Warnings {
				fmt.Fprintf(cmd.OutOrStdout(), "Warning: %s\n", w)
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&depth, "depth", 2, "Traversal depth")
	cmd.Flags().StringVar(&nodeID, "node", "", "Changed node id")
	cmd.Flags().StringVar(&applyPath, "apply", "", "Apply impact analysis results from JSON/NDJSON and create follow-up issues")
	cmd.Flags().StringVar(&output, "output", "", "Output format override: text|json")
	return cmd
}

func colorBand(band string) string {
	switch strings.ToLower(strings.TrimSpace(band)) {
	case "gray":
		return "\x1b[90mgray\x1b[0m"
	case "amber":
		return "\x1b[33mamber\x1b[0m"
	case "green":
		return "\x1b[32mgreen\x1b[0m"
	default:
		return band
	}
}

func runImpactApply(cmd *cobra.Command, path string) error {
	projectRoot, err := projectRootFromCmd(cmd)
	if err != nil {
		return err
	}
	results, err := loadImpactResults(path)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return fmt.Errorf("no impact results found in %s", path)
	}
	if _, err := impactLookPath("gh"); err != nil {
		return fmt.Errorf("gh CLI not found: %w", err)
	}

	out := impactApplyResult{
		Created: make([]impactApplyIssue, 0),
		Skipped: make([]impactApplyIssue, 0),
	}
	seen := map[string]struct{}{}

	for _, res := range results {
		for _, node := range res.ReviewNeeded {
			key := "amber:" + node
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			title := fmt.Sprintf("Review impact on %s", node)
			body := generateAmberReviewBody(res, node)
			url, err := createIssue(projectRoot, title, body, []string{"impact", "amber"})
			if err != nil {
				return fmt.Errorf("create amber issue for %s: %w", node, err)
			}
			out.Created = append(out.Created, impactApplyIssue{
				NodeID:  node,
				Band:    "amber",
				Title:   title,
				URL:     url,
				Created: true,
			})
		}

		for _, node := range res.RegenRequired {
			key := "gray:" + node
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			title := fmt.Sprintf("Regenerate artifact for %s", node)
			body := generateGrayRegenBody(res, node)
			url, err := createIssue(projectRoot, title, body, []string{"impact", "gray", "regeneration"})
			if err != nil {
				return fmt.Errorf("create gray issue for %s: %w", node, err)
			}
			out.Created = append(out.Created, impactApplyIssue{
				NodeID:  node,
				Band:    "gray",
				Title:   title,
				URL:     url,
				Created: true,
			})
		}
	}

	format, err := outputFormatFromCmd(cmd)
	if err != nil {
		return err
	}
	if format == "json" {
		return writeJSON(cmd, out)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Created %d issues\n", len(out.Created))
	for _, item := range out.Created {
		fmt.Fprintf(cmd.OutOrStdout(), "- [%s] %s", item.Band, item.Title)
		if item.URL != "" {
			fmt.Fprintf(cmd.OutOrStdout(), " -> %s", item.URL)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}
	return nil
}

func loadImpactResults(path string) ([]pipeline.ImpactResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, nil
	}

	var array []pipeline.ImpactResult
	if err := json.Unmarshal(data, &array); err == nil {
		return array, nil
	}
	var single pipeline.ImpactResult
	if err := json.Unmarshal(data, &single); err == nil && single.ChangedNode != "" {
		return []pipeline.ImpactResult{single}, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	out := make([]pipeline.ImpactResult, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var item pipeline.ImpactResult
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, fmt.Errorf("invalid NDJSON line: %w", err)
		}
		if item.ChangedNode == "" {
			continue
		}
		out = append(out, item)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func createIssue(projectRoot, title, body string, labels []string) (string, error) {
	args := []string{"issue", "create", "--title", title, "--body", body}
	for _, label := range labels {
		if label != "" {
			args = append(args, "--label", label)
		}
	}

	cmd := impactExecCommand("gh", args...)
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		if slices.Contains(labels, "amber") || slices.Contains(labels, "gray") || slices.Contains(labels, "regeneration") || slices.Contains(labels, "impact") {
			cmd = impactExecCommand("gh", "issue", "create", "--title", title, "--body", body)
			cmd.Dir = projectRoot
			output, err = cmd.CombinedOutput()
			if err != nil {
				msg = strings.TrimSpace(string(output))
				if msg == "" {
					msg = err.Error()
				}
				return "", fmt.Errorf("gh issue create failed: %s", msg)
			}
			return strings.TrimSpace(string(output)), nil
		}
		return "", fmt.Errorf("gh issue create failed: %s", msg)
	}
	return strings.TrimSpace(string(output)), nil
}

func generateAmberReviewBody(result pipeline.ImpactResult, nodeID string) string {
	return fmt.Sprintf(
		"## Impact Review Required\n\n"+
			"**Changed**: %q\n"+
			"**Affected**: %q (Band: Amber)\n\n"+
			"### 影響サマリー（自動生成）\n"+
			"- 変更ノードからの依存チェーン経由で影響を検出\n"+
			"- 推定影響度: 中（限定範囲の追従修正を推奨）\n"+
			"- 重大な破綻は未検出だが、人手レビューが必要\n\n"+
			"### 推奨アクション\n"+
			"- [ ] %s の本文を変更内容に追従更新する\n"+
			"- [ ] 関連する verifies / verified_by の整合性を再確認する\n"+
			"- [ ] 関連テスト仕様を更新し、`teraflow validate --full` を再実行する\n",
		result.ChangedNode, nodeID, nodeID,
	)
}

func generateGrayRegenBody(result pipeline.ImpactResult, nodeID string) string {
	return fmt.Sprintf(
		"## Regeneration Required\n\n"+
			"**Changed**: %q\n"+
			"**Affected**: %q (Band: Gray)\n\n"+
			"### 判定理由\n"+
			"- 依存距離と変更種別から高影響と判定\n"+
			"- 下流成果物の整合性破綻リスクが高い\n\n"+
			"### 対応\n"+
			"- [ ] 対象成果物を再生成する\n"+
			"- [ ] 再生成後に `teraflow validate --full` を実行する\n"+
			"- [ ] 必要に応じて関連ノードにも伝播確認を行う\n",
		result.ChangedNode, nodeID,
	)
}
