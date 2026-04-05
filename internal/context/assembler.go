package context

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/taka-sho/teraflow/internal/index"
	"github.com/taka-sho/teraflow/internal/skill"
)

var sanitizeRE = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// Assembler creates final prompt context from index entries and summaries.
type Assembler struct {
	idx         *index.Index
	summaryDir  string
	projectRoot string
	budget      TokenBudget
}

// NewAssembler creates a context assembler.
func NewAssembler(idx *index.Index, summaryDir, projectRoot string) *Assembler {
	return &Assembler{
		idx:         idx,
		summaryDir:  summaryDir,
		projectRoot: projectRoot,
		budget:      DefaultBudget(),
	}
}

// Assemble builds a context bundle according to include/exclude and scoring.
func (a *Assembler) Assemble(cfg skill.ContextCfg, userInput, conversationHistory string) (*AssembledContext, error) {
	if a.idx == nil {
		return nil, fmt.Errorf("index is nil")
	}

	candidates := a.filterEntries(cfg)
	sort.SliceStable(candidates, func(i, j int) bool {
		iScore := ScoreEntry(candidates[i], userInput, conversationHistory)
		jScore := ScoreEntry(candidates[j], userInput, conversationHistory)
		if iScore == jScore {
			return candidates[i].Path < candidates[j].Path
		}
		return iScore > jScore
	})

	ctx := &AssembledContext{UserInput: userInput, IncludedDocs: make([]string, 0, len(candidates))}
	summaryParts := make([]string, 0)
	summaryTokens := 0
	included := map[string]struct{}{}

	for _, entry := range candidates {
		if summaryTokens >= a.budget.Summaries {
			break
		}
		summary, err := a.loadSummary(entry.NodeID)
		if err != nil || strings.TrimSpace(summary) == "" {
			continue
		}
		t := EstimateTokens(summary)
		if summaryTokens+t > a.budget.Summaries {
			continue
		}
		summaryParts = append(summaryParts, summary)
		summaryTokens += t
		if _, ok := included[entry.NodeID]; !ok {
			included[entry.NodeID] = struct{}{}
			ctx.IncludedDocs = append(ctx.IncludedDocs, entry.NodeID)
		}
	}

	fullText := ""
	if len(candidates) > 0 {
		full, err := a.loadFullDocument(candidates[0])
		if err != nil {
			return nil, err
		}
		est := EstimateTokens(full)
		if est > a.budget.FullDocument {
			full = trimToTokens(full, a.budget.FullDocument)
		}
		fullText = full
		if _, ok := included[candidates[0].NodeID]; !ok {
			included[candidates[0].NodeID] = struct{}{}
			ctx.IncludedDocs = append(ctx.IncludedDocs, candidates[0].NodeID)
		}
	}

	summaryText := strings.Join(summaryParts, "\n\n")
	switch {
	case summaryText != "" && fullText != "":
		ctx.Context = summaryText + "\n\n---\n\n" + fullText
	case summaryText != "":
		ctx.Context = summaryText
	default:
		ctx.Context = fullText
	}

	ctx.TotalTokens = EstimateTokens(ctx.Context) + EstimateTokens(ctx.UserInput)
	if cfg.MaxContextTokens > 0 && ctx.TotalTokens > cfg.MaxContextTokens {
		ctx.Context = trimToTokens(ctx.Context, max(0, cfg.MaxContextTokens-EstimateTokens(ctx.UserInput)))
		ctx.TotalTokens = EstimateTokens(ctx.Context) + EstimateTokens(ctx.UserInput)
	}

	return ctx, nil
}

func (a *Assembler) filterEntries(cfg skill.ContextCfg) []index.Entry {
	filtered := make([]index.Entry, 0, len(a.idx.Entries))
	for _, e := range a.idx.Entries {
		if !matchesAny(e.Path, cfg.Include, true) {
			continue
		}
		if matchesAny(e.Path, cfg.Exclude, false) {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered
}

func matchesAny(p string, patterns []string, emptyDefault bool) bool {
	if len(patterns) == 0 {
		return emptyDefault
	}
	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}
		if ok, _ := path.Match(pattern, p); ok {
			return true
		}
		if ok, _ := filepath.Match(pattern, p); ok {
			return true
		}
	}
	return false
}

func (a *Assembler) loadSummary(nodeID string) (string, error) {
	safe := sanitizeNodeID(nodeID)
	data, err := os.ReadFile(filepath.Join(a.summaryDir, safe+".txt"))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (a *Assembler) loadFullDocument(entry index.Entry) (string, error) {
	fullPath := filepath.Join(a.projectRoot, filepath.FromSlash(entry.Path))
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("read full document %s: %w", fullPath, err)
	}
	return strings.TrimSpace(string(data)), nil
}

func sanitizeNodeID(nodeID string) string {
	safe := sanitizeRE.ReplaceAllString(nodeID, "_")
	safe = strings.Trim(safe, "_")
	if safe == "" {
		return "node"
	}
	return safe
}

func trimToTokens(text string, maxTokens int) string {
	if maxTokens <= 0 || text == "" {
		return ""
	}
	runes := []rune(text)
	if EstimateTokens(text) <= maxTokens {
		return text
	}

	lo, hi := 0, len(runes)
	best := ""
	for lo <= hi {
		mid := (lo + hi) / 2
		candidate := string(runes[:mid])
		t := EstimateTokens(candidate)
		if t <= maxTokens {
			best = candidate
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return strings.TrimSpace(best)
}
