package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type extChild = int

const (
	home extChild = iota
	extDirNav
	extSend
	extReceive
	preference
	download
)

type extensionSpaceModel struct {
	extDirNav                    extDirNavModel
	extSend                      extSendModel
	extReceive                   extReceiveModel
	preference                   preferenceModel
	download                     downloadModel
	titleStyle                   lipgloss.Style
	extendedSpace                focusSpace
	dirToExtend                  string
	activeChild, prevActiveChild extChild
	prevFocus                    focusSpace
	disableKeymap                bool
}

func initialExtensionSpaceModel() extensionSpaceModel {
	return extensionSpaceModel{
		extDirNav:     initialExtDirNavModel(),
		extSend:       initialExtSendModel(),
		extReceive:    initialExtReceiveModel(),
		preference:    initialPreferenceModel(),
		download:      initialDownloadModel(),
		titleStyle:    titleStyle,
		disableKeymap: true,
	}
}

func (m extensionSpaceModel) capturesKeyEvent(msg tea.KeyMsg) bool {
	switch m.activeChild {
	case home:
		return false
	case extDirNav:
		return m.extDirNav.capturesKeyEvent(msg)
	case extSend:
		return m.extSend.capturesKeyEvent(msg)
	case extReceive:
		return m.extReceive.capturesKeyEvent(msg)
	case preference:
		return m.preference.capturesKeyEvent(msg)
	case download:
		return m.download.capturesKeyEvent(msg)
	default:
		return false
	}
}

func (m extensionSpaceModel) Init() tea.Cmd {
	return tea.Batch(m.extDirNav.Init(), m.preference.Init())
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
			return m, msgToCmd(extensionChildSwitchMsg{child: home, focus: true})

		case "backspace":
			currentFocus = local
			return m, msgToCmd(spaceFocusSwitchMsg{})

		}

	case extensionChildSwitchMsg:
		if msg.child != m.activeChild {
			m.prevActiveChild = m.activeChild
			m.activeChild = msg.child
		}
		// so we can switch focus back to prev active space
		// after preference OR download is deactivated
		if msg.child == preference || msg.child == download {
			m.prevFocus = currentFocus
		}
		if msg.child == home {
			m.disableKeymap = true
		}
		if msg.focus {
			currentFocus = extension
			return m, tea.Batch(msgToCmd(spaceFocusSwitchMsg{}), m.handleChildModelUpdate(msg))
		}

	case serverLogsTimeoutMsg:
		if m.activeChild != extDirNav {
			return m, msgToCmd(extensionChildSwitchMsg{child: extDirNav, focus: false})
		}

	case preferenceInactiveMsg, downloadInactiveMsg:
		if m.prevFocus != extension {
			currentFocus = m.prevFocus
			return m, tea.Batch(
				msgToCmd(spaceFocusSwitchMsg{}),
				msgToCmd(extensionChildSwitchMsg{child: m.prevActiveChild, focus: false}),
			)
		}
		return m, msgToCmd(extensionChildSwitchMsg{child: m.prevActiveChild, focus: true})

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
	var view string
	switch m.activeChild {
	case home:
		title := m.titleStyle.Render("Home Space")
		b := banner.Height(extContainerWorkableH() - lipgloss.Height(title)).Render()
		b = lipgloss.JoinVertical(lipgloss.Center, title, b)
		view = lipgloss.PlaceHorizontal(largeContainerW(), lipgloss.Center, b)
	case extDirNav:
		view = m.extDirNav.View()
	case extSend:
		view = m.extSend.View()
	case extReceive:
		view = m.extReceive.View()
	case preference:
		view = m.preference.View()
	case download:
		view = m.download.View()
	default:
		view = ""
	}
	return largeContainerStyle.Width(largeContainerW()).Height(workableH()).Render(view)
}

func (m *extensionSpaceModel) handleChildModelUpdate(msg tea.Msg) tea.Cmd {
	var cmds [5]tea.Cmd
	m.extDirNav, cmds[0] = m.extDirNav.Update(msg)
	m.extSend, cmds[1] = m.extSend.Update(msg)
	m.extReceive, cmds[2] = m.extReceive.Update(msg)
	m.preference, cmds[3] = m.preference.Update(msg)
	m.download, cmds[4] = m.download.Update(msg)
	return tea.Batch(cmds[:]...)
}

func (m *extensionSpaceModel) updateTitleStyleAsFocus(focus bool) {
	if focus {
		m.titleStyle = titleStyle.
			Background(highlightColor).
			Foreground(subduedHighlightColor)
	} else {
		m.titleStyle = titleStyle.
			Background(grayColor).
			Foreground(highlightColor)
	}
}

func (m *extensionSpaceModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
	m.extDirNav.updateKeymap(disable || m.activeChild != extDirNav)
	m.extSend.updateKeymap(disable || m.activeChild != extSend)
	m.extReceive.updateKeymap(disable || m.activeChild != extReceive)
	m.preference.updateKeymap(disable || m.activeChild != preference)
	m.download.updateKeymap(disable || m.activeChild != download)
}
