package tui

import (
	"github.com/MuhamedUsman/letshare/internal/server"
	tea "github.com/charmbracelet/bubbletea"
)

type errMsg struct {
	// errHeader: header to display in the error message
	// do not set if fatal is set to true
	errHeader string
	// err to log
	err error
	// errStr: user-friendly err
	// display if fatal is set to false
	errStr string
	// flag to signal log the err to stderr and exit
	// if true, no need to processed errStr, as it may be zero valued
	fatal bool
}

type fsErrMsg string

// spaceFocusSwitchMsg manages child switching using tab & shift+tab
type spaceFocusSwitchMsg struct{}

type localChildSwitchMsg struct {
	child localChild
	// marker to decide whether to make the local child focused
	focus bool
}

// extensionChildSwitchMsg is sent by extDirNavModel to extensionSpaceModel to signal
// the title change and whether to make it focused
type extensionChildSwitchMsg struct {
	child extChild
	// marker to decide whether to make the extension child focused
	focus bool
}

type remoteChildSwitchMsg struct {
	child remoteChild
	focus bool
}

// extendDirMsg is sent by the dirNavModel to signal extDirNavModel
// to extend selected directory, and whether to make the table focused or not
type extendDirMsg struct {
	path string
	// marker to decide whether to make the extDirNavModel.extDirTable table focused
	focus bool
}

type resetExtDirTableSelectionsMsg struct{}

type resetExtFileIndexTableSelectionsMsg struct{}

type dirListUpdatedMsg struct{}

type processSelectionsMsg struct {
	parentPath  string
	filenames   []string
	dirs, files int
}

type downloadSelectionsMsg struct {
	files []string
}

// preferencesSavedMsg signals the changes to the preferences are saved,
// bool indicates whether to inactivate the preferences model or not
type preferencesSavedMsg bool

// to signal extensionSpaceModel to switch back to the previous child model
// as the user is done with the preference model.
type preferenceInactiveMsg struct{}

type downloadInactiveMsg struct{}

type progressMsg uint64

type zippingLogMsg string

type zippingDoneMsg []string

type zippingErrMsg error

type zippingCanceledMsg struct{}

type serverLogMsg server.Log

type rerenderPreferencesMsg struct{}

type sendFilesMsg []string

type instanceServingMsg struct{}

type instanceShutdownMsg struct{}

type shutdownReqWhenNotIdleMsg string

type handleExtSendCh struct {
	logCh        chan server.Log
	activeDownCh <-chan int
}

type activeDownsMsg int

type serverLogsTimeoutMsg struct{}

type instanceAvailabilityMsg bool

type fetchFileIndexesMsg string // string hold the instance name

type fileIndexesMsg []fileIndex

func msgToCmd[t tea.Msg](msg t) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}
