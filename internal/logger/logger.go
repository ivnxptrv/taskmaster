package logger

import (
	"log/slog"
	"taskmaster/internal/fileutil"
)

func NewDefaultLogger() (*slog.Logger, error) {
	output, err := fileutil.OpenOutput("./logs.out", false)
	if err != nil {
		return nil, err
	}

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	handler := slog.NewTextHandler(output, opts)
	logger := slog.New(handler)

	logger.Debug("logger object created")

	return logger, nil
}
