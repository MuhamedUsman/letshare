package tui

import tea "github.com/charmbracelet/bubbletea"

type infoModel struct {
	sendInfo sendInfoModel
}

func initialInfoModel() infoModel {
	return infoModel{
		sendInfo: initialSendInfoModel(),
	}
}

func (m infoModel) Init() tea.Cmd {
	return m.sendInfo.Init()
}

func (m infoModel) Update(msg tea.Msg) (infoModel, tea.Cmd) {
	var cmd tea.Cmd
	m.sendInfo, cmd = m.sendInfo.Update(msg)
	return m, cmd
}

func (m infoModel) View() string {
	return infoContainerStyle.
		Width(largeContainerW()).
		Height(termH - (mainContainerStyle.GetVerticalFrameSize())).
		Render(m.sendInfo.View())
}
