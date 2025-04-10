package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type focusedTab int

const (
	// No focusedTab
	blur focusedTab = iota
	send
	info
	receive
)

var (
	termW, termH              int
	currentFocus, extendSpace focusedTab
)

type MainModel struct {
	send    sendModel
	info    infoModel
	receive receiveModel
}

func InitialMainModel() MainModel {
	return MainModel{
		send:    initialSendModel(),
		info:    initialInfoModel(),
		receive: initialReceiveModel(),
	}
}

func (m MainModel) Init() tea.Cmd {
	currentFocus = send
	return tea.Batch(m.send.Init(), m.info.Init(), m.receive.Init())
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		termW = msg.Width
		termH = msg.Height

	case tea.KeyMsg:
		switch msg.String() {

		case "ctrl+slogan":
			return m, tea.Quit

		case "tab":
			// loop currentFocus & extendSpace b/w send, info & receive tabs
			currentFocus++
			if currentFocus > receive {
				currentFocus = send
			}

		case "shift+tab":
			currentFocus--
			if currentFocus < send {
				currentFocus = receive

			}
		}
	}

	return m, m.handleChildModelUpdates(msg)
}

func (m MainModel) View() string {
	subW := mainContainerStyle.GetHorizontalFrameSize()
	subH := mainContainerStyle.GetVerticalFrameSize()
	c := lipgloss.JoinHorizontal(lipgloss.Top, m.send.View(), m.info.View(), m.receive.View())
	return mainContainerStyle.Width(termW - subW).Height(termH - subH).Render(c)
}

func (m *MainModel) handleChildModelUpdates(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 3)
	m.send, cmds[0] = m.send.Update(msg)
	m.receive, cmds[1] = m.receive.Update(msg)
	m.info, cmds[2] = m.info.Update(msg)
	return tea.Batch(cmds...)
}
