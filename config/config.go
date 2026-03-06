package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap/zapcore"
)

// Config is the top-level application configuration. Add new subsections here
// as the application grows — each section is a self-contained struct with its
// own defaults and env var bindings below.
type Config struct {
	App      AppConfig
	Server   ServerConfig
	DB       DBConfig
	Auth     AuthConfig
	Discord  DiscordConfig
	Hiscores HiscoresConfig
}

// AppConfig holds general application settings.
type AppConfig struct {
	Env      string        // "development" | "production"
	LogLevel zapcore.Level // parsed from LOG_LEVEL; defaults to zapcore.InfoLevel
}

// ServerConfig holds HTTP server tuning parameters.
type ServerConfig struct {
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// DBConfig holds database connection settings.
type DBConfig struct {
	URL string // full postgres connection string (DATABASE_URL)
}

// AuthConfig holds session cookie and token settings.
type AuthConfig struct {
	CookieName    string
	CookieSecure  bool // set true in production (requires HTTPS)
	SessionMaxAge time.Duration
}

// DiscordConfig holds Discord OAuth 2.0 settings.
// If ClientID is empty, Discord login is disabled.
type DiscordConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// HiscoresConfig holds settings for the OSRS Hiscores proxy.
type HiscoresConfig struct {
	BaseURL          string
	Timeout          time.Duration
	RateLimitSeconds int // minimum seconds between syncs per user
}

// Load reads configuration from environment variables and, optionally, a
// runeplan.config file in $HOME/.runeplan or the working directory.
// Environment variables always take precedence over the config file.
// Returns a populated *Config or an error if required values are missing or
// any value cannot be parsed.
func Load() (*Config, error) {
	v := viper.New()

	// --- Config file (optional) ---
	v.SetConfigName("runeplan.config")
	v.AddConfigPath("$HOME/.runeplan")
	v.AddConfigPath(".")

	// --- Env var support ---
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// --- Defaults ---
	v.SetDefault("app.env", "development")
	v.SetDefault("app.log_level", "info")

	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "10s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.shutdown_timeout", "10s")

	v.SetDefault("auth.cookie_name", "runeplan_session")
	v.SetDefault("auth.cookie_secure", false)
	v.SetDefault("auth.session_max_age", "720h") // 30 days

	v.SetDefault("hiscores.base_url", "https://secure.runescape.com/m=hiscore_oldschool/index_lite.ws")
	v.SetDefault("hiscores.timeout", "10s")
	v.SetDefault("hiscores.rate_limit_seconds", 5)

	// --- Explicit env var bindings ---
	// These map well-known env var names to Viper keys so callers don't need to
	// know the nested key path. Add new bindings here when expanding Config.
	bindings := map[string]string{
		"app.env":               "APP_ENV",
		"app.log_level":         "LOG_LEVEL",
		"server.port":           "PORT",
		"db.url":                "DATABASE_URL",
		"auth.cookie_secure":    "COOKIE_SECURE",
		"discord.client_id":     "DISCORD_CLIENT_ID",
		"discord.client_secret": "DISCORD_CLIENT_SECRET",
		"discord.redirect_url":  "DISCORD_REDIRECT_URL",
	}
	for key, env := range bindings {
		if err := v.BindEnv(key, env); err != nil {
			return nil, fmt.Errorf("config: binding %s to %s: %w", key, env, err)
		}
	}

	// --- Read config file (optional — missing file is not an error) ---
	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("config: reading config file: %w", err)
		}
	}

	// --- Parse log level ---
	logLevel, err := zapcore.ParseLevel(v.GetString("app.log_level"))
	if err != nil {
		return nil, fmt.Errorf("config: app.log_level %q: %w", v.GetString("app.log_level"), err)
	}

	// --- Parse duration fields ---
	sessionMaxAge, err := time.ParseDuration(v.GetString("auth.session_max_age"))
	if err != nil {
		return nil, fmt.Errorf("config: auth.session_max_age %q: %w", v.GetString("auth.session_max_age"), err)
	}

	readTimeout, err := time.ParseDuration(v.GetString("server.read_timeout"))
	if err != nil {
		return nil, fmt.Errorf("config: server.read_timeout %q: %w", v.GetString("server.read_timeout"), err)
	}

	writeTimeout, err := time.ParseDuration(v.GetString("server.write_timeout"))
	if err != nil {
		return nil, fmt.Errorf("config: server.write_timeout %q: %w", v.GetString("server.write_timeout"), err)
	}

	shutdownTimeout, err := time.ParseDuration(v.GetString("server.shutdown_timeout"))
	if err != nil {
		return nil, fmt.Errorf("config: server.shutdown_timeout %q: %w", v.GetString("server.shutdown_timeout"), err)
	}

	hiscoresTimeout, err := time.ParseDuration(v.GetString("hiscores.timeout"))
	if err != nil {
		return nil, fmt.Errorf("config: hiscores.timeout %q: %w", v.GetString("hiscores.timeout"), err)
	}

	cfg := &Config{
		App: AppConfig{
			Env:      v.GetString("app.env"),
			LogLevel: logLevel,
		},
		Server: ServerConfig{
			Port:            v.GetInt("server.port"),
			ReadTimeout:     readTimeout,
			WriteTimeout:    writeTimeout,
			ShutdownTimeout: shutdownTimeout,
		},
		DB: DBConfig{
			URL: v.GetString("db.url"),
		},
		Auth: AuthConfig{
			CookieName:    v.GetString("auth.cookie_name"),
			CookieSecure:  v.GetBool("auth.cookie_secure"),
			SessionMaxAge: sessionMaxAge,
		},
		Discord: DiscordConfig{
			ClientID:     v.GetString("discord.client_id"),
			ClientSecret: v.GetString("discord.client_secret"),
			RedirectURL:  v.GetString("discord.redirect_url"),
		},
		Hiscores: HiscoresConfig{
			BaseURL:          v.GetString("hiscores.base_url"),
			Timeout:          hiscoresTimeout,
			RateLimitSeconds: v.GetInt("hiscores.rate_limit_seconds"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate checks that all required fields are present and valid.
func (c *Config) validate() error {
	if c.DB.URL == "" {
		return fmt.Errorf("config: DATABASE_URL is required")
	}
	if c.App.LogLevel == zapcore.InvalidLevel {
		return fmt.Errorf("config: invalid log level")
	}
	return nil
}

// IsProduction reports whether the app is running in production mode.
func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}

// DiscordEnabled reports whether Discord OAuth is configured.
func (c *Config) DiscordEnabled() bool {
	return c.Discord.ClientID != ""
}
