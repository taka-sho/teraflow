package pipeline

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/taka-sho/teraflow/internal/agent"
	"github.com/taka-sho/teraflow/internal/index"
	"github.com/taka-sho/teraflow/internal/validate"
	"gopkg.in/yaml.v3"
)

// ImplementEngine generates implementation modules from a detailed design document.
type ImplementEngine struct {
	provider    agent.Provider
	waveEngine  any
	validator   *validate.Validator
	projectRoot string
	idx         *index.Index
}

func NewImplementEngine(provider agent.Provider, waveEngine any, validator *validate.Validator, projectRoot string, idx *index.Index) *ImplementEngine {
	return &ImplementEngine{
		provider:    provider,
		waveEngine:  waveEngine,
		validator:   validator,
		projectRoot: projectRoot,
		idx:         idx,
	}
}

type ImplementRequest struct {
	DesignDocPath string
	Modules       []ModuleSpec
	MaxParallel   int
	CreatePR      bool
	DryRun        bool
}

type ModuleSpec struct {
	Path           string
	DesignSection  string
	DependsOn      []string
	ReviewRequired string
	TestSpec       string
}

type ImplementResult struct {
	Module         ModuleSpec `json:"module"`
	Status         string     `json:"status"`
	OutputPath     string     `json:"output_path"`
	ReviewRequired string     `json:"review_required"`
	Warnings       []string   `json:"warnings,omitempty"`
	Error          string     `json:"error,omitempty"`
	TestFiles      []string   `json:"test_files,omitempty"`
	StartedAt      time.Time  `json:"started_at"`
	FinishedAt     time.Time  `json:"finished_at"`
}

type IntegrationPR struct {
	Title          string   `json:"title"`
	Body           string   `json:"body"`
	ReviewRequired string   `json:"review_required"`
	Modules        []string `json:"modules"`
	DryRun         bool     `json:"dry_run"`
	Created        bool     `json:"created"`
}

type ImplementReport struct {
	DesignDocPath string             `json:"design_doc_path"`
	Modules       []ImplementResult  `json:"modules"`
	Summary       map[string]int     `json:"summary"`
	Validation    *ValidationSummary `json:"validation,omitempty"`
	IntegrationPR *IntegrationPR     `json:"integration_pr,omitempty"`
}

type ValidationSummary struct {
	Valid        bool `json:"valid"`
	ErrorCount   int  `json:"error_count"`
	WarningCount int  `json:"warning_count"`
}

type designDocSpec struct {
	NodeID         string
	Title          string
	ReviewRequired string
	Modules        []ModuleSpec
	VerifiedBy     []string
	Raw            string
}

// Execute runs module generation. Module-level failures are aggregated and do not stop other modules.
func (e *ImplementEngine) Execute(ctx context.Context, req ImplementRequest) (*ImplementReport, error) {
	if strings.TrimSpace(req.DesignDocPath) == "" {
		return nil, fmt.Errorf("design doc path is required")
	}
	if req.MaxParallel <= 0 {
		req.MaxParallel = 3
	}

	designPath, err := resolveToAbsPath(e.projectRoot, req.DesignDocPath)
	if err != nil {
		return nil, err
	}
	design, err := loadDesignDocSpec(designPath)
	if err != nil {
		return nil, err
	}

	modules, err := selectModules(design.Modules, req.Modules)
	if err != nil {
		return nil, err
	}
	if len(modules) == 0 {
		return nil, fmt.Errorf("no modules resolved from design doc")
	}

	levels, err := executionLevels(modules)
	if err != nil {
		return nil, err
	}

	report := &ImplementReport{
		DesignDocPath: req.DesignDocPath,
		Modules:       make([]ImplementResult, 0, len(modules)),
		Summary: map[string]int{
			"total":     len(modules),
			"completed": 0,
			"failed":    0,
			"skipped":   0,
		},
	}

	resultsByPath := make(map[string]ImplementResult, len(modules))
	for _, level := range levels {
		batch, batchErr := e.runLevel(ctx, design, level, req)
		if batchErr != nil {
			return nil, batchErr
		}
		for _, result := range batch {
			resultsByPath[result.Module.Path] = result
		}
	}

	ordered := make([]ImplementResult, 0, len(modules))
	for _, mod := range modules {
		res, ok := resultsByPath[mod.Path]
		if !ok {
			res = ImplementResult{Module: mod, Status: "skipped", Error: "module result missing"}
		}
		ordered = append(ordered, res)
		switch res.Status {
		case "completed", "dry_run":
			report.Summary["completed"]++
		case "failed":
			report.Summary["failed"]++
		default:
			report.Summary["skipped"]++
		}
	}
	report.Modules = ordered

	if e.validator != nil {
		e.validator.SetLevel(2)
		v := e.validator.ValidatePhaseTransition("", "")
		report.Validation = &ValidationSummary{
			Valid:        v.Valid,
			ErrorCount:   len(v.Errors),
			WarningCount: len(v.Warnings),
		}
	}

	if req.CreatePR {
		report.IntegrationPR = e.CreateIntegrationPR(*report, req.DryRun)
	}

	return report, nil
}

