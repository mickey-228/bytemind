package policy

import (
	"testing"

	planpkg "bytemind/internal/plan"
)

func TestEvaluatePromptHintInjected(t *testing.T) {
	result := EvaluatePromptHint("Find implementation in GitHub repository")
	if result.Decision != PromptHintDecisionHint {
		t.Fatalf("expected hint decision, got %#v", result)
	}
	if result.ReasonCode != ReasonWebLookupHintInjected {
		t.Fatalf("expected web hint reason code, got %#v", result)
	}
	if result.Instruction == "" {
		t.Fatalf("expected non-empty instruction, got %#v", result)
	}
}

func TestEvaluatePromptHintSkipped(t *testing.T) {
	result := EvaluatePromptHint("Use search_text in current workspace")
	if result.Decision != PromptHintDecisionNone {
		t.Fatalf("expected no hint decision, got %#v", result)
	}
	if result.ReasonCode != ReasonPromptHintSkipped {
		t.Fatalf("expected hint skipped reason code, got %#v", result)
	}
}

func TestEvaluateRunShellDangerousCommandBlocksBeforeApproval(t *testing.T) {
	result := Evaluate(EvaluateInput{
		ToolName:       "run_shell",
		ShellCommand:   "rm -rf .",
		Mode:           planpkg.ModeBuild,
		ApprovalPolicy: "always",
	})
	if result.MainDecision != MainDecisionDeny {
		t.Fatalf("expected deny for dangerous command, got %#v", result)
	}
	if result.MainReasonCode != ReasonDangerousCommandBlocked {
		t.Fatalf("expected dangerous command reason code, got %#v", result)
	}
}

func TestEvaluateRunShellPlanModeReturnsPlanBlockedReasonCode(t *testing.T) {
	result := Evaluate(EvaluateInput{
		ToolName:       "run_shell",
		ShellCommand:   "go test ./...",
		Mode:           planpkg.ModePlan,
		ApprovalPolicy: "always",
	})
	if result.MainDecision != MainDecisionDeny {
		t.Fatalf("expected deny in plan mode, got %#v", result)
	}
	if result.MainReasonCode != ReasonPlanModeToolBlocked {
		t.Fatalf("expected plan blocked reason code, got %#v", result)
	}
}

func TestEvaluateRunShellOnRequestEscalatesForRiskyCommand(t *testing.T) {
	result := Evaluate(EvaluateInput{
		ToolName:       "run_shell",
		ShellCommand:   "go test ./...",
		Mode:           planpkg.ModeBuild,
		ApprovalPolicy: "on-request",
	})
	if result.MainDecision != MainDecisionEscalate {
		t.Fatalf("expected escalate for risky command, got %#v", result)
	}
	if result.MainReasonCode != ReasonDestructiveToolRequiresApproval {
		t.Fatalf("expected approval reason code, got %#v", result)
	}
}

func TestEvaluateDenylistOverridesAllowlist(t *testing.T) {
	result := Evaluate(EvaluateInput{
		ToolName: "read_file",
		Allowed:  map[string]struct{}{"read_file": {}},
		Denied:   map[string]struct{}{"read_file": {}},
	})
	if result.MainDecision != MainDecisionDeny {
		t.Fatalf("expected deny when denylist and allowlist both include tool, got %#v", result)
	}
	if result.MainReasonCode != ReasonToolDeniedByDenylist {
		t.Fatalf("expected denylist reason code, got %#v", result)
	}
}

func TestEvaluateHintDoesNotChangeMainDecision(t *testing.T) {
	result := Evaluate(EvaluateInput{
		ToolName:  "read_file",
		UserInput: "Find implementation in GitHub repository",
	})
	if result.MainDecision != MainDecisionAllow {
		t.Fatalf("expected main decision allow, got %#v", result)
	}
	if result.PromptHint.Decision != PromptHintDecisionHint {
		t.Fatalf("expected prompt hint to be injected, got %#v", result.PromptHint)
	}
}

func TestEvaluateSkipRuntimeChecksForRunShell(t *testing.T) {
	result := Evaluate(EvaluateInput{
		ToolName:          "run_shell",
		ShellCommand:      "rm -rf .",
		SkipRuntimeChecks: true,
	})
	if result.MainDecision != MainDecisionAllow {
		t.Fatalf("expected runtime checks to be skipped, got %#v", result)
	}
}
