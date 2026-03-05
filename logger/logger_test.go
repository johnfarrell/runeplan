package loggerd

import (
	"testing"

	"go.uber.org/zap/zapcore"

	"github.com/johnfarrell/runeplan/config"
)

// enabled is a helper that reports whether the logger emits messages at level.
func enabled(log interface{ Core() zapcore.Core }, level zapcore.Level) bool {
	return log.Core().Enabled(level)
}

// TestNew_LevelIsApplied is the primary test: for each valid log level,
// verify that the correct levels are enabled and suppressed on the resulting
// logger. Both development and production presets are exercised.
func TestNew_LevelIsApplied(t *testing.T) {
	tests := []struct {
		level       zapcore.Level
		wantEnabled map[zapcore.Level]bool
	}{
		{
			level: zapcore.DebugLevel,
			wantEnabled: map[zapcore.Level]bool{
				zapcore.DebugLevel: true,
				zapcore.InfoLevel:  true,
				zapcore.WarnLevel:  true,
				zapcore.ErrorLevel: true,
			},
		},
		{
			level: zapcore.InfoLevel,
			wantEnabled: map[zapcore.Level]bool{
				zapcore.DebugLevel: false,
				zapcore.InfoLevel:  true,
				zapcore.WarnLevel:  true,
				zapcore.ErrorLevel: true,
			},
		},
		{
			level: zapcore.WarnLevel,
			wantEnabled: map[zapcore.Level]bool{
				zapcore.DebugLevel: false,
				zapcore.InfoLevel:  false,
				zapcore.WarnLevel:  true,
				zapcore.ErrorLevel: true,
			},
		},
		{
			level: zapcore.ErrorLevel,
			wantEnabled: map[zapcore.Level]bool{
				zapcore.DebugLevel: false,
				zapcore.InfoLevel:  false,
				zapcore.WarnLevel:  false,
				zapcore.ErrorLevel: true,
			},
		},
	}

	envs := []string{"development", "production"}

	for _, env := range envs {
		for _, tt := range tests {
			t.Run(env+"/"+tt.level.String(), func(t *testing.T) {
				cfg := config.AppConfig{Env: env, LogLevel: tt.level}

				log, err := New(cfg)
				if err != nil {
					t.Fatalf("New() error = %v", err)
				}
				defer log.Sync() //nolint:errcheck

				for level, want := range tt.wantEnabled {
					got := enabled(log, level)
					if got != want {
						t.Errorf("env=%q logLevel=%v: Enabled(%v) = %v, want %v",
							env, tt.level, level, got, want)
					}
				}
			})
		}
	}
}

// TestNew_LevelConsistencyAcrossEnvs verifies that the same zapcore.Level
// produces identical filtering behaviour regardless of preset (dev vs prod).
func TestNew_LevelConsistencyAcrossEnvs(t *testing.T) {
	levels := []zapcore.Level{
		zapcore.DebugLevel,
		zapcore.InfoLevel,
		zapcore.WarnLevel,
		zapcore.ErrorLevel,
	}

	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			dev, err := New(config.AppConfig{Env: "development", LogLevel: level})
			if err != nil {
				t.Fatalf("New(development, %v) error = %v", level, err)
			}
			defer dev.Sync() //nolint:errcheck

			prod, err := New(config.AppConfig{Env: "production", LogLevel: level})
			if err != nil {
				t.Fatalf("New(production, %v) error = %v", level, err)
			}
			defer prod.Sync() //nolint:errcheck

			for _, l := range levels {
				devEnabled := enabled(dev, l)
				prodEnabled := enabled(prod, l)
				if devEnabled != prodEnabled {
					t.Errorf("level %v: Enabled(%v) differs between envs: dev=%v prod=%v",
						level, l, devEnabled, prodEnabled)
				}
			}
		})
	}
}

// TestNew_InvalidLevel verifies that zapcore.InvalidLevel is rejected.
func TestNew_InvalidLevel(t *testing.T) {
	cfg := config.AppConfig{Env: "development", LogLevel: zapcore.InvalidLevel}

	_, err := New(cfg)
	if err == nil {
		t.Fatal("New() with InvalidLevel: expected error, got nil")
	}
}
