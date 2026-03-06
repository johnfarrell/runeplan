package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/johnfarrell/runeplan/config"
)

// New builds a *zap.Logger configured from cfg.
//
// In production (cfg.Env == "production") it uses zap's production preset:
// JSON output, no caller/stack info on every line, sampling enabled.
//
// In all other environments it uses the development preset: console-formatted
// output, colored log levels, full caller info, panic on DPanic.
//
// The log level is taken from cfg.LogLevel (a zapcore.Level parsed by the
// config package). Returns an error if the level is zapcore.InvalidLevel or
// if zap's internal config build fails.
func New(cfg config.AppConfig) (*zap.Logger, error) {
	if cfg.LogLevel == zapcore.InvalidLevel {
		return nil, fmt.Errorf("logger: invalid log level")
	}

	if cfg.Env == "production" {
		return newProduction(cfg.LogLevel)
	}
	return newDevelopment(cfg.LogLevel)
}

func newProduction(level zapcore.Level) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(level)
	return cfg.Build()
}

func newDevelopment(level zapcore.Level) (*zap.Logger, error) {
	cfg := zap.NewDevelopmentConfig()
	cfg.Level = zap.NewAtomicLevelAt(level)
	return cfg.Build()
}
