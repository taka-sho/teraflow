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
}

func NewGenerator(provider agent.Provider, projectRoot string, dryRun bool) *Generator {
	return &Generator{
		provider:    provider,
		projectRoot: projectRoot,
		dryRun:      dryRun,
	}
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
      comments(first: 100) {
        nodes {
          author { login }
          body
          createdAt
          isAnswer
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
		comments = append(comments, Comment{
			Author:    c.Author.Login,
			Body:      c.Body,
			CreatedAt: c.CreatedAt,
			IsAnswer:  c.IsAnswer,
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

	var b strings.Builder
	b.WriteString("次のGitHub DiscussionをCoDD文書として構造化し、JSONのみを出力せよ。\\n")
	b.WriteString("必要キー: node_id, title, category, summary, sections, depends_on, status\\n")
	b.WriteString("statusは draft/review/confirmed のいずれか。\\n\\n")
	b.WriteString("# Discussion\\n")
	b.WriteString("Title: ")
	b.WriteString(disc.Title)
	b.WriteString("\\n\\n")
	b.WriteString(disc.Body)
	b.WriteString("\\n\\n# Comments\\n")
	for _, c := range disc.Comments {
		b.WriteString("- ")
		b.WriteString(c.Author)
		b.WriteString(": ")
		b.WriteString(c.Body)
		b.WriteString("\\n")
	}

	out, _, err := g.provider.Complete(ctx,
		"You structure GitHub discussions into CoDD JSON. Return JSON only.",
		b.String(),
		2048,
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

	sections := asStringSlice(structured["sections"])
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

func buildBody(summary string, sections []string) string {
	var b strings.Builder
	if strings.TrimSpace(summary) != "" {
		b.WriteString("# Summary\n\n")
		b.WriteString(strings.TrimSpace(summary))
		b.WriteString("\n\n")
	}
	if len(sections) > 0 {
		b.WriteString("# Sections\n\n")
		for _, s := range sections {
			if strings.TrimSpace(s) == "" {
				continue
			}
			b.WriteString("- ")
			b.WriteString(strings.TrimSpace(s))
			b.WriteString("\n")
		}
	}
	if strings.TrimSpace(b.String()) == "" {
		return "# Summary\n\nGenerated from discussion."
	}
	return strings.TrimSpace(b.String())
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
