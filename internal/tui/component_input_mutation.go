package tui

import (
	"strings"
	"time"
)

func (m *model) noteInputMutation(before, after, source string) {
	now := time.Now()
	delta := len(after) - len(before)
	if delta < 0 {
		delta = 0
	}

	if now.Sub(m.lastInputAt) <= 80*time.Millisecond {
		m.inputBurstSize += max(1, delta)
	} else {
		m.inputBurstSize = max(1, delta)
	}
	m.lastInputAt = now

	if shouldRecordPasteSignal(before, after, source) ||
		(m.inputBurstSize >= 4 && isLikelyPathInput(strings.TrimSpace(after))) {
		m.lastPasteAt = now
	}
}

func shouldRecordPasteSignal(before, after, source string) bool {
	if source == "paste-enter" || isPasteLikeSource(source) {
		return true
	}
	_, inserted, _ := insertionDiff(before, after)
	inserted = strings.ReplaceAll(inserted, ctrlVMarkerRune, "")
	trimmed := strings.TrimSpace(inserted)
	if trimmed == "" {
		return false
	}
	if strings.Contains(inserted, "\n") {
		return true
	}
	return len(inserted) > 1 && len(trimmed) >= pasteBurstImmediateMinChars
}

func (m *model) handleInputMutation(before, after, source string) {
	m.noteInputMutation(before, after, source)

	updated, note := m.applyInputImagePipeline(before, after, source)
	if updated == after {
		fallbackUpdated, fallbackNote := m.applyWholeInputImagePathFallback(after, source)
		if fallbackUpdated != after {
			updated = fallbackUpdated
		}
		if strings.TrimSpace(note) == "" {
			note = fallbackNote
		}
	}

	pasteUpdated, pasteNote := m.applyLongPastedTextPipeline(before, updated, source)
	if pasteUpdated != updated {
		updated = pasteUpdated
	}
	if strings.TrimSpace(note) == "" {
		note = pasteNote
	}
	if locked, changed := m.protectCompressedMarkerChain(before, updated, source); changed {
		updated = locked
		if strings.TrimSpace(note) == "" {
			note = "Paste marker is locked to prevent accidental edits."
		}
	}

	if updated != after {
		m.setInputValue(updated)
	}
	if strings.TrimSpace(note) != "" {
		m.statusNote = note
	}
}

func lenCommonPrefix(a, b string) int {
	limit := min(len(a), len(b))
	for i := 0; i < limit; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return limit
}
