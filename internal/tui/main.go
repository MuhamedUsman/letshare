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

type appShutdownState int

const (
	// whole app is in a clean state, no active server, no zipping files, no partial downloads
	clean appShutdownState = iota
	// app is in the state of serving files and has partial downloads (downloading || paused || failed)
	servAndDown
	// server is idling(not serving files) and has partial downloads
	idleAndDown
	// app is in the state of zipping files and has partial downloads
	zipAndDown
	zippingFiles
	servingFiles
	idleServer
	partialDowns
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
	currentFocus = extension
	return tea.Batch(
		m.localSpace.Init(),
		m.extensionSpace.Init(),
		m.remoteSpace.Init(),
		m.alertDialog.Init(),
	)
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
			// loop currentFocus b/w local, extension and remote tabs
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

		case "ctrl+c":
			state := m.getAppShutdownState()
			msg, cmd := m.getMsgAndCmd(state)

			// no dirty state, just quit
			if state == clean {
				shutdownBgTasks()
				return m, tea.Quit
			}

			// no msg just silently quit
			if msg == "" && cmd != nil {
				shutdownBgTasks()
				return m, tea.Sequence(cmd, tea.Quit)
			}

			return m, msgToCmd(alertDialogMsg{
				header:         "FORCE SHUTDOWN?",
				body:           msg,
				positiveBtnTxt: "YUP!",
				negativeBtnTxt: "NOPE",
				cursor:         positive,
				positiveFunc: func() tea.Cmd {
					shutdownBgTasks()
					return tea.Sequence(cmd, tea.Quit)
				},
			})
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

func (m MainModel) getAppShutdownState() appShutdownState {
	zipState := m.localSpace.processFiles.zipTracker.state
	zipInProgress := zipState == processing || zipState == canceling
	servingInProgress := m.localSpace.send.server != nil && m.localSpace.send.server.ActiveDowns > 0
	serverIdle := m.localSpace.send.isServing && !servingInProgress
	dirtyDowns := m.extensionSpace.download.hasPartialDownloads()

	switch {
	case zipInProgress && dirtyDowns:
		return zipAndDown
	case servingInProgress && dirtyDowns:
		return servAndDown
	case serverIdle && dirtyDowns:
		return idleAndDown
	case zipInProgress:
		return zippingFiles
	case servingInProgress:
		return servingFiles
	case serverIdle:
		return idleServer
	case dirtyDowns:
		return partialDowns
	default:
		return clean
	}
}

func (m MainModel) getMsgAndCmd(s appShutdownState) (string, tea.Cmd) {
	switch s {
	case clean: // no-op
	case servAndDown:
		return "Are you sure, it will close all the active server connections, and will delete all the partial downloads.",
			tea.Batch(m.localSpace.send.shutdownServer(true), m.extensionSpace.download.deletePartailDownloads())
	case idleAndDown:
		return "Are you sure, it will stop the idle server, and will delete all the partial downloads.",
			tea.Batch(m.localSpace.send.shutdownServer(true), m.extensionSpace.download.deletePartailDownloads())
	case zipAndDown:
		return "Are you sure, it will stop the file zipping, and will delete all the partial downloads.",
			tea.Batch(m.localSpace.processFiles.stopZipping(true), m.extensionSpace.download.deletePartailDownloads())
	case idleServer:
		return "", m.localSpace.send.shutdownServer(true)
	case zippingFiles:
		return "Do you want to stop zipping the files, progress will be lost.",
			m.localSpace.processFiles.stopZipping(true)
	case servingFiles:
		return "The server instance is being shutdown forcefully, all active downloads will abruptly halt.",
			m.localSpace.send.shutdownServer(true)
	case partialDowns:
		return "All the partially downloaded files will be deleted from the disk. If you want them to complete them make sure to keep the app running.",
			m.extensionSpace.download.deletePartailDownloads()
	}
	return "", nil
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
