package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type infoModel struct {
	sendInfo   sendInfoModel
	titleStyle lipgloss.Style
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
	case tea.WindowSizeMsg:
		m.updateDimensions()
	case tea.KeyMsg:
		switch msg.String() {

		case "tab", "shift+tab":
			if currentFocus == info {
				m.titleStyle = titleStyle.
					Background(highlightColor).
					Foreground(subduedHighlightColor)
			} else {
				m.titleStyle = titleStyle.
					Background(subduedGrayColor).
					Foreground(highlightColor)
			}

		case "esc":
			if currentFocus == info {
				extendSpace, currentFocus = blur, blur
			}
		}
	}
	return m, m.handleInfoModelUpdate(msg)
}

func (m infoModel) View() string {
	title := "Home Space"
	if currentFocus == info {
		if extendSpace == send {
			title = "Extended Local Space"
		} else if extendSpace == receive {
			title = "Extended Remote Space"
		}
	}
	title = m.titleStyle.Render(title)
	infoContent := banner.Height(infoContainerWorkableH()).Render()
	if extendSpace == send {
		infoContent = m.sendInfo.View()
	}
	if extendSpace == receive {
		infoContent = ""
	}
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

func (m infoModel) updateDimensions() {

}
