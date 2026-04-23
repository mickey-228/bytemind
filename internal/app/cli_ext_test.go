package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunExtHelpRendersUsage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := RunExt([]string{"help"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	output := stdout.String()
	if !strings.Contains(output, "bytemind ext list") || !strings.Contains(output, "bytemind ext status") {
		t.Fatalf("unexpected help output: %q", output)
	}
}

func TestRunExtListStatusAndUnload(t *testing.T) {
	workspace := t.TempDir()
	t.Setenv("BYTEMIND_HOME", t.TempDir())
	skillDir := filepath.Join(workspace, ".bytemind", "skills", "review")
	writeExtSkillFixture(t, skillDir, "review")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := RunExt([]string{"list", "--workspace", workspace}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("RunExt list failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "skill.review") {
		t.Fatalf("expected list output to include skill.review, got %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := RunExt([]string{"status", "skill.review", "--workspace", workspace}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("RunExt status failed: %v", err)
	}
	statusOutput := stdout.String()
	if !strings.Contains(statusOutput, "id: skill.review") || !strings.Contains(statusOutput, "status: active") {
		t.Fatalf("expected status output to include id and active status, got %q", statusOutput)
	}

	stdout.Reset()
	stderr.Reset()
	if err := RunExt([]string{"unload", "skill.review", "--workspace", workspace}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("RunExt unload failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "unloaded extension skill.review") {
		t.Fatalf("expected unload confirmation, got %q", stdout.String())
	}
}

func TestRunExtLoadExternalSource(t *testing.T) {
	workspace := t.TempDir()
	t.Setenv("BYTEMIND_HOME", t.TempDir())

	externalSource := filepath.Join(t.TempDir(), "docs")
	writeExtSkillFixture(t, externalSource, "docs")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := RunExt([]string{"load", externalSource, "--workspace", workspace}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("RunExt load failed: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "id: skill.docs") || !strings.Contains(output, "status: active") {
		t.Fatalf("expected load output to include skill.docs active, got %q", output)
	}
}

func writeExtSkillFixture(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skill.json"), []byte(`{"name":"`+name+`"}`), 0o644); err != nil {
		t.Fatalf("write skill.json failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# /"+name+"\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md failed: %v", err)
	}
}
