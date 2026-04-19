package app

import (
	"bytes"
	"testing"
	"time"

	"bytemind/internal/config"
)

func TestMaybePrintUpdateReminderSkipsWhenDisabled(t *testing.T) {
	t.Setenv("BYTEMIND_HOME", t.TempDir())

	cfg := config.Default(t.TempDir())
	cfg.UpdateCheck.Enabled = false

	previousVersion := buildVersion
	buildVersion = "v1.0.0"
	defer func() { buildVersion = previousVersion }()

	previousFetcher := updateCheckFetchLatestVersion
	calls := 0
	updateCheckFetchLatestVersion = func(currentVersion string) (string, error) {
		calls++
		return "v1.1.0", nil
	}
	defer func() { updateCheckFetchLatestVersion = previousFetcher }()

	var output bytes.Buffer
	maybePrintUpdateReminder(cfg, &output)

	if calls != 0 {
		t.Fatalf("expected disabled update check to skip fetch, got %d calls", calls)
	}
	if output.Len() != 0 {
		t.Fatalf("expected no output when update check disabled, got %q", output.String())
	}
}

func TestMaybePrintUpdateReminderChecksAtMostOncePerDay(t *testing.T) {
	t.Setenv("BYTEMIND_HOME", t.TempDir())

	cfg := config.Default(t.TempDir())
	cfg.UpdateCheck.Enabled = true

	previousVersion := buildVersion
	buildVersion = "v1.0.0"
	defer func() { buildVersion = previousVersion }()

	previousNow := updateCheckNow
	now := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)
	updateCheckNow = func() time.Time { return now }
	defer func() { updateCheckNow = previousNow }()

	previousFetcher := updateCheckFetchLatestVersion
	calls := 0
	updateCheckFetchLatestVersion = func(currentVersion string) (string, error) {
		calls++
		return "v1.1.0", nil
	}
	defer func() { updateCheckFetchLatestVersion = previousFetcher }()

	var first bytes.Buffer
	maybePrintUpdateReminder(cfg, &first)
	if calls != 1 {
		t.Fatalf("expected first run to fetch once, got %d", calls)
	}
	if first.Len() == 0 {
		t.Fatal("expected first run to print update reminder")
	}

	var second bytes.Buffer
	maybePrintUpdateReminder(cfg, &second)
	if calls != 1 {
		t.Fatalf("expected second run within cache window not to re-fetch, got %d", calls)
	}
	if second.Len() != 0 {
		t.Fatalf("expected no second reminder within cache window, got %q", second.String())
	}

	now = now.Add(25 * time.Hour)
	var third bytes.Buffer
	maybePrintUpdateReminder(cfg, &third)
	if calls != 2 {
		t.Fatalf("expected reminder to re-check after cache window, got %d fetches", calls)
	}
	if third.Len() == 0 {
		t.Fatal("expected reminder after cache window")
	}
}

func TestMaybePrintUpdateReminderSkipsDevVersion(t *testing.T) {
	t.Setenv("BYTEMIND_HOME", t.TempDir())

	cfg := config.Default(t.TempDir())
	cfg.UpdateCheck.Enabled = true

	previousVersion := buildVersion
	buildVersion = "dev"
	defer func() { buildVersion = previousVersion }()

	previousFetcher := updateCheckFetchLatestVersion
	calls := 0
	updateCheckFetchLatestVersion = func(currentVersion string) (string, error) {
		calls++
		return "v1.1.0", nil
	}
	defer func() { updateCheckFetchLatestVersion = previousFetcher }()

	var output bytes.Buffer
	maybePrintUpdateReminder(cfg, &output)
	if calls != 0 {
		t.Fatalf("expected dev version to skip update fetch, got %d calls", calls)
	}
	if output.Len() != 0 {
		t.Fatalf("expected no output for dev version, got %q", output.String())
	}
}
