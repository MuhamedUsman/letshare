package main

import (
	"context"
	"github.com/MuhamedUsman/letshare/internal/mdns"
	"github.com/MuhamedUsman/letshare/internal/util/bgtask"
	"github.com/lmittmann/tint"
	"log/slog"
	"os"
)

func init() {
	h := tint.NewHandler(os.Stderr, &tint.Options{AddSource: true})
	slog.SetDefault(slog.New(h))
}

//func main() {
//	root := "D:\\BSCS Spring 2022"
//	dirs := []string{"1st Semester", "2nd Semester", "3rd Semester", "4th Semester", "5th Semester", "6th Semester", "7th Semester"}
//
//	progCh := make(chan uint64)
//	logCh := make(chan string)
//
//	go func() {
//		isFirst := true
//		var total uint64
//	main:
//		for {
//			select {
//			case log, ok := <-logCh:
//				if !ok {
//					break main
//				}
//				fmt.Printf("\rZipping %s", log)
//			case i, ok := <-progCh:
//				if !ok {
//					break main
//				}
//				if isFirst {
//					total = i
//					isFirst = false
//				} else {
//					fmt.Printf("\rProgress: %s/%s ", humanize.Bytes(i), humanize.Bytes(total))
//				}
//			}
//		}
//		fmt.Println()
//	}()
//
//	progCh = nil
//
//	zipper := zipr.Get(progCh, logCh, zipr.Store)
//	tNow := time.Now()
//	archive, err := zipper.CreateArchive(context.Background(), os.TempDir(), "Letshare.zip", root, dirs...)
//	if err != nil {
//		slog.Error(err.Error())
//		return
//	}
//	tAfter := time.Since(tNow)
//	zipper.Close()
//
//	slog.Info("Time taken to zip directories: ", "sec", tAfter.Seconds())
//	/*for _, a := range archive {
//		_ = os.Remove(a)
//	}*/
//	_ = os.Remove(archive)
//}

func main() {
	m := mdns.Get()
	bg := bgtask.Get()

	bg.Run(func(shutdownCtx context.Context) {
		if err := m.Publish(shutdownCtx, mdns.DefaultInstance, mdns.DefaultInstance, "Muhammad Usman", 80); err != nil {
			slog.Error("Error registering mDNS service", "error", err)
			return
		}
	})

	ch := make(chan struct{})
	<-ch

	/*time.Sleep(15 * time.Second)
	if err := bg.Shutdown(5 * time.Second); err != nil {
		slog.Error("Error shutting down background tasks", "error", err)
		return
	}*/
}
