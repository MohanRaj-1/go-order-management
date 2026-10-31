package logger

import (
	"io"
	"log/slog"
)

func New(output io.Writer, minimumLevel slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(output, &slog.HandlerOptions{
		Level: minimumLevel,
	})

	return slog.New(handler)
}
