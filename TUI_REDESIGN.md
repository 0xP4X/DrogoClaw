# TUI Redesign Plan: DrogonClaw

## OpenCode TUI Analysis

OpenCode's TUI has these key design elements:
1. **Clean Header**: Minimal with session name, model info, and status
2. **Sidebar**: Always visible with session details, tools, and context
3. **Message Display**: Clean markdown rendering with proper spacing
4. **Status Bar**: Bottom bar with mode indicator, keybinds, and status
5. **Input Area**: Multi-line with autocomplete and file references
6. **Theme System**: Consistent color palette with dark/light support
7. **Leader Key**: ctrl+x as modifier for most actions
8. **Proper Borders**: Subtle borders separating sections

## Current DrogonClaw TUI Issues

1. **Header**: Too cluttered with runtime info
2. **Sidebar**: Hidden by default, not integrated properly
3. **Status Bar**: Minimal, missing keybind hints
4. **Input**: Basic textarea without proper styling
5. **Colors**: Inconsistent theme application
6. **Layout**: No proper panel separation

## Redesign Plan

### 1. New Layout Structure (view.go)

Create a proper 3-panel layout:
```
┌─────────────────────────────────────────────────────────────┐
│ HEADER: DrogonClaw · operator@agent · model · phase        │
├──────────────────────────────────────┬──────────────────────┤
│                                      │                      │
│  MAIN CONTENT AREA                   │  SIDEBAR             │
│  (messages, tool output, findings)   │  (session, tools,    │
│                                      │   memory, cost)      │
│                                      │                      │
├──────────────────────────────────────┴──────────────────────┤
│ STATUS BAR: [mode] · [phase] · [step] · [keybind hints]    │
├─────────────────────────────────────────────────────────────┤
│ INPUT: drogonclaw > [multi-line input with autocomplete]    │
└─────────────────────────────────────────────────────────────┘
```

### 2. Header Redesign (view.go:renderHeaderLine)

New format:
```
DrogonClaw · operator@agent · claude-3.5-sonnet · ● EXECUTING
```

- Left: Brand name (styled)
- Middle: Operator@Agent, Model name
- Right: Phase badge with icon

### 3. Sidebar Redesign (view.go:renderSidebar)

Always-visible sidebar (toggle with Ctrl+B) with sections:

```
┌─ SESSION ─────────────────────────┐
│ ID: abc123-def456                  │
│ Workflow: pentest                  │
│ Elapsed: 00:12:34                  │
│ Phase: ● EXECUTING                 │
├─ TOOLS ───────────────────────────┤
│ ▶ nmap -sV target.com            │
│ ✓ nuclei --target target.com     │
│ ▶ sqlmap --url ...               │
├─ FINDINGS ────────────────────────┤
│ 🔴 CVE-2024-XXXX (critical)     │
│ 🟡 admin:password123            │
│ 🟢 flag{...}                    │
├─ MEMORY ──────────────────────────┤
│ Entities: 24                      │
│ Relations: 12                     │
├─ COST ────────────────────────────┤
│ Tokens: 12,450                    │
│ Est. cost: $0.12                  │
└───────────────────────────────────┘
```

### 4. Status Bar Redesign (view.go:renderStatusBar)

New format:
```
[MANUAL] · [EXECUTING] · Step 3/10 · Ctrl+B sidebar · /help
```

- Left: Mode indicator (MANUAL/AUTOPILOT)
- Center: Phase, step progress
- Right: Keybind hints

### 5. Input Area Redesign (view.go:renderInputLine)

New format:
```
drogonclaw > [multi-line input with placeholder]
            [@ file reference] [Tab autocomplete]
```

- Proper prompt glyph
- Multi-line support
- File reference autocomplete with @
- Command autocomplete with /

### 6. Theme System (styles.go)

