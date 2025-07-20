package main

import (
	"context"
	"errors"
	"github.com/MuhamedUsman/letshare/internal/bgtask"
	"github.com/MuhamedUsman/letshare/internal/mdns"
	"github.com/MuhamedUsman/letshare/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmittmann/tint"
	"log/slog"
	"os"
	"time"
)

func init() {
	h := tint.NewHandler(os.Stderr, &tint.Options{TimeFormat: time.Kitchen, NoColor: true})
	slog.SetDefault(slog.New(h))
}

func main() {
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
