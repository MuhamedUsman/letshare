package tui

import tea "github.com/charmbracelet/bubbletea"

type errMsg struct {
	// err to log
	err error
	// errStr: user-friendly err
	errStr string
}

func (msg errMsg) cmd() tea.Msg { return msg }

type fsErrMsg string

func (msg fsErrMsg) cmd() tea.Msg { return msg }

// extendDirMsg is sent by the dirNavigationModel to signal extendDirModel
// to extend selected directory, and whether to make the table focused or not
type extendDirMsg struct {
	path string
	// marker to decide whether to make the extendDirModel.infoTable table focused
	focus bool
}

func (msg extendDirMsg) cmd() tea.Msg { return msg }

// extendSpaceMsg is sent by extendDirModel to extensionSpaceModel to signal
// the title change and whether to make it focused
type extendSpaceMsg struct {
	space focusedSpace
	// marker to decide whether to make the extension space focused
	focus bool
}

func (msg extendSpaceMsg) cmd() tea.Msg { return msg }

// spaceFocusSwitchMsg manages space switching using tab & shift+tab
type spaceFocusSwitchMsg focusedSpace

func (msg spaceFocusSwitchMsg) cmd() tea.Msg { return msg }

type hideInfoSpaceTitle bool

func (msg hideInfoSpaceTitle) cmd() tea.Msg { return msg }
