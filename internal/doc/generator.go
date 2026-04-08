package doc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/taka-sho/teraflow/internal/agent"
	"github.com/taka-sho/teraflow/internal/index"
	"gopkg.in/yaml.v3"
)

var fetchCommandContext = exec.CommandContext

// Generator builds CoDD documents from GitHub Discussions.
type Generator struct {
	provider    agent.Provider
	projectRoot string
	dryRun      bool
	maxTokensCfg int
}

func NewGenerator(provider agent.Provider, projectRoot string, dryRun bool) *Generator {
	return &Generator{
		provider:    provider,
		projectRoot: projectRoot,
		dryRun:      dryRun,
	}
}

func (g *Generator) maxTokens() int {
	if g.maxTokensCfg > 0 {
		return g.maxTokensCfg
	}
	return 8192
}

func (g *Generator) Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error) {
	if strings.TrimSpace(req.ProjectRoot) != "" {
		g.projectRoot = req.ProjectRoot
	}
	if req.DryRun {
		g.dryRun = true
	}
	if strings.TrimSpace(g.projectRoot) == "" {
		return nil, fmt.Errorf("projectRoot is required")
	}

	disc, err := g.fetchDiscussion(ctx, req.DiscussionID)
	if err != nil {
		return nil, err
	}

	structured, err := g.structurize(ctx, disc)
	if err != nil {
		return nil, err
	}

	doc := g.buildDocument(disc, structured)
	res := &GenerateResult{Document: doc}
	if g.dryRun {
		return res, nil
	}

	path, err := g.writeFile(doc, outputDir(req.OutputDir, structured))
	if err != nil {
		return nil, err
	}
	res.FilePath = path

	builder := index.NewBuilder(g.projectRoot)
	idx, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("index build failed: %w", err)
	}
	if err := builder.Save(idx); err != nil {
		return nil, fmt.Errorf("index save failed: %w", err)
	}
	res.IndexUpdate = true

	return res, nil
}

