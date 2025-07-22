package main

import (
	"context"
	"errors"
	"flag"
	"github.com/MuhamedUsman/letshare/internal/bgtask"
	"github.com/MuhamedUsman/letshare/internal/mdns"
	"github.com/MuhamedUsman/letshare/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmittmann/tint"
	"log/slog"
	"os"
	"time"
)

var (
	version     = "UNKNOWN"
	showVersion bool
)

func init() {
	// Set up the logger to use tint for formatting
	h := tint.NewHandler(os.Stderr, &tint.Options{TimeFormat: time.Kitchen})
	slog.SetDefault(slog.New(h))

	flag.BoolVar(&showVersion, "version", false, "Print the version and exit") // long
	flag.BoolVar(&showVersion, "v", false, "Print the version and exit")       // short
	flag.Parse()
}

func main() {
	if showVersion {
		println(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ECFD65")).
			Render("Letshare ", version))
		return
	}

	// start the discovery of mDNS services on startup
	bgtask.Get().Run(func(shutdownCtx context.Context) {
		if err := mdns.Get().Browse(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
			println()
			slog.Error("Error discovering mDNS services", "err", err)
			os.Exit(1)
		}
	})

	finalErrCh := make(chan error, 1)
	_, err := tea.NewProgram(
		tui.InitialMainModel(finalErrCh),
		tea.WithAltScreen(),
		tea.WithoutBracketedPaste(),
	).Run()
	if err != nil {
		err = <-finalErrCh
		slog.Error(err.Error())
	}
}
