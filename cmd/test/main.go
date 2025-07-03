package main

import (
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/client"
	"github.com/dustin/go-humanize"
	"github.com/lmittmann/tint"
	"log/slog"
	"os"
)

func init() {
	h := tint.NewHandler(os.Stderr, &tint.Options{AddSource: true})
	slog.SetDefault(slog.New(h))
}

func main() {
	downloadTo := "C:/Users/usman/Downloads/Letshare.zip"
	ch := make(chan client.Progress, 10)

	go func() {
		for p := range ch {
			d := humanize.Bytes(uint64(p.D))
			t := humanize.Bytes(uint64(p.T))
			s := humanize.Bytes(uint64(p.S))
			fmt.Printf("\rDownloaded: %s/%s\tSpeed: %s/s", d, t, s)
		}
	}()

	dt, err := client.NewDownloadTracker(downloadTo, ch)
	if err != nil {
		slog.Error("initializing download tracker", "err", err.Error())
		return
	}
	defer dt.Close()

	var status int
	if status, err = client.Get().DownloadFile("letshare", 2550687330, dt); err != nil {
		slog.Error("downloading file", "err", err)
		return
	}
	slog.Info("Download completed", "status", status, "file", downloadTo)
}
