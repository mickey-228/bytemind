package tools

import (
	"path/filepath"
	"testing"
)

func TestResolvePathRejectsEscape(t *testing.T) {
	workspace := t.TempDir()
	if _, err := resolvePath(workspace, filepath.Join("..", "bad.txt")); err == nil {
		t.Fatal("expected path escape error")
	}
}

func TestResolvePathAllowsWorkspaceFile(t *testing.T) {
	workspace := t.TempDir()
	got, err := resolvePath(workspace, "ok.txt")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(got) != workspace {
		t.Fatalf("unexpected path: %s", got)
	}
}

func TestResolvePathAllowsConfiguredWritableRoot(t *testing.T) {
	workspace := t.TempDir()
	writableRoot := filepath.Join(t.TempDir(), "external-output")
	got, err := resolvePath(workspace, filepath.Join(writableRoot, "ok.txt"), writableRoot)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(got) != writableRoot {
		t.Fatalf("expected writable root path, got %s", got)
	}
}

func TestResolvePathRejectsOutsideWritableRoots(t *testing.T) {
	workspace := t.TempDir()
	writableRoot := filepath.Join(t.TempDir(), "external-output")
	anotherRoot := filepath.Join(t.TempDir(), "blocked-output")
	if _, err := resolvePath(workspace, filepath.Join(anotherRoot, "blocked.txt"), writableRoot); err == nil {
		t.Fatal("expected path outside writable roots to be rejected")
	}
}
