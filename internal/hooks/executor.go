package hooks

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/taka-sho/teraflow/internal/agent"
	cfgpkg "github.com/taka-sho/teraflow/internal/config"
	"github.com/taka-sho/teraflow/internal/index"
)

var hookExecCommandContext = exec.CommandContext

type Executor struct {
	projectRoot string
	dryRun      bool
}

func NewExecutor(projectRoot string, dryRun bool) *Executor {
	return &Executor{
		projectRoot: projectRoot,
		dryRun:      dryRun,
	}
}

// Execute runs a single hook action.
func (e *Executor) Execute(ctx HookContext, action HookAction) ActionResult {
	switch action.Action {
	case "respond":
		return e.executeRespond(ctx, action)
	case "summarize":
		return e.executeSummarize(ctx, action)
	case "index_update":
		return e.executeIndexUpdate()
	case "summary_update":
		return e.executeSummaryUpdate()
	case "generate":
		return e.executeGenerate(ctx, action)
	default:
		return ActionResult{
			Action:  action.Action,
			Success: false,
			Message: fmt.Sprintf("unsupported action: %s", action.Action),
		}
	}
}

// ExecuteAll runs all matched actions.
func (e *Executor) ExecuteAll(ctx HookContext, actions []HookAction) HookResult {
	results := make([]ActionResult, 0, len(actions))
	for _, action := range actions {
		results = append(results, e.Execute(ctx, action))
	}

	return HookResult{
		Event:   ctx.Event,
		Matched: actions,
		DryRun:  e.dryRun,
		Results: results,
	}
}

func (e *Executor) executeRespond(ctx HookContext, action HookAction) ActionResult {
	agentType := action.Skill
	if strings.TrimSpace(agentType) == "" {
		agentType = "requirements"
	}

	args := []string{
		"agent", "assign",
		"--type", agentType,
		"--discussion-id", ctx.DiscussionID,
		"--input", ctx.Input,
		"--config", e.defaultConfigPath(),
	}
	if strings.TrimSpace(action.Skill) != "" {
		args = append(args, "--skill", action.Skill)
	}
	return e.runCommandAction("respond", args)
}

func (e *Executor) executeSummarize(ctx HookContext, action HookAction) ActionResult {
	args := []string{
		"agent", "assign",
		"--type", "requirements",
		"--mode", "confirm",
		"--discussion-id", ctx.DiscussionID,
		"--input", ctx.Input,
		"--config", e.defaultConfigPath(),
	}
	if strings.TrimSpace(action.Skill) != "" {
		args = append(args, "--skill", action.Skill)
	}
	return e.runCommandAction("summarize", args)
}

func (e *Executor) executeGenerate(ctx HookContext, _ HookAction) ActionResult {
	if strings.TrimSpace(ctx.DiscussionID) == "" {
		return ActionResult{
			Action:  "generate",
			Success: false,
			Message: "discussion_id is required for generate action",
		}
	}

	args := []string{
		"doc", "generate",
		"--discussion", ctx.DiscussionID,
		"--config", e.defaultConfigPath(),
		"--create-pr",
	}
	return e.runCommandAction("generate", args)
}

func (e *Executor) executeIndexUpdate() ActionResult {
	if e.dryRun {
		return ActionResult{
			Action:  "index_update",
			Success: true,
			Message: "would run: index.NewBuilder(projectRoot).Build()+Save()",
		}
	}

	builder := index.NewBuilder(e.projectRoot)
	idx, err := builder.Build()
	if err != nil {
		return ActionResult{
			Action:  "index_update",
			Success: false,
			Message: fmt.Sprintf("index build failed: %v", err),
		}
	}
	if err := builder.Save(idx); err != nil {
		return ActionResult{
			Action:  "index_update",
			Success: false,
			Message: fmt.Sprintf("index save failed: %v", err),
		}
	}
	return ActionResult{
		Action:  "index_update",
		Success: true,
		Message: fmt.Sprintf("index updated: %d entries", len(idx.Entries)),
	}
}

func (e *Executor) executeSummaryUpdate() ActionResult {
	if e.dryRun {
		return ActionResult{
			Action:  "summary_update",
			Success: true,
			Message: "would run: summary cache update",
		}
	}

	builder := index.NewBuilder(e.projectRoot)
	idx, err := builder.LoadIndex()
	if err != nil {
		return ActionResult{
			Action:  "summary_update",
			Success: false,
			Message: fmt.Sprintf("load index failed: %v", err),
		}
	}

	cfg, err := cfgpkg.Load(e.defaultConfigPath())
	if err != nil {
		return ActionResult{
			Action:  "summary_update",
			Success: false,
			Message: fmt.Sprintf("load config failed: %v", err),
		}
	}

	providerName, model := cfgpkg.ResolveProviderForType(cfg, "requirements")
	provider, err := agent.NewProviderFromConfig(agent.ProviderConfig{
		Provider: providerName,
		Model:    model,
	})
	if err != nil {
		return ActionResult{
			Action:  "summary_update",
			Success: false,
			Message: fmt.Sprintf("create provider failed: %v", err),
		}
	}

	s := index.NewSummarizer(provider, filepath.Join(e.projectRoot, ".teraflow", "summaries"))
	updated, skipped, err := s.UpdateAll(context.Background(), idx)
	if err != nil {
		return ActionResult{
			Action:  "summary_update",
			Success: false,
			Message: fmt.Sprintf("summary update failed: %v", err),
		}
	}
	return ActionResult{
		Action:  "summary_update",
		Success: true,
		Message: fmt.Sprintf("summary updated: updated=%d skipped=%d", updated, skipped),
	}
}

func (e *Executor) runCommandAction(name string, args []string) ActionResult {
	cmdline := "teraflow " + strings.Join(args, " ")
	if e.dryRun {
		return ActionResult{
			Action:  name,
			Success: true,
			Message: "would run: " + cmdline,
		}
	}

	cmd := hookExecCommandContext(context.Background(), "teraflow", args...)
	cmd.Dir = e.projectRoot
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return ActionResult{
			Action:  name,
			Success: false,
			Message: fmt.Sprintf("command failed: %s", msg),
		}
	}
	return ActionResult{
		Action:  name,
		Success: true,
		Message: strings.TrimSpace(string(out)),
	}
}

func (e *Executor) defaultConfigPath() string {
	return filepath.Join(e.projectRoot, ".github", "teraflow.yml")
}
