package testlog

import (
	"io"
	"log/slog"
)

var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

// Discard returns a shared logger that drops all output.
func Discard() *slog.Logger {
	return discard
}
