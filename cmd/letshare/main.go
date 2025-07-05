package main

import (
	"flag"
	"github.com/MuhamedUsman/letshare/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmittmann/tint"
	"log/slog"
	"os"
	"time"
)

func init() {
	flag.Parse()
}

func main() {
	f, err := tea.LogToFile("Letshare.log", "Letshare")
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	defer f.Close()

	h := tint.NewHandler(f, &tint.Options{TimeFormat: time.Kitchen, NoColor: true})
	slog.SetDefault(slog.New(h))

	/*bgtask.Get().Run(func(shutdownCtx context.Context) {
		if err = mdns.Get().Browse(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
			println()
			slog.Error("Error discovering mDNS services", "err", err)
			os.Exit(1)
		}
	})*/

	finalErrCh := make(chan error, 1) // writer wil close
	_, err = tea.NewProgram(
		tui.InitialMainModel(finalErrCh),
		tea.WithAltScreen(),
		tea.WithoutBracketedPaste(),
	).Run()
	if err != nil {
		err = <-finalErrCh
		slog.Error(err.Error())
	}
}
