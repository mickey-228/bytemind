package tui

import (
	"strings"
	"testing"
)

func TestFormatChatBodyModeHandlesPasteMarkersForRenderAndCopy(t *testing.T) {
	item := chatEntry{
		Kind: "user",
		Body: "Please inspect [Paste #1 ~12 lines]",
	}

	rendered := stripANSI(formatChatBodyMode(item, 80, false))
	if !strings.Contains(rendered, "[Paste #1 ~12 lines] [click]") {
		t.Fatalf("expected rendered paste marker with click hint, got %q", rendered)
	}

	copied := stripANSI(formatChatBodyMode(item, 80, true))
	if !strings.Contains(copied, "[Paste #1 ~12 lines]") {
		t.Fatalf("expected copied paste marker text to be preserved, got %q", copied)
	}
	if strings.Contains(copied, "[click]") {
		t.Fatalf("expected copy-mode output to omit click hint, got %q", copied)
	}
}

func TestResolveUserBodyPastesRendersCollapsedPreviewAndFullModes(t *testing.T) {
	m := model{
		pastedContents: map[string]pastedContent{
			"1": {
				ID:      "1",
				Content: "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11\nline12",
				Lines:   12,
			},
		},
		pasteExpandLevel: map[string]int{},
	}

	collapsed := stripANSI(m.resolveUserBodyPastes("Before [Paste #1 ~12 lines] after"))
	if !strings.Contains(collapsed, "Before [Paste #1 ~12 lines] after") {
		t.Fatalf("expected collapsed paste body to preserve marker inline, got %q", collapsed)
	}

	m.pasteExpandLevel["1"] = 1
	preview := stripANSI(m.resolveUserBodyPastes("[Paste #1 ~12 lines]"))
	for _, want := range []string{"[Paste #1 ~12 lines] [preview]", "line1", "line10", "Ctrl+E expand all"} {
		if !strings.Contains(preview, want) {
			t.Fatalf("expected preview paste body to contain %q, got %q", want, preview)
		}
	}
	if strings.Contains(preview, "line12") {
		t.Fatalf("expected preview paste body to hide final lines, got %q", preview)
	}

	m.pasteExpandLevel["1"] = 2
	full := stripANSI(m.resolveUserBodyPastes("[Paste #1 ~12 lines]"))
	for _, want := range []string{"[Paste #1 ~12 lines] [full]", "line12", "click again to collapse"} {
		if !strings.Contains(full, want) {
			t.Fatalf("expected full paste body to contain %q, got %q", want, full)
		}
	}
}
