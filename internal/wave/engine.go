package wave

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/taka-sho/teraflow/internal/agent"
	"github.com/taka-sho/teraflow/internal/graphbridge"
	"github.com/taka-sho/teraflow/internal/index"
	"github.com/taka-sho/teraflow/internal/pipeline"
	"github.com/taka-sho/teraflow/internal/validate"
)

// WaveEngine executes wave definitions and materializes generated artifacts.
type WaveEngine struct {
	provider      agent.Provider
	validator     *validate.Validator
	graphRAG      *graphbridge.Bridge
	idx           *index.Index
	projectRoot   string
	maxContextTok int
	templates     *TemplateLoader
}

func NewEngine(provider agent.Provider, validator *validate.Validator, graphRAG *graphbridge.Bridge, idx *index.Index, projectRoot string) *WaveEngine {
	return &WaveEngine{
		provider:      provider,
		validator:     validator,
		graphRAG:      graphRAG,
		idx:           idx,
		projectRoot:   projectRoot,
		maxContextTok: DefaultContextTokenLimit,
		templates:     NewTemplateLoader(projectRoot),
	}
}

func (e *WaveEngine) ExecutePhase(ctx context.Context, phase string, waves []WaveDefinition, opts ExecuteOptions) ([]WaveResult, error) {
	if len(waves) == 0 {
		return nil, fmt.Errorf("no waves defined for phase %q", phase)
	}
	sort.Slice(waves, func(i, j int) bool { return waves[i].Number < waves[j].Number })

	completed := make(map[int]bool, len(waves))
	results := make([]WaveResult, 0, len(waves))

	for _, w := range waves {
		for _, dep := range w.DependsOn {
			if !completed[dep] {
				return nil, fmt.Errorf("wave %d depends on unfinished wave %d", w.Number, dep)
			}
		}
		res, err := e.ExecuteWave(ctx, phase, w, opts)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
		completed[w.Number] = true
	}
	return results, nil
}

func (e *WaveEngine) ExecuteWave(ctx context.Context, phase string, wave WaveDefinition, opts ExecuteOptions) (WaveResult, error) {
	ctxData, warns, err := e.buildContext(ctx, phase, wave)
	if err != nil {
		return WaveResult{}, err
	}

	tplName := wave.Template
	if strings.TrimSpace(opts.TemplateOverride) != "" {
		tplName = opts.TemplateOverride
	}

	prompt, err := e.templates.Render(tplName, map[string]any{
		"Phase":   phase,
		"Wave":    wave,
		"Context": ctxData,
		"Now":     time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return WaveResult{}, err
	}

	content := prompt
	status := "completed"
	if opts.DryRun {
		status = "dry_run"
	} else {
		if e.provider == nil {
			return WaveResult{}, fmt.Errorf("provider is required for non-dry-run wave execution")
		}
		generated, _, err := e.provider.Complete(
			ctx,
			"You are a senior software architect. Generate CoDD-compatible artifacts in markdown.",
			prompt,
			4096,
		)
		if err != nil {
			return WaveResult{}, fmt.Errorf("wave %d generation failed: %w", wave.Number, err)
		}
		content = strings.TrimSpace(generated) + "\n"
	}

	outPath := buildArtifactPath(e.projectRoot, phase, wave)
	if !opts.DryRun {
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return WaveResult{}, fmt.Errorf("create output directory: %w", err)
		}
		if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
			return WaveResult{}, fmt.Errorf("write artifact: %w", err)
		}
	}

	if e.validator != nil && !opts.DryRun {
		validation := e.validator.ValidateArtifact(nodeIDForWave(phase, wave))
		if !validation.Valid {
			status = "pending_review"
			warns = append(warns, "generated artifact has validation issues")
		}
	}

	return WaveResult{
		WaveNumber: wave.Number,
		WaveName:   wave.Name,
		Artifacts: []GeneratedArtifact{{
			Path:         outPath,
			NodeID:       nodeIDForWave(phase, wave),
			ArtifactType: wave.ArtifactType,
			ReviewReq:    wave.ReviewReq,
		}},
		Status:   status,
		Warnings: warns,
	}, nil
}

