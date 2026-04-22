package tools

import (
	"fmt"
	"runtime"
	"strings"
)

// ValidateSystemSandboxRuntime verifies whether the configured system sandbox mode
// is usable in the current runtime. It is fail-closed for required mode.
func ValidateSystemSandboxRuntime(sandboxEnabled bool, mode string) error {
	return validateSystemSandboxRuntimeWith(sandboxEnabled, mode, runtime.GOOS, runShellLookPath)
}

func validateSystemSandboxRuntimeWith(
	sandboxEnabled bool,
	mode string,
	goos string,
	lookPath func(string) (string, error),
) error {
	if !sandboxEnabled {
		return nil
	}
	normalized := normalizeSystemSandboxMode(&ExecutionContext{SystemSandboxMode: mode})
	if normalized == systemSandboxModeOff {
		return nil
	}
	if _, err := resolveSystemSandboxBackend(normalized, strings.TrimSpace(goos), lookPath); err != nil {
		return fmt.Errorf("system sandbox backend unavailable for mode %q: %w", normalized, err)
	}
	return nil
}
