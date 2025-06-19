package tui

import tea "github.com/charmbracelet/bubbletea"

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

func (msg errMsg) cmd() tea.Msg { return msg }

type fsErrMsg string

func (msg fsErrMsg) cmd() tea.Msg { return msg }

// extendDirMsg is sent by the dirNavModel to signal extDirNavModel
// to extend selected directory, and whether to make the table focused or not
type extendDirMsg struct {
	path string
	// marker to decide whether to make the extDirNavModel.extDirTable table focused
	focus bool
}

func (msg extendDirMsg) cmd() tea.Msg { return msg }

// extensionChildSwitchMsg is sent by extDirNavModel to extensionSpaceModel to signal
// the title change and whether to make it focused
type extensionChildSwitchMsg struct {
	child extChild
	// marker to decide whether to make the extension child focused
	focus bool
}

func (msg extensionChildSwitchMsg) cmd() tea.Msg { return msg }

type localChildSwitchMsg struct {
	child localChild
	// marker to decide whether to make the local child focused
	focus bool
}

func (msg localChildSwitchMsg) cmd() tea.Msg { return msg }

// spaceFocusSwitchMsg manages child switching using tab & shift+tab
type spaceFocusSwitchMsg struct{}

func spaceFocusSwitchCmd() tea.Msg { return spaceFocusSwitchMsg{} }

type processSelectionsMsg struct {
	parentPath  string
	filenames   []string
	dirs, files int
}

func (msg processSelectionsMsg) cmd() tea.Msg { return msg }

// preferencesSavedMsg signals the changes to the preferences are saved,
// bool indicates whether to inactivate the preferences model or not
type preferencesSavedMsg bool

func preferencesSavedCmd(exit bool) tea.Cmd {
	return func() tea.Msg {
		return preferencesSavedMsg(exit)
	}
}

type progressMsg uint64

type logMsg string

type zippingDoneMsg []string

type zippingErrMsg error

type zippingCanceledMsg struct{}

type rerenderPreferencesMsg struct{}

func rerenderPreferencesCmd() tea.Msg { return rerenderPreferencesMsg{} }

type sendFilesMsg []string

func (f sendFilesMsg) cmd() tea.Msg {
	return f
}

func instanceStateCmd(s instanceState) tea.Cmd {
	return func() tea.Msg { return s }
}
