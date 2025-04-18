package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type processSelectionModel struct {
}

func (m processSelectionModel) Init() tea.Cmd {
	return nil
}

func (m processSelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m processSelectionModel) View() string {
	return ""
}
