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
	extDirNav                    extDirNavModel
	preference                   preferenceModel
	titleStyle                   lipgloss.Style
	extendedSpace                focusSpace
	dirToExtend                  string
	activeChild, prevActiveChild extChild
	disableKeymap                bool
}

func initialExtensionSpaceModel() extensionSpaceModel {
	return extensionSpaceModel{
		extDirNav:   initialExtDirNavModel(),
		preference:  initialPreferenceModel(),
		titleStyle:  titleStyle,
		activeChild: home,
	}
}

func (m extensionSpaceModel) capturesKeyEvent(msg tea.KeyMsg) bool {
	switch m.activeChild {
	case home:
		return false
	case extDirNav:
		return m.extDirNav.capturesKeyEvent(msg)
	case preference:
		return m.preference.capturesKeyEvent(msg)
	default:
		return false
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
		if m.capturesKeyEvent(msg) {
			// some child will capture the key event, let them handle it
			return m, m.handleChildModelUpdate(msg)
		}
		switch msg.String() {

		case "esc":
			return m, extendChildMsg{child: home, focus: true}.cmd

		case "backspace":
			//return m, tea.Sequence(m.grantSpaceFocusSwitch(local), m.handleChildModelUpdate(msg))

		}

	case extendChildMsg:
		if msg.child != m.activeChild {
			m.prevActiveChild = m.activeChild
			m.activeChild = msg.child
		}
		if msg.child == home {
			m.disableKeymap = true
		}
		if msg.focus {
			currentFocus = extension
			return m, tea.Batch(spaceFocusSwitchCmd, m.handleChildModelUpdate(msg))
		}

	case preferenceInactiveMsg:
		return m, extendChildMsg{child: m.prevActiveChild, focus: true}.cmd

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
	var cmd tea.Cmd
	switch m.activeChild {
	case extDirNav:
		m.extDirNav, cmd = m.extDirNav.Update(msg)
		return cmd
	case preference:
		m.preference, cmd = m.preference.Update(msg)
		return cmd
	default:
		var cmds [2]tea.Cmd
		m.extDirNav, cmds[0] = m.extDirNav.Update(msg)
		m.preference, cmds[1] = m.preference.Update(msg)
		return tea.Batch(cmds[:]...)
	}
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
	m.preference.updateKeymap(disable || m.activeChild == home)
}
