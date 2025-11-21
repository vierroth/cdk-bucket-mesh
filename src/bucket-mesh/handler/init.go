package main

import (
	"log/slog"
	"os"
)

var (
	Logger *slog.Logger
)

func init() {
	Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
}
