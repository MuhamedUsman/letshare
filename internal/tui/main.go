package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type focus int

const (
	// No focus
	blur focus = iota - 1
	main
	send
	receive
	info
)

var (
	termW, termH int
	curFocus     focus
)

type MainModel struct {
	send    sendModel
	receive receiveModel
	info    infoModel
}

func InitialMainModel() MainModel {
	return MainModel{
		send: initialSendModel(),
	}
}

func (m MainModel) Init() tea.Cmd {
	return tea.Batch(m.send.Init())
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		termW = msg.Width
		termH = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	}

	return m, m.handleChildModelUpdates(msg)
}

func (m MainModel) View() string {
	subW := mainContainerStyle.GetHorizontalFrameSize()
	subH := mainContainerStyle.GetVerticalFrameSize()
	c := mainContainerStyle.
		Width(termW - subW).
		Height(termH - subH).
		Render(m.send.View())
	return c
}

func (m *MainModel) handleChildModelUpdates(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 3)
	m.send, cmds[0] = m.send.Update(msg)
	m.receive, cmds[1] = m.receive.Update(msg)
	m.info, cmds[2] = m.info.Update(msg)
	return tea.Batch(cmds...)
}
