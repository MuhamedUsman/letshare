package main

import (
	"context"
	"github.com/MuhamedUsman/letshare/internal/common"
	"github.com/MuhamedUsman/letshare/internal/server"
)

func main() {
	s := server.Server{
		BT: common.NewBackgroundTask(),
	}
	s.BT.Run(func(shutdownCtx context.Context) {
		server.PublishEntry(shutdownCtx, "Letshare", "Sharing Files")
	})
	s.SrvDir("C:/Users/usman/Downloads/Programs")
}
