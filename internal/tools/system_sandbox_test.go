package tools

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateSystemSandboxRuntimeWithSkipsWhenSandboxDisabled(t *testing.T) {
	if err := validateSystemSandboxRuntimeWith(false, "required", "windows", func(string) (string, error) {
		t.Fatal("lookPath should not be called when sandbox is disabled")
		return "", nil
	}); err != nil {
		t.Fatalf("expected disabled sandbox to skip system sandbox validation, got %v", err)
	}
}

func TestValidateSystemSandboxRuntimeWithSkipsWhenModeOff(t *testing.T) {
	if err := validateSystemSandboxRuntimeWith(true, "off", "linux", func(string) (string, error) {
		t.Fatal("lookPath should not be called when mode is off")
		return "", nil
	}); err != nil {
		t.Fatalf("expected mode=off to skip validation, got %v", err)
	}
}

func TestValidateSystemSandboxRuntimeWithFailsClosedForRequiredModeWithoutBackend(t *testing.T) {
	err := validateSystemSandboxRuntimeWith(true, "required", "windows", func(string) (string, error) {
		t.Fatal("lookPath should not be called on unsupported OS")
		return "", nil
	})
	if err == nil {
		t.Fatal("expected required mode to fail closed when backend is unavailable")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSystemSandboxRuntimeWithBestEffortFallsBackWithoutBackend(t *testing.T) {
	err := validateSystemSandboxRuntimeWith(true, "best_effort", "linux", func(string) (string, error) {
		return "", errors.New("missing unshare")
	})
	if err != nil {
		t.Fatalf("expected best_effort mode to allow fallback, got %v", err)
	}
}

func TestValidateSystemSandboxRuntimeWithRequiredPassesWhenBackendAvailable(t *testing.T) {
	err := validateSystemSandboxRuntimeWith(true, "required", "linux", func(name string) (string, error) {
		if name != "unshare" {
			t.Fatalf("unexpected binary lookup: %q", name)
		}
		return "/usr/bin/unshare", nil
	})
	if err != nil {
		t.Fatalf("expected required mode to pass with backend available, got %v", err)
	}
}
