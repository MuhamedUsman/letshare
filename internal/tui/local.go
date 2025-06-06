package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type localChild = int

const (
	dirNav localChild = iota
	send
)

type localSpaceModel struct {
	dirNavigation dirNavModel
	send          sendModel
	activeChild   localChild
	disableKeymap bool
}

func initialLocalSpaceModel() localSpaceModel {
	return localSpaceModel{
		dirNavigation: initialDirNavModel(),
		send:          initialSendModel(),
		activeChild:   dirNav,
	}
}

func (m localSpaceModel) capturesKeyEvent(msg tea.KeyMsg) bool {
	switch m.activeChild {
	case dirNav:
		return m.dirNavigation.capturesKeyEvent(msg)
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
		switch msg.String() {
		case "esc":
			m.activeChild = dirNav
		}

	case localChildSwitchMsg:
		m.activeChild = msg.child
		if msg.focus {
			currentFocus = local
			return m, tea.Batch(spaceFocusSwitchCmd, m.handleChildModelUpdate(msg))
		}
	}

	return m, m.handleChildModelUpdate(msg)
}

func (m localSpaceModel) View() string {
	switch m.activeChild {
	case dirNav:
		return m.dirNavigation.View()
	case send:
		return m.send.View()
	default:
		return ""
	}
}

func (m *localSpaceModel) handleChildModelUpdate(msg tea.Msg) tea.Cmd {
	var cmds [2]tea.Cmd
	m.dirNavigation, cmds[0] = m.dirNavigation.Update(msg)
	m.send, cmds[1] = m.send.Update(msg)
	return tea.Batch(cmds[:]...)
}

func (m *localSpaceModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
	m.dirNavigation.updateKeymap(disable || m.activeChild != dirNav)
	m.send.updateKeymap(disable || m.activeChild != send)
}
