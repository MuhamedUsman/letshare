package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type extChild = int

const (
	home extChild = iota
	extDirNav
	preference
)

type extensionSpaceModel struct {
	extDirNav     extDirNavModel
	preference    preferenceModel
	titleStyle    lipgloss.Style
	extendedSpace focusedSpace
	dirToExtend   string
	activeChild   extChild
	disableKeymap bool
}

func initialExtensionSpaceModel() extensionSpaceModel {
	return extensionSpaceModel{
		extDirNav:   initialExtDirNavModel(),
		preference:  initialPreferenceModel(),
		titleStyle:  titleStyle,
		activeChild: extDirNav,
	}
}

func (m extensionSpaceModel) Init() tea.Cmd {
	return m.extDirNav.Init()
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
			return m, tea.Sequence(m.grantExtensionSwitch(home), m.handleChildModelUpdate(msg))

		case "backspace":
			return m, tea.Sequence(m.grantSpaceFocusSwitch(local), m.handleChildModelUpdate(msg))

		}

	case extendChildMsg:
		m.activeChild = msg.child
		if msg.child == home {
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
	style := largeContainerStyle.
		Width(largeContainerW()).
		Height(termH - (mainContainerStyle.GetVerticalFrameSize()))
	switch m.activeChild {
	case home:
		title := m.titleStyle.Render("Home Space")
		b := banner.Height(extContainerWorkableH() - lipgloss.Height(title)).Render()
		bt := lipgloss.JoinVertical(lipgloss.Center, title, b)
		return style.Render(lipgloss.PlaceHorizontal(largeContainerW(), lipgloss.Center, bt))
	case extDirNav:
		return style.Render(m.extDirNav.View())
	case preference:
		return style.Render(m.preference.View())
	default:
		return ""
	}
}

func (m *extensionSpaceModel) handleChildModelUpdate(msg tea.Msg) tea.Cmd {
	var cmds [2]tea.Cmd
	m.extDirNav, cmds[0] = m.extDirNav.Update(msg)
	m.preference, cmds[1] = m.preference.Update(msg)
	return tea.Batch(cmds[:]...)
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
	m.extDirNav.updateKeymap(disable || m.activeChild == home)
	//m.preference.updateKeymap(disable || m.extendedSpace == home)
}

func (m extensionSpaceModel) grantExtensionSwitch(child extChild) tea.Cmd {
	return m.extDirNav.grantExtensionSwitch(child)
}

func (m extensionSpaceModel) grantSpaceFocusSwitch(space focusedSpace) tea.Cmd {
	return m.extDirNav.grantSpaceFocusSwitch(space)
}
