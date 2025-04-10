package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

type receiveModel struct {
	titleStyle lipgloss.Style
}

func initialReceiveModel() receiveModel {
	return receiveModel{
		titleStyle: titleStyle.Margin(0, 2),
	}
}

func (m receiveModel) Init() tea.Cmd {
	return nil
}

func (m receiveModel) Update(msg tea.Msg) (receiveModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.updateDimensions()

	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "shift+tab":
			if currentFocus == receive {
				m.titleStyle = titleStyle.
					Background(highlightColor).
					Foreground(subduedHighlightColor)
			} else {
				m.titleStyle = titleStyle.
					Background(subduedGrayColor).
					Foreground(highlightColor)
			}
		}
	}
	return m, nil
}

func (m receiveModel) View() string {
	s := m.titleStyle.Render("RemoteSpace")
	s = runewidth.Truncate(s, runewidth.StringWidth(s), "…")
	return smallContainerStyle.Render(s)
}

func (m *receiveModel) updateDimensions() {

}
