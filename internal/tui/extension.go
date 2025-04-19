package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type extensionSpaceModel struct {
	localExtension localExtensionModel
	titleStyle     lipgloss.Style
	extendedSpace  focusedSpace
	dirToExtend    string
	disableKeymap  bool
}

func initialExtensionSpaceModel() extensionSpaceModel {
	return extensionSpaceModel{
		localExtension: initialLocalExtensionModel(),
		titleStyle:     titleStyle,
	}
}

func (m extensionSpaceModel) Init() tea.Cmd {
	return m.localExtension.Init()
}

func (m extensionSpaceModel) Update(msg tea.Msg) (extensionSpaceModel, tea.Cmd) {

	switch msg := msg.(type) {

	case tea.KeyMsg:
		if m.disableKeymap {
			return m, nil
		}
		switch msg.String() {

		case "esc":
			// this must be run in a sequence, we need to get grant access on
			// current state of the localExtension model before it handles esc key msg
			return m, tea.Sequence(m.localExtension.grantExtensionSwitch(home), m.handleChildModelUpdate(msg))

		case "backspace":
			return m, tea.Sequence(m.localExtension.grantSpaceFocusSwitch(local), m.handleChildModelUpdate(msg))

		}

	case extendSpaceMsg:
		m.extendedSpace = msg.space
		if msg.space == home {
			m.disableKeymap = true
		}
		if msg.focus {
			currentFocus = extension
			return m, spaceFocusSwitchCmd
		}

	case spaceFocusSwitchMsg:
		if currentFocus == extension {
			m.updateTitleStyleAsFocus(true)
		} else {
			m.updateTitleStyleAsFocus(false)
		}

	}
	return m, m.handleChildModelUpdate(msg)
}

func (m extensionSpaceModel) View() string {
	style := infoContainerStyle.
		Width(largeContainerW()).
		Height(termH - (mainContainerStyle.GetVerticalFrameSize()))
	switch m.extendedSpace {
	case home:
		title := m.titleStyle.Render("Home Space")
		b := banner.Height(infoContainerWorkableH() - lipgloss.Height(title)).Render()
		return style.Render(lipgloss.JoinVertical(lipgloss.Center, title, b))
	case local:
		return style.Render(m.localExtension.View())
	default:
		return ""
	}
}

func (m *extensionSpaceModel) handleChildModelUpdate(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.localExtension, cmd = m.localExtension.Update(msg)
	return cmd
}

func (m *extensionSpaceModel) updateTitleStyleAsFocus(focus bool) {
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

func (m *extensionSpaceModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
	m.localExtension.updateKeymap(disable || m.extendedSpace == home)
}
