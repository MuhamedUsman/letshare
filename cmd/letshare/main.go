package main

import (
	"github.com/MuhamedUsman/letshare/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmittmann/tint"
	"log/slog"
	"os"
	"time"
)

func init() {
	h := tint.NewHandler(os.Stderr, &tint.Options{TimeFormat: time.Kitchen})
	slog.SetDefault(slog.New(h))
}

func main() {
	f, err := tea.LogToFile("Letshare.log", "Letshare")
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	defer f.Close()

	_ = lipgloss.DefaultRenderer().HasDarkBackground()
	_, err = tea.NewProgram(
		tui.InitialMainModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithoutBracketedPaste(),
		tea.WithReportFocus(),
	).Run()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
