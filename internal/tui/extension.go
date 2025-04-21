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
		switch msg.String() {

		case "esc":
			// if preference is active, we need to switch back to the previous active child
			if m.activeChild == preference {
				// esc must be handled by the preference model, so it can deactivate itself
				// for more info see preferenceModel.Update method
				p, cmd := m.preference.Update(msg)
				m.preference = p
				if p.grantSpaceFocusSwitch() {
					return m, tea.Batch(cmd, extendChildMsg{child: m.prevActiveChild, focus: true}.cmd)
				}
			}
			// this must be run in a sequence, we need to get grant access on
			// current state of the localExtension model before it handles esc key msg
			return m, tea.Batch(m.grantExtensionSwitch(home), m.handleChildModelUpdate(msg))

		case "backspace":
		//return m, tea.Sequence(m.grantSpaceFocusSwitch(local), m.handleChildModelUpdate(msg))

		case "ctrl+p":
			return m, extendChildMsg{child: preference, focus: true}.cmd
		}

	case extendChildMsg:
		m.prevActiveChild = m.activeChild
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
	// For key messages that were captured, only update the capturing component
	if keyMsg, isKeyMsg := msg.(tea.KeyMsg); isKeyMsg {
		switch m.activeChild {
		case extDirNav:
			if m.extDirNav.capturesKeyEvent(keyMsg) {
				var cmd tea.Cmd
				m.extDirNav, cmd = m.extDirNav.Update(msg)
				return cmd
			}
		case preference:
			if m.preference.capturesKeyEvent(keyMsg) {
				var cmd tea.Cmd
				m.preference, cmd = m.preference.Update(msg)
				return cmd
			}
		case home:
			// Handle if there's a specific home component or logic
		}
	}

	// For non-key messages or uncaptured key events, update all components
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

func (m extensionSpaceModel) grantSpaceFocusSwitch() bool {
	switch m.activeChild {
	case home:
		return true
	case extDirNav:
		return m.extDirNav.grantSpaceFocusSwitch()
	case preference:
		return m.preference.grantSpaceFocusSwitch()
	default:
		return true
	}
}
