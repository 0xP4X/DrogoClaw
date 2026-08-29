package tui

import (
	"strings"
	"testing"
)

func TestSlashCommandRegistryConsistent(t *testing.T) {
	seencan := make(map[string]bool)
	seenname := make(map[string]bool)
	for _, c := range slashCommands {
		if c.category != catOperations && c.category != catControls && c.category != catSession && c.category != catUI {
			t.Errorf("command %s has invalid category %q", c.canonical(), c.category)
		}
		if c.desc == "" {
			t.Errorf("command %s has no description", c.canonical())
		}
		if c.run == nil {
			t.Errorf("command %s has no handler", c.canonical())
		}
		if try := c.canonical(); try != c.names[0] {
			t.Errorf("canonical %s mismatch with names[0] %s", try, c.names[0])
		}
		if seencan[c.canonical()] {
			t.Errorf("duplicate canonical command %s", c.canonical())
		}
		seencan[c.canonical()] = true
		if len(c.names) == 0 {
			t.Fatalf("command entry with empty names")
		}
		for _, n := range c.names {
			if !strings.HasPrefix(n, "/") {
				t.Errorf("command name %q does not start with '/'", n)
			}
			if seenname[n] {
				t.Errorf("duplicate command name %q", n)
			}
			seenname[n] = true
		}
	}

	if !seenname["/help"] {
		t.Error("registry missing /help")
	}
	if !seenname["/commands"] {
		t.Error("registry missing /commands alias")
	}
	if !seenname["/exit"] || !seenname["/quit"] {
		t.Error("registry missing /exit or /quit alias")
	}
	if !seenname["/config"] {
		t.Error("registry missing /config")
	}

	if len(allHints) != len(slashCommands) {
		t.Errorf("palette hints (%d) drifted from registry (%d)", len(allHints), len(slashCommands))
	}
	for i := range allHints {
		if allHints[i].cmd != slashCommands[i].canonical() {
			t.Errorf("hint command %s mismatched with registry %s", allHints[i].cmd, slashCommands[i].canonical())
		}
	}
}

func TestHandleSlashCommandBareSlashShowsHelp(t *testing.T) {
	m := &Model{}
	m, _ = m.handleSlashCommand("/")
	if len(m.lines) == 0 {
		t.Fatal("expected help output appended to the transcript")
	}
	if !strings.Contains(strings.Join(m.lines, "\n"), "COMMAND REFERENCE") {
		t.Error("bare '/' did not render the command reference")
	}
}

func TestHandleSlashCommandUnknown(t *testing.T) {
	m := &Model{}
	m, cmd := m.handleSlashCommand("/definitely_not_a_command")
	if cmd != nil {
		t.Error("unknown command should not produce a tea.Cmd")
	}
	if len(m.lines) == 0 ||
		!strings.Contains(strings.Join(m.lines, "\n"), "Unknown command") {
		t.Error("unknown command did not surface a warning")
	}
}

func TestHandleSlashCommandAliasesResolveToSameEntry(t *testing.T) {
	var helpEntry *slashCommand
	for i := range slashCommands {
		if slashCommands[i].matches("/help") {
			helpEntry = &slashCommands[i]
			break
		}
	}
	if helpEntry == nil {
		t.Fatal("could not resolve /help entry")
	}
	if !helpEntry.matches("/commands") {
		t.Error("/commands should resolve to the same entry as /help")
	}
	for i := range slashCommands {
		if &slashCommands[i] == helpEntry {
			continue
		}
		if slashCommands[i].matches("/commands") {
			t.Error("/commands matched more than one registry entry")
		}
	}
}
