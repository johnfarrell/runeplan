package config

import (
	"strings"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
)

const testDBURL = "postgres://runeplan:secret@localhost:5432/runeplan_test"

// TestLoad_Defaults verifies that Load returns sensible defaults when only
// the required DATABASE_URL is set.
func TestLoad_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", testDBURL)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.App.Env != "development" {
		t.Errorf("App.Env = %q, want %q", cfg.App.Env, "development")
	}
	if cfg.App.LogLevel != zapcore.InfoLevel {
		t.Errorf("App.LogLevel = %v, want %v", cfg.App.LogLevel, zapcore.InfoLevel)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 10*time.Second {
		t.Errorf("Server.ReadTimeout = %v, want 10s", cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != 30*time.Second {
		t.Errorf("Server.WriteTimeout = %v, want 30s", cfg.Server.WriteTimeout)
	}
	if cfg.Server.ShutdownTimeout != 10*time.Second {
		t.Errorf("Server.ShutdownTimeout = %v, want 10s", cfg.Server.ShutdownTimeout)
	}
	if cfg.Auth.CookieName != "runeplan_session" {
		t.Errorf("Auth.CookieName = %q, want %q", cfg.Auth.CookieName, "runeplan_session")
	}
	if cfg.Auth.CookieSecure != false {
		t.Errorf("Auth.CookieSecure = %v, want false", cfg.Auth.CookieSecure)
	}
	if cfg.Auth.SessionMaxAge != 720*time.Hour {
		t.Errorf("Auth.SessionMaxAge = %v, want 720h", cfg.Auth.SessionMaxAge)
	}
	if cfg.Hiscores.RateLimitSeconds != 5 {
		t.Errorf("Hiscores.RateLimitSeconds = %d, want 5", cfg.Hiscores.RateLimitSeconds)
	}
	if cfg.Hiscores.Timeout != 10*time.Second {
		t.Errorf("Hiscores.Timeout = %v, want 10s", cfg.Hiscores.Timeout)
	}
	if cfg.Hiscores.BaseURL != "https://secure.runescape.com/m=hiscore_oldschool/index_lite.ws" {
		t.Errorf("Hiscores.BaseURL = %q, unexpected default", cfg.Hiscores.BaseURL)
	}
}

// TestLoad_DatabaseURL verifies that DATABASE_URL is read correctly.
func TestLoad_DatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", testDBURL)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DB.URL != testDBURL {
		t.Errorf("DB.URL = %q, want %q", cfg.DB.URL, testDBURL)
	}
}

// TestLoad_MissingDatabaseURL verifies that Load returns an error when
// DATABASE_URL is not set.
func TestLoad_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for missing DATABASE_URL, got nil")
	}
}

// TestLoad_LogLevelParsing verifies that LOG_LEVEL is parsed into the correct
// zapcore.Level value for each valid input string.
func TestLoad_LogLevelParsing(t *testing.T) {
	tests := []struct {
		input string
		want  zapcore.Level
	}{
		{"debug", zapcore.DebugLevel},
		{"info", zapcore.InfoLevel},
		{"warn", zapcore.WarnLevel},
		{"error", zapcore.ErrorLevel},
		{"DEBUG", zapcore.DebugLevel}, // case-insensitive
		{"INFO", zapcore.InfoLevel},
		{"WARN", zapcore.WarnLevel},
		{"ERROR", zapcore.ErrorLevel},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Setenv("DATABASE_URL", testDBURL)
			t.Setenv("LOG_LEVEL", tt.input)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if cfg.App.LogLevel != tt.want {
				t.Errorf("App.LogLevel = %v, want %v", cfg.App.LogLevel, tt.want)
			}
		})
	}
}

// TestLoad_InvalidLogLevel verifies that an unrecognised LOG_LEVEL string
// causes Load to return an error mentioning the field name.
func TestLoad_InvalidLogLevel(t *testing.T) {
	invalids := []string{"verbose", "trace", "LOUD", "0", "1"}

	for _, level := range invalids {
		t.Run(level, func(t *testing.T) {
			t.Setenv("DATABASE_URL", testDBURL)
			t.Setenv("LOG_LEVEL", level)

			_, err := Load()
			if err == nil {
				t.Fatalf("Load() with LOG_LEVEL=%q: expected error, got nil", level)
			}
			if !strings.Contains(err.Error(), "app.log_level") {
				t.Errorf("error %q does not mention field \"app.log_level\"", err.Error())
			}
		})
	}
}

