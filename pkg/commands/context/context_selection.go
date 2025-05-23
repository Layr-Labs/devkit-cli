package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// RunSelection is a variable alias for runSelection, so it can be stubbed in tests.
var RunSelection = runSelection

type model struct {
	Label    string
	Choices  []string
	cursor   int
	selected int
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "j":
			if m.cursor < len(m.Choices)-1 {
				m.cursor++
			}
			return m, nil
		case "enter", " ":
			m.selected = m.cursor
			return m, tea.Quit
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString(m.Label + "\n")
	b.WriteString("Use ↑/↓ to navigate, press space or enter to select\n\n")
	for i, choice := range m.Choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		check := " "
		if m.selected == i {
			check = "x"
		}
		fmt.Fprintf(&b, "%s [%s] %s\n", cursor, check, choice)
	}
	return b.String()
}

func runSelection(label string, opts []string) (string, error) {
	m := NewModel(label, opts)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}
	selIdx := finalModel.(model).selected
	if selIdx < 0 || selIdx >= len(opts) {
		return "", fmt.Errorf("no selection made")
	}
	return opts[selIdx], nil
}

// NewModel creates a new bubbletea model with a label and options
func NewModel(label string, choices []string) model {
	return model{
		Label:    label,
		Choices:  choices,
		selected: -1,
	}
}

// ListContexts reads YAML contexts and returns user's selection
func ListContexts(contextDir string, isList bool) ([]string, error) {
	entries, err := os.ReadDir(contextDir)
	if err != nil {
		return nil, fmt.Errorf("reading contexts dir: %w", err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := filepath.Ext(name)
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		names = append(names, strings.TrimSuffix(name, ext))
	}

	var operation = "set"
	if isList {
		operation = "list"
	}
	sel, err := RunSelection(fmt.Sprintf("Which context would you like to %s?", operation), names)
	if err != nil {
		return nil, err
	}
	return []string{sel}, nil
}
