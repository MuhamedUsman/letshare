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

type extentDirMsg string

func (msg extentDirMsg) cmd() tea.Msg { return msg }
