package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type localSpaceModel struct {
	dirNavigation dirNavModel
	disableKeymap bool
}

func initialLocalSpaceModel() localSpaceModel {
	return localSpaceModel{
		dirNavigation: initialDirNavModel(),
	}
}

func (m localSpaceModel) Init() tea.Cmd {
	return m.dirNavigation.Init()
}

func (m localSpaceModel) Update(msg tea.Msg) (localSpaceModel, tea.Cmd) {
	m.dirNavigation.disableKeymap = m.disableKeymap
	return m, m.handleChildModelUpdate(msg)
}

func (m localSpaceModel) View() string {
	return m.dirNavigation.View()
}

func (m *localSpaceModel) handleChildModelUpdate(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.dirNavigation, cmd = m.dirNavigation.Update(msg)
	return cmd
}

func (m *localSpaceModel) updateKeymap(disable bool) {
	m.disableKeymap = disable
	m.dirNavigation.updateKeymap(disable)
}
