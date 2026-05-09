package tui

import (
	"strings"
	"testing"

	extensionast "github.com/yuin/goldmark/extension/ast"

	tea "github.com/charmbracelet/bubbletea"
)

func TestComputeColumnWidthsFallsBackToHardBreakWhenSpaceIsTight(t *testing.T) {
	table := markdownTableBlock{
		Header: []markdownTableCell{
			{Spans: []markdownSpan{{Text: "verylongheader"}}},
			{Spans: []markdownSpan{{Text: "anotherlongheader"}}},
		},
		Rows: [][]markdownTableCell{
			{
				{Spans: []markdownSpan{{Text: "supercalifragilistic"}}},
				{Spans: []markdownSpan{{Text: "pneumonoultramicroscopic"}}},
			},
		},
	}

	widths, hardBreak, ok := computeColumnWidths(table, 2, 24)
	if !ok {
		t.Fatal("expected computeColumnWidths to return widths")
	}
	if !hardBreak {
		t.Fatalf("expected hardBreak mode for tight width, got widths=%v", widths)
	}
	if len(widths) != 2 || widths[0] < 1 || widths[1] < 1 {
		t.Fatalf("expected positive widths for both columns, got %v", widths)
	}
}

func TestRenderMarkdownTableUsesStackedFallbackForNarrowWidth(t *testing.T) {
	table := markdownTableBlock{
		Header: []markdownTableCell{
			{Spans: []markdownSpan{{Text: "Name"}}},
			{Spans: []markdownSpan{{Text: "Notes"}}},
		},
		Rows: [][]markdownTableCell{
			{
				{Spans: []markdownSpan{{Text: "ByteMind"}}},
				{Spans: []markdownSpan{{Text: "compact terminal renderer output"}}},
			},
		},
		Align: []extensionast.Alignment{extensionast.AlignLeft, extensionast.AlignLeft},
	}

	lines := renderMarkdownTable(markdownSurfaceAssistant, table, 18)
	if len(lines) == 0 {
		t.Fatal("expected rendered table lines")
	}
	copyText := make([]string, 0, len(lines))
	for _, l := range lines {
		copyText = append(copyText, l.Copy)
	}
	joined := strings.Join(copyText, "\n")
	if !strings.Contains(joined, "Name: ByteMind") {
		t.Fatalf("expected stacked row label/value in copy output, got %q", joined)
	}
}

func TestWrapANSILinesHardBreakSkipsLeadingSpaceAfterFlush(t *testing.T) {
	lines := wrapANSILines("word next", 4, true)
	for _, line := range lines {
		if strings.TrimSpace(line) == "" && line != "" {
			t.Fatalf("expected no whitespace-only wrapped line, got %#v", lines)
		}
	}
}

func TestRenderSkillsModalAndApprovalHelpers(t *testing.T) {
	m := model{width: 80}
	skillsModal := m.renderSkillsModal()
	if !strings.Contains(skillsModal, "No loaded skills available.") {
		t.Fatalf("expected empty skills hint in modal, got %q", skillsModal)
	}

	if got := trimToWidth("abcdef", 3); got == "abcdef" {
		t.Fatalf("expected trimToWidth to truncate long text, got %q", got)
	}
	if got := trimToWidth("abcdef", 0); got != "" {
		t.Fatalf("expected trimToWidth with non-positive width to return empty, got %q", got)
	}

	selected := renderApprovalChoice("Enable", "warning", true)
	idle := renderApprovalChoice("Enable", "warning", false)
	if selected == idle {
		t.Fatalf("expected selected and idle approval choices to render differently")
	}
}

func TestHandleHiddenPasteProbeKeyFlushesOnClipboardMismatch(t *testing.T) {
	m := newImagePipelineModel(t)
	m.hiddenPasteProbe = hiddenPasteProbeState{
		active:    true,
		baseInput: "before ",
		clipboard: "prefix\nbody",
		buffered:  "prefix",
		flushID:   1,
	}

	consumed, cmd := m.handleHiddenPasteProbeKey(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("x"),
	})
	if consumed {
		t.Fatalf("expected mismatch to flush probe and not consume key")
	}
	if cmd != nil {
		t.Fatalf("expected mismatch flush to return nil cmd")
	}
	if m.hiddenPasteProbe.active {
		t.Fatalf("expected hidden probe to be cleared after mismatch")
	}
	if !strings.Contains(m.input.Value(), "prefix") {
		t.Fatalf("expected flushed buffered text to return to input, got %q", m.input.Value())
	}
}

func TestTryStartImplicitClipboardPasteFromKeyStartsProbeForSingleRune(t *testing.T) {
	m := newImagePipelineModel(t)
	m.input.SetValue("")
	m.clipboardRead = fakeClipboardTextReader{text: "你 line 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7\nline 8\nline 9\nline 10\nline 11\nline 12"}

	cmd := m.tryStartImplicitClipboardPasteFromKey(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("你"),
	})
	if cmd == nil {
		t.Fatal("expected single-rune implicit paste capture to schedule probe flush")
	}
	if !m.hiddenPasteProbe.active {
		t.Fatal("expected hidden paste probe to become active")
	}
}
