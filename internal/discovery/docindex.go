package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DocIndex is a static index of project documentation files.
type DocIndex struct {
	Version     string     `yaml:"version"`
	GeneratedAt string     `yaml:"generated_at"`
	Generator   string     `yaml:"generator"`
	Docs        []DocEntry `yaml:"docs"`
}

// DocEntry describes one document in docs/.
type DocEntry struct {
	Path        string       `yaml:"path"`
	Title       string       `yaml:"title"`
	Summary     string       `yaml:"summary"`
	Categories  []string     `yaml:"categories"`
	Keywords    []string     `yaml:"keywords"`
	Sections    []DocSection `yaml:"sections"`
	TotalTokens int          `yaml:"total_tokens"`
	CoddNodeID  string       `yaml:"codd_node_id,omitempty"`
}

// DocSection maps a section heading to source lines and estimated token usage.
type DocSection struct {
	Heading       string `yaml:"heading"`
	LineStart     int    `yaml:"line_start"`
	LineEnd       int    `yaml:"line_end"`
	TokenEstimate int    `yaml:"token_estimate"`
}

// DocChunk is a selected section body loaded from source docs.
type DocChunk struct {
	Path    string
	Section string
	Content string
	Tokens  int
}

// LoadDocIndex reads doc-index.yaml and returns a parsed DocIndex.
func LoadDocIndex(path string) (*DocIndex, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read doc index %s: %w", path, err)
	}
	var idx DocIndex
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parse doc index %s: %w", path, err)
	}
	if strings.TrimSpace(idx.Version) == "" {
		idx.Version = "1"
	}
	return &idx, nil
}

// GenerateDocIndex scans docsDir and creates a static documentation index.
// llm is currently reserved for future Phase 2 summary enrichment.
func GenerateDocIndex(ctx context.Context, docsDir string, _ LLMGenerator) (*DocIndex, error) {
	root := strings.TrimSpace(docsDir)
	if root == "" {
		return nil, fmt.Errorf("docs directory is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve docs directory: %w", err)
	}

	docs := make([]DocEntry, 0, 64)
	err = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() || strings.ToLower(filepath.Ext(path)) != ".md" {
			return nil
		}

		entry, err := buildDocEntry(absRoot, path)
		if err != nil {
			return fmt.Errorf("build doc entry %s: %w", path, err)
		}
		docs = append(docs, entry)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].Path < docs[j].Path
	})

	return &DocIndex{
		Version:     "1",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Generator:   "teraflow doc index",
		Docs:        docs,
	}, nil
}

// SelectRelevantDocs picks documents relevant to the latest discussion context.
// Signature keeps task-level compatibility while using deterministic matching fallback.
func SelectRelevantDocs(ctx context.Context, index DocIndex, discussion string) ([]DocEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(index.Docs) == 0 {
		return nil, nil
	}

	tokens := tokenizeQuery(discussion)
	type scored struct {
		doc   DocEntry
		score int
	}
	scoredDocs := make([]scored, 0, len(index.Docs))
	for _, doc := range index.Docs {
		score := scoreDocEntry(doc, tokens)
		scoredDocs = append(scoredDocs, scored{doc: doc, score: score})
	}

	sort.SliceStable(scoredDocs, func(i, j int) bool {
		if scoredDocs[i].score != scoredDocs[j].score {
			return scoredDocs[i].score > scoredDocs[j].score
		}
		if scoredDocs[i].doc.TotalTokens != scoredDocs[j].doc.TotalTokens {
			return scoredDocs[i].doc.TotalTokens < scoredDocs[j].doc.TotalTokens
		}
		return scoredDocs[i].doc.Path < scoredDocs[j].doc.Path
	})

	selected := make([]DocEntry, 0, 5)
	for _, item := range scoredDocs {
		if len(selected) >= 5 {
			break
		}
		if len(tokens) > 0 && item.score <= 0 {
			continue
		}
		selected = append(selected, item.doc)
	}

	if len(selected) > 0 {
		return selected, nil
	}

	fallbackCount := min(3, len(scoredDocs))
	for i := 0; i < fallbackCount; i++ {
		selected = append(selected, scoredDocs[i].doc)
	}
	return selected, nil
}

