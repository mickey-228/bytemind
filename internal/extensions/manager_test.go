package extensions

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestManagerLoadDiscoversExtensionFromSource(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, ".bytemind", "skills")
	if err := os.MkdirAll(filepath.Join(project, "review"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "review", "skill.json"), []byte(`{"name":"review","description":"Review code changes","prompts":[{"id":"p1","path":"prompt.md"}],"tools":{"items":["read_file","search_text"]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "review", "SKILL.md"), []byte("# /review"), 0o644); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager(root)
	item, err := mgr.Load(context.Background(), filepath.Join(root, ".bytemind", "skills", "review"))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if item.ID != "skill.review" {
		t.Fatalf("unexpected id: %q", item.ID)
	}
	if item.Status != ExtensionStatusReady {
		t.Fatalf("unexpected status: %q", item.Status)
	}
	if item.Capabilities.Prompts != 1 || item.Capabilities.Tools != 2 {
		t.Fatalf("unexpected capabilities: %#v", item.Capabilities)
	}
}

func TestManagerListDiscoversAcrossScopesWithPriority(t *testing.T) {
	root := t.TempDir()
	builtin := filepath.Join(root, "builtin")
	user := filepath.Join(root, "user")
	project := filepath.Join(root, "project")
	for _, dir := range []string{
		filepath.Join(builtin, "review"),
		filepath.Join(user, "review"),
		filepath.Join(project, "review"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(builtin, "review", "skill.json"), []byte(`{"name":"review","description":"builtin"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(user, "review", "skill.json"), []byte(`{"name":"review","description":"user"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "review", "skill.json"), []byte(`{"name":"review","description":"project"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "review", "SKILL.md"), []byte("# /review"), 0o644); err != nil {
		t.Fatal(err)
	}

	mgr := NewManagerWithDirs(root, builtin, user, project)
	items, err := mgr.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 extension, got %d", len(items))
	}
	if items[0].Description != "project" {
		t.Fatalf("expected project scope to win, got %q", items[0].Description)
	}
	if items[0].Source.Scope != ExtensionScopeProject {
		t.Fatalf("unexpected scope: %q", items[0].Source.Scope)
	}
}

func TestManagerGetReturnsNotFound(t *testing.T) {
	mgr := NewManager(t.TempDir())
	item, err := mgr.Get(context.Background(), "skill.review")
	if item != (ExtensionInfo{}) {
		t.Fatal("expected zero extension info")
	}
	if err == nil {
		t.Fatal("expected not found error")
	}
	var extErr *ExtensionError
	if !errors.As(err, &extErr) {
		t.Fatalf("expected ExtensionError, got %T", err)
	}
	if extErr.Code != ErrCodeNotFound {
		t.Fatalf("unexpected code: %s", extErr.Code)
	}
}

func TestManagerUnloadRemovesLoadedExtension(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, ".bytemind", "skills")
	if err := os.MkdirAll(filepath.Join(project, "review"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "review", "skill.json"), []byte(`{"name":"review"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "review", "SKILL.md"), []byte("# /review"), 0o644); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager(root)
	if _, err := mgr.Load(context.Background(), filepath.Join(project, "review")); err != nil {
		t.Fatal(err)
	}
	impl, ok := mgr.(*extensionManager)
	if !ok {
		t.Fatalf("expected extensionManager, got %T", mgr)
	}
	if err := mgr.Unload(context.Background(), "skill.review"); err != nil {
		t.Fatalf("Unload failed: %v", err)
	}
	impl.mu.RLock()
	_, exists := impl.catalog["skill.review"]
	impl.mu.RUnlock()
	if exists {
		t.Fatal("expected loaded extension to be removed from manager cache")
	}
	items, err := mgr.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "skill.review" {
		t.Fatalf("expected discovery pipeline to rehydrate extension from disk, got %#v", items)
	}
}

func TestManagerRejectsInvalidSourceAndManifest(t *testing.T) {
	root := t.TempDir()
	mgr := NewManager(root)
	if _, err := mgr.Load(context.Background(), ""); err == nil {
		t.Fatal("expected invalid source error")
	}
	bad := filepath.Join(root, ".bytemind", "skills", "bad")
	if err := os.MkdirAll(bad, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bad, "skill.json"), []byte(`{"name":`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Load(context.Background(), bad); err == nil {
		t.Fatal("expected invalid manifest error")
	} else {
		var extErr *ExtensionError
		if !errors.As(err, &extErr) {
			t.Fatalf("expected ExtensionError, got %T", err)
		}
		if extErr.Code != ErrCodeInvalidManifest {
			t.Fatalf("unexpected code: %s", extErr.Code)
		}
	}
}

func TestNopManagerLoad(t *testing.T) {
	mgr := NopManager{}
	item, err := mgr.Load(nil, ".bytemind/skills/review")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if !item.IsZero() {
		t.Fatal("expected zero extension info")
	}
}

func TestNopManagerUnload(t *testing.T) {
	mgr := NopManager{}
	if err := mgr.Unload(nil, "skill.review"); err != nil {
		t.Fatalf("Unload failed: %v", err)
	}
}

func TestNopManagerGet(t *testing.T) {
	mgr := NopManager{}
	item, err := mgr.Get(nil, "skill.review")
	if item != (ExtensionInfo{}) {
		t.Fatal("expected zero extension info")
	}
	if err == nil {
		t.Fatal("expected not found error")
	}
	var extErr *ExtensionError
	if !errors.As(err, &extErr) {
		t.Fatalf("expected ExtensionError, got %T", err)
	}
	if extErr.Code != ErrCodeNotFound {
		t.Fatalf("unexpected code: %s", extErr.Code)
	}
}

func TestNopManagerList(t *testing.T) {
	mgr := NopManager{}
	items, err := mgr.List(nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected no extensions, got %d", len(items))
	}
}

func TestExtensionInfoValid(t *testing.T) {
	valid := ExtensionInfo{
		ID:   "skill.review",
		Name: "review",
		Kind: ExtensionSkill,
		Source: ExtensionSource{
			Scope: ExtensionScopeProject,
			Ref:   ".bytemind/skills/review",
		},
		Status:       ExtensionStatusReady,
		Capabilities: CapabilitySet{Prompts: 1, Tools: 2},
	}
	if !valid.Valid() {
		t.Fatal("expected extension info to be valid")
	}

	cases := []ExtensionInfo{
		{Name: "review", Kind: ExtensionSkill},
		{ID: "skill.review", Kind: ExtensionSkill},
		{ID: "skill.review", Name: "review"},
		{ID: "skill.review", Name: "review", Kind: ExtensionKind("unknown")},
		{ID: "skill.review", Name: "review", Kind: ExtensionSkill, Source: ExtensionSource{Ref: ".bytemind/skills/review"}},
		{ID: "skill.review", Name: "review", Kind: ExtensionSkill, Source: ExtensionSource{Scope: ExtensionScopeProject}},
		{ID: "skill.review", Name: "review", Kind: ExtensionSkill, Source: ExtensionSource{Scope: ExtensionScope("bad"), Ref: ".bytemind/skills/review"}},
	}
	for _, tc := range cases {
		if tc.Valid() {
			t.Fatalf("expected invalid extension info: %+v", tc)
		}
	}
}

func TestExtensionInfoIsZero(t *testing.T) {
	if !((ExtensionInfo{}).IsZero()) {
		t.Fatal("expected zero extension info")
	}

	cases := []ExtensionInfo{
		{ID: "skill.review"},
		{Version: "1.0.0"},
		{Title: "Review"},
		{Description: "desc"},
		{Source: ExtensionSource{Scope: ExtensionScopeProject}},
		{Source: ExtensionSource{Ref: ".bytemind/skills/review"}},
		{Capabilities: CapabilitySet{Tools: 1}},
		{Manifest: Manifest{Name: "review"}},
		{Manifest: Manifest{Kind: ExtensionSkill}},
		{Manifest: Manifest{Source: ExtensionSource{Ref: "manifest.json"}}},
		{Health: HealthSnapshot{Status: ExtensionStatusReady}},
		{Health: HealthSnapshot{Message: "ok"}},
		{Health: HealthSnapshot{LastError: ErrCodeLoadFailed}},
		{Health: HealthSnapshot{CheckedAtUTC: "2026-04-17T00:00:00Z"}},
		{Status: ExtensionStatusReady},
	}
	for _, tc := range cases {
		if tc.IsZero() {
			t.Fatalf("expected non-zero extension info: %+v", tc)
		}
	}
}

func TestExtensionErrorWrap(t *testing.T) {
	err := wrapError(ErrCodeLoadFailed, "load extension", nil)
	extErr, ok := err.(*ExtensionError)
	if !ok {
		t.Fatalf("expected ExtensionError, got %T", err)
	}
	if extErr.Code != ErrCodeLoadFailed {
		t.Fatalf("unexpected code: %s", extErr.Code)
	}
	if extErr.Message == "" {
		t.Fatal("expected message")
	}
	if extErr.Unwrap() != nil {
		t.Fatal("expected nil unwrap")
	}
	if extErr.CodeString() != string(ErrCodeLoadFailed) {
		t.Fatalf("unexpected code string: %q", extErr.CodeString())
	}
}

func TestExtensionErrorWithCause(t *testing.T) {
	cause := errors.New("boom")
	err := wrapError(ErrCodeUnloadFailed, "unload extension", cause)
	extErr, ok := err.(*ExtensionError)
	if !ok {
		t.Fatalf("expected ExtensionError, got %T", err)
	}
	if !errors.Is(extErr, cause) {
		t.Fatal("expected wrapped cause")
	}
	if extErr.Error() == "" {
		t.Fatal("expected error string")
	}
}

func TestNilExtensionErrorBehaviors(t *testing.T) {
	var err *ExtensionError
	if err.Error() != "" {
		t.Fatalf("expected empty error string, got %q", err.Error())
	}
	if err.Unwrap() != nil {
		t.Fatal("expected nil unwrap")
	}
	if err.CodeString() != "" {
		t.Fatalf("expected empty code string, got %q", err.CodeString())
	}
}
