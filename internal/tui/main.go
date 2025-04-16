package tui

import (
	"github.com/MuhamedUsman/letshare/internal/tui/overlay"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type focusedTab int

const (
	// No focusedTab
	home focusedTab = iota
	send
	info
	receive
	confirmation
)

var (
	termW, termH int
	currentFocus focusedTab
)

type MainModel struct {
	send         sendModel
	info         infoModel
	receive      receiveModel
	confirmation confirmDialogModel
}

func InitialMainModel() MainModel {
	return MainModel{
		send:         initialSendModel(),
		info:         initialInfoModel(),
		receive:      initialReceiveModel(),
		confirmation: initialConfirmDialogModel(),
	}
}

func (m MainModel) Init() tea.Cmd {
	currentFocus = send
	return tea.Batch(m.send.Init(), m.info.Init(), m.receive.Init(), m.confirmation.Init())
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		termW = msg.Width
		termH = msg.Height

	case tea.KeyMsg:

		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}

		// if confirmation dialog is in focus, disable keys of this model
		if currentFocus == confirmation {
			return m, m.handleChildModelUpdates(msg)
		}

		switch msg.String() {

		case "tab":
			// loop currentFocus & extendSpace b/w send, info & receive tabs
			currentFocus++
			if currentFocus > receive {
				currentFocus = send
			}
			return m, spaceFocusSwitchMsg(currentFocus).cmd

		case "shift+tab":
			currentFocus--
			if currentFocus < send {
				currentFocus = receive
			}
			return m, spaceFocusSwitchMsg(currentFocus).cmd

		}
	}

	return m, m.handleChildModelUpdates(msg)
}

func (m MainModel) View() string {
	subW := mainContainerStyle.GetHorizontalFrameSize()
	subH := mainContainerStyle.GetVerticalFrameSize()
	c := lipgloss.JoinHorizontal(lipgloss.Top, m.send.View(), m.info.View(), m.receive.View())
	if m.confirmation.render {
		w, h := mainContainerStyle.GetFrameSize()
		w, h = termW-w, termH-h
		c = overlay.Place(w, h, lipgloss.Center, lipgloss.Center, c, m.confirmation.View())
	}
	return mainContainerStyle.Width(termW - subW).Height(termH - subH).Render(c)
}

func (m *MainModel) handleChildModelUpdates(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 4)
	m.send, cmds[0] = m.send.Update(msg)
	m.receive, cmds[1] = m.receive.Update(msg)
	m.info, cmds[2] = m.info.Update(msg)
	m.confirmation, cmds[3] = m.confirmation.Update(msg)
	return tea.Batch(cmds...)
}
