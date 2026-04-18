package agent

import (
	"fmt"
	"io"
	"sort"
	"strings"

	planpkg "bytemind/internal/plan"
)

var destructiveApprovalTools = map[string]struct{}{
	"apply_patch":     {},
	"replace_in_file": {},
	"write_file":      {},
}

func (r *Runner) renderApprovalPrecheck(out io.Writer, setup runPromptSetup) {
	if out == nil || r == nil || r.registry == nil {
		return
	}
	if setup.RunMode != planpkg.ModeBuild {
		return
	}
	toolNames := filteredToolNamesForMode(r.registry, setup.RunMode, setup.AllowedToolNames, setup.DeniedToolNames)
	summary := buildApprovalPrecheckSummary(approvalPrecheckSummaryInput{
		ToolNames:      toolNames,
		ApprovalPolicy: r.config.ApprovalPolicy,
		ApprovalMode:   r.config.ApprovalMode,
		AwayPolicy:     r.config.AwayPolicy,
	})
	if strings.TrimSpace(summary) == "" {
		return
	}
	_, _ = io.WriteString(out, summary)
}

type approvalPrecheckSummaryInput struct {
	ToolNames      []string
	ApprovalPolicy string
	ApprovalMode   string
	AwayPolicy     string
}

func buildApprovalPrecheckSummary(input approvalPrecheckSummaryInput) string {
	policy := strings.ToLower(strings.TrimSpace(input.ApprovalPolicy))
	if policy == "" {
		policy = "on-request"
	}
	if policy == "never" {
		return ""
	}

	toolSet := make(map[string]struct{}, len(input.ToolNames))
	for _, name := range input.ToolNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		toolSet[name] = struct{}{}
	}

	hasShell := false
	if _, ok := toolSet["run_shell"]; ok {
		hasShell = true
	}

	destructive := make([]string, 0, len(destructiveApprovalTools))
	for name := range destructiveApprovalTools {
		if _, ok := toolSet[name]; ok {
			destructive = append(destructive, name)
		}
	}
	sort.Strings(destructive)

	if !hasShell && len(destructive) == 0 {
		return ""
	}

	lines := []string{
		fmt.Sprintf("%sapproval precheck%s potential approval-required actions:", ansiDim, ansiReset),
	}

	if hasShell {
		if policy == "always" {
			lines = append(lines, "  - run_shell commands (approval_policy=always)")
		} else {
			lines = append(lines, "  - run_shell commands that are not read-only")
		}
	}

	if len(destructive) > 0 {
		lines = append(lines, fmt.Sprintf("  - workspace-modifying tools: %s", strings.Join(destructive, ", ")))
	}

	approvalMode := strings.ToLower(strings.TrimSpace(input.ApprovalMode))
	if approvalMode == "" {
		approvalMode = "interactive"
	}
	if approvalMode == "away" {
		awayPolicy := strings.ToLower(strings.TrimSpace(input.AwayPolicy))
		if awayPolicy == "" {
			awayPolicy = "auto_deny_continue"
		}
		lines = append(lines, fmt.Sprintf("  away mode: approvals are unavailable; matched actions will be denied (away_policy=%s)", awayPolicy))
		if awayPolicy == "fail_fast" {
			lines = append(lines, "  fail_fast: run stops after the first denied approval-required action")
		}
	} else {
		lines = append(lines, "  interactive mode: approvals are requested only when an action is actually attempted")
	}

	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func filteredToolNamesForMode(registry ToolRegistry, mode planpkg.AgentMode, allowlist, denylist []string) []string {
	if registry == nil {
		return nil
	}
	defs := registry.DefinitionsForModeWithFilters(mode, allowlist, denylist)
	if len(defs) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(defs))
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		name := strings.TrimSpace(def.Function.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