Create a consistent theme with:
- Primary: Blue (#58a6ff)
- Secondary: Green (#238636)
- Accent: Cyan (#39c5cf)
- Error: Red (#da3633)
- Warning: Yellow (#bf8700)
- Background: Dark (#0d1117)
- Panel: Slightly lighter (#161b22)
- Border: Subtle (#30363d)

### 7. Message Display (view.go:renderMessages)

Clean message formatting:
```
┌─ USER ─────────────────────────────────────────────────────┐
│ Profile example.com and identify vulnerabilities           │
└────────────────────────────────────────────────────────────┘

┌─ AGENT ────────────────────────────────────────────────────┐
│ I'll profile the target and identify potential attack      │
│ vectors. Let me start with reconnaissance...               │
│                                                            │
│ ▶ nmap -sV -sC example.com                               │
│ ✓ nmap (12.3s) — 3 ports open                            │
│                                                            │
│ Finding: OpenSSH 8.9 on port 22                           │
│ Finding: nginx 1.18 on port 80                            │
└────────────────────────────────────────────────────────────┘
```

### 8. Keybind System

Implement leader key (ctrl+x) with:
- ctrl+x b: Toggle sidebar
- ctrl+x n: New session
- ctrl+x l: List sessions
- ctrl+x m: List models
- ctrl+x t: List themes
- ctrl+x e: Open editor
- ctrl+x x: Export session
- ctrl+x q: Exit

### 9. Files to Modify

1. **internal/tui/view.go** - Complete rewrite of View(), renderHeaderLine(), renderSidebar(), renderStatusBar(), renderInputLine()
2. **internal/tui/styles.go** - New theme system with consistent colors
3. **internal/tui/model.go** - Add sidebar state, keybind state, theme state
4. **internal/tui/layout.go** - New layout calculation for 3-panel design
5. **internal/tui/commands.go** - Update commands for new keybinds
6. **internal/tui/messages.go** - Add new message types for sidebar updates

### 10. Implementation Order

1. Create new theme system (styles.go)
2. Rewrite layout calculation (layout.go)
3. Redesign header (view.go)
4. Redesign sidebar (view.go)
5. Redesign status bar (view.go)
6. Redesign input area (view.go)
7. Update message display (view.go)
8. Implement leader key system (model.go)
9. Update commands (commands.go)
10. Test and refine

### 11. Detailed Implementation Steps

#### Step 1: New Theme System (styles.go)

Replace the existing color definitions with a structured theme system:

```go
// Theme holds all color definitions for the TUI
type Theme struct {
    // Primary colors
    Primary   lipgloss.Color
    Secondary lipgloss.Color
    Accent    lipgloss.Color
    
    // Status colors
    Success lipgloss.Color
    Warning lipgloss.Color
    Error   lipgloss.Color
    Info    lipgloss.Color
    
    // Text colors
    Text     lipgloss.Color
    TextMuted lipgloss.Color
    TextDim  lipgloss.Color
    
    // Background colors
    Background      lipgloss.Color
    BackgroundPanel lipgloss.Color
    BackgroundSurface lipgloss.Color
    
    // Border colors
    Border        lipgloss.Color
    BorderActive  lipgloss.Color
    BorderSubtle  lipgloss.Color
}

// DefaultTheme is the dark theme matching OpenCode's style
var DefaultTheme = Theme{
    Primary:   lipgloss.Color("#58a6ff"),
    Secondary: lipgloss.Color("#238636"),
    Accent:    lipgloss.Color("#39c5cf"),
    
    Success: lipgloss.Color("#238636"),
    Warning: lipgloss.Color("#bf8700"),
    Error:   lipgloss.Color("#da3633"),
    Info:    lipgloss.Color("#58a6ff"),
    
    Text:      lipgloss.Color("#f0f6fc"),
    TextMuted: lipgloss.Color("#7d8590"),
    TextDim:   lipgloss.Color("#484f58"),
    
    Background:        lipgloss.Color("#0d1117"),
    BackgroundPanel:   lipgloss.Color("#161b22"),
    BackgroundSurface: lipgloss.Color("#21262d"),
    
    Border:       lipgloss.Color("#30363d"),
    BorderActive: lipgloss.Color("#58a6ff"),
    BorderSubtle: lipgloss.Color("#21262d"),
}

// Style helpers
func (t Theme) HeaderStyle() lipgloss.Style {
    return lipgloss.NewStyle().
        Foreground(t.Primary).
        Bold(true)
}

func (t Theme) PanelStyle() lipgloss.Style {
    return lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(t.Border).
        Padding(0, 1)
}
```

#### Step 2: Layout Calculation (layout.go)

Create a new layout system for 3-panel design:

```go
// Layout holds the calculated dimensions for all panels
type Layout struct {
    Header  Rect
    Content Rect
    Sidebar Rect
    Status  Rect
    Input   Rect
}

type Rect struct {
    X, Y, Width, Height int
}

func CalculateLayout(width, height int, showSidebar bool) Layout {
    layout := Layout{}
    
    // Header: full width, 1 line
    layout.Header = Rect{X: 0, Y: 0, Width: width, Height: 1}
    
    // Status bar: full width, 1 line
    layout.Status = Rect{X: 0, Y: height - 2, Width: width, Height: 1}
    
    // Input: full width, 3 lines (prompt + input + autocomplete)
    layout.Input = Rect{X: 0, Y: height - 1, Width: width, Height: 3}
    
    // Content area: between header and status
    contentHeight := height - 4 // header + status + input + margins
    
    if showSidebar {
        // Sidebar: 30 chars wide, right side
        sidebarWidth := min(30, width/3)
        layout.Sidebar = Rect{
            X: width - sidebarWidth,
            Y: 1,
            Width: sidebarWidth,
            Height: contentHeight,
        }
        
        // Content: remaining space
        layout.Content = Rect{
            X: 0,
            Y: 1,
            Width: width - sidebarWidth - 1,
            Height: contentHeight,
        }
    } else {
        // Full width content
        layout.Content = Rect{
            X: 0,
            Y: 1,
            Width: width,
            Height: contentHeight,
        }
        
        layout.Sidebar = Rect{X: 0, Y: 0, Width: 0, Height: 0}
    }
    
    return layout
}
```

#### Step 3: Header Redesign (view.go:renderHeaderLine)

New header format:
```go
func (m *Model) renderHeaderLine() string {
    var parts []string
    
    // Brand
    parts = append(parts, HeaderBrandStyle.Render("DrogonClaw"))
    
    // Separator
    parts = append(parts, SeparatorStyle.Render("·"))
    
    // Operator@Agent
    op := m.graph.GetOperatorProfile()
    ag := m.graph.GetAgentProfile()
    if op != nil && ag != nil {
        parts = append(parts, 
            InfoStyle.Render(fmt.Sprintf("%s@%s", op.Name, ag.Name)))
    }
    
    // Model name
    if m.model != "" {
        parts = append(parts, SeparatorStyle.Render("·"))
        parts = append(parts, ModelStyle.Render(m.model))
    }
    
    // Phase badge
    phaseIcon := m.phaseIcon()
    parts = append(parts, SeparatorStyle.Render("·"))
    parts = append(parts, PhaseStyle.Render(phaseIcon))
    
    return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
}
```

#### Step 4: Sidebar Redesign (view.go:renderSidebar)

New sidebar with sections:
```go
func (m *Model) renderSidebar(width, height int) string {
    var sections []string
    
    // Session section
    sessionSection := m.renderSidebarSection("SESSION", []string{
        fmt.Sprintf("ID: %s", m.sessionID[:8]),
        fmt.Sprintf("Mode: %s", m.mode),
        fmt.Sprintf("Time: %s", m.elapsed()),
    })
    sections = append(sections, sessionSection)
    
    // Tools section
    toolsSection := m.renderSidebarSection("TOOLS", m.recentTools())
    sections = append(sections, toolsSection)
    
    // Findings section
    findingsSection := m.renderSidebarSection("FINDINGS", m.findingsList())
    sections = append(sections, findingsSection)
    
    // Memory section
    memorySection := m.renderSidebarSection("MEMORY", []string{
        fmt.Sprintf("Entities: %d", m.entityCount()),
        fmt.Sprintf("Relations: %d", m.relationCount()),
    })
    sections = append(sections, memorySection)
    
    // Cost section
    costSection := m.renderSidebarSection("COST", []string{
        fmt.Sprintf("Tokens: %s", m.formatTokens()),
        fmt.Sprintf("Cost: $%.2f", m.cost),
    })
    sections = append(sections, costSection)
    
    return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m *Model) renderSidebarSection(title string, items []string) string {
    var lines []string
    
    // Section header
    lines = append(lines, SidebarTitleStyle.Render(fmt.Sprintf("─ %s %s", title, 
        strings.Repeat("─", max(0, 20-len(title))))))
    
    // Items
    for _, item := range items {
        lines = append(lines, SidebarItemStyle.Render("  "+item))
    }
    
    return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
```

#### Step 5: Status Bar Redesign (view.go:renderStatusBar)

New status bar format:
```go
func (m *Model) renderStatusBar() string {
    var parts []string
    
    // Mode indicator
    mode := "MANUAL"
    if m.autopilot {
        mode = "AUTOPILOT"
    }
    parts = append(parts, ModeStyle.Render(fmt.Sprintf("[%s]", mode)))
    
    // Separator
    parts = append(parts, SeparatorStyle.Render("·"))
    
    // Phase
    parts = append(parts, PhaseStyle.Render(m.phase))
    
    // Separator
    parts = append(parts, SeparatorStyle.Render("·"))
    
    // Step progress
    if m.totalSteps > 0 {
        parts = append(parts, StepStyle.Render(fmt.Sprintf("Step %d/%d", 
            m.stepCount, m.totalSteps)))
    }
    
    // Keybind hints (right-aligned)
    hints := []string{
        HintStyle.Render("Ctrl+B sidebar"),
        HintStyle.Render("/help"),
    }
    
    rightPart := lipgloss.JoinHorizontal(lipgloss.Right, hints...)
    parts = append(parts, rightPart)
    
    return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
}
```

#### Step 6: Input Area Redesign (view.go:renderInputLine)

New input format:
```go
func (m *Model) renderInputLine() string {
    var parts []string
    
    // Prompt glyph
    parts = append(parts, PromptStyle.Render("drogonclaw > "))
    
    // Input text
    if m.textarea.Value() == "" {
        parts = append(parts, PlaceholderStyle.Render("Type a message..."))
    } else {
        parts = append(parts, InputStyle.Render(m.textarea.Value()))
    }
    
    return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}
```

#### Step 7: Message Display (view.go:renderMessages)

Clean message formatting:
```go
func (m *Model) renderMessages() string {
    var messages []string
    
    for _, msg := range m.messages {
        switch msg.Role {
        case "user":
            messages = append(messages, m.renderUserMessage(msg))
        case "assistant":
            messages = append(messages, m.renderAgentMessage(msg))
        case "tool":
            messages = append(messages, m.renderToolMessage(msg))
        }
    }
    
    return lipgloss.JoinVertical(lipgloss.Left, messages...)
}

func (m *Model) renderUserMessage(msg Message) string {
    content := UserMessageStyle.Render(msg.Content)
    return PanelStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(BorderColor).
        Width(m.layout.Content.Width).
        Render(content)
}

func (m *Model) renderAgentMessage(msg Message) string {
    content := AgentMessageStyle.Render(msg.Content)
    return PanelStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(BorderActiveColor).
        Width(m.layout.Content.Width).
        Render(content)
}
```

#### Step 8: Leader Key System (model.go)

Implement leader key handling:
```go
// Leader key state
type LeaderState struct {
    Active    bool
    Timeout   time.Duration
    StartTime time.Time
}

func (m *Model) handleLeaderKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "ctrl+x":
        // Activate leader key
        m.leader.Active = true
        m.leader.StartTime = time.Now()
        return m, nil
        
    case "b":
        if m.leader.Active {
            m.showSidebar = !m.showSidebar
            m.leader.Active = false
            return m, nil
        }
        
    case "n":
        if m.leader.Active {
            // New session
            m.leader.Active = false
            return m, m.newSession()
        }
        
    // ... other leader key commands
    }
    
    // Check leader timeout
    if m.leader.Active && time.Since(m.leader.StartTime) > m.leader.Timeout {
        m.leader.Active = false
    }
    
    return m, nil
}
```

#### Step 9: Commands Update (commands.go)

Update command dispatch:
```go
func (m *Model) handleCommand(input string) (tea.Model, tea.Cmd) {
    parts := strings.Fields(input)
    if len(parts) == 0 {
        return m, nil
    }
    
    cmd := parts[0]
    args := parts[1:]
    
    switch cmd {
    case "/help":
        return m, m.showHelp()
    case "/timeline":
        return m, m.showTimeline()
    case "/details":
        return m, m.toggleToolDetails()
    case "/sidebar":
        m.showSidebar = !m.showSidebar
        return m, nil
    case "/new":
        return m, m.newSession()
    case "/sessions":
        return m, m.listSessions()
    case "/models":
        return m, m.listModels()
    case "/themes":
        return m, m.listThemes()
    case "/export":
        return m, m.exportSession()
    case "/quit":
        return m, tea.Quit
    default:
        return m, m.showError(fmt.Sprintf("Unknown command: %s", cmd))
    }
}
```

### 12. Visual Example

Full TUI layout:
```
┌─────────────────────────────────────────────────────────────┐
│  DrogonClaw · operator@agent · claude-3.5-sonnet · ● EXEC  │
├──────────────────────────────────────┬──────────────────────┤
│                                      │                      │
│  [Welcome Message]                   │  SESSION             │
│                                      │  ID: abc123          │
│  ▶ nmap -sV example.com             │  Mode: pentest       │
│  ✓ nmap (12.3s)                     │  Time: 00:12:34      │
│                                      │                      │
│  OpenSSH 8.9 on port 22             │  TOOLS               │
│  nginx 1.18 on port 80              │  ▶ nmap ✓ nuclei    │
│                                      │                      │
│  ▶ nuclei --target example.com      │  FINDINGS            │
│  ✓ nuclei (45.2s)                   │  🔴 CVE-2024-XXXX   │
│                                      │  🟡 admin:pass      │
│  CVE-2024-XXXX detected             │  🟢 flag{...}       │
│                                      │                      │
│                                      │  MEMORY              │
│                                      │  Entities: 24        │
│                                      │                      │
│                                      │  COST                │
│                                      │  $0.12               │
│                                      │                      │
├──────────────────────────────────────┴──────────────────────┤
│  [MANUAL] · [EXECUTING] · Step 3/10 · Ctrl+B sidebar      │
├─────────────────────────────────────────────────────────────┤
│  drogonclaw > profile example.com and identify vulns       │
└─────────────────────────────────────────────────────────────┘
```
