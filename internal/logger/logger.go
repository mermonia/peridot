package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/mermonia/peridot/internal/paths"
)

var defaultHandler = NewCustomHandler(os.Stdout,
	&CustomOptions{Level: slog.LevelDebug, HidePC: true})
var defaultLogger = slog.New(defaultHandler)

var defaultLogFile *os.File
var (
	verboseMode bool
	quietMode   bool
)

func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
	if verboseMode {
		fmt.Println(msg)
	}
}

func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
	if !quietMode {
		fmt.Println(msg)
	}
}

func Warn(msg string, args ...any) {
	color.Set(color.FgMagenta)
	defaultLogger.Warn(msg, args...)
	if !quietMode {
		fmt.Fprintln(os.Stderr, msg)
	}
	color.Unset()
}

func Error(msg string, args ...any) {
	color.Set(color.FgRed)
	fmt.Fprintln(os.Stderr, msg)
	defaultLogger.Error(msg, args...)
	color.Unset()
}

func InitFileLogging(dotfilesDir string) error {
	logFilePath := paths.LogFilePath(dotfilesDir)
	if err := os.MkdirAll(filepath.Dir(logFilePath), 0755); err != nil {
		return fmt.Errorf("could not create parent dir: %w", err)
	}

	f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("could not open log file: %w", err)
	}

	defaultLogFile = f
	SetDefaultLoggerWriter(f)
	return nil
}

func CloseDefaultLogFile() error {
	if defaultLogFile != nil {
		err := defaultLogFile.Close()
		defaultLogFile = nil
		return err
	}
	return nil
}

func SetDefaultLoggerLevel(level slog.Level) {
	defaultHandler.opts.Level = level
}

func SetDefaultLoggerWriter(out io.Writer) {
	defaultHandler.out = out
	defaultLogger = slog.New(defaultHandler)
}

func SetQuietMode(enabled bool) {
	quietMode = enabled
}

func SetVerboseMode(enabled bool) {
	verboseMode = enabled
}
