package main

import (
	"context"
	"github.com/MuhamedUsman/letshare/internal/mdns"
	"github.com/MuhamedUsman/letshare/internal/server"
	"github.com/MuhamedUsman/letshare/internal/util"
	"log"
	"log/slog"
	"os"
	"time"
)

func main() {
	util.ConfigureSlog(os.Stderr)
	s := server.New()
	m := mdns.New()
	instance := "Letshare"
	// Publishing DNS Entry
	s.BT.Run(func(shutdownCtx context.Context) {
		slog.Info("Publishing Multicast DNS Entry", "instance", instance)
		if err := m.Publish(shutdownCtx, instance, "Sharing Files"); err != nil {
			slog.Error(err.Error())
		}
	})
	go func() {
		if err := s.StartServerForDir("C:/Users/usman/Downloads/Programs"); err != nil {
			slog.Error(err.Error())
		}
	}()
	errCh := m.DiscoverMDNSEntries(10*time.Second, 250*time.Millisecond)
	log.Println(<-errCh)
}