// ReadDocChunks reads sections from selected docs under token budget.
// Budget is capped at 2000 tokens and applies graceful degradation by skipping
// sections that would exceed the remaining budget.
func ReadDocChunks(entries []DocEntry, tokenBudget int) ([]DocChunk, error) {
	if tokenBudget <= 0 {
		return nil, fmt.Errorf("token budget must be > 0")
	}
	if tokenBudget > 2000 {
		tokenBudget = 2000
	}

	chunks := make([]DocChunk, 0, 16)
	remaining := tokenBudget
	for _, entry := range entries {
		if remaining < 100 {
			break
		}

		path, ok := resolveDocPath(entry.Path)
		if !ok {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")

		sections := entry.Sections
		if len(sections) == 0 {
			sections = []DocSection{{
				Heading:       "Document",
				LineStart:     1,
				LineEnd:       len(lines),
				TokenEstimate: estimateTokens(strings.Join(lines, "\n")),
			}}
		}

		for _, section := range sections {
			if remaining < 100 {
				break
			}
			body := readSection(lines, section.LineStart, section.LineEnd)
			if strings.TrimSpace(body) == "" {
				continue
			}
			tokens := section.TokenEstimate
			if tokens <= 0 {
				tokens = estimateTokens(body)
			}
			if tokens > remaining {
				continue
			}

			heading := strings.TrimSpace(section.Heading)
			if heading == "" {
				heading = "Document"
			}
			chunks = append(chunks, DocChunk{
				Path:    filepath.ToSlash(entry.Path),
				Section: heading,
				Content: body,
				Tokens:  tokens,
			})
			remaining -= tokens
		}
	}
	return chunks, nil
}

// FormatDocContext converts chunks into prompt-ready markdown.
func FormatDocContext(chunks []DocChunk) string {
	if len(chunks) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## 既存ドキュメント参照\n\n")
	for i, chunk := range chunks {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("### ")
		b.WriteString(strings.TrimSpace(chunk.Path))
		b.WriteString(" / ")
		b.WriteString(strings.TrimSpace(chunk.Section))
		b.WriteString("\n")
		b.WriteString(strings.TrimSpace(chunk.Content))
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func buildDocEntry(root, path string) (DocEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DocEntry{}, err
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")

	meta := parseDocFrontmatter(lines)
	bodyStart := meta.bodyStart
	if bodyStart < 1 {
		bodyStart = 1
	}

	title := strings.TrimSpace(meta.title)
	if title == "" {
		title = firstHeading(lines[bodyStart-1:], "# ")
	}
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	sections := collectSections(lines, bodyStart)
	keywords := collectKeywords(sections, meta.tags)
	categories := inferCategories(meta.tags, keywords, path)

	summary := buildSummary(lines, bodyStart)
	if summary == "" {
		summary = title
	}

	body := strings.Join(lines[bodyStart-1:], "\n")
	totalTokens := estimateTokens(body)

	rel, err := filepath.Rel(root, path)
	if err != nil {
		return DocEntry{}, err
	}

	return DocEntry{
		Path:        filepath.ToSlash(rel),
		Title:       title,
		Summary:     summary,
		Categories:  categories,
		Keywords:    keywords,
		Sections:    sections,
		TotalTokens: totalTokens,
		CoddNodeID:  meta.nodeID,
	}, nil
}

type docMeta struct {
	nodeID    string
	title     string
	tags      []string
	bodyStart int
}

func parseDocFrontmatter(lines []string) docMeta {
	meta := docMeta{bodyStart: 1}
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return meta
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return meta
	}

	fm := strings.Join(lines[1:end], "\n")
	var raw struct {
		Codd struct {
			NodeID string `yaml:"node_id"`
			Title  string `yaml:"title"`
			Tags   any    `yaml:"tags"`
		} `yaml:"codd"`
		NodeID string `yaml:"node_id"`
		Title  string `yaml:"title"`
		Tags   any    `yaml:"tags"`
	}
	if err := yaml.Unmarshal([]byte(fm), &raw); err != nil {
		meta.bodyStart = end + 2
		return meta
	}

	meta.nodeID = strings.TrimSpace(raw.Codd.NodeID)
	meta.title = strings.TrimSpace(raw.Codd.Title)
	meta.tags = normalizeStringList(raw.Codd.Tags)
	if meta.nodeID == "" {
		meta.nodeID = strings.TrimSpace(raw.NodeID)
	}
	if meta.title == "" {
		meta.title = strings.TrimSpace(raw.Title)
	}
	if len(meta.tags) == 0 {
		meta.tags = normalizeStringList(raw.Tags)
	}
	meta.bodyStart = end + 2
	return meta
}

func normalizeStringList(v any) []string {
	switch typed := v.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, it := range typed {
			s := strings.TrimSpace(fmt.Sprint(it))
			if s != "" && s != "<nil>" {
				out = append(out, s)
			}
		}
		return uniqueStrings(out)
	case []string:
		return uniqueStrings(typed)
	case string:
		s := strings.TrimSpace(typed)
		if s == "" {
			return nil
		}
		return []string{s}
	default:
		return nil
	}
}

func firstHeading(lines []string, prefix string) string {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
		}
	}
	return ""
}

