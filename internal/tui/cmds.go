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

// extendDirMsg is sent by the sendModel to signal sendInfoModel
// to extend selected directory, and whether to make the table focused or not
type extendDirMsg struct {
	path string
	// marker to decide whether to make the sendInfoModel.infoTable table focused
	focus bool
}

func (msg extendDirMsg) cmd() tea.Msg { return msg }

// extendSpaceMsg is sent by sendInfoModel to infoModel to signal
// the title change and whether to make it focused
type extendSpaceMsg struct {
	space focusedTab
	// marker to decide whether to make the info space focused
	focus bool
}

func (msg extendSpaceMsg) cmd() tea.Msg { return msg }

// spaceTabSwitchMsg manages space switching using tab & shift+tab
type spaceTabSwitchMsg focusedTab

func (msg spaceTabSwitchMsg) cmd() tea.Msg { return msg }
