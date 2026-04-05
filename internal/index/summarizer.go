package index

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/taka-sho/teraflow/internal/agent"
)

const defaultSummaryPrompt = "以下の技術文書を200-300 tokensで要約してください。\nキーポイント、主要な定義、依存関係を含めてください。"

// Summarizer generates and caches summaries for indexed documents.
type Summarizer struct {
	provider  agent.Provider
	cacheDir  string
	maxTokens int
}

// NewSummarizer constructs a summarizer with sane defaults.
func NewSummarizer(provider agent.Provider, cacheDir string) *Summarizer {
	if cacheDir == "" {
		cacheDir = filepath.Join(".teraflow", "summaries")
	}

	return &Summarizer{
		provider:  provider,
		cacheDir:  cacheDir,
		maxTokens: 300,
	}
}

// Summarize returns a cached summary when valid; otherwise it regenerates and stores it.
func (s *Summarizer) Summarize(ctx context.Context, entry Entry, content string) (string, error) {
	if s.IsCacheValid(entry) {
		return s.LoadSummary(entry.NodeID)
	}
	if s.provider == nil {
		return "", fmt.Errorf("provider is nil")
	}

	summary, _, err := s.provider.Complete(ctx, defaultSummaryPrompt, content, s.maxTokens)
	if err != nil {
		return "", fmt.Errorf("generate summary for %s: %w", entry.NodeID, err)
	}

	path := s.cachePath(entry.NodeID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create summary directory: %w", err)
	}

	cacheBody := fmt.Sprintf("# hash:%s\n%s\n", entry.ContentHash, strings.TrimSpace(summary))
	if err := os.WriteFile(path, []byte(cacheBody), 0o644); err != nil {
		return "", fmt.Errorf("write summary cache: %w", err)
	}

	return strings.TrimSpace(summary), nil
}

// LoadSummary loads a cached summary body for the given node id.
func (s *Summarizer) LoadSummary(nodeID string) (string, error) {
	path := s.cachePath(nodeID)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("load summary cache: %w", err)
	}

	text := string(data)
	parts := strings.SplitN(text, "\n", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid summary cache format: %s", path)
	}
	return strings.TrimSpace(parts[1]), nil
}

// IsCacheValid checks whether cache hash line matches the entry content hash.
func (s *Summarizer) IsCacheValid(entry Entry) bool {
	path := s.cachePath(entry.NodeID)
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	firstLine := string(data)
	if i := strings.IndexByte(firstLine, '\n'); i >= 0 {
		firstLine = firstLine[:i]
	}

	return firstLine == "# hash:"+entry.ContentHash
}

// UpdateAll updates summaries for all index entries.
func (s *Summarizer) UpdateAll(ctx context.Context, idx *Index) (updated int, skipped int, err error) {
	if idx == nil {
		return 0, 0, fmt.Errorf("index is nil")
	}

	for _, entry := range idx.Entries {
		if s.IsCacheValid(entry) {
			skipped++
			continue
		}

		content, readErr := os.ReadFile(entry.FilePath)
		if readErr != nil {
			return updated, skipped, fmt.Errorf("read source %s: %w", entry.FilePath, readErr)
		}
		if _, sumErr := s.Summarize(ctx, entry, string(content)); sumErr != nil {
			return updated, skipped, sumErr
		}
		updated++
	}

	return updated, skipped, nil
}

func (s *Summarizer) cachePath(nodeID string) string {
	safe := strings.ReplaceAll(nodeID, ":", "--")
	safe = strings.ReplaceAll(safe, "/", "-")
	return filepath.Join(s.cacheDir, safe+".txt")
}
