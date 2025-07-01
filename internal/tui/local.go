package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type localChild = int

const (
	dirNav localChild = iota
	processFiles
	send
)

type localSpaceModel struct {
	dirNavigation dirNavModel
	processFiles  processFilesModel
	send          sendModel
	activeChild   localChild
	disableKeymap bool
}

func initialLocalSpaceModel() localSpaceModel {
	return localSpaceModel{
		dirNavigation: initialDirNavModel(),
		processFiles:  initialProcessFilesModel(),
		send:          initialSendModel(),
		activeChild:   dirNav,
	}
}

func (m localSpaceModel) capturesKeyEvent(msg tea.KeyMsg) bool {
	switch m.activeChild {
	case dirNav:
		return m.dirNavigation.capturesKeyEvent(msg)
	case processFiles:
		return m.processFiles.capturesKeyEvent(msg)
	case send:
		return m.send.capturesKeyEvent(msg)
	default:
		return false
	}
}

func (m localSpaceModel) Init() tea.Cmd {
	return tea.Batch(m.dirNavigation.Init())
}

func (m localSpaceModel) Update(msg tea.Msg) (localSpaceModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		if m.disableKeymap {
			return m, nil
		}
		if m.capturesKeyEvent(msg) {
			// some child will capture the key event, let them handle it
			return m, m.handleChildModelUpdate(msg)
		}

	case localChildSwitchMsg:
		m.activeChild = msg.child
		if msg.focus {
			currentFocus = local
			return m, tea.Batch(msgToCmd(spaceFocusSwitchMsg{}), m.handleChildModelUpdate(msg))
		}

	case spaceFocusSwitchMsg:
		// once switched with focus, update the view of the extension space
		// with corresponding local extension child
		if currentFocus == local {
			var switchExtChild extChild
			switch m.activeChild {
			case dirNav:
				// ask the child before switching,
				// they will answer based on their internal state
				// if false defaults to home
				if m.dirNavigation.grantExtSpaceSwitch() {
					switchExtChild = extDirNav
				}
			case processFiles:
				if m.processFiles.grantExtSpaceSwitch() {
					switchExtChild = home // no extProcessFiles model
				}
			case send:
				if m.send.grantExtSpaceSwitch() {
					switchExtChild = extSend
				}
			default:
				switchExtChild = home
			}
			return m, tea.Batch(
				msgToCmd(extensionChildSwitchMsg{child: switchExtChild}),
				m.handleChildModelUpdate(msg),
			)
		}
	}

	return m, m.handleChildModelUpdate(msg)
}

func (m localSpaceModel) View() string {
	var v string
	switch m.activeChild {
	case dirNav:
		v = m.dirNavigation.View()
	case processFiles:
		v = m.processFiles.View()
	case send:
		v = m.send.View()
	default:
		v = ""
	}
	return smallContainerStyle.Width(smallContainerW()).Render(v)
}

func (m *localSpaceModel) handleChildModelUpdate(msg tea.Msg) tea.Cmd {
	var cmds [3]tea.Cmd
	m.dirNavigation, cmds[0] = m.dirNavigation.Update(msg)
	m.processFiles, cmds[1] = m.processFiles.Update(msg)
	m.send, cmds[2] = m.send.Update(msg)
	return tea.Batch(cmds[:]...)
}

func (m *localSpaceModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
	m.dirNavigation.updateKeymap(disable || m.activeChild != dirNav)
	m.processFiles.updateKeymap(disable || m.activeChild != processFiles)
	m.send.updateKeymap(disable || m.activeChild != send)
}
