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

	f, err := tea.LogToFile("Letschat.log", "Letschat")

	slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{AddSource: true})))
	if err != nil {
		slg.Error(err.Error())
		os.Exit(1)
	}
	defer f.Close()

	_ = lipgloss.DefaultRenderer().HasDarkBackground()
	_, err = tea.NewProgram(
		tui.InitialMainModel(),
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
