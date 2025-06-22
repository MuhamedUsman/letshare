package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// extSendModel is the model to read & view logs when server is running
// such as who is connected, what files are being downloaded, etc.
type extSendModel struct {
}

func (m extSendModel) Init() tea.Cmd {
	return nil
}

func (m extSendModel) Update(msg tea.Msg) (extSendModel, tea.Cmd) {
	return m, nil
}

func (m extSendModel) View() string {
	return ""
}
