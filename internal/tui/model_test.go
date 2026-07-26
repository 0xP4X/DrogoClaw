package tui

import "testing"

func TestCalculateLayoutPreservesMainPaneOnNarrowTerminals(t *testing.T) {
	narrow := calculateLayout(80, 24)
	if narrow.sidebarWidth != 0 || narrow.mainWidth != 80 {
		t.Fatalf("narrow layout should not reserve a sidebar: %#v", narrow)
	}
	if narrow.contentWidth < 8 || narrow.inputWidth < 8 {
		t.Fatalf("narrow layout produced unusable content sizes: %#v", narrow)
	}

	wide := calculateLayout(140, 40)
	if wide.sidebarWidth != 0 || wide.mainWidth != 140 {
		t.Fatalf("wide layout should preserve a full-width workspace: %#v", wide)
	}

	nearThreshold := calculateLayout(115, 24)
	if nearThreshold.sidebarWidth != 0 || nearThreshold.mainWidth != 115 {
		t.Fatalf("terminal below sidebar threshold should stay single-pane: %#v", nearThreshold)
	}

	minimumWide := calculateLayout(116, 24)
	if minimumWide.sidebarWidth != 0 || minimumWide.mainWidth != 116 {
		t.Fatalf("wide layouts should not reserve a permanent sidebar: %#v", minimumWide)
	}
}

func TestMatchHintsSupportsRecognitionWithoutChangingPrefixCompletion(t *testing.T) {
	prefix := matchHints("/hea")
	if len(prefix) != 1 || prefix[0].cmd != "/health" {
		t.Fatalf("expected /health prefix suggestion, got %#v", prefix)
	}

	intent := matchHints("/diagnostic")
	if len(intent) == 0 || intent[0].cmd != "/health" {
		t.Fatalf("expected a diagnostic-related visible action, got %#v", intent)
	}
}

func TestTruncateHandlesSmallAndUnicodeWidths(t *testing.T) {
	if got := truncate("abcdef", 2); got != "ab" {
		t.Fatalf("small truncation = %q, want %q", got, "ab")
	}
	if got := truncate("éclair", 4); got != "écl…" {
		t.Fatalf("unicode truncation = %q, want %q", got, "écl…")
	}
}
