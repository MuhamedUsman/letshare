package main

import (
	"context"
	"errors"
	"github.com/MuhamedUsman/letshare/internal/bgtask"
	"github.com/MuhamedUsman/letshare/internal/mdns"
	"github.com/MuhamedUsman/letshare/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmittmann/tint"
	"log"
	"log/slog"
	"os"
	"runtime"
	"time"
)

func main() {

	f, err := os.OpenFile("letshare.log", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		println("Error opening log file:", err.Error())
		os.Exit(1)
	}
	h := tint.NewHandler(f, &tint.Options{TimeFormat: time.Kitchen, NoColor: true})
	slog.SetDefault(slog.New(h))

	sog := slog.New(h)

	bgtask.Get().Run(func(shutdownCtx context.Context) {
		if err := mdns.Get().Browse(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
			println()
			sog.Error("Error discovering mDNS services", "err", err)
			os.Exit(1)
		}
	})

	go func() {
		prev := 0
		for {
			<-time.After(time.Second)
			current := runtime.NumGoroutine()
			if prev != current {
				log.Println("Current goroutine count: ", runtime.NumGoroutine())
				prev = runtime.NumGoroutine()
			}
		}
	}()

	finalErrCh := make(chan error, 1) // writer wil close
	_, err = tea.NewProgram(
		tui.InitialMainModel(finalErrCh),
		tea.WithAltScreen(),
		tea.WithoutBracketedPaste(),
	).Run()
	if err != nil {
		err = <-finalErrCh
		sog.Error(err.Error())
	}
}