func buildArtifactPath(projectRoot, phase string, wave WaveDefinition) string {
	cleanPhase := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(phase)), "_", "-")
	name := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(wave.Name)), " ", "-")
	if name == "" {
		name = strings.TrimSpace(wave.ArtifactType)
	}
	if name == "" {
		name = "artifact"
	}
	base := fmt.Sprintf("%s-wave-%02d-%s.md", cleanPhase, wave.Number, name)

	dir := "design"
	switch cleanPhase {
	case "requirements":
		dir = "requirements"
	case "detailed-design":
		dir = "detailed-design"
	case "implementation":
		dir = "implementation"
	case "unit-test", "integration-test", "system-test", "acceptance-test":
		dir = "test"
	}
	return filepath.Join(projectRoot, "docs", dir, base)
}

func nodeIDForWave(phase string, wave WaveDefinition) string {
	p := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(phase)), "_", "-")
	t := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(wave.ArtifactType)), " ", "-")
	if t == "" {
		t = fmt.Sprintf("wave-%d", wave.Number)
	}
	return fmt.Sprintf("%s:%s", p, t)
}

func (e *WaveEngine) buildContext(ctx context.Context, phase string, wave WaveDefinition) (*WaveContext, []string, error) {
	contextData := &WaveContext{Phase: phase, Wave: wave}
	warnings := make([]string, 0)

	remaining := e.maxContextTok
	for _, inputNode := range wave.Inputs {
		if e.idx == nil {
			break
		}
		entry := e.idx.FindByNodeID(inputNode)
		if entry == nil {
			warnings = append(warnings, fmt.Sprintf("input node not found in index: %s", inputNode))
			continue
		}

		data, err := os.ReadFile(filepath.Join(e.projectRoot, filepath.FromSlash(entry.Path)))
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("failed reading %s: %v", entry.Path, err))
			continue
		}

		body := string(data)
		docBudget := 1500
		if remaining < docBudget {
			docBudget = remaining
		}
		body = trimToTokenBudget(body, docBudget)
		remaining -= estimateTokens(body)
		contextData.Inputs = append(contextData.Inputs, ContextDocument{NodeID: inputNode, Path: entry.Path, Body: body})
		if remaining <= 0 {
			break
		}
	}

	if e.graphRAG != nil && e.graphRAG.Available() {
		resp, err := e.graphRAG.Execute(graphbridge.Request{
			Command: "query",
			Args: map[string]any{
				"query":        wave.Name,
				"mode":         "local",
				"graph_path":   filepath.Join(e.projectRoot, ".teraflow", "graphrag", "graph.graphml"),
				"storage_path": filepath.Join(e.projectRoot, ".teraflow", "graphrag"),
			},
		})
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("graphrag query failed: %v", err))
		} else if resp != nil {
			related := compactLines(fmt.Sprintf("%v", resp.Data), 20)
			contextData.RelatedSummary = trimToTokenBudget(related, min(1000, remaining))
			remaining -= estimateTokens(contextData.RelatedSummary)
		}
	} else {
		warnings = append(warnings, "graphrag not available, using index-only context")
	}

	if e.idx != nil && len(wave.Inputs) > 0 {
		analyzer := pipeline.NewImpactAnalyzer(e.idx, e.graphRAG, e.projectRoot)
		impact, err := analyzer.Analyze(ctx, wave.Inputs[0], 2)
		if err == nil {
			summary := fmt.Sprintf("changed=%s review=%s regen=%s", impact.ChangedNode, summarizeNodeIDs(impact.ReviewNeeded), summarizeNodeIDs(impact.RegenRequired))
			contextData.ImpactSummary = trimToTokenBudget(summary, min(500, remaining))
			remaining -= estimateTokens(contextData.ImpactSummary)
		} else {
			warnings = append(warnings, fmt.Sprintf("impact summary unavailable: %v", err))
		}
	}

	used := e.maxContextTok - remaining
	if used < 0 {
		used = 0
	}
	contextData.TokenEstimate = used
	return contextData, warnings, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
