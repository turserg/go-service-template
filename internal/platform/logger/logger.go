package logger

import (
	"log/slog"
	"os"
)

func NewJSON(serviceName string) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: false})
	return slog.New(handler).With("service", serviceName)
}