// TestLoad_EnvOverrides verifies that environment variables override defaults.
func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", testDBURL)
	t.Setenv("APP_ENV", "production")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("PORT", "9090")
	t.Setenv("COOKIE_SECURE", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.App.Env != "production" {
		t.Errorf("App.Env = %q, want %q", cfg.App.Env, "production")
	}
	if cfg.App.LogLevel != zapcore.DebugLevel {
		t.Errorf("App.LogLevel = %v, want %v", cfg.App.LogLevel, zapcore.DebugLevel)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want 9090", cfg.Server.Port)
	}
	if !cfg.Auth.CookieSecure {
		t.Errorf("Auth.CookieSecure = false, want true")
	}
}

// TestLoad_DiscordConfig verifies that Discord OAuth env vars are read.
func TestLoad_DiscordConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", testDBURL)
	t.Setenv("DISCORD_CLIENT_ID", "client123")
	t.Setenv("DISCORD_CLIENT_SECRET", "secret456")
	t.Setenv("DISCORD_REDIRECT_URL", "https://example.com/auth/discord/callback")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Discord.ClientID != "client123" {
		t.Errorf("Discord.ClientID = %q, want %q", cfg.Discord.ClientID, "client123")
	}
	if cfg.Discord.ClientSecret != "secret456" {
		t.Errorf("Discord.ClientSecret = %q, want %q", cfg.Discord.ClientSecret, "secret456")
	}
	if cfg.Discord.RedirectURL != "https://example.com/auth/discord/callback" {
		t.Errorf("Discord.RedirectURL = %q, want %q", cfg.Discord.RedirectURL, "https://example.com/auth/discord/callback")
	}
}

// TestLoad_InvalidDuration verifies that Load returns a descriptive error when
// a duration field cannot be parsed.
func TestLoad_InvalidDuration(t *testing.T) {
	tests := []struct {
		env     string
		value   string
		wantErr string
	}{
		{"AUTH_SESSION_MAX_AGE", "notaduration", "auth.session_max_age"},
		{"SERVER_READ_TIMEOUT", "notaduration", "server.read_timeout"},
		{"SERVER_WRITE_TIMEOUT", "notaduration", "server.write_timeout"},
		{"SERVER_SHUTDOWN_TIMEOUT", "notaduration", "server.shutdown_timeout"},
		{"HISCORES_TIMEOUT", "notaduration", "hiscores.timeout"},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			t.Setenv("DATABASE_URL", testDBURL)
			t.Setenv(tt.env, tt.value)

			_, err := Load()
			if err == nil {
				t.Fatalf("Load() expected error for invalid %s, got nil", tt.env)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not mention field %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// TestIsProduction verifies the IsProduction helper.
func TestIsProduction(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{"production", true},
		{"development", false},
		{"staging", false},
		{"", false},
	}

	for _, tt := range tests {
		cfg := &Config{App: AppConfig{Env: tt.env}}
		if got := cfg.IsProduction(); got != tt.want {
			t.Errorf("IsProduction() with env=%q = %v, want %v", tt.env, got, tt.want)
		}
	}
}

// TestDiscordEnabled verifies the DiscordEnabled helper.
func TestDiscordEnabled(t *testing.T) {
	tests := []struct {
		clientID string
		want     bool
	}{
		{"abc123", true},
		{"", false},
	}

	for _, tt := range tests {
		cfg := &Config{Discord: DiscordConfig{ClientID: tt.clientID}}
		if got := cfg.DiscordEnabled(); got != tt.want {
			t.Errorf("DiscordEnabled() with clientID=%q = %v, want %v", tt.clientID, got, tt.want)
		}
	}
}

// TestLoad_NoDiscordByDefault verifies that Discord is disabled when no env
// vars are set.
func TestLoad_NoDiscordByDefault(t *testing.T) {
	t.Setenv("DATABASE_URL", testDBURL)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DiscordEnabled() {
		t.Error("DiscordEnabled() = true, want false when DISCORD_CLIENT_ID is unset")
	}
}

// TestLoad_IsProductionIntegration verifies the IsProduction helper via Load.
func TestLoad_IsProductionIntegration(t *testing.T) {
	t.Run("production env", func(t *testing.T) {
		t.Setenv("DATABASE_URL", testDBURL)
		t.Setenv("APP_ENV", "production")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if !cfg.IsProduction() {
			t.Error("IsProduction() = false, want true")
		}
	})

	t.Run("development env", func(t *testing.T) {
		t.Setenv("DATABASE_URL", testDBURL)
		t.Setenv("APP_ENV", "development")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.IsProduction() {
			t.Error("IsProduction() = true, want false")
		}
	})
}