func (e *ImplementEngine) runLevel(ctx context.Context, design designDocSpec, modules []ModuleSpec, req ImplementRequest) ([]ImplementResult, error) {
	results := make([]ImplementResult, 0, len(modules))
	resultCh := make(chan ImplementResult, len(modules))
	sem := make(chan struct{}, req.MaxParallel)
	var wg sync.WaitGroup

	for _, mod := range modules {
		mod := mod
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			resultCh <- e.GenerateModule(ctx, design, mod, len(modules), req.DryRun)
		}()
	}

	wg.Wait()
	close(resultCh)
	for res := range resultCh {
		results = append(results, res)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Module.Path < results[j].Module.Path
	})
	return results, nil
}

// GenerateModule generates one module and optional test skeletons.
func (e *ImplementEngine) GenerateModule(ctx context.Context, design designDocSpec, spec ModuleSpec, affectedModuleCount int, dryRun bool) ImplementResult {
	started := time.Now().UTC()
	res := ImplementResult{
		Module:     spec,
		Status:     "failed",
		StartedAt:  started,
		FinishedAt: started,
	}
	defer func() { res.FinishedAt = time.Now().UTC() }()

	outPath, err := resolveToAbsPath(e.projectRoot, spec.Path)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.OutputPath = outPath
	res.ReviewRequired = DetermineReviewLevel(spec, design.ReviewRequired, affectedModuleCount)

	prompt := buildImplementPrompt(design, spec)
	generated := ""
	if dryRun {
		generated = fmt.Sprintf("// dry-run: generated for %s\n", spec.Path)
		if strings.HasSuffix(spec.Path, ".py") {
			generated = fmt.Sprintf("# dry-run: generated for %s\n", spec.Path)
		}
		res.Status = "dry_run"
	} else {
		if e.provider == nil {
			res.Error = "provider is required for non-dry-run implement execution"
			return res
		}
		output, _, genErr := e.provider.Complete(
			ctx,
			"You are a senior software engineer. Produce compile-ready code for one module.",
			prompt,
			4096,
		)
		if genErr != nil {
			res.Error = genErr.Error()
			return res
		}
		generated = strings.TrimSpace(output) + "\n"
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			res.Error = fmt.Sprintf("create output dir: %v", err)
			return res
		}
		if err := os.WriteFile(outPath, []byte(generated), 0o644); err != nil {
			res.Error = fmt.Sprintf("write module: %v", err)
			return res
		}
		res.Status = "completed"
	}

	tests, warns := e.generateTestSkeletons(spec, design.VerifiedBy, dryRun)
	res.TestFiles = tests
	res.Warnings = append(res.Warnings, warns...)
	return res
}

// CreateIntegrationPR builds integration PR metadata after module generation.
func (e *ImplementEngine) CreateIntegrationPR(report ImplementReport, dryRun bool) *IntegrationPR {
	modules := make([]string, 0, len(report.Modules))
	for _, r := range report.Modules {
		modules = append(modules, r.Module.Path)
	}
	sort.Strings(modules)
	return &IntegrationPR{
		Title:          "feat(pipeline): integrate generated implementation modules",
		Body:           fmt.Sprintf("Generated from %s\n\nModules:\n- %s", report.DesignDocPath, strings.Join(modules, "\n- ")),
		ReviewRequired: "approve",
		Modules:        modules,
		DryRun:         dryRun,
		Created:        !dryRun,
	}
}

