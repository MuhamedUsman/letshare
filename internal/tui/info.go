package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type infoModel struct {
	sendInfo      sendInfoModel
	titleStyle    lipgloss.Style
	extendedSpace focusedTab
	dirToExtend   string
	hideTitle     bool
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
			// when title is hidden, the esc will be used in sendInfoModel
			if currentFocus == info && !m.hideTitle && m.sendInfo.filterState != filterApplied {
				m.extendedSpace = none
			}

		case "backspace":
			if currentFocus == info && !m.hideTitle {
				currentFocus = send
				return m, spaceFocusSwitchMsg(send).cmd
			}

		}

	case extendSpaceMsg:
		m.updateTitleStyleAsFocus(msg.focus)
		m.extendedSpace = msg.space
		if msg.focus {
			currentFocus = info
			return m, spaceFocusSwitchMsg(info).cmd
		}

	case spaceFocusSwitchMsg:
		if focusedTab(msg) == info {
			m.updateTitleStyleAsFocus(true)
		} else {
			m.updateTitleStyleAsFocus(false)
		}

	case hideInfoSpaceTitle:
		m.hideTitle = bool(msg)

	}
	return m, m.handleInfoModelUpdate(msg)
}

func (m infoModel) View() string {
	title := "Home Space"
	infoContent := banner.Height(infoContainerWorkableH(!m.hideTitle)).Render() // !m.hideTitle == showTitle
	if m.extendedSpace == send {
		title = "Extended Local Space"
		infoContent = m.sendInfo.View()
	}
	if m.extendedSpace == receive {
		title = "Extended Remote Space"
		infoContent = ""
	}
	title = m.titleStyle.Render(title)
	style := infoContainerStyle.
		Width(largeContainerW()).
		Height(termH - (mainContainerStyle.GetVerticalFrameSize()))
	if m.hideTitle {
		return style.Render(infoContent)
	}
	return style.Render(title, infoContent)
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
