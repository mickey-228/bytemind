package tui

import (
	"context"
	"strings"
	"testing"

	"bytemind/internal/mcpctl"
)

type stubMCPService struct {
	listStatuses []mcpctl.ServerStatus
	lastEnableID string
	lastEnabled  bool
}

func (s *stubMCPService) List(context.Context) ([]mcpctl.ServerStatus, error) {
	out := make([]mcpctl.ServerStatus, len(s.listStatuses))
	copy(out, s.listStatuses)
	return out, nil
}

func (s *stubMCPService) Add(context.Context, mcpctl.AddRequest) (mcpctl.ServerStatus, error) {
	return mcpctl.ServerStatus{ID: "added", Enabled: true, Status: "ready"}, nil
}

func (s *stubMCPService) Remove(context.Context, string) error {
	return nil
}

func (s *stubMCPService) Enable(_ context.Context, serverID string, enabled bool) (mcpctl.ServerStatus, error) {
	s.lastEnableID = serverID
	s.lastEnabled = enabled
	return mcpctl.ServerStatus{ID: strings.TrimSpace(serverID), Enabled: enabled, Status: "ready"}, nil
}

func (s *stubMCPService) Test(context.Context, string) (mcpctl.ServerStatus, error) {
	return mcpctl.ServerStatus{ID: "local", Enabled: true, Status: "active", Message: "ok"}, nil
}

func (s *stubMCPService) Reload(context.Context) error {
	return nil
}

func TestRunMCPCommandList(t *testing.T) {
	service := &stubMCPService{
		listStatuses: []mcpctl.ServerStatus{
			{ID: "local", Enabled: true, Status: "active", Tools: 3, Message: "ok"},
		},
	}
	m := model{mcpService: service}
	if err := m.runMCPCommand("/mcp list", []string{"/mcp", "list"}); err != nil {
		t.Fatalf("runMCPCommand list failed: %v", err)
	}
	if len(m.chatItems) < 2 {
		t.Fatalf("expected command exchange in chat, got %#v", m.chatItems)
	}
	got := m.chatItems[len(m.chatItems)-1].Body
	if !strings.Contains(got, "local") || !strings.Contains(got, "active") {
		t.Fatalf("expected status output to include server and status, got %q", got)
	}
}

func TestRunMCPCommandEnable(t *testing.T) {
	service := &stubMCPService{}
	m := model{mcpService: service}
	if err := m.runMCPCommand("/mcp enable local", []string{"/mcp", "enable", "local"}); err != nil {
		t.Fatalf("runMCPCommand enable failed: %v", err)
	}
	if service.lastEnableID != "local" || !service.lastEnabled {
		t.Fatalf("expected enable call for local=true, got id=%q enabled=%v", service.lastEnableID, service.lastEnabled)
	}
}

func TestRunMCPCommandAddRequiresCommand(t *testing.T) {
	service := &stubMCPService{}
	m := model{mcpService: service}
	err := m.runMCPCommand("/mcp add local", []string{"/mcp", "add", "local"})
	if err == nil {
		t.Fatal("expected missing command error")
	}
	if !strings.Contains(err.Error(), "usage: /mcp add") {
		t.Fatalf("unexpected error: %v", err)
	}
}