func (e *ImplementEngine) generateTestSkeletons(spec ModuleSpec, verifiedBy []string, dryRun bool) ([]string, []string) {
	if len(verifiedBy) == 0 {
		return nil, nil
	}
	outputs := make([]string, 0, len(verifiedBy))
	warnings := make([]string, 0)

	for _, nodeID := range verifiedBy {
		lang := inferTestLanguage(nodeID)
		target := defaultTestPath(spec.Path, nodeID, lang)
		if e.idx != nil {
			if entry := e.idx.FindByNodeID(nodeID); entry != nil && strings.TrimSpace(entry.Path) != "" {
				target = filepath.Join(e.projectRoot, filepath.FromSlash(entry.Path))
				if strings.HasSuffix(strings.ToLower(entry.Path), ".py") {
					lang = "python"
				} else {
					lang = "go"
				}
			}
		}

		body := renderTestSkeleton(lang, nodeID)
		outputs = append(outputs, target)
		if dryRun {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			warnings = append(warnings, fmt.Sprintf("create test dir for %s: %v", nodeID, err))
			continue
		}
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			warnings = append(warnings, fmt.Sprintf("write test skeleton %s: %v", nodeID, err))
		}
	}

	return outputs, warnings
}

func renderTestSkeleton(lang, nodeID string) string {
	if lang == "python" {
		funcName := sanitizeIdentifier(strings.ReplaceAll(nodeID, ":", "_"))
		return "import pytest\n\n\ndef test_" + funcName + "():\n    # TODO: implement from test spec\n    assert True\n"
	}

	funcName := sanitizeIdentifier(strings.ReplaceAll(nodeID, ":", "_"))
	return "package generated\n\nimport \"testing\"\n\nfunc Test" + toExportedIdentifier(funcName) + "(t *testing.T) {\n\t// TODO: implement from test spec\n}\n"
}

func buildImplementPrompt(design designDocSpec, spec ModuleSpec) string {
	var b strings.Builder
	b.WriteString("Detailed design document:\n")
	b.WriteString(design.Raw)
	b.WriteString("\n\nTarget module:\n")
	b.WriteString(spec.Path)
	if spec.DesignSection != "" {
		b.WriteString("\nDesign section:\n")
		b.WriteString(spec.DesignSection)
	}
	if len(spec.DependsOn) > 0 {
		b.WriteString("\nDepends on:\n- ")
		b.WriteString(strings.Join(spec.DependsOn, "\n- "))
	}
	if spec.TestSpec != "" {
		b.WriteString("\nTest spec node:\n")
		b.WriteString(spec.TestSpec)
	}
	b.WriteString("\n\nGenerate only the source code for this module.")
	return b.String()
}

func resolveToAbsPath(projectRoot, path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("path is required")
	}
	p := filepath.FromSlash(strings.TrimSpace(path))
	if filepath.IsAbs(p) {
		return filepath.Clean(p), nil
	}
	if strings.TrimSpace(projectRoot) == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		projectRoot = cwd
	}
	return filepath.Clean(filepath.Join(projectRoot, p)), nil
}

func loadDesignDocSpec(path string) (designDocSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return designDocSpec{}, fmt.Errorf("read design doc: %w", err)
	}
	content := string(data)
	parsed, err := parseDesignFrontmatter(content)
	if err != nil {
		return designDocSpec{}, err
	}
	if len(parsed.Modules) == 0 {
		return designDocSpec{}, fmt.Errorf("design doc modules field is required")
	}
	parsed.Raw = content
	return parsed, nil
}

