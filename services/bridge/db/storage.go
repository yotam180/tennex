package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Storage struct {
	db *gorm.DB
}

func NewStorage() (*Storage, error) {
	db, err := gorm.Open(postgres.Open(GetConnectionString()), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// No local schema migration needed anymore - using backend for account connections

	return &Storage{db: db}, nil
}

// All account connection methods removed - now using backend gRPC for account management
