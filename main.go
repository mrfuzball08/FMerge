package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type model struct {
	input string // input prompt
	width int // terminal width
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		key := msg.Key()
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		case "enter":
			m.input += "\n"
		default:
			if key.Text != "" {
				m.input += key.Text
			}
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12"))

	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(40)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	value := m.input
	if value == "" {
		value = "Type a prompt"
		inputStyle = inputStyle.Foreground(lipgloss.Color("8"))
	}

	block := lipgloss.JoinVertical(
		lipgloss.Left,
		labelStyle.Render("Input"),
		inputStyle.Render(value),
		helpStyle.Render("Type to edit. Backspace deletes. Ctrl+C to quit."),
	)

	if m.width > 0 {
		block = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, block)
	}

	return tea.NewView(block + "\n")
}

func main() {
	p := tea.NewProgram(model{})
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