func parseDesignFrontmatter(content string) (designDocSpec, error) {
	parts := strings.Split(content, "\n---\n")
	if len(parts) < 2 || !strings.HasPrefix(strings.TrimSpace(parts[0]), "---") {
		return designDocSpec{}, errors.New("design doc must include YAML frontmatter")
	}
	front := strings.TrimPrefix(parts[0], "---\n")
	type rawSpec struct {
		NodeID         string `yaml:"node_id"`
		Title          string `yaml:"title"`
		ReviewRequired string `yaml:"review_required"`
		Modules        any    `yaml:"modules"`
		VerifiedBy     any    `yaml:"verified_by"`
	}
	var raw struct {
		CoDD    rawSpec `yaml:"codd"`
		rawSpec `yaml:",inline"`
	}
	if err := yaml.Unmarshal([]byte(front), &raw); err != nil {
		return designDocSpec{}, fmt.Errorf("parse design frontmatter: %w", err)
	}
	base := raw.CoDD
	if strings.TrimSpace(base.NodeID) == "" && strings.TrimSpace(raw.rawSpec.NodeID) != "" {
		base = raw.rawSpec
	}
	modules := normalizeModules(base.Modules)
	if len(modules) == 0 {
		modules = normalizeModules(raw.rawSpec.Modules)
	}
	verifiedBy := normalizeStrings(base.VerifiedBy)
	if len(verifiedBy) == 0 {
		verifiedBy = normalizeStrings(raw.rawSpec.VerifiedBy)
	}
	return designDocSpec{
		NodeID:         strings.TrimSpace(base.NodeID),
		Title:          strings.TrimSpace(base.Title),
		ReviewRequired: normalizeReviewLevel(base.ReviewRequired),
		Modules:        modules,
		VerifiedBy:     verifiedBy,
	}, nil
}

func normalizeModules(v any) []ModuleSpec {
	switch typed := v.(type) {
	case []any:
		out := make([]ModuleSpec, 0, len(typed))
		for _, item := range typed {
			switch mod := item.(type) {
			case string:
				p := strings.TrimSpace(mod)
				if p != "" {
					out = append(out, ModuleSpec{Path: filepath.ToSlash(filepath.Clean(filepath.FromSlash(p)))})
				}
			case map[string]any:
				path, _ := mod["path"].(string)
				if strings.TrimSpace(path) == "" {
					continue
				}
				spec := ModuleSpec{Path: filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))}
				spec.DesignSection, _ = mod["design_section"].(string)
				spec.TestSpec, _ = mod["test_spec"].(string)
				spec.ReviewRequired, _ = mod["review_required"].(string)
				spec.DependsOn = normalizeStrings(mod["depends_on"])
				for i := range spec.DependsOn {
					spec.DependsOn[i] = filepath.ToSlash(filepath.Clean(filepath.FromSlash(spec.DependsOn[i])))
				}
				out = append(out, spec)
			}
		}
		return dedupeModules(out)
	default:
		return nil
	}
}

func normalizeStrings(v any) []string {
	switch typed := v.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{strings.TrimSpace(typed)}
	default:
		return nil
	}
}

func dedupeModules(in []ModuleSpec) []ModuleSpec {
	byPath := make(map[string]ModuleSpec, len(in))
	order := make([]string, 0, len(in))
	for _, spec := range in {
		p := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(spec.Path))))
		if p == "" || p == "." {
			continue
		}
		spec.Path = p
		if existing, ok := byPath[p]; ok {
			if spec.DesignSection == "" {
				spec.DesignSection = existing.DesignSection
			}
			if spec.TestSpec == "" {
				spec.TestSpec = existing.TestSpec
			}
			if spec.ReviewRequired == "" {
				spec.ReviewRequired = existing.ReviewRequired
			}
			if len(spec.DependsOn) == 0 {
				spec.DependsOn = existing.DependsOn
			}
		} else {
			order = append(order, p)
		}
		byPath[p] = spec
	}
	out := make([]ModuleSpec, 0, len(order))
	for _, p := range order {
		out = append(out, byPath[p])
	}
	return out
}

