package config

import (
	"fmt"
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
	
	// WhatsApp settings
	WhatsApp WhatsAppConfig `mapstructure:"whatsapp"`
	
	// Database settings
	MongoDB MongoDBConfig `mapstructure:"mongodb"`
	
	// Message Queue settings
	NATS NATSConfig `mapstructure:"nats"`
	
	// Development settings
	Dev DevConfig `mapstructure:"dev"`
}

type WhatsAppConfig struct {
	SessionPath   string        `mapstructure:"session_path"`
	DBLogLevel    string        `mapstructure:"db_log_level"`
	HistorySync   bool          `mapstructure:"history_sync"`
	ConnectTimeout time.Duration `mapstructure:"connect_timeout"`
}

type MongoDBConfig struct {
	URI      string `mapstructure:"uri"`
	Database string `mapstructure:"database"`
}

type NATSConfig struct {
	URLs []string `mapstructure:"urls"`
}

type DevConfig struct {
	EnablePprof  bool `mapstructure:"enable_pprof"`
	EnableMetrics bool `mapstructure:"enable_metrics"`
	QRInTerminal bool `mapstructure:"qr_in_terminal"`
}

// Load loads configuration from environment variables and config files
func Load() (*Config, error) {
	v := viper.New()
	
	// Set defaults
	setDefaults(v)
	
	// Environment variables
	v.SetEnvPrefix("TENNEX_BRIDGE")
	v.AutomaticEnv()
	
	// Config file (optional)
	v.SetConfigName("bridge")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/tennex")
	
	// Read config file if it exists
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
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
	
	// WhatsApp defaults
	v.SetDefault("whatsapp.session_path", "./session")
	v.SetDefault("whatsapp.db_log_level", "WARN")
	v.SetDefault("whatsapp.history_sync", false)
	v.SetDefault("whatsapp.connect_timeout", "30s")
	
	// MongoDB defaults
	v.SetDefault("mongodb.uri", "mongodb://localhost:27017")
	v.SetDefault("mongodb.database", "tennex")
	
	// NATS defaults
	v.SetDefault("nats.urls", []string{"nats://localhost:4222"})
	
	// Dev defaults
	v.SetDefault("dev.enable_pprof", false)
	v.SetDefault("dev.enable_metrics", true)
	v.SetDefault("dev.qr_in_terminal", true)
}

func validate(cfg *Config) error {
	if cfg.HTTPPort < 1 || cfg.HTTPPort > 65535 {
		return fmt.Errorf("invalid http_port: %d", cfg.HTTPPort)
	}
	
	if cfg.MongoDB.URI == "" {
		return fmt.Errorf("mongodb.uri is required")
	}
	
	if cfg.MongoDB.Database == "" {
		return fmt.Errorf("mongodb.database is required")
	}
	
	if len(cfg.NATS.URLs) == 0 {
		return fmt.Errorf("nats.urls cannot be empty")
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
		zap.String("session_path", c.WhatsApp.SessionPath),
		zap.Bool("history_sync", c.WhatsApp.HistorySync),
		zap.String("mongodb_database", c.MongoDB.Database),
		zap.Strings("nats_urls", c.NATS.URLs),
		zap.Bool("dev_mode", c.Dev.EnablePprof || c.Dev.QRInTerminal),
	)
}
