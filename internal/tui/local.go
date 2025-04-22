package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type activeLocalChild = int

const (
	dirNav activeLocalChild = iota
	prepSel
)

type localSpaceModel struct {
	dirNavigation dirNavModel
	activeChild   activeLocalChild
	disableKeymap bool
}

func initialLocalSpaceModel() localSpaceModel {
	return localSpaceModel{
		dirNavigation: initialDirNavModel(),
		//activeChild:   prepSel,
	}
}

func (m localSpaceModel) capturesKeyEvent(msg tea.KeyMsg) bool {
	return m.dirNavigation.capturesKeyEvent(msg)
}

func (m localSpaceModel) Init() tea.Cmd {
	return tea.Batch(m.dirNavigation.Init())
}

func (m localSpaceModel) Update(msg tea.Msg) (localSpaceModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.capturesKeyEvent(msg) {
			// some child will capture the key event, let them handle it
			return m, m.handleChildModelUpdate(msg)
		}
		switch msg.String() {
		case "ctrl+s":
			m.activeChild = prepSel
		case "esc":
			m.activeChild = dirNav
		}
	}
	return m, m.handleChildModelUpdate(msg)
}

func (m localSpaceModel) View() string {
	switch m.activeChild {
	case dirNav:
		return m.dirNavigation.View()
	default:
		return ""
	}
}

func (m *localSpaceModel) handleChildModelUpdate(msg tea.Msg) tea.Cmd {
	var cmds [1]tea.Cmd
	m.dirNavigation, cmds[0] = m.dirNavigation.Update(msg)
	return tea.Batch(cmds[:]...)
}

func (m *localSpaceModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
	m.dirNavigation.updateKeymap(disable)
}

func (m localSpaceModel) grantSpaceFocusSwitch() bool {
	switch m.activeChild {
	case dirNav:
		return m.dirNavigation.grantSpaceFocusSwitch()
	default:
		return true
	}
}
