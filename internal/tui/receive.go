package tui

import tea "github.com/charmbracelet/bubbletea"

type receiveModel struct {
}

func (m receiveModel) Init() tea.Cmd {
	return nil
}

func (m receiveModel) Update(msg tea.Msg) (receiveModel, tea.Cmd) {
	return m, nil
}

func (m receiveModel) View() string {
	return ""
}

func (m *receiveModel) updateDimensions() {

}
