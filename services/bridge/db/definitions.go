package db

import (
	"fmt"
)

// TODO: Take from environment variables
const (
	DatabaseName     = "tennex"
	DatabaseHost     = "localhost"
	DatabasePort     = "5432"
	DatabaseUser     = "tennex"
	DatabasePassword = "tennex123"
)

func GetConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", DatabaseUser, DatabasePassword, DatabaseHost, DatabasePort, DatabaseName)
}
