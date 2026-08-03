package tui

const (
	sidebarMinWidth   = 28
	sidebarMaxWidth   = 36
	contentPadding    = 2
	headerHeight      = 1
	footerHeight      = 1
	inputMinHeight    = 3
	spacerMinHeight   = 1
)

type tuiLayout struct {
	width         int
	height        int
	mainWidth     int
	mainHeight    int
	sidebarWidth  int
	sidebarHeight int
	contentWidth  int
	contentHeight int
	inputHeight   int
	headerHeight  int
	footerHeight  int
	hasSidebar    bool
	isCompact     bool
}

func calculateLayout(width, height int) tuiLayout {
	width = max(1, width)
	height = max(1, height)

	l := tuiLayout{
		width:        width,
		height:       height,
		hasSidebar:   width >= 120,
		isCompact:    width < 80,
		headerHeight: headerHeight,
		footerHeight: footerHeight,
		inputHeight:  inputMinHeight,
	}

	if l.hasSidebar {
		if width >= 140 {
			l.sidebarWidth = sidebarMaxWidth
		} else {
			l.sidebarWidth = sidebarMinWidth
		}
		l.mainWidth = width - l.sidebarWidth - 1
	} else {
		l.mainWidth = width
	}

	l.sidebarHeight = max(1, height-l.headerHeight-l.footerHeight-l.inputHeight-spacerMinHeight)
	l.mainHeight = max(1, height-l.headerHeight-l.footerHeight-l.inputHeight-spacerMinHeight)
	l.contentWidth = max(8, l.mainWidth-contentPadding*2)
	l.contentHeight = max(3, l.mainHeight)

	return l
}

func (l tuiLayout) mainPaneBounds() (width, height int) {
	return l.contentWidth, l.contentHeight
}

func (l tuiLayout) sidebarBounds() (width, height int) {
	if !l.hasSidebar {
		return 0, 0
	}
	return l.sidebarWidth, l.sidebarHeight
}
