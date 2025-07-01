package main

import (
	"github.com/lmittmann/tint"
	"log/slog"
	"os"
)

func init() {
	h := tint.NewHandler(os.Stderr, &tint.Options{AddSource: true})
	slog.SetDefault(slog.New(h))
}

func main() {
	x := 1
	defer println(x)
	x = 2
}
