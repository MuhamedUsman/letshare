package tui

import (
	"errors"
	"github.com/MuhamedUsman/letshare/internal/bgtask"
	"github.com/MuhamedUsman/letshare/internal/tui/overlay"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"log/slog"
	"time"
)

type focusSpace int

const (
	local focusSpace = iota
	extension
	remote
	alert
)

var (
	termW, termH int
	currentFocus focusSpace
)

// eventCapturer defines an interface for types that can determine if they capture a specific key event.
// exists mainly for documentation purpose
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

type FinalErrCh chan<- error

type MainModel struct {
	localSpace         localSpaceModel
	extensionSpace     extensionSpaceModel
	remoteSpace        remoteSpaceModel
	alertDialog        alertDialogModel
	finalErrCh         FinalErrCh
	isFinalErrChClosed bool // true if finalErrCh is already closed
}

// InitialMainModel initializes the main model
// make sure to have FinalErrCh buffered by 1
// and read from it when the main tea.Program has exited
func InitialMainModel(ch FinalErrCh) MainModel {
	return MainModel{
		localSpace:     initialLocalSpaceModel(),
		extensionSpace: initialExtensionSpaceModel(),
		remoteSpace:    initialRemoteSpaceModel(),
		alertDialog:    initialAlertDialogModel(),
		finalErrCh:     ch,
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
	case alert:
		return m.alertDialog.capturesKeyEvent(msg)
	default:
		return false
	}
}

func (m MainModel) Init() tea.Cmd {
	currentFocus = local
	return tea.Batch(m.localSpace.Init(), m.extensionSpace.Init(), m.remoteSpace.Init(), m.alertDialog.Init())
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
			// don't show preferences if the alert dialog is active
			if !m.alertDialog.active {
				return m, msgToCmd(extensionChildSwitchMsg{child: preference, focus: true})
			}

		case "ctrl+d":
			if !m.alertDialog.active {
				return m, msgToCmd(extensionChildSwitchMsg{child: download, focus: true})
			}

		case "tab":
			// loop currentFocus & extendSpace b/w local, extensionSpace and remote tabs
			currentFocus++
			if currentFocus > remote {
				currentFocus = local
			}
			return m, msgToCmd(spaceFocusSwitchMsg{})

		case "shift+tab":
			currentFocus--
			if currentFocus < local {
				currentFocus = remote
			}
			return m, msgToCmd(spaceFocusSwitchMsg{})

			// shutdown works as follows:
			// there is a global state - shutdownOk []bool (bool slice)
			// after ctrl+c this gives an opportunity to each model to handle the shutdown (as seen in the ctrl+c case below)
			// each command needs to do its cleanup, show any alert that it needs, and append a true to the shutdownOk slice
			//  each command that is called in this block (specific to a model) should return a tea.KeyCtrlC command,
			// that is, it should not consume the ctrl+c key event
			// when the key event comes back to the main model, it calls the next conditional statement till it shuts down.
		case "ctrl+c":
			if m.localSpace.send.isServing {
				return m, m.localSpace.send.shutdownServer(true)
			}
			state := m.localSpace.processFiles.zipTracker.state
			if state == processing || state == canceling {
				return m, m.localSpace.processFiles.confirmStopZipping(true)
			}
			if m.localSpace.activeChild == send {
				m.localSpace.send.deleteTempFiles()
			}
			shutdownBgTasks()
			return m, tea.Quit
		}

	case spaceFocusSwitchMsg:
		m.updateKeymapsByFocus()

	case errMsg:
		if msg.fatal {
			if !m.isFinalErrChClosed {
				m.finalErrCh <- msg.err
				close(m.finalErrCh)
				m.isFinalErrChClosed = true
			}
			return m, tea.Interrupt
		}
		return m, msgToCmd(alertDialogMsg{header: msg.errHeader, body: msg.errStr})
	}

	return m, m.handleChildModelUpdates(msg)
}

func (m MainModel) View() string {
	v := lipgloss.JoinHorizontal(lipgloss.Top, m.localSpace.View(), m.extensionSpace.View(), m.remoteSpace.View())
	if m.alertDialog.active {
		v = overlay.Place(lipgloss.Center, lipgloss.Center, v, m.alertDialog.View())
	}
	return mainContainerStyle.Width(workableW()).Height(workableH()).Render(v)
}

func (m *MainModel) handleChildModelUpdates(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 4)
	m.localSpace, cmds[0] = m.localSpace.Update(msg)
	m.extensionSpace, cmds[1] = m.extensionSpace.Update(msg)
	m.remoteSpace, cmds[2] = m.remoteSpace.Update(msg)
	m.alertDialog, cmds[3] = m.alertDialog.Update(msg)
	return tea.Batch(cmds...)
}

// disables keymaps for the spaces that are not focused
func (m *MainModel) updateKeymapsByFocus() {
	m.localSpace.updateKeymap(currentFocus != local)
	m.extensionSpace.updateKeymap(currentFocus != extension)
	m.remoteSpace.updateKeymap(currentFocus != remote)
	m.alertDialog.updateKeymap(currentFocus != alert)
}

func shutdownBgTasks() {
	bgTask := bgtask.Get()
	if err := bgTask.Shutdown(5 * time.Second); err != nil {
		slog.Error(err.Error())
	}
}

func unwrapErr(err error) error {
	for {
		unwrapped := errors.Unwrap(err)
		if unwrapped == nil {
			return err
		}
		err = unwrapped
	}
}
