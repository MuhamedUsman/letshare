package tui

import tea "github.com/charmbracelet/bubbletea"

type errMsg struct {
	// err to log
	err error
	// errStr: user-friendly err
	// display if fatal is set to false
	errStr string
	// flag to signal log the err to stderr and exit
	// if true, no need to read errStr, as it may be zero valued
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

// extendChildMsg is sent by extDirNavModel to extensionSpaceModel to signal
// the title change and whether to make it focused
type extendChildMsg struct {
	child extChild
	// marker to decide whether to make the extension child focused
	focus bool
}

func (msg extendChildMsg) cmd() tea.Msg { return msg }

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
