package main

import (
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/file"
	"github.com/lmittmann/tint"
	"log/slog"
	"os"
)

func init() {
	h := tint.NewHandler(os.Stderr, &tint.Options{AddSource: true})
	slog.SetDefault(slog.New(h))
}

func main() {
	root := "D:\\BSCS Spring 2022\\1st Semester"
	files := []string{"Applied Physics", "Math", "Complete Numericals.pdf"}

	progress := make(chan int64, 5)
	isTotal := true
	var total int64
	go func() {
		for p := range progress {
			if isTotal {
				total = p
				isTotal = false
			} else {
				fmt.Printf("\rWritten: %s/ Total: %s", file.HumanizeSize(p), file.HumanizeSize(total))
			}
		}
	}()

	path, err := file.Zip(progress, "1stSemester.zip", root, files...)
	if err != nil {
		slog.Error(err.Error())
	} else {
		slog.Info("Zipping completed successfully", "path", path)
	}
}
