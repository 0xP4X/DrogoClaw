package tui

import (
	"testing"
)

func TestCalculateLayout(t *testing.T) {
	tests := []struct {
		name        string
		width       int
		height      int
		fixedHeight int
		wantMain    int
		wantContent int
	}{
		{
			name:        "Standard terminal",
			width:       80,
			height:      24,
			fixedHeight: 0,
			wantMain:    80,
			wantContent: 76, // width - padding
		},
		{
			name:        "Wide terminal",
			width:       120,
			height:      40,
			fixedHeight: 0,
			wantMain:    120,
			wantContent: 116,
		},
		{
			name:        "Narrow terminal",
			width:       40,
			height:      20,
			fixedHeight: 0,
			wantMain:    40,
			wantContent: 36,
		},
		{
			name:        "Minimum size",
			width:       1,
			height:      1,
			fixedHeight: 0,
			wantMain:    1,
			wantContent: 1, // Clamped to minimum
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout := calculateLayout(tt.width, tt.height, tt.fixedHeight)

			if layout.mainWidth != tt.wantMain {
				t.Errorf("mainWidth = %d, want %d", layout.mainWidth, tt.wantMain)
			}

			if layout.contentWidth < 1 {
				t.Error("contentWidth should be at least 1")
			}

			if layout.contentHeight < 1 {
				t.Error("contentHeight should be at least 1")
			}
		})
	}
}

func TestCalculateLayoutWithSidebar(t *testing.T) {
	tests := []struct {
		name            string
		width           int
		height          int
		showSidebar     bool
		wantHasSidebar  bool
		wantSidebarW    int
		minMainWidth    int
	}{
		{
			name:           "Wide with sidebar",
			width:          100,
			height:         30,
			showSidebar:    true,
			wantHasSidebar: true,
			wantSidebarW:   36, // min(36,100/3)=33 + padding*2+1 =36
			minMainWidth:   60,
		},
		{
			name:           "Wide without sidebar",
			width:          100,
			height:         30,
			showSidebar:    false,
			wantHasSidebar: false,
			wantSidebarW:   0,
			minMainWidth:   100,
		},
		{
			name:           "Narrow with sidebar request",
			width:          40,
			height:         20,
			showSidebar:    true,
			wantHasSidebar: false, // width < sidebarMinWidth+20 (44)
			wantSidebarW:   0,
			minMainWidth:   40,
		},
		{
			name:           "Minimum sidebar width",
			width:          50,
			height:         20,
			showSidebar:    true,
			wantHasSidebar: true,
			wantSidebarW:   19, // min(36,50/3=16)+3=19
			minMainWidth:   20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout := calculateLayoutWithSidebar(tt.width, tt.height, tt.showSidebar)

			if layout.hasSidebar != tt.wantHasSidebar {
				t.Errorf("hasSidebar = %v, want %v", layout.hasSidebar, tt.wantHasSidebar)
			}

			if layout.sidebarWidth != tt.wantSidebarW {
				t.Errorf("sidebarWidth = %d, want %d", layout.sidebarWidth, tt.wantSidebarW)
			}

			if layout.mainWidth < tt.minMainWidth {
				t.Errorf("mainWidth = %d, should be at least %d", layout.mainWidth, tt.minMainWidth)
			}

			// Verify total width adds up
			if tt.wantHasSidebar {
				totalWidth := layout.mainWidth + layout.sidebarWidth + sidebarGap
				if totalWidth > tt.width {
					t.Errorf("total width %d exceeds terminal width %d", totalWidth, tt.width)
				}
			}
		})
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		min      int
		max      int
		want     int
	}{
		{
			name:  "Within range",
			value: 50,
			min:   0,
			max:   100,
			want:  50,
		},
		{
			name:  "Below minimum",
			value: -10,
			min:   0,
			max:   100,
			want:  0,
		},
		{
			name:  "Above maximum",
			value: 150,
			min:   0,
			max:   100,
			want:  100,
		},
		{
			name:  "At minimum",
			value: 0,
			min:   0,
			max:   100,
			want:  0,
		},
		{
			name:  "At maximum",
			value: 100,
			min:   0,
			max:   100,
			want:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clamp(tt.value, tt.min, tt.max)
			if result != tt.want {
				t.Errorf("clamp(%d, %d, %d) = %d, want %d",
					tt.value, tt.min, tt.max, result, tt.want)
			}
		})
	}
}

func TestLayoutBounds(t *testing.T) {
	layout := calculateLayoutWithSidebar(100, 30, true)

	mainW, mainH := layout.mainPaneBounds()
	if mainW <= 0 || mainH <= 0 {
		t.Error("mainPaneBounds() should return positive values")
	}

	sidebarW, sidebarH := layout.sidebarBounds()
	if layout.hasSidebar {
		if sidebarW <= 0 || sidebarH <= 0 {
			t.Error("sidebarBounds() should return positive values when sidebar enabled")
		}
	}

	headerW, headerH := layout.headerBounds()
	if headerW <= 0 || headerH <= 0 {
		t.Error("headerBounds() should return positive values")
	}

	statusW, statusH := layout.statusBounds()
	if statusW <= 0 || statusH <= 0 {
		t.Error("statusBounds() should return positive values")
	}

	inputW, inputH := layout.inputBounds()
	if inputW <= 0 || inputH <= 0 {
		t.Error("inputBounds() should return positive values")
	}
}

func TestLayoutResponsiveness(t *testing.T) {
	widths := []int{40, 60, 80, 100, 120, 160}

	for _, width := range widths {
		layout := calculateLayoutWithSidebar(width, 30, true)

		if layout.width != width {
			t.Errorf("layout.width = %d, want %d", layout.width, width)
		}

		if width < sidebarMinWidth+20 {
			if layout.hasSidebar {
				t.Errorf("width %d: sidebar should be disabled when too narrow", width)
			}
		} else if !layout.hasSidebar {
			t.Errorf("width %d: sidebar should be enabled when wide enough", width)
		}

		// Invariant: widths must add up correctly, not strictly monotonic mainWidth
		if layout.hasSidebar {
			total := layout.mainWidth + layout.sidebarWidth + sidebarGap
			if total != width {
				t.Errorf("width %d: main(%d)+sidebar(%d)+gap(%d)=%d want %d", width, layout.mainWidth, layout.sidebarWidth, sidebarGap, total, width)
			}
		} else if layout.mainWidth != width {
			t.Errorf("width %d: without sidebar mainWidth=%d want %d", width, layout.mainWidth, width)
		}
	}
}

func BenchmarkCalculateLayout(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = calculateLayout(100, 30, 0)
	}
}

func BenchmarkCalculateLayoutWithSidebar(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = calculateLayoutWithSidebar(100, 30, true)
	}
}
