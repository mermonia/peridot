package logger

import (
	"log/slog"
	"os"
)

var defaultHandler = NewCustomHandler(os.Stdout,
	&CustomOptions{Level: slog.LevelInfo, HidePC: true})

var defaultLogger = slog.New(defaultHandler)
