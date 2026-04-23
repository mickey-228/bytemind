package extensions

import (
	"testing"
	"time"
)

func TestHealthManagerIsolationOpensCircuitAtThreshold(t *testing.T) {
	now := time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC)
	manager := NewHealthManager(IsolationPolicy{
		FailureThreshold: 2,
		RecoveryCooldown: 10 * time.Second,
	}, WithHealthManagerClock(func() time.Time {
		return now
	}))

	if !manager.AllowProbe("mcp.docs") {
		t.Fatal("expected closed circuit to allow probe")
	}
	first := manager.RecordFailure("mcp.docs")
	if first.CircuitState != CircuitClosed || first.FailureCount != 1 {
		t.Fatalf("expected first failure to stay closed, got %#v", first)
	}
	second := manager.RecordFailure("mcp.docs")
	if second.CircuitState != CircuitOpen || second.FailureCount != 2 {
		t.Fatalf("expected threshold failure to open circuit, got %#v", second)
	}
	if second.NextRetryAtUTC == "" {
		t.Fatalf("expected open circuit to expose next retry time, got %#v", second)
	}
	if manager.AllowProbe("mcp.docs") {
		t.Fatal("expected open circuit to block probe before cooldown")
	}
}

func TestHealthManagerRecoveryTransitionsHalfOpenToClosedOnSuccess(t *testing.T) {
	now := time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC)
	manager := NewHealthManager(IsolationPolicy{
		FailureThreshold: 2,
		RecoveryCooldown: 10 * time.Second,
	}, WithHealthManagerClock(func() time.Time {
		return now
	}))

	manager.RecordFailure("mcp.docs")
	manager.RecordFailure("mcp.docs")
	now = now.Add(11 * time.Second)

	if !manager.AllowProbe("mcp.docs") {
		t.Fatal("expected cooldown elapsed probe to enter half-open")
	}
	halfOpen := manager.Snapshot("mcp.docs")
	if halfOpen.CircuitState != CircuitHalfOpen {
		t.Fatalf("expected half-open state after probe allowance, got %#v", halfOpen)
	}

	recovered := manager.RecordSuccess("mcp.docs")
	if recovered.CircuitState != CircuitClosed || recovered.FailureCount != 0 {
		t.Fatalf("expected successful probe to close circuit, got %#v", recovered)
	}
}

func TestHealthManagerRecoveryExtendsCooldownOnHalfOpenFailure(t *testing.T) {
	now := time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC)
	manager := NewHealthManager(IsolationPolicy{
		FailureThreshold: 2,
		RecoveryCooldown: 10 * time.Second,
	}, WithHealthManagerClock(func() time.Time {
		return now
	}))

	manager.RecordFailure("mcp.docs")
	openSnapshot := manager.RecordFailure("mcp.docs")
	firstRetry, err := time.Parse(time.RFC3339, openSnapshot.NextRetryAtUTC)
	if err != nil {
		t.Fatalf("expected valid first retry time, got %q (%v)", openSnapshot.NextRetryAtUTC, err)
	}
	now = firstRetry.Add(time.Second)

	if !manager.AllowProbe("mcp.docs") {
		t.Fatal("expected probe after first cooldown")
	}
	reopened := manager.RecordFailure("mcp.docs")
	secondRetry, err := time.Parse(time.RFC3339, reopened.NextRetryAtUTC)
	if err != nil {
		t.Fatalf("expected valid second retry time, got %q (%v)", reopened.NextRetryAtUTC, err)
	}
	if !secondRetry.After(firstRetry) {
		t.Fatalf("expected second retry to be later after cooldown extension, first=%s second=%s", firstRetry, secondRetry)
	}
	if reopened.CircuitState != CircuitOpen {
		t.Fatalf("expected half-open failure to reopen circuit, got %#v", reopened)
	}
}