func (g *Generator) fetchDiscussion(ctx context.Context, discussionID string) (*DiscussionData, error) {
	number, err := strconv.Atoi(strings.TrimSpace(discussionID))
	if err != nil {
		return nil, fmt.Errorf("invalid discussion id %q: %w", discussionID, err)
	}

	owner, repo, err := g.resolveRepository()
	if err != nil {
		return nil, err
	}

	const query = `query($number: Int!, $owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
    discussion(number: $number) {
      title
      body
      createdAt
      labels(first: 10) { nodes { name } }
      comments(first: 50) {
        nodes {
          author { login }
          body
          createdAt
          isAnswer
          replies(first: 100) {
            nodes {
              author { login }
              body
              createdAt
            }
          }
        }
      }
    }
  }
}`

	cmd := fetchCommandContext(ctx, "gh", "api", "graphql",
		"-f", "query="+query,
		"-F", "number="+strconv.Itoa(number),
		"-f", "owner="+owner,
		"-f", "name="+repo,
	)
	cmd.Dir = g.projectRoot
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("fetch discussion failed: %s", msg)
	}

	var resp struct {
		Data struct {
			Repository struct {
				Discussion struct {
					Title     string `json:"title"`
					Body      string `json:"body"`
					CreatedAt string `json:"createdAt"`
					Labels    struct {
						Nodes []struct {
							Name string `json:"name"`
						} `json:"nodes"`
					} `json:"labels"`
					Comments struct {
						Nodes []struct {
							Author struct {
								Login string `json:"login"`
							} `json:"author"`
							Body      string `json:"body"`
							CreatedAt string `json:"createdAt"`
							IsAnswer  bool   `json:"isAnswer"`
							Replies   struct {
								Nodes []struct {
									Author struct {
										Login string `json:"login"`
									} `json:"author"`
									Body      string `json:"body"`
									CreatedAt string `json:"createdAt"`
								} `json:"nodes"`
							} `json:"replies"`
						} `json:"nodes"`
					} `json:"comments"`
				} `json:"discussion"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parse discussion response: %w", err)
	}

	d := resp.Data.Repository.Discussion
	if strings.TrimSpace(d.Title) == "" {
		return nil, fmt.Errorf("discussion %d not found", number)
	}

	labels := make([]string, 0, len(d.Labels.Nodes))
	for _, label := range d.Labels.Nodes {
		if strings.TrimSpace(label.Name) != "" {
			labels = append(labels, label.Name)
		}
	}
	comments := make([]Comment, 0, len(d.Comments.Nodes))
	for _, c := range d.Comments.Nodes {
		replies := make([]Reply, 0, len(c.Replies.Nodes))
		for _, r := range c.Replies.Nodes {
			replies = append(replies, Reply{
				Author:    r.Author.Login,
				Body:      r.Body,
				CreatedAt: r.CreatedAt,
			})
		}
		comments = append(comments, Comment{
			Author:    c.Author.Login,
			Body:      c.Body,
			CreatedAt: c.CreatedAt,
			IsAnswer:  c.IsAnswer,
			Replies:   replies,
		})
	}

	return &DiscussionData{
		Number:    number,
		Title:     d.Title,
		Body:      d.Body,
		Comments:  comments,
		Labels:    labels,
		CreatedAt: d.CreatedAt,
	}, nil
}

func (g *Generator) structurize(ctx context.Context, disc *DiscussionData) (map[string]interface{}, error) {
	if disc == nil {
		return nil, fmt.Errorf("discussion is nil")
	}

	out, _, err := g.provider.Complete(ctx,
		"You are a senior product manager. You structure GitHub Discussions into rich CoDD requirement JSON. Return JSON only without markdown fences.",
		g.buildStructurizePrompt(disc),
		g.maxTokens(),
	)
	if err != nil {
		return nil, fmt.Errorf("structurize failed: %w", err)
	}

	out = stripCodeFence(out)
	var structured map[string]interface{}
	if err := json.Unmarshal([]byte(out), &structured); err != nil {
		return nil, fmt.Errorf("parse structured json: %w", err)
	}
	return structured, nil
}

func (g *Generator) buildStructurizePrompt(disc *DiscussionData) string {
	var commentsBuf strings.Builder
	for _, c := range disc.Comments {
		fmt.Fprintf(&commentsBuf, "\n## %s (%s)\n%s\n", c.Author, c.CreatedAt, c.Body)
		for _, r := range c.Replies {
			fmt.Fprintf(&commentsBuf, "\n  ### Reply by %s (%s)\n  %s\n",
				r.Author, r.CreatedAt,
				strings.ReplaceAll(r.Body, "\n", "\n  "))
		}
	}
	labels := strings.Join(disc.Labels, ", ")
	return fmt.Sprintf(`次のGitHub DiscussionをCoDD要件定義文書として構造化し、JSONのみを出力せよ。

必要キー:
- node_id: snake_case の識別子
- title: 文書タイトル
- category: docs配下のサブディレクトリ名（例: "システム要件"）
- summary: 1〜3文の要約
- sections: [{ "heading": string, "body": string }] のオブジェクト配列。
  以下の見出しを必ず含めること:
  - 背景・課題
  - 目的・ゴール
  - スコープ
  - 機能要件
  - 非機能要件
  - 受入条件
  - 依存関係・前提条件
  - リスク・未確定事項
- depends_on: [string] 他CoDDへのnode_id参照
- status: "draft" / "review" / "confirmed" のいずれか

# Discussion
Title: %s
Labels: %s
CreatedAt: %s

## Body
%s

## 壁打ち履歴（コメントとリプライ）
%s

---
JSONのみを返せ。マークダウンコードフェンスは付けるな。
summary は短く、sections.body は詳細に書け。可能な限り壁打ちの発言を反映せよ。
`, disc.Title, labels, disc.CreatedAt, disc.Body, commentsBuf.String())
}

func (g *Generator) buildDocument(disc *DiscussionData, structured map[string]interface{}) *CoDDDocument {
	now := time.Now().UTC().Format(time.RFC3339)
	if disc != nil && strings.TrimSpace(disc.CreatedAt) != "" {
		now = disc.CreatedAt
	}

	nodeID := asString(structured["node_id"])
	if strings.TrimSpace(nodeID) == "" {
		nodeID = fmt.Sprintf("discussion:%d", disc.Number)
	}
	title := asString(structured["title"])
	if strings.TrimSpace(title) == "" {
		title = disc.Title
	}
	status := asString(structured["status"])
	if status == "" {
		status = "draft"
	}

	sections := parseSections(structured["sections"])
	summary := asString(structured["summary"])
	body := buildBody(summary, sections)

	return &CoDDDocument{
		NodeID:    nodeID,
		Title:     title,
		DependsOn: asStringSlice(structured["depends_on"]),
		Status:    status,
		Source:    fmt.Sprintf("discussion:#%d", disc.Number),
		CreatedAt: now,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Summary:   summary,
		Sections:  sections,
		Body:      body,
	}
}

func (g *Generator) writeFile(doc *CoDDDocument, outputDir string) (string, error) {
	if doc == nil {
		return "", fmt.Errorf("document is nil")
	}
	if strings.TrimSpace(outputDir) == "" {
		outputDir = filepath.Join(g.projectRoot, "docs")
	}
	if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(g.projectRoot, outputDir)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}

	type frontmatter struct {
		CoDD CoDDDocument `yaml:"codd"`
	}
	fm := frontmatter{CoDD: *doc}
	yamlBytes, err := yaml.Marshal(fm)
	if err != nil {
		return "", fmt.Errorf("marshal frontmatter: %w", err)
	}

	var content strings.Builder
	content.WriteString("---\n")
	content.Write(yamlBytes)
	content.WriteString("---\n\n")
	content.WriteString(doc.Body)
	content.WriteString("\n")

	filename := doc.NodeID + ".md"
	path := filepath.Join(outputDir, filename)
	if err := os.WriteFile(path, []byte(content.String()), 0o644); err != nil {
		return "", fmt.Errorf("write document: %w", err)
	}
	return path, nil
}

func (g *Generator) resolveRepository() (string, string, error) {
	cmd := fetchCommandContext(context.Background(), "git", "remote", "get-url", "origin")
	cmd.Dir = g.projectRoot
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", "", fmt.Errorf("resolve repository failed: %s", msg)
	}
	remote := strings.TrimSpace(string(out))
	return parseGitHubRepository(remote)
}

func parseGitHubRepository(remote string) (string, string, error) {
	normalized := strings.TrimSpace(remote)
	if normalized == "" {
		return "", "", fmt.Errorf("empty remote url")
	}

	normalized = strings.TrimPrefix(normalized, "https://github.com/")
	normalized = strings.TrimPrefix(normalized, "http://github.com/")
	normalized = strings.TrimPrefix(normalized, "ssh://git@github.com/")
	normalized = strings.TrimPrefix(normalized, "git@github.com:")
	normalized = strings.TrimSuffix(normalized, ".git")
	normalized = strings.Trim(normalized, "/")

	parts := strings.Split(normalized, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid repository url: %s", remote)
	}
	return parts[0], parts[1], nil
}

func outputDir(reqOutput string, structured map[string]interface{}) string {
	if strings.TrimSpace(reqOutput) != "" {
		return reqOutput
	}
	category := asString(structured["category"])
	if strings.TrimSpace(category) == "" {
		return "docs"
	}
	return filepath.Join("docs", category)
}

func buildBody(summary string, sections []DocSection) string {
	var b strings.Builder
	if strings.TrimSpace(summary) != "" {
		b.WriteString("# 概要\n\n")
		b.WriteString(strings.TrimSpace(summary))
		b.WriteString("\n\n")
	}
	for _, s := range sections {
		heading := strings.TrimSpace(s.Heading)
		body := strings.TrimSpace(s.Body)
		if heading == "" && body == "" {
			continue
		}
		if heading == "" {
			heading = "詳細"
		}
		fmt.Fprintf(&b, "## %s\n\n", heading)
		if body != "" {
			b.WriteString(body)
			b.WriteString("\n\n")
		}
	}
	if b.Len() == 0 {
		b.WriteString("> （構造化に失敗しました。Discussion 元投稿を確認してください）\n")
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

func parseSections(raw interface{}) []DocSection {
	arr, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	result := make([]DocSection, 0, len(arr))
	for _, item := range arr {
		switch v := item.(type) {
		case map[string]interface{}:
			result = append(result, DocSection{
				Heading: asString(v["heading"]),
				Body:    asString(v["body"]),
			})
		case string:
			if strings.TrimSpace(v) != "" {
				result = append(result, DocSection{Heading: strings.TrimSpace(v)})
			}
		}
	}
	return result
}

func asString(v interface{}) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func asStringSlice(v interface{}) []string {
	raw, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		s, ok := item.(string)
		if ok && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}

func stripCodeFence(s string) string {
	t := strings.TrimSpace(s)
	if strings.HasPrefix(t, "```") {
		t = strings.TrimPrefix(t, "```")
		t = strings.TrimSpace(t)
		if strings.HasPrefix(strings.ToLower(t), "json") {
			t = strings.TrimSpace(t[4:])
		}
		if idx := strings.LastIndex(t, "```"); idx >= 0 {
			t = strings.TrimSpace(t[:idx])
		}
	}
	return t
}