func collectSections(lines []string, bodyStart int) []DocSection {
	start := bodyStart - 1
	if start < 0 {
		start = 0
	}

	type marker struct {
		line    int
		heading string
	}
	markers := make([]marker, 0, 16)
	for i := start; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "## ") {
			heading := strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
			if heading != "" {
				markers = append(markers, marker{line: i + 1, heading: heading})
			}
		}
	}

	if len(markers) == 0 {
		last := len(lines)
		if last < bodyStart {
			last = bodyStart
		}
		body := strings.Join(lines[start:], "\n")
		return []DocSection{{
			Heading:       "Document",
			LineStart:     bodyStart,
			LineEnd:       last,
			TokenEstimate: estimateTokens(body),
		}}
	}

	sections := make([]DocSection, 0, len(markers))
	for i, marker := range markers {
		end := len(lines)
		if i+1 < len(markers) {
			end = markers[i+1].line - 1
		}
		if end < marker.line {
			end = marker.line
		}
		chunk := strings.Join(lines[marker.line-1:end], "\n")
		sections = append(sections, DocSection{
			Heading:       marker.heading,
			LineStart:     marker.line,
			LineEnd:       end,
			TokenEstimate: estimateTokens(chunk),
		})
	}
	return sections
}

func collectKeywords(sections []DocSection, tags []string) []string {
	out := make([]string, 0, len(tags)+len(sections))
	out = append(out, tags...)
	for _, sec := range sections {
		heading := strings.TrimSpace(sec.Heading)
		if heading == "" || strings.EqualFold(heading, "Document") {
			continue
		}
		out = append(out, heading)
	}
	out = uniqueStrings(out)
	if len(out) > 12 {
		out = out[:12]
	}
	return out
}

func inferCategories(tags, keywords []string, path string) []string {
	known := map[string]struct{}{
		"scope":          {},
		"functional":     {},
		"non_functional": {},
		"acceptance":     {},
		"risk":           {},
		"dependency":     {},
		"priority":       {},
	}
	out := make([]string, 0, 3)
	push := func(v string) {
		v = strings.TrimSpace(v)
		if _, ok := known[v]; ok {
			out = append(out, v)
		}
	}
	for _, tag := range tags {
		push(tag)
	}
	for _, kw := range keywords {
		push(strings.ToLower(kw))
	}

	lowerPath := strings.ToLower(filepath.ToSlash(path))
	switch {
	case strings.Contains(lowerPath, "/requirements/"):
		out = append(out, "functional", "scope")
	case strings.Contains(lowerPath, "/design/"):
		out = append(out, "functional", "dependency")
	case strings.Contains(lowerPath, "/adr/"):
		out = append(out, "dependency", "risk")
	}

	out = uniqueStrings(out)
	if len(out) == 0 {
		return []string{"functional"}
	}
	return out
}

func buildSummary(lines []string, bodyStart int) string {
	start := bodyStart - 1
	if start < 0 {
		start = 0
	}
	for i := start; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "|") ||
			strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "---") {
			continue
		}
		r := []rune(trimmed)
		if len(r) > 140 {
			return string(r[:140])
		}
		return trimmed
	}
	return ""
}

func estimateTokens(s string) int {
	chars := len([]rune(strings.TrimSpace(s)))
	if chars == 0 {
		return 0
	}
	tokens := int(float64(chars) * 0.4)
	if tokens < 1 {
		return 1
	}
	return tokens
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		clean := strings.TrimSpace(v)
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	return out
}

func scoreDocEntry(doc DocEntry, tokens []string) int {
	if len(tokens) == 0 {
		return 0
	}
	full := strings.ToLower(strings.Join([]string{doc.Title, doc.Summary}, " "))
	score := 0
	for _, token := range tokens {
		if strings.Contains(full, token) {
			score += 6
		}
		for _, kw := range doc.Keywords {
			if strings.Contains(strings.ToLower(kw), token) {
				score += 4
			}
		}
		for _, c := range doc.Categories {
			if strings.Contains(strings.ToLower(c), token) {
				score += 3
			}
		}
		for _, sec := range doc.Sections {
			if strings.Contains(strings.ToLower(sec.Heading), token) {
				score += 2
			}
		}
	}
	return score
}

func tokenizeQuery(s string) []string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return nil
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return !(r == '_' || r == '-' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') || (r >= 0x3040 && r <= 0x30ff) || (r >= 0x4e00 && r <= 0x9faf))
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len([]rune(p)) < 2 {
			continue
		}
		out = append(out, p)
	}
	return uniqueStrings(out)
}

func resolveDocPath(rel string) (string, bool) {
	candidate := strings.TrimSpace(rel)
	if candidate == "" {
		return "", false
	}
	candidates := []string{candidate}
	if !filepath.IsAbs(candidate) {
		candidates = append(candidates, filepath.Join("docs", candidate))
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c, true
		}
	}
	return "", false
}

func readSection(lines []string, start, end int) string {
	if len(lines) == 0 {
		return ""
	}
	if start < 1 {
		start = 1
	}
	if end <= 0 || end > len(lines) {
		end = len(lines)
	}
	if end < start {
		end = start
	}
	if start > len(lines) {
		return ""
	}
	return strings.TrimSpace(strings.Join(lines[start-1:end], "\n"))
}
