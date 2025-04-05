package tui

import tea "github.com/charmbracelet/bubbletea"

type errMsg struct {
	// err to log
	err error
	// errStr: user-friendly err
	errStr string
}

func (em errMsg) cmd() tea.Msg { return em }
