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

func main() {
	h := tint.NewHandler(os.Stderr, &tint.Options{TimeFormat: time.Kitchen, NoColor: true})
	sog := slog.New(h)

	bgtask.Get().Run(func(shutdownCtx context.Context) {
		if err := mdns.Get().Browse(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
			println()
			sog.Error("Error discovering mDNS services", "err", err)
			os.Exit(1)
		}
	})

	finalErrCh := make(chan error, 1) // writer wil close
	_, err := tea.NewProgram(
		tui.InitialMainModel(finalErrCh),
		tea.WithAltScreen(),
		tea.WithoutBracketedPaste(),
	).Run()
	if err != nil {
		err = <-finalErrCh
		sog.Error(err.Error())
	}
}
