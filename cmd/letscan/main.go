package main

import (
	"context"
	"github.com/MuhamedUsman/letshare/internal/mdns"
	"github.com/MuhamedUsman/letshare/internal/server"
	"github.com/MuhamedUsman/letshare/internal/util"
	"log/slog"
	"os"
)

func main() {
	util.ConfigureSlog(os.Stderr)
	s := server.New()
	m := mdns.New()
	// Publishing DNS Entry
	s.BT.Run(func(shutdownCtx context.Context) {
		instance := "Letshare"
		slog.Info("Publishing Multicast DNS Entry", "instance", instance)
		if err := m.Publish(shutdownCtx, instance, "Sharing Files"); err != nil {
			slog.Error(err.Error())
		}
	})
	if err := s.StartServerForDir("C:/Users/usman/Downloads/Programs"); err != nil {
		slog.Error(err.Error())
	}
}
