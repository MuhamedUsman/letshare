package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type sendModel struct {
	dtm dirListModel
}

func initialSendModel() sendModel {
	return sendModel{dtm: initialDirListModel()}
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
	s := sendContainerStyle.
		Height(termH - mainContainerStyle.GetVerticalFrameSize()).
		Width(smallContainerW())
	return s.Render(m.dtm.View())
}
