package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type focus int

const (
	// No focus
	blur focus = iota - 1
	main
	send
	receive
	info
)

var curFocus focus

type MainModel struct {
	send    sendModel
	receive receiveModel
	info    infoModel
}

func (m MainModel) Init() tea.Cmd {
	return nil
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		terminalHeight = msg.Height
		terminalWidth = msg.Width

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m MainModel) View() string {
	container := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), true).
		Width(terminalWidth - 2).
		Height(terminalHeight - 2).
		Align(lipgloss.Center).
		Render()
	return lipgloss.Place(terminalWidth, terminalHeight, lipgloss.Center, lipgloss.Center, container)
}
