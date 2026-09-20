package config

import (
	"log/slog"
	"os"
)

// SetupLogger instala o logger global: texto legível em desenvolvimento,
// JSON estruturado em produção.
func (c *Config) SetupLogger() {
	opts := &slog.HandlerOptions{Level: c.slogLevel()}

	var handler slog.Handler
	if c.IsProduction() {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}

func (c *Config) slogLevel() slog.Level {
	switch c.LogLevel {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
