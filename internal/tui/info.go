package tui

import tea "github.com/charmbracelet/bubbletea"

type infoModel struct {
}

func (m infoModel) Init() tea.Cmd {
	return nil
}

func (m infoModel) Update(msg tea.Msg) (infoModel, tea.Cmd) {
	return m, nil
}

func (m infoModel) View() string {
	return ""
}
