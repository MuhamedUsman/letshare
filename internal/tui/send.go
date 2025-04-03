package tui

import tea "github.com/charmbracelet/bubbletea"

type sendModel struct {
	dirTreeModel dirTree
}

func (m sendModel) Init() tea.Cmd {
	return nil
}

func (m sendModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m sendModel) View() string {
	return ""
}
