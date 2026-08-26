package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestCalculateLayoutPreservesMainPaneOnNarrowTerminals(t *testing.T) {
	narrow := calculateLayout(80, 24, inputMinHeight)
	if narrow.sidebarWidth != 0 || narrow.mainWidth != 80 {
		t.Fatalf("narrow layout should not reserve a sidebar: %#v", narrow)
	}
	if narrow.contentWidth < 8 {
		t.Fatalf("narrow layout produced unusable content sizes: %#v", narrow)
	}

	wide := calculateLayout(140, 40, inputMinHeight)
	if wide.sidebarWidth != 0 || wide.mainWidth != 140 {
		t.Fatalf("wide layout should not reserve a sidebar: %#v", wide)
	}

	minimumWide := calculateLayout(200, 40, inputMinHeight)
	if minimumWide.sidebarWidth != 0 || minimumWide.mainWidth != 200 {
		t.Fatalf("wide layouts should not reserve a sidebar: %#v", minimumWide)
	}
}

func TestMatchHintsSupportsRecognitionWithoutChangingPrefixCompletion(t *testing.T) {
	prefix := matchHints("/hea")
	if len(prefix) != 1 || prefix[0].cmd != "/health" {
		t.Fatalf("expected /health prefix suggestion, got %#v", prefix)
	}

	intent := matchHints("/health")
	if len(intent) == 0 || intent[0].cmd != "/health" {
		t.Fatalf("expected a health-related visible action, got %#v", intent)
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

func TestTruncateVisiblePreservesANSIEscapeSequences(t *testing.T) {
	styled := lipgloss.NewStyle().Foreground(ColorAccent).Render("abcdef")
	got := truncateVisible(styled, 4)
	if ansi.StringWidth(got) > 4 {
		t.Fatalf("styled truncation width = %d, want <= 4: %q", ansi.StringWidth(got), got)
	}
	if ansi.Strip(got) != "abc…" {
		t.Fatalf("styled truncation text = %q, want %q", ansi.Strip(got), "abc…")
	}
}
