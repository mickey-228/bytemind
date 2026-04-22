package tools

import (
	"errors"
	"strings"
	"testing"
)

func TestResolveSystemSandboxRuntimeBackendRequiredFailsOnUnsupportedOS(t *testing.T) {
	_, err := resolveSystemSandboxRuntimeBackend(systemSandboxModeRequired, "windows", func(string) (string, error) {
		t.Fatal("lookPath should not be called for unsupported OS")
		return "", nil
	})
	if err == nil {
		t.Fatal("expected required mode to fail on unsupported OS")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveSystemSandboxRuntimeBackendBestEffortFallsBackWhenLinuxBackendMissing(t *testing.T) {
	backend, err := resolveSystemSandboxRuntimeBackend(systemSandboxModeBestEffort, "linux", func(string) (string, error) {
		return "", errors.New("not found")
	})
	if err != nil {
		t.Fatalf("expected best_effort fallback, got %v", err)
	}
	if backend.Enabled {
		t.Fatalf("expected disabled backend when probe fails in best_effort, got %#v", backend)
	}
}

func TestResolveSystemSandboxRuntimeBackendLinuxProfiles(t *testing.T) {
	backend, err := resolveSystemSandboxRuntimeBackend(systemSandboxModeRequired, "linux", func(name string) (string, error) {
		if name != "unshare" {
			t.Fatalf("unexpected binary lookup: %q", name)
		}
		return "/usr/bin/unshare", nil
	})
	if err != nil {
		t.Fatalf("resolve backend: %v", err)
	}
	if !backend.Enabled {
		t.Fatalf("expected enabled backend, got %#v", backend)
	}
	if backend.Name != "linux_unshare" {
		t.Fatalf("unexpected backend name: %#v", backend)
	}
	if !containsString(backend.Shell.ArgPrefix, "--net") {
		t.Fatalf("shell launch should include --net isolation, got %#v", backend.Shell.ArgPrefix)
	}
	if containsString(backend.Worker.ArgPrefix, "--net") {
		t.Fatalf("worker launch should not include --net isolation, got %#v", backend.Worker.ArgPrefix)
	}
	if !backend.Shell.Policy.NetworkIsolation {
		t.Fatalf("expected shell policy network isolation, got %#v", backend.Shell.Policy)
	}
	if backend.Worker.Policy.NetworkIsolation {
		t.Fatalf("expected worker policy without network isolation, got %#v", backend.Worker.Policy)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
