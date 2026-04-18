package agent

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"bytemind/internal/config"
	"bytemind/internal/llm"
	planpkg "bytemind/internal/plan"
	"bytemind/internal/session"
	"bytemind/internal/tools"
)

func TestBuildApprovalPrecheckSummaryInteractive(t *testing.T) {
	summary := buildApprovalPrecheckSummary(approvalPrecheckSummaryInput{
		ToolNames:      []string{"list_files", "run_shell", "write_file", "apply_patch"},
		ApprovalPolicy: "on-request",
		ApprovalMode:   "interactive",
	})
	for _, want := range []string{
		"approval precheck",
		"run_shell",
		"workspace-modifying tools: apply_patch, write_file",
		"interactive mode",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("expected summary to contain %q, got %q", want, summary)
		}
	}
}

func TestBuildApprovalPrecheckSummaryAwayFailFast(t *testing.T) {
	summary := buildApprovalPrecheckSummary(approvalPrecheckSummaryInput{
		ToolNames:      []string{"run_shell"},
		ApprovalPolicy: "always",
		ApprovalMode:   "away",
		AwayPolicy:     "fail_fast",
	})
	for _, want := range []string{
		"approval precheck",
		"approval_policy=always",
		"away_policy=fail_fast",
		"fail_fast",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("expected summary to contain %q, got %q", want, summary)
		}
	}
}

func TestRunPromptWritesApprovalPrecheckNotice(t *testing.T) {
	workspace := t.TempDir()
	store, err := session.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sess := session.New(workspace)
	client := &fakeClient{replies: []llm.Message{{
		Role:    llm.RoleAssistant,
		Content: "done",
	}}}

	var out bytes.Buffer
	runner := NewRunner(Options{
		Workspace: workspace,
		Config: config.Config{
			Provider:       config.ProviderConfig{Model: "test-model"},
			ApprovalPolicy: "on-request",
			ApprovalMode:   "interactive",
			AwayPolicy:     "auto_deny_continue",
			MaxIterations:  2,
			Stream:         false,
		},
		Client:   client,
		Store:    store,
		Registry: tools.DefaultRegistry(),
		Stdin:    strings.NewReader(""),
		Stdout:   io.Discard,
	})

	answer, err := runner.RunPrompt(context.Background(), sess, "hello", string(planpkg.ModeBuild), &out)
	if err != nil {
		t.Fatal(err)
	}
	if answer != "done" {
		t.Fatalf("unexpected answer: %q", answer)
	}
	for _, want := range []string{
		"approval precheck",
		"run_shell",
		"workspace-modifying tools",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("expected output to contain %q, got %q", want, out.String())
		}
	}
}

func TestRunPromptSkipsApprovalPrecheckWhenPolicyNever(t *testing.T) {
	workspace := t.TempDir()
	store, err := session.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sess := session.New(workspace)
	client := &fakeClient{replies: []llm.Message{{
		Role:    llm.RoleAssistant,
		Content: "done",
	}}}

	var out bytes.Buffer
	runner := NewRunner(Options{
		Workspace: workspace,
		Config: config.Config{
			Provider:       config.ProviderConfig{Model: "test-model"},
			ApprovalPolicy: "never",
			ApprovalMode:   "interactive",
			MaxIterations:  2,
			Stream:         false,
		},
		Client:   client,
		Store:    store,
		Registry: tools.DefaultRegistry(),
		Stdin:    strings.NewReader(""),
		Stdout:   io.Discard,
	})

	_, err = runner.RunPrompt(context.Background(), sess, "hello", string(planpkg.ModeBuild), &out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "approval precheck") {
		t.Fatalf("did not expect approval precheck under policy never, got %q", out.String())
	}
}
