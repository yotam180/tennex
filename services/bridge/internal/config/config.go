package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Config holds all configuration for the bridge service
type Config struct {
	// Application settings
	AppName    string `mapstructure:"app_name"`
	AppVersion string `mapstructure:"app_version"`
	LogLevel   string `mapstructure:"log_level"`

	// HTTP Server settings
	HTTPPort int `mapstructure:"http_port"`

	// Database settings
	DatabaseURL string `mapstructure:"database_url"`

	// WhatsApp settings
	WhatsApp WhatsAppConfig `mapstructure:"whatsapp"`

	// Development settings
	Dev DevConfig `mapstructure:"dev"`
}

type WhatsAppConfig struct {
	DBLogLevel     string        `mapstructure:"db_log_level"`
	HistorySync    bool          `mapstructure:"history_sync"`
	ConnectTimeout time.Duration `mapstructure:"connect_timeout"`
}

type DevConfig struct {
	EnablePprof   bool `mapstructure:"enable_pprof"`
	EnableMetrics bool `mapstructure:"enable_metrics"`
	QRInTerminal  bool `mapstructure:"qr_in_terminal"`
}

// Load loads configuration from environment variables and config files
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Environment variables
	v.SetEnvPrefix("TENNEX_BRIDGE")
	// Allow mapping nested keys (e.g., whatsapp.session_path) from env like TENNEX_BRIDGE_WHATSAPP_SESSION_PATH
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Explicitly bind nested environment variables
	v.BindEnv("database_url", "TENNEX_BRIDGE_DATABASE_URL")
	v.BindEnv("whatsapp.db_log_level", "TENNEX_BRIDGE_WHATSAPP_DB_LOG_LEVEL")
	v.BindEnv("whatsapp.history_sync", "TENNEX_BRIDGE_WHATSAPP_HISTORY_SYNC")
	v.BindEnv("dev.qr_in_terminal", "TENNEX_BRIDGE_DEV_QR_IN_TERMINAL")
	v.BindEnv("dev.enable_pprof", "TENNEX_BRIDGE_DEV_ENABLE_PPROF")
	v.BindEnv("dev.enable_metrics", "TENNEX_BRIDGE_DEV_ENABLE_METRICS")
	v.BindEnv("log_level", "TENNEX_BRIDGE_LOG_LEVEL")
	v.BindEnv("http_port", "TENNEX_BRIDGE_HTTP_PORT")

	// Config file (optional)
	v.SetConfigName("bridge")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/tennex")

	// Read config file if it exists (but don't fail if there are issues)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Log warning but continue with environment variables and defaults
			fmt.Printf("Warning: error reading config file: %v. Continuing with environment variables and defaults.\n", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate configuration
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app_name", "tennex-bridge")
	v.SetDefault("app_version", "0.1.0")
	v.SetDefault("log_level", "info")
	v.SetDefault("http_port", 8080)

	// Database defaults
	v.SetDefault("database_url", "postgres://tennex:tennex123@localhost:5432/tennex?sslmode=disable")

	// WhatsApp defaults
	v.SetDefault("whatsapp.db_log_level", "WARN")
	v.SetDefault("whatsapp.history_sync", false)
	v.SetDefault("whatsapp.connect_timeout", "30s")

	// Dev defaults
	v.SetDefault("dev.enable_pprof", false)
	v.SetDefault("dev.enable_metrics", true)
	v.SetDefault("dev.qr_in_terminal", true)
}

func validate(cfg *Config) error {
	if cfg.HTTPPort < 1 || cfg.HTTPPort > 65535 {
		return fmt.Errorf("invalid http_port: %d", cfg.HTTPPort)
	}

	if cfg.DatabaseURL == "" {
		return fmt.Errorf("database_url is required")
	}

	// Validate log level
	switch cfg.LogLevel {
	case "debug", "info", "warn", "error", "fatal":
		// valid
	default:
		return fmt.Errorf("invalid log_level: %s", cfg.LogLevel)
	}

	return nil
}

// LogConfig logs the configuration (with sensitive data masked)
func (c *Config) LogConfig(logger *zap.Logger) {
	logger.Info("Bridge service configuration",
		zap.String("app_name", c.AppName),
		zap.String("app_version", c.AppVersion),
		zap.String("log_level", c.LogLevel),
		zap.Int("http_port", c.HTTPPort),
		zap.String("database_url", c.DatabaseURL),
		zap.Bool("history_sync", c.WhatsApp.HistorySync),
		zap.Bool("dev_mode", c.Dev.EnablePprof || c.Dev.QRInTerminal),
	)
}
