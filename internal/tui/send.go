package tui

import tea "github.com/charmbracelet/bubbletea"

type sendModel struct {
	dtm dirTreeModel
}

func initialSendModel() sendModel {
	return sendModel{dtm: initialDirTreeModel()}
}
func (m sendModel) Init() tea.Cmd {
	return tea.Batch(m.dtm.Init())
}

func (m sendModel) Update(msg tea.Msg) (sendModel, tea.Cmd) {
	var cmd tea.Cmd
	m.dtm, cmd = m.dtm.Update(msg)
	return m, cmd
}

func (m sendModel) View() string {
	return m.dtm.View()
}
