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
	slog.SetLogLoggerLevel(slog.LevelWarn)
	s := server.New()
	m := mdns.New()
	instance := "letshare"
	// Publishing DNS Entry
	s.BT.Run(func(shutdownCtx context.Context) {
		slog.Info("Publishing Multicast DNS Entry", "instance", instance)
		if err := m.Publish(shutdownCtx, instance, instance, 80); err != nil {
			slog.Error(err.Error())
		}
	})
	s.BT.Run(func(shutdownCtx context.Context) {
		slog.Info("Discovering Multicast DNS Entries")
		if err := m.Discover(shutdownCtx); err != nil {
			slog.Error(err.Error())
		}
	})
	if err := s.StartServerForDir("C:/Users/usman/Downloads/Programs"); err != nil {
		slog.Error(err.Error())
	}
}
