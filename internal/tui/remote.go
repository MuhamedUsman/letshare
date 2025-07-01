package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type remoteChild = int

const (
	recv remoteChild = iota
)

type remoteSpaceModel struct {
	receive       receiveModel
	activeChild   remoteChild
	disableKeymap bool
}

func initialRemoteSpaceModel() remoteSpaceModel {
	return remoteSpaceModel{
		receive:       initialReceiveModel(),
		disableKeymap: true,
	}
}

func (m remoteSpaceModel) capturesKeyEvent(msg tea.KeyMsg) bool {
	return m.receive.capturesKeyEvent(msg)
}

func (m remoteSpaceModel) Init() tea.Cmd {
	return m.receive.Init()
}

func (m remoteSpaceModel) Update(msg tea.Msg) (remoteSpaceModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.disableKeymap {
			return m, nil
		}
		if m.capturesKeyEvent(msg) {
			return m, m.handleChildModelUpdate(msg)
		}

	case remoteChildSwitchMsg:
		m.activeChild = msg.child
		if msg.focus {
			currentFocus = remote
			return m, tea.Batch(msgToCmd(spaceFocusSwitchMsg{}), m.handleChildModelUpdate(msg))
		}

	case spaceFocusSwitchMsg:
		// once switched with focus, update the view of the extension space
		// with corresponding local extension child
		if currentFocus == remote {
			var switchExtChild extChild
			switch m.activeChild {
			case recv:
				if m.receive.grantExtSpaceSwitch() {
					switchExtChild = extReceive
				}
			default:
				switchExtChild = home
			}
			return m, tea.Batch(msgToCmd(extensionChildSwitchMsg{child: switchExtChild}), m.handleChildModelUpdate(msg))
		}
	}

	return m, m.handleChildModelUpdate(msg)
}

func (m remoteSpaceModel) View() string {
	var view string
	switch m.activeChild {
	case recv:
		view = m.receive.View()
	default:
		return ""
	}
	return smallContainerStyle.Render(view)
}

func (m *remoteSpaceModel) handleChildModelUpdate(msg tea.Msg) tea.Cmd {
	var cmds [1]tea.Cmd
	m.receive, cmds[0] = m.receive.Update(msg)
	return tea.Batch(cmds[:]...)
}

func (m *remoteSpaceModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
	m.receive.updateKeymap(disable || m.activeChild != recv)
}
