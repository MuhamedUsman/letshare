package tui

import (
	"github.com/MuhamedUsman/letshare/internal/tui/overlay"
	"github.com/MuhamedUsman/letshare/internal/util/bgtask"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"log/slog"
	"os"
	"time"
)

type focusSpace int

const (
	// No focusSpace
	local focusSpace = iota
	extension
	remote
	confirmation
)

var (
	termW, termH int
	currentFocus focusSpace
)

// eventCapturer defines an interface for types that can determine if they capture a specific key event.
type eventCapturer interface {
	// CapturesKeyEvent returns true if the implementation captures the given key event.
	// When true is returned, the application typically should consider the event "consumed",
	// and must be passed to the child component.
	//
	// Example:
	//  case tea.KeyMsg:
	//		// some child will capture the key event, let them handle it
	//		if m.capturesKeyEvent(msg) {
	//			return m, m.handleChildModelUpdates(msg)
	//		}
	//
	// and not pass it to other components, In case the msg should be passed to other components,
	// it must be handled with care (not recommended, but possible).
	capturesKeyEvent(msg tea.KeyMsg) bool
}

type MainModel struct {
	localSpace     localSpaceModel
	extensionSpace extensionSpaceModel
	remoteSpace    remoteSpaceModel
	confirmation   alertDialogModel
}

func InitialMainModel() MainModel {
	return MainModel{
		localSpace:     initialLocalSpaceModel(),
		extensionSpace: initialExtensionSpaceModel(),
		remoteSpace:    initialRemoteSpaceModel(),
		confirmation:   initialAlertDialogModel(),
	}
}

func (m MainModel) capturesKeyEvent(msg tea.KeyMsg) bool {
	switch currentFocus {
	case local:
		return m.localSpace.capturesKeyEvent(msg)
	case extension:
		return m.extensionSpace.capturesKeyEvent(msg)
	case remote:
		return m.remoteSpace.capturesKeyEvent(msg)
	case confirmation:
		return m.confirmation.capturesKeyEvent(msg)
	default:
		return false
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
		// some child will capture the key event, let them handle it
		if m.capturesKeyEvent(msg) {
			return m, m.handleChildModelUpdates(msg)
		}

		switch msg.String() {

		case "ctrl+p":
			// don't show preferences if the confirmation dialog is active
			if !m.confirmation.active {
				return m, extensionChildSwitchMsg{child: preference, focus: true}.cmd
			}

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

		case "ctrl+c":
			bgTask := bgtask.Get()
			if err := bgTask.Shutdown(5 * time.Second); err != nil {
				slog.Error("failed to shutdown background task", "tasks", bgTask.Tasks, "error", err)
			}
			return m, tea.Quit
		}

	case spaceFocusSwitchMsg:
		m.updateKeymapsByFocus()

	case errMsg:
		if msg.fatal {
			println()
			slog.Error(msg.err.Error())
			os.Exit(1)
		}
		return m, alertDialogMsg{header: msg.errHeader, body: msg.errStr}.cmd
	}
	
	return m, m.handleChildModelUpdates(msg)
}

func (m MainModel) View() string {
	c := lipgloss.JoinHorizontal(lipgloss.Top, m.localSpace.View(), m.extensionSpace.View(), m.remoteSpace.View())
	if m.confirmation.active {
		c = overlay.Place(lipgloss.Center, lipgloss.Center, c, m.confirmation.View())
	}
	return mainContainerStyle.Width(workableW()).Height(workableH()).Render(c)
}

func (m *MainModel) handleChildModelUpdates(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 4)
	m.localSpace, cmds[0] = m.localSpace.Update(msg)
	m.extensionSpace, cmds[1] = m.extensionSpace.Update(msg)
	m.remoteSpace, cmds[2] = m.remoteSpace.Update(msg)
	m.confirmation, cmds[3] = m.confirmation.Update(msg)
	return tea.Batch(cmds...)
}

func (m *MainModel) updateKeymapsByFocus() {
	m.localSpace.updateKeymap(currentFocus != local)
	m.extensionSpace.updateKeymap(currentFocus != extension)
	m.remoteSpace.disableKeymap = currentFocus != remote
	m.confirmation.disableKeymap = currentFocus != confirmation
}
