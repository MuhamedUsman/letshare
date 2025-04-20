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
	prepSel       prepSelModel
	activeChild   activeLocalChild
	disableKeymap bool
}

func initialLocalSpaceModel() localSpaceModel {
	return localSpaceModel{
		dirNavigation: initialDirNavModel(),
		prepSel:       initialPrepSelModel(),
		//activeChild:   prepSel,
	}
}

func (m localSpaceModel) Init() tea.Cmd {
	return tea.Batch(m.dirNavigation.Init(), m.prepSel.Init())
}

func (m localSpaceModel) Update(msg tea.Msg) (localSpaceModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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
	case prepSel:
		return m.prepSel.View()
	default:
		return ""
	}
}

func (m *localSpaceModel) handleChildModelUpdate(msg tea.Msg) tea.Cmd {
	var cmds [2]tea.Cmd
	m.dirNavigation, cmds[0] = m.dirNavigation.Update(msg)
	m.prepSel, cmds[1] = m.prepSel.Update(msg)
	return tea.Batch(cmds[:]...)
}

func (m *localSpaceModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
	m.dirNavigation.updateKeymap(disable)
}
