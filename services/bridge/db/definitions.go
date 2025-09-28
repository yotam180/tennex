package db

import (
	"os"
)

// Database connection defaults
const (
	DefaultDatabaseURL = "postgres://tennex:tennex123@localhost:5432/tennex?sslmode=disable"
)

func GetConnectionString() string {
	// Check for TENNEX_DATABASE_URL environment variable first
	if dbURL := os.Getenv("TENNEX_DATABASE_URL"); dbURL != "" {
		return dbURL
	}

	// Fallback to default connection string
	return DefaultDatabaseURL
}
