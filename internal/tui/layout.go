package tui

const (
	contentPadding  = 1
	headerHeight    = 1
	footerHeight    = 1
	inputMinHeight  = 1
	spacerMinHeight = 0
	sidebarMaxWidth = 30
	sidebarMinWidth = 20
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

func calculateLayout(width, height, fixedHeight int) tuiLayout {
	width = max(1, width)
	height = max(1, height)

	if fixedHeight == 0 {
		fixedHeight = headerHeight + footerHeight + inputMinHeight
	}

	l := tuiLayout{
		width:        width,
		height:       height,
		hasSidebar:   false,
		isCompact:    width < 80,
		headerHeight: headerHeight,
		footerHeight: footerHeight,
		inputHeight:  fixedHeight,
	}

	l.mainWidth = width
	l.mainHeight = max(1, height-fixedHeight)
	l.contentWidth = max(8, l.mainWidth-contentPadding*2)
	l.contentHeight = max(3, l.mainHeight)

	return l
}

func calculateLayoutWithSidebar(width, height int, showSidebar bool) tuiLayout {
	width = max(1, width)
	height = max(1, height)

	l := tuiLayout{
		width:        width,
		height:       height,
		hasSidebar:   showSidebar,
		isCompact:    width < 80,
		headerHeight: headerHeight,
		footerHeight: footerHeight,
		inputHeight:  3, // prompt + input + autocomplete
	}

	if showSidebar && width >= sidebarMinWidth+20 {
		// Sidebar takes up to 1/3 of width, max 30 chars
		l.sidebarWidth = min(sidebarMaxWidth, width/3)
		l.sidebarHeight = max(1, height-4) // header + status + input + margins

		// Main content takes remaining space
		l.mainWidth = width - l.sidebarWidth - 1
		l.mainHeight = max(1, height-4)
	} else {
		l.sidebarWidth = 0
		l.sidebarHeight = 0
		l.mainWidth = width
		l.mainHeight = max(1, height-4)
	}

	l.contentWidth = max(8, l.mainWidth-contentPadding*2)
	l.contentHeight = max(3, l.mainHeight)

	return l
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func (l tuiLayout) mainPaneBounds() (width, height int) {
	return l.contentWidth, l.contentHeight
}

func (l tuiLayout) sidebarBounds() (width, height int) {
	return l.sidebarWidth, l.sidebarHeight
}

func (l tuiLayout) headerBounds() (width, height int) {
	return l.width, l.headerHeight
}

func (l tuiLayout) statusBounds() (width, height int) {
	return l.width, l.footerHeight
}

func (l tuiLayout) inputBounds() (width, height int) {
	return l.width, l.inputHeight
}