func selectModules(fromDesign, requested []ModuleSpec) ([]ModuleSpec, error) {
	if len(requested) == 0 {
		return dedupeModules(fromDesign), nil
	}
	all := dedupeModules(fromDesign)
	allMap := make(map[string]ModuleSpec, len(all))
	for _, mod := range all {
		allMap[mod.Path] = mod
	}

	need := make(map[string]struct{}, len(requested))
	for _, mod := range requested {
		p := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(mod.Path))))
		if p == "" || p == "." {
			continue
		}
		need[p] = struct{}{}
	}
	if len(need) == 0 {
		return nil, fmt.Errorf("--module provided but no valid module path found")
	}

	queue := make([]string, 0, len(need))
	for p := range need {
		queue = append(queue, p)
	}

	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		if spec, ok := allMap[p]; ok {
			for _, dep := range spec.DependsOn {
				d := filepath.ToSlash(filepath.Clean(filepath.FromSlash(dep)))
				if d == "" || d == "." {
					continue
				}
				if _, seen := need[d]; !seen {
					need[d] = struct{}{}
					queue = append(queue, d)
				}
			}
		}
	}

	selected := make([]ModuleSpec, 0, len(need))
	for _, mod := range all {
		if _, ok := need[mod.Path]; ok {
			selected = append(selected, mod)
		}
	}

	for p := range need {
		if _, ok := allMap[p]; !ok {
			selected = append(selected, ModuleSpec{Path: p})
		}
	}

	return dedupeModules(selected), nil
}

func executionLevels(modules []ModuleSpec) ([][]ModuleSpec, error) {
	byPath := make(map[string]ModuleSpec, len(modules))
	inDeg := make(map[string]int, len(modules))
	graph := make(map[string][]string, len(modules))

	for _, mod := range modules {
		byPath[mod.Path] = mod
		inDeg[mod.Path] = 0
	}
	for _, mod := range modules {
		for _, dep := range mod.DependsOn {
			d := filepath.ToSlash(filepath.Clean(filepath.FromSlash(dep)))
			if d == "" || d == "." {
				continue
			}
			if _, ok := byPath[d]; !ok {
				continue
			}
			graph[d] = append(graph[d], mod.Path)
			inDeg[mod.Path]++
		}
	}

	remaining := len(modules)
	levels := make([][]ModuleSpec, 0)
	for remaining > 0 {
		ready := make([]string, 0)
		for path, d := range inDeg {
			if d == 0 {
				ready = append(ready, path)
			}
		}
		if len(ready) == 0 {
			cycle := make([]string, 0, remaining)
			for p := range inDeg {
				cycle = append(cycle, p)
			}
			sort.Strings(cycle)
			return nil, fmt.Errorf("module dependency cycle detected: %s", strings.Join(cycle, ", "))
		}
		sort.Strings(ready)
		level := make([]ModuleSpec, 0, len(ready))
		for _, path := range ready {
			level = append(level, byPath[path])
			delete(inDeg, path)
			remaining--
			for _, next := range graph[path] {
				if _, ok := inDeg[next]; ok {
					inDeg[next]--
				}
			}
		}
		levels = append(levels, level)
	}
	return levels, nil
}

func inferTestLanguage(nodeID string) string {
	n := strings.ToLower(nodeID)
	if strings.Contains(n, "pytest") || strings.Contains(n, "python") {
		return "python"
	}
	return "go"
}

func defaultTestPath(modulePath, nodeID, lang string) string {
	dir := filepath.Dir(modulePath)
	base := strings.TrimSuffix(filepath.Base(modulePath), filepath.Ext(modulePath))
	if base == "" || base == "." || base == string(filepath.Separator) {
		base = sanitizeIdentifier(strings.ReplaceAll(nodeID, ":", "_"))
	}
	if lang == "python" {
		if strings.HasSuffix(modulePath, ".py") {
			return filepath.Join(dir, "test_"+base+".py")
		}
		return filepath.Join("tests", base+"_test.py")
	}
	return filepath.Join(dir, base+"_test.go")
}

func sanitizeIdentifier(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "generated"
	}
	replacer := strings.NewReplacer("-", "_", ":", "_", "/", "_", ".", "_")
	s = replacer.Replace(s)
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	s = strings.Trim(s, "_")
	if s == "" {
		return "generated"
	}
	return s
}

func toExportedIdentifier(s string) string {
	if s == "" {
		return "Generated"
	}
	parts := strings.Split(s, "_")
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		runes := []rune(parts[i])
		if len(runes) == 0 {
			continue
		}
		if runes[0] >= 'a' && runes[0] <= 'z' {
			runes[0] = runes[0] - ('a' - 'A')
		}
		parts[i] = string(runes)
	}
	out := strings.Join(parts, "")
	if out == "" {
		return "Generated"
	}
	return out
}
