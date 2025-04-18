package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

type remoteSpaceModel struct {
	titleStyle    lipgloss.Style
	disableKeymap bool
}

func initialRemoteSpaceModel() remoteSpaceModel {
	return remoteSpaceModel{
		titleStyle: titleStyle.Margin(0, 2),
	}
}

func (m remoteSpaceModel) Init() tea.Cmd {
	return nil
}

func (m remoteSpaceModel) Update(msg tea.Msg) (remoteSpaceModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.updateDimensions()

	case tea.KeyMsg:
		if m.disableKeymap {
			return m, nil
		}

	case spaceFocusSwitchMsg:
		if focusedSpace(msg) == remote {
			m.titleStyle = titleStyle.
				Background(highlightColor).
				Foreground(subduedHighlightColor)
		} else {
			m.titleStyle = titleStyle.
				Background(subduedGrayColor).
				Foreground(highlightColor)
		}
	}

	return m, nil
}

func (m remoteSpaceModel) View() string {
	s := m.titleStyle.Render("Remote Space")
	s = runewidth.Truncate(s, runewidth.StringWidth(s), "…")
	return smallContainerStyle.Render(s)
}

func (m *remoteSpaceModel) updateDimensions() {

}
