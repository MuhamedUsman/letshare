package tui

import (
	"github.com/MuhamedUsman/letshare/internal/tui/overlay"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type focusedSpace int

const (
	// No focusedSpace
	none focusedSpace = iota
	local
	extension
	remote
	confirmation
)

var (
	termW, termH int
	currentFocus focusedSpace
)

type MainModel struct {
	localSpace     localSpaceModel
	extensionSpace extensionSpaceModel
	remoteSpace    remoteSpaceModel
	confirmation   confirmDialogModel
}

func InitialMainModel() MainModel {
	return MainModel{
		localSpace:     initialLocalSpaceModel(),
		extensionSpace: initialExtensionSpaceModel(),
		remoteSpace:    initialRemoteSpaceModel(),
		confirmation:   initialConfirmDialogModel(),
	}
}

func (m MainModel) Init() tea.Cmd {
	currentFocus = local
	return tea.Batch(m.localSpace.Init(), m.extensionSpace.Init(), m.remoteSpace.Init(), m.confirmation.Init())
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		termW = msg.Width
		termH = msg.Height

	case tea.KeyMsg:
		m.updateKeymapsByFocus()

		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}

		// if the confirmation dialog is in focus, disable keys of this model
		if currentFocus == confirmation {
			return m, m.handleChildModelUpdates(msg)
		}

		switch msg.String() {

		case "tab":
			// loop currentFocus & extendSpace b/w local, extensionSpace and remote tabs
			currentFocus++
			if currentFocus > remote {
				currentFocus = local
			}
			return m, spaceFocusSwitchCmd

		case "shift+tab":
			currentFocus--
			if currentFocus < local {
				currentFocus = remote
			}
			return m, spaceFocusSwitchCmd

		}

	}

	return m, m.handleChildModelUpdates(msg)
}

func (m MainModel) View() string {
	c := lipgloss.JoinHorizontal(lipgloss.Top, m.localSpace.View(), m.extensionSpace.View(), m.remoteSpace.View())
	if m.confirmation.render {
		w, h := mainContainerStyle.GetFrameSize()
		w, h = termW-w, termH-h
		c = overlay.Place(w, h, lipgloss.Center, lipgloss.Center, c, m.confirmation.View())
	}
	return mainContainerStyle.Width(workableW()).Height(workableH()).Render(c)
}

func (m *MainModel) handleChildModelUpdates(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 4)
	m.localSpace, cmds[0] = m.localSpace.Update(msg)
	m.remoteSpace, cmds[1] = m.remoteSpace.Update(msg)
	m.extensionSpace, cmds[2] = m.extensionSpace.Update(msg)
	m.confirmation, cmds[3] = m.confirmation.Update(msg)
	return tea.Batch(cmds...)
}

func (m *MainModel) updateKeymapsByFocus() {
	m.localSpace.updateKeymap(currentFocus != local)
	m.extensionSpace.updateKeymap(currentFocus != extension)
	m.remoteSpace.disableKeymap = currentFocus != remote
	m.confirmation.disableKeymap = currentFocus != confirmation
}
