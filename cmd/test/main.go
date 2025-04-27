package main

import (
	"github.com/MuhamedUsman/letshare/internal/client"
	"github.com/lmittmann/tint"
	"log/slog"
	"os"
)

func init() {
	logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{AddSource: true}))
	slog.SetDefault(logger)
}

func main() {
	cfg, err := client.LoadConfig()
	if err != nil {
		slog.Error(err.Error())
	}
	slog.Info("Loaded Config", "config", cfg)
}
