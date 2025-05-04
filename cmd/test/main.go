package main

import (
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/util/file"
	"github.com/MuhamedUsman/letshare/internal/zipr"
	"github.com/lmittmann/tint"
	"log/slog"
	"os"
	"time"
)

func init() {
	h := tint.NewHandler(os.Stderr, &tint.Options{AddSource: true})
	slog.SetDefault(slog.New(h))
}

func main() {
	root := "D:\\BSCS Spring 2022"
	dirs := []string{"1st Semester", "2nd Semester", "3rd Semester", "4th Semester", "5th Semester", "6th Semester", "7th Semester"}
	progCh := make(chan uint64)

	go func() {
		isFirst := true
		var total uint64
		for i := range progCh {
			if isFirst {
				total = i
				isFirst = false
			} else {
				fmt.Printf("\rProgress: %s/%s ", file.HumanizeSize(i), file.HumanizeSize(total))
			}
		}
		fmt.Println()
	}()

	zipper := zipr.New(progCh, zipr.Deflate)
	tNow := time.Now()
	archive, err := zipper.CreateArchive(os.TempDir(), "Letshare.zip", root, dirs...)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	tAfter := time.Since(tNow)
	zipper.Close()

	slog.Info("Time taken to zip directories: ", "sec", tAfter.Seconds())
	/*for _, a := range archive {
		_ = os.Remove(a)
	}*/
	_ = os.Remove(archive)
}
