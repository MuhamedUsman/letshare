package main

import (
	"context"
	"errors"
	"github.com/MuhamedUsman/letshare/internal/mdns"
	"github.com/MuhamedUsman/letshare/internal/tui"
	"github.com/MuhamedUsman/letshare/internal/util/bgtask"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmittmann/tint"
	"log"
	"log/slog"
	"os"
	"time"
)

func main() {
	f, err := tea.LogToFile("Letshare.log", "Letshare")
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	defer f.Close()

	h := tint.NewHandler(f, &tint.Options{TimeFormat: time.Kitchen})
	slog.SetDefault(slog.New(h))

	// mdns discovery
	bgtask.Get().Run(func(shutdownCtx context.Context) {
		if err = mdns.Get().Discover(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
			log.Fatal(err)
		}
	})

	_, err = tea.NewProgram(
		tui.InitialMainModel(),
		tea.WithAltScreen(),
		tea.WithoutBracketedPaste(),
	).Run()
	if err != nil {
		println()
		slog.Error(err.Error())
		os.Exit(1)
	}
}
