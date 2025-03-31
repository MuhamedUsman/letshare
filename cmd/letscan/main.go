package main

import (
	"context"
	"github.com/MuhamedUsman/letshare/internal/server"
	"github.com/MuhamedUsman/letshare/internal/utility"
	"log/slog"
	"os"
)

func main() {
	utility.ConfigureSlog(os.Stderr)
	s := server.New()

	// Publishing DNS Entry
	s.BT.Run(func(shutdownCtx context.Context) {
		instance := "Letshare"
		slog.Info("Publishing Multicast DNS Entry", "instance", instance)
		if err := server.PublishEntry(shutdownCtx, instance, "Sharing Files"); err != nil {
			slog.Error(err.Error())
		}
	})
	err := s.StartServerForDir("C:/Users/usman/Downloads/Programs")
	if err != nil {
		slog.Error(err.Error())
	}
}
