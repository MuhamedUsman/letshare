package main

import (
	"github.com/MuhamedUsman/letshare/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmittmann/tint"
	"log/slog"
	"os"
)

func main() {

	slg := slog.New(tint.NewHandler(os.Stderr, nil))

	_ = lipgloss.DefaultRenderer().HasDarkBackground()
	_, err := tea.NewProgram(
		tui.MainModel{},
		tea.WithAltScreen(),
		tea.WithMouseAllMotion(),
		tea.WithoutBracketedPaste(),
		tea.WithReportFocus(),
	).Run()
	if err != nil {
		slg.Error(err.Error())
		os.Exit(1)
	}

}
