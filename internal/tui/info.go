package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type infoModel struct {
	sendInfo      sendInfoModel
	titleStyle    lipgloss.Style
	extendedSpace focusedTab
}

func initialInfoModel() infoModel {
	return infoModel{
		sendInfo:   initialSendInfoModel(),
		titleStyle: titleStyle,
	}
}

func (m infoModel) Init() tea.Cmd {
	return m.sendInfo.Init()
}

func (m infoModel) Update(msg tea.Msg) (infoModel, tea.Cmd) {

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {

		case "esc":
			if currentFocus == info {
				m.extendedSpace = none
			}

		}

	case extendSpaceMsg:
		m.updateTitleStyleAsFocus(msg.focus)
		m.extendedSpace = msg.space
		if msg.focus {
			currentFocus = info
			return m, spaceTabSwitchMsg(info).cmd
		}

	case spaceTabSwitchMsg:
		if focusedTab(msg) == info {
			m.updateTitleStyleAsFocus(true)
		} else {
			m.updateTitleStyleAsFocus(false)
		}

	}
	return m, m.handleInfoModelUpdate(msg)
}

func (m infoModel) View() string {
	title := "Home Space"
	infoContent := banner.Height(infoContainerWorkableH()).Render()
	if m.extendedSpace == send {
		title = "Extended Local Space"
		infoContent = m.sendInfo.View()
	}
	if m.extendedSpace == receive {
		title = "Extended Remote Space"
		infoContent = ""
	}
	title = m.titleStyle.Render(title)
	return infoContainerStyle.
		Width(largeContainerW()).
		Height(termH-(mainContainerStyle.GetVerticalFrameSize())).
		Render(title, infoContent)
}

func (m *infoModel) handleInfoModelUpdate(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.sendInfo, cmd = m.sendInfo.Update(msg)
	return cmd
}

func (m *infoModel) updateTitleStyleAsFocus(focus bool) {
	if focus {
		m.titleStyle = titleStyle.
			Background(highlightColor).
			Foreground(subduedHighlightColor)
	} else {
		m.titleStyle = titleStyle.
			Background(subduedGrayColor).
			Foreground(highlightColor)
	}
}
