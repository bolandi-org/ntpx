package logger

import (
	"log/slog"
	"os"
)

// Init initializes the structured logger
func Init() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	log := slog.New(handler)
	slog.SetDefault(log)
}

// ErrorLog returns a logger specifically for errors, useful if needed explicitly
// Note: Slog default will handle structured errors.
func ErrorLog() *slog.Logger {
	return slog.Default()
}
