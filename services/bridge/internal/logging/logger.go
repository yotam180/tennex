package logging

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a new structured logger with the specified level
func NewLogger(level string) (*zap.Logger, error) {
	// Parse log level
	zapLevel, err := parseLogLevel(level)
	if err != nil {
		return nil, err
	}

	// Production-like config with some development-friendly changes
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapLevel)
	config.DisableStacktrace = zapLevel > zapcore.ErrorLevel

	// Use console encoder in development for better readability
	if isDevelopment() {
		config.Encoding = "console"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.DisableCaller = false
	}

	logger, err := config.Build(
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.AddCallerSkip(0),
	)
	if err != nil {
		return nil, err
	}

	return logger, nil
}

// NewContextLogger creates a logger with service context fields
func NewContextLogger(baseLogger *zap.Logger, serviceName, version string) *zap.Logger {
	return baseLogger.With(
		zap.String("service", serviceName),
		zap.String("version", version),
		zap.Int("pid", os.Getpid()),
	)
}

// WhatsAppLogger creates a logger specifically for WhatsApp operations
func WhatsAppLogger(baseLogger *zap.Logger) *zap.Logger {
	return baseLogger.With(
		zap.String("component", "whatsapp"),
	)
}

// HTTPLogger creates a logger for HTTP operations
func HTTPLogger(baseLogger *zap.Logger) *zap.Logger {
	return baseLogger.With(
		zap.String("component", "http"),
	)
}

// DatabaseLogger creates a logger for database operations
func DatabaseLogger(baseLogger *zap.Logger) *zap.Logger {
	return baseLogger.With(
		zap.String("component", "database"),
	)
}

// EventLogger creates a logger for event processing
func EventLogger(baseLogger *zap.Logger) *zap.Logger {
	return baseLogger.With(
		zap.String("component", "events"),
	)
}

func parseLogLevel(level string) (zapcore.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	case "fatal":
		return zapcore.FatalLevel, nil
	default:
		return zapcore.InfoLevel, nil
	}
}

func isDevelopment() bool {
	env := os.Getenv("TENNEX_ENV")
	return env == "" || env == "dev" || env == "development"
}

// LoggedPanic logs a panic with context and re-panics
func LoggedPanic(logger *zap.Logger, msg string, fields ...zap.Field) {
	logger.Fatal(msg, fields...)
	panic(msg) // This won't be reached due to Fatal, but makes intent clear
}

// SafeLog logs at info level but falls back gracefully if logger is nil
func SafeLog(logger *zap.Logger, msg string, fields ...zap.Field) {
	if logger != nil {
		logger.Info(msg, fields...)
	}
}

// SafeLogError logs at error level but falls back gracefully if logger is nil
func SafeLogError(logger *zap.Logger, msg string, err error, fields ...zap.Field) {
	if logger != nil {
		allFields := append(fields, zap.Error(err))
		logger.Error(msg, allFields...)
	}
}
