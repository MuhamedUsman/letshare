package main

import (
	"context"
	"github.com/MuhamedUsman/letshare/internal/mdns"
	"github.com/MuhamedUsman/letshare/internal/server"
	"github.com/MuhamedUsman/letshare/internal/util"
	"github.com/MuhamedUsman/letshare/internal/util/bgtask"
	"log/slog"
	"os"
)

func main() {
	util.ConfigureSlog(os.Stderr)
	slog.SetLogLoggerLevel(slog.LevelWarn)
	s := server.New()
	m := mdns.Get()
	instance := "letshare"
	// Publishing DNS Entry
	bgtask.Get().Run(func(shutdownCtx context.Context) {
		slog.Info("Publishing Multicast DNS Entry", "instance", instance)
		if err := m.Publish(shutdownCtx, instance, instance, 80); err != nil {
			slog.Error(err.Error())
		}
	})
	/*bgtask.Get().Run(func(shutdownCtx context.Context) {
		slog.Info("Discovering Multicast DNS Entries")
		if err := m.Discover(shutdownCtx); err != nil {
			slog.Error(err.Error())
		}
	})*/
	if err := s.StartServer(); err != nil {
		slog.Error(err.Error())
	}
}
